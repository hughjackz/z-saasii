package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/config"
	"github.com/yourorg/csms-backend/internal/handler"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/mylog"
	"github.com/yourorg/csms-backend/internal/ocppws"
	_ "github.com/yourorg/csms-backend/internal/ocppws/v16"  // register V16Handler
	_ "github.com/yourorg/csms-backend/internal/ocppws/v201" // register V201Handler
	"github.com/yourorg/csms-backend/internal/repository"
	"github.com/yourorg/csms-backend/internal/vdv261"
	ws "github.com/yourorg/csms-backend/internal/websocket"
)

func main() {
	// 1. Load config
	config.Load()
	gin.SetMode(config.Cfg.Server.Mode)

	// 2. Connect to database
	repository.Connect()

	// 2b. Bridge mylog file logger → event_log DB table.
	// All logs written via mylog (HTTP, OCPP, VDV, errors) are also persisted
	// to the database so the frontend Events view can query them.
	mylog.DBHook = func(tenantID, level, device, message string) {
		ev := &model.Event{
			Time:    time.Now().UTC().Format(time.RFC3339),
			Level:   level,
			Device:  device,
			Message: message,
		}
		repository.SaveEvent(ev, tenantID)
	}

	// 3. Create event channel (buffers events between OCPP WS server and WS hub)
	eventCh := make(chan *model.Event, 512)

	// 4. Start OCPP WebSocket server (replaces TCP client)
	deviceHub := ocppws.NewDeviceHub()
	ocppwsServer := ocppws.NewServer(config.Cfg.OCPP.WSAddr, deviceHub, eventCh)
	ocppws.Default = ocppwsServer
	ocppwsServer.Start()

	// 4b. Start VDV261 HTTPS service (EVCC communication)
	vdv261.Start()

	// 5. Start WebSocket hub (frontend event broadcast)
	hub := ws.NewHub(eventCh)
	go hub.Run()

	// 6. Build HTTP router
	r := handler.NewRouter(hub)

	// 7. Start HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:         ":" + config.Cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[server] listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] fatal: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[server] shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[server] forced shutdown: %v", err)
	}
	log.Println("[server] stopped")
}
