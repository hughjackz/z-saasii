package ocppws

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/mylog"
	"github.com/yourorg/csms-backend/internal/repository"
)

// V16Handler is set by the v16 package (via init or explicit call) to avoid
// a circular import between ocppws and v16.
var V16Handler func(dc *DeviceConnection, call *CallMessage, eventCh chan<- *model.Event)

// V201Handler is set by the v201 package, mirroring V16Handler.
var V201Handler func(dc *DeviceConnection, call *CallMessage, eventCh chan<- *model.Event)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  40960,
	WriteBufferSize: 40960,
	Subprotocols:    []string{"ocpp1.6", "ocpp1.5", "ocpp2.0.1", "ocpp2.1"},
}

// Server handles incoming OCPP WebSocket connections from charge points.
type Server struct {
	hub     *DeviceHub
	addr    string
	eventCh chan<- *model.Event

	mu      sync.Mutex
	pending map[string]chan *rawResponse // msgID → response channel
}

// rawResponse holds a raw CALLRESULT or CALLERROR from a device.
type rawResponse struct {
	msgType int
	msgID   string
	payload json.RawMessage
	errCode string
	errDesc string
}

// NewServer creates a new OCPP WebSocket server.
func NewServer(addr string, hub *DeviceHub, eventCh chan<- *model.Event) *Server {
	return &Server{
		hub:     hub,
		addr:    addr,
		eventCh: eventCh,
		pending: make(map[string]chan *rawResponse),
	}
}

// Start begins listening for OCPP WebSocket connections on the configured address.
func (s *Server) Start() {
	// Use a plain handler (not ServeMux) to avoid Go's automatic redirect
	// on URLs containing // (e.g. old-format /csocpp16/kk//device).
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean double slashes etc. without redirecting
		r.URL.Path = cleanPath(r.URL.Path)
		switch {
		case strings.HasPrefix(r.URL.Path, "/csocpp16"):
			s.handleV16(w, r)
		case strings.HasPrefix(r.URL.Path, "/csocpp201"):
			s.handleV201(w, r)
		case strings.HasPrefix(r.URL.Path, "/csocpp21"):
			s.handleV21(w, r)
		default:
			http.NotFound(w, r)
		}
	})

	go s.monitorLoop()

	go func() {
		log.Printf("[ocppws] listening on %s", s.addr)
		if err := http.ListenAndServe(s.addr, handler); err != nil {
			log.Fatalf("[ocppws] fatal: %v", err)
		}
	}()
}

// monitorLoop periodically checks connected devices and drops connections
// that have been silent longer than 2×heartbeat_interval (README 4.1).
func (s *Server) monitorLoop() {
	const monitorInterval = 10 * time.Second
	t := time.NewTicker(monitorInterval)
	defer t.Stop()
	for range t.C {
		for _, id := range s.hub.ConnectedDevices() {
			dc := s.hub.Get(id)
			if dc == nil {
				continue
			}
			timeout := time.Duration(dc.HeartbeatInterval) * 2 * time.Second
			if time.Since(dc.LastSeenAt()) > timeout {
				log.Printf("[ocppws] heartbeat timeout for %s (silent %v > %v)", dc.DeviceName, time.Since(dc.LastSeenAt()), timeout)
				s.hub.Unregister(dc) // closes Conn+WriteCh, removes from map
				_ = repository.UpdateDeviceStatus(dc.DeviceName, "Offline")
				s.emitEvent("error", dc.TenantID, dc.DeviceName, "Heartbeat timeout — device marked offline")
			}
		}
	}
}

// disconnect tears down a connection that is no longer live and, if it was
// still the registered connection, marks the device offline in the DB.
// A stale connection (superseded by a re-register) is ignored.
func (s *Server) disconnect(dc *DeviceConnection) {
	if !s.hub.Unregister(dc) {
		return
	}
	_ = repository.UpdateDeviceStatus(dc.DeviceName, "Offline")
	s.emitEvent("warning", dc.TenantID, dc.DeviceName, "Device disconnected — marked offline")
}

// cleanPath returns the canonical path, collapsing // and removing . and .. elements.
// Unlike http.ServeMux, this does NOT redirect — it cleans internally.
func cleanPath(p string) string {
	if p == "" {
		return "/"
	}
	if p[0] != '/' {
		p = "/" + p
	}
	// Remove double slashes
	for strings.Contains(p, "//") {
		p = strings.ReplaceAll(p, "//", "/")
	}
	return p
}

// ─── Version dispatchers ─────────────────────────────────────────────────────

func (s *Server) handleV16(w http.ResponseWriter, r *http.Request) {
	s.handleConnection(w, r, "16")
}

func (s *Server) handleV201(w http.ResponseWriter, r *http.Request) {
	s.handleConnection(w, r, "201")
}

func (s *Server) handleV21(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "OCPP 2.1 not yet supported", http.StatusNotImplemented)
}

// ─── Connection handling ─────────────────────────────────────────────────────

