package ocppws

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// DeviceConnection represents a connected charge point over WebSocket.
type DeviceConnection struct {
	DeviceID   string // database device.id
	DeviceName string // database device.name
	TenantID   string // tenant_id from device record
	CPOPName   string // CP_OP username from URL path
	Version    string // "16", "201", "21"
	Conn       *websocket.Conn
	WriteCh    chan []byte

	HeartbeatInterval int // seconds; from device record at connect time (fallback 60)

	lastSeenMu sync.Mutex
	lastSeen   time.Time // updated on ANY inbound frame (readPump)
}

// Touch records that an inbound frame was just received from the device.
func (dc *DeviceConnection) Touch() {
	dc.lastSeenMu.Lock()
	dc.lastSeen = time.Now()
	dc.lastSeenMu.Unlock()
}

// LastSeenAt returns when the last inbound frame was received.
func (dc *DeviceConnection) LastSeenAt() time.Time {
	dc.lastSeenMu.Lock()
	defer dc.lastSeenMu.Unlock()
	return dc.lastSeen
}

// DeviceHub manages all connected charge points.
type DeviceHub struct {
	mu      sync.RWMutex
	devices map[string]*DeviceConnection // keyed by device.id
}

// NewDeviceHub creates a new DeviceHub.
func NewDeviceHub() *DeviceHub {
	return &DeviceHub{
		devices: make(map[string]*DeviceConnection),
	}
}

// Register adds a connected device to the hub.
func (h *DeviceHub) Register(conn *DeviceConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	// Close any existing connection for the same device
	if old, ok := h.devices[conn.DeviceID]; ok {
		close(old.WriteCh)
		old.Conn.Close()
	}
	h.devices[conn.DeviceID] = conn
	log.Printf("[ocppws] device %s registered (id=%s, version=%s)", conn.DeviceName, conn.DeviceID, conn.Version)
}

// Unregister removes a device connection from the hub.
// Returns true only if the given connection was still the registered one;
// a stale connection (superseded by a re-register) is a no-op returning false.
// Callers use the return value to decide whether to mark the device offline.
func (h *DeviceHub) Unregister(conn *DeviceConnection) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cur, ok := h.devices[conn.DeviceID]; ok && cur == conn {
		close(conn.WriteCh)
		conn.Conn.Close()
		delete(h.devices, conn.DeviceID)
		log.Printf("[ocppws] device %s unregistered (offline)", conn.DeviceID)
		return true
	}
	return false
}

// Get returns the connection for a device, or nil if not connected.
func (h *DeviceHub) Get(deviceID string) *DeviceConnection {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.devices[deviceID]
}

// SendToDevice writes a message to a connected device's write channel.
func (h *DeviceHub) SendToDevice(deviceID string, msg []byte) error {
	conn := h.Get(deviceID)
	if conn == nil {
		return fmt.Errorf("device %s not connected", deviceID)
	}
	select {
	case conn.WriteCh <- msg:
		return nil
	default:
		return fmt.Errorf("device %s write buffer full", deviceID)
	}
}

// ConnectedDevices returns a list of currently connected device IDs.
func (h *DeviceHub) ConnectedDevices() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.devices))
	for id := range h.devices {
		ids = append(ids, id)
	}
	return ids
}
