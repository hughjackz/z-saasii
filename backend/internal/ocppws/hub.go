package ocppws

import (
	"fmt"
	"log"
	"sync"

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

// Unregister removes a device from the hub. Connection state is managed
// purely in-memory; the device's operational status in the DB is unchanged.
func (h *DeviceHub) Unregister(deviceID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conn, ok := h.devices[deviceID]; ok {
		close(conn.WriteCh)
		conn.Conn.Close()
		delete(h.devices, deviceID)
		log.Printf("[ocppws] device %s unregistered (offline)", deviceID)
	}
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