// parseDevicePath extracts cpop_name, device_name from the URL path.
// URL format: /csocpp{version}/{CP_OP.name}/{device.name}
func parseDevicePath(path string) (cpopName, deviceName string, ok bool) {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return "", "", false
	}
	if !strings.HasPrefix(parts[0], "csocpp") {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request, ocppVersion string) {
	cpopName, deviceName, ok := parseDevicePath(r.URL.Path)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	device, err := repository.GetDeviceByName(deviceName)
	if err != nil {
		log.Printf("[ocppws] device %q not found: %v", deviceName, err)
		http.Error(w, "device not found", http.StatusNotFound)
		return
	}

	if !device.Enabled {
		log.Printf("[ocppws] device %q is disabled, rejecting connection", deviceName)
		http.Error(w, "device disabled", http.StatusForbidden)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ocppws] upgrade error for %q: %v", deviceName, err)
		return
	}

	hbInterval := device.HeartbeatInterval
	if hbInterval <= 0 {
		hbInterval = 60
	}

	dc := &DeviceConnection{
		DeviceID:          device.ID,
		DeviceName:        device.Name,
		TenantID:          device.TenantID,
		CPOPName:          cpopName,
		Version:           ocppVersion,
		Conn:              conn,
		WriteCh:           make(chan []byte, 256),
		HeartbeatInterval: hbInterval,
	}
	dc.Touch()

	s.hub.Register(dc)

	// Push the device-reported status from the DB to the frontend (README 4.1).
	s.emitEvent("info", dc.TenantID, dc.DeviceName, "Device connected (status="+device.Status+")")

	go s.writePump(dc)
	go s.readPump(dc)
}

func (s *Server) writePump(dc *DeviceConnection) {
	defer s.disconnect(dc)
	for msg := range dc.WriteCh {
		log.Printf("[ocppws] >> %s %s", dc.DeviceName, truncate(msg, 500))
		mylog.Device(dc.DeviceName, ">>", string(msg))
		if err := dc.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("[ocppws] write error for %s: %v", dc.DeviceName, err)
			return
		}
	}
}

func (s *Server) readPump(dc *DeviceConnection) {
	defer s.disconnect(dc)
	for {
		_, msg, err := dc.Conn.ReadMessage()
		if err == nil {
			log.Printf("[ocppws] << %s %s", dc.DeviceName, truncate(msg, 500))
			mylog.Device(dc.DeviceName, "<<", string(msg))
		}
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ocppws] read error for %s: %v", dc.DeviceName, err)
			}
			return
		}

		msgType, msgID, payload, errCode, errDesc, err := ParseResult(msg)
		if err != nil {
			call, callErr := ParseCall(msg)
			if callErr != nil {
				log.Printf("[ocppws] parse error from %s: %v (raw: %s)", dc.DeviceName, err, string(msg[:min(len(msg), 200)]))
				continue
			}
			dc.Touch()
			s.dispatchCall(dc, call)
			continue
		}

		dc.Touch()

		s.mu.Lock()
		ch, ok := s.pending[msgID]
		s.mu.Unlock()
		if ok {
			ch <- &rawResponse{
				msgType: msgType,
				msgID:   msgID,
				payload: payload,
				errCode: errCode,
				errDesc: errDesc,
			}
		}
	}
}

func (s *Server) dispatchCall(dc *DeviceConnection, call *CallMessage) {
	switch dc.Version {
	case "16":
		if V16Handler != nil {
			V16Handler(dc, call, s.eventCh)
		}
	case "201":
		if V201Handler != nil {
			V201Handler(dc, call, s.eventCh)
		}
	default:
		s.sendError(dc, call.MsgID, "NotSupported", "protocol version not supported")
	}
}

// IsConnected returns true if the device is currently connected via WebSocket.
func (s *Server) IsConnected(deviceID string) bool {
	return s.hub.Get(deviceID) != nil
}

// ─── Server-initiated request ────────────────────────────────────────────────

func (s *Server) SendRequest(deviceID, action string, payload interface{}) (interface{}, error) {
	dc := s.hub.Get(deviceID)
	if dc == nil {
		return nil, fmt.Errorf("device %s not connected", deviceID)
	}

	msgID := uuid.New().String()
	callMsg, err := BuildCall(msgID, action, payload)
	if err != nil {
		return nil, fmt.Errorf("build call: %w", err)
	}

	ch := make(chan *rawResponse, 1)
	s.mu.Lock()
	s.pending[msgID] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, msgID)
		s.mu.Unlock()
	}()

	if err := s.hub.SendToDevice(deviceID, callMsg); err != nil {
		return nil, err
	}

	select {
	case resp := <-ch:
		if resp.msgType == CallError {
			return nil, fmt.Errorf("%s: %s", resp.errCode, resp.errDesc)
		}
		var result interface{}
		if len(resp.payload) > 0 {
			if err := json.Unmarshal(resp.payload, &result); err != nil {
				return nil, fmt.Errorf("unmarshal response: %w", err)
			}
		}
		return result, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for %s response", action)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func (s *Server) sendError(dc *DeviceConnection, msgID, code, desc string) {
	errMsg, _ := BuildCallError(msgID, code, desc, nil)
	dc.WriteCh <- errMsg
}

// emitEvent persists an event to the event log and fans it out to frontend
// WebSocket clients (non-blocking).
func (s *Server) emitEvent(level, tenantID, device, message string) {
	ev := &model.Event{
		Time:    time.Now().UTC().Format(time.RFC3339),
		Level:   level,
		Device:  device,
		Message: message,
	}
	repository.SaveEvent(ev, tenantID)
	select {
	case s.eventCh <- ev:
	default:
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(b []byte, maxLen int) string {
	s := string(b)
	if len(s) <= maxLen {
		return s
	}
	return s
	//return s[:maxLen] + "..."
}
