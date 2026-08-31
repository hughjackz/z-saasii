package vdv261

// server.go — VDV261 HTTPS server (README 6.3).
// Handles EVCC communication with HTTP Basic Auth + JSON request/response.
// Uses config.Cfg.VDV261 for certificates, network mode, and listen address.

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/yourorg/csms-backend/config"
	"github.com/yourorg/csms-backend/internal/repository"
)

// Start begins the VDV261 HTTPS service if enabled in config.
func Start() {
	cfg := config.Cfg.VDV261
	if !cfg.Enable {
		log.Println("[vdv261] disabled in config")
		return
	}

	// Check cert files exist
	if _, err := os.Stat(cfg.CertFile); err != nil {
		log.Printf("[vdv261] cert file not found: %s — service not started", cfg.CertFile)
		return
	}
	if _, err := os.Stat(cfg.KeyFile); err != nil {
		log.Printf("[vdv261] key file not found: %s — service not started", cfg.KeyFile)
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc(cfg.URLPath, handleVDV)

	// Configure TLS
	tlsCfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if cfg.RootCA != "" {
		// Optional: add client CA
	}

	server := &http.Server{
		Addr:      cfg.ListenAddr,
		Handler:   mux,
		TLSConfig: tlsCfg,
	}

	// Start with network mode awareness
	go func() {
		ln, err := listenWithMode(cfg.NetworkMode, cfg.ListenAddr)
		if err != nil {
			log.Printf("[vdv261] listen error: %v", err)
			return
		}
		log.Printf("[vdv261] HTTPS serving on %s (mode=%s, path=%s)", cfg.ListenAddr, cfg.NetworkMode, cfg.URLPath)
		if err := server.ServeTLS(ln, cfg.CertFile, cfg.KeyFile); err != nil && err != http.ErrServerClosed {
			log.Printf("[vdv261] serve error: %v", err)
		}
	}()
}

// listenWithMode creates a listener for the given network mode.
func listenWithMode(mode, addr string) (net.Listener, error) {
	switch mode {
	case "ipv4":
		return net.Listen("tcp4", addr)
	case "ipv6":
		return net.Listen("tcp6", addr)
	case "dual":
		return net.Listen("tcp", addr)
	default:
		return net.Listen("tcp", addr)
	}
}

// handleVDV processes EVCC POST requests (6.3.3 / 6.3.4).
func handleVDV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 6.3.1 HTTP Basic Auth
	vin, ok := checkBasicAuth(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="VDV261"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request body
	bodyBytes, _ := io.ReadAll(r.Body)
	log.Printf("[vdv261] request from %s (vin=%s): %s", r.RemoteAddr, vin, string(bodyBytes))
	var req vdvRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		log.Printf("[vdv261] bad request from %s: %v", r.RemoteAddr, err)
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update car info (evccid, odo) from the request
	car, err := repository.GetVDVCarInfoByVIN(vin)
	if err == nil {
		fields := map[string]interface{}{}
		if req.EVCCID != "" {
			fields["evccid"] = req.EVCCID
		}
		fields["odo"] = req.Odo
		_ = repository.UpdateVDVCarInfo(car.ID, fields)
	}

	// 6.3.4 Look up VDVProfile for this VIN → build response
	var resp vdvResponse
	resp.Seq = req.Seq
	resp.VIN = vin

	// Defaults
	resp.DriveOff = 0
	resp.PrecDsrd = 0
	resp.PrecHvac = 0
	resp.AmbientTemp = 22

	if car != nil && car.VDVProfileID != nil {
		profile, err := repository.GetVDVProfile(*car.VDVProfileID)
		if err == nil {
			// Map VDVProfile to VDV261 response
			resp.DriveOff = parseTimeToMinutes(profile.DriveOff)
			resp.PrecDsrd = profile.PrecDsrd
			resp.PrecHvac = profile.PrecHvac
			resp.AmbientTemp = profile.AmbientTemp
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// checkBasicAuth validates HTTP Basic Auth credentials against the VDVCarInfo table.
func checkBasicAuth(r *http.Request) (vin string, ok bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" || !strings.HasPrefix(auth, "Basic ") {
		return "", false
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return "", false
	}
	parts := strings.SplitN(string(payload), ":", 2)
	if len(parts) != 2 {
		return "", false
	}
	vin, password := parts[0], parts[1]
	car, err := repository.GetVDVCarInfoByVIN(vin)
	if err != nil {
		log.Printf("[vdv261] auth: VIN %q not found in DB", vin)
		return "", false
	}
	if car.Password != password {
		log.Printf("[vdv261] auth: VIN %q password mismatch", vin)
		return "", false
	}
	return vin, true
}

// ─── VDV261 JSON types ──────────────────────────────────────────────────────

type vdvRequest struct {
	Seq          int    `json:"seq"`
	VIN          string `json:"vin"`
	EVCCID       string `json:"evccid"`
	Odo          int    `json:"odo"`
	BatReqTime   int    `json:"bat_reqtime"`
	BatEAmount   int    `json:"bat_eamount"`
	PrecEAmount  int    `json:"prec_eamount"`
	PrecReqTime  int    `json:"prec_reqtime"`
	ChrgStat     int    `json:"chrg_stat"`
	DataNum      string `json:"data_num"`
}

type vdvResponse struct {
	Seq         int    `json:"seq"`
	VIN         string `json:"vin"`
	DriveOff    int    `json:"driveoff"`
	PrecDsrd    int    `json:"prec_dsrd"`
	PrecHvac    int    `json:"prec_hvac"`
	AmbientTemp int    `json:"ambienttemp"`
}

func parseTimeToMinutes(t string) int {
	var h, m int
	if n, _ := fmt.Sscanf(t, "%d:%d", &h, &m); n >= 2 {
		return h*60 + m
	}
	return 0
}
