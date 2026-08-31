package ocppws

// Default is the global OCPP WebSocket server instance.
// Set by main.go after creation; used by HTTP handlers to send requests to devices.
var Default *Server
