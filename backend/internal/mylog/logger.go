package mylog

// mylog — Daily log system, separated by CP_OP tenant.
// Log files: backend/mylog/{tenantID}_{YYYY-MM-DD}.log
// A special "system" tenant ID is used for non-tenant-scoped events.

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// DBHook is an optional callback that receives every log entry for DB persistence.
// Set from main.go after DB is connected. Parameters match event_log columns.
var DBHook func(tenantID, level, device, message string)

var (
	mu      sync.Mutex
	writers = make(map[string]*os.File)
	baseDir = "mylog"
)

// Write logs a line to the daily file for the given tenant.
// If tenantID is empty, uses "system".
// Also forwards to DBHook (if set) with level="info", device="".
func Write(tenantID, format string, args ...interface{}) {
	writeEntry(tenantID, "info", "", format, args...)
}

// WriteEvent logs to file AND forwards to DBHook with explicit level and device.
func WriteEvent(tenantID, level, device, format string, args ...interface{}) {
	writeEntry(tenantID, level, device, format, args...)
}

func writeEntry(tenantID, level, device, format string, args ...interface{}) {
	if tenantID == "" {
		tenantID = "system"
	}
	message := fmt.Sprintf(format, args...)
	today := time.Now().Format("2006-01-02")
	key := tenantID + "_" + today

	mu.Lock()
	w, ok := writers[key]
	if !ok {
		_ = os.MkdirAll(baseDir, 0755)
		path := filepath.Join(baseDir, fmt.Sprintf("%s_%s.log", tenantID, today))
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			mu.Unlock()
			return
		}
		// Close old writers from previous days
		for k, old := range writers {
			if k != key {
				old.Close()
				delete(writers, k)
			}
		}
		w = f
		writers[key] = f
	}
	mu.Unlock()

	line := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), message)
	w.WriteString(line)

	// Persist to DB via hook (set from main.go after DB is connected)
	if DBHook != nil {
		DBHook(tenantID, level, device, message)
	}
}

// HTTP logs an HTTP request.
func HTTP(userID, role, method, path, body string) {
	WriteEvent(userID, "info", "", "[HTTP] %s %s %s %s", role, method, path, truncateStr(body, 200))
}

// Device logs OCPP device communication.
func Device(deviceName, direction, data string) {
	WriteEvent("", "info", deviceName, "[DEVICE] %s %s", direction, truncateStr(data, 500))
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Flush closes all open log writers.
func Flush() {
	mu.Lock()
	defer mu.Unlock()
	for _, w := range writers {
		w.Close()
	}
	writers = make(map[string]*os.File)
}
