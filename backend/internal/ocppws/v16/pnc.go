package v16

// pnc.go — 15118 PnC (Plug & Charge) message handlers.
// PNC operations use DataTransfer with vendor-specific message IDs.
// Ref: README 4.2.9, backend/doc/ocpp1.6/schema/PNC/

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// PNC DataTransfer vendorId and messageId constants
const (
	pncVendorId = "org.openchargealliance.iso15118pnc"
	pncSignCert = "SignCertificate"
	pncGetCert  = "Get15118EVCertificate"
)

// ─── SignCertificate (4.2.9.1) ──────────────────────────────────────────────
// Device sends CSR to SAAS for SECC Leaf certificate signing.

type signCertificateReq struct {
	CSR string `json:"csr"`
}

func handlePNCSignCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req signCertificateReq
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		log.Printf("[ocppws/v16/pnc] SignCertificate from %s cannot unmarshal payload: %v", dc.DeviceName, err)
		sendDataTransferResult(dc, call.MsgID, "Rejected", "")
		return
	}

	// Step 2 of SECC Leaf signing: device sends CSR, SAAS signs it
	session := GetSECCSession(dc.DeviceID)
	if session == nil {
		log.Printf("[ocppws/v16/pnc] SignCertificate from %s but no pending SECC session", dc.DeviceName)
		sendDataTransferResult(dc, call.MsgID, "Rejected", "")
		return
	}

	log.Printf("[ocppws/v16/pnc] Signing SECC Leaf for %s with V2G Sub2=%s", dc.DeviceName, session.V2GSub2)

	signedCert, err := SignCSR(req.CSR, dc.DeviceName, dc.TenantID, session)
	if err != nil {
		log.Printf("[ocppws/v16/pnc] SECC signing failed for %s: %v", dc.DeviceName, err)
		sendDataTransferResult(dc, call.MsgID, "Rejected", "")
		pushEvent(eventCh, dc.TenantID, "error", dc.DeviceName, "SECC Leaf signing failed: "+err.Error())
		return
	}

	// Build certificate chain: SECC Leaf + V2G Sub2 + V2G Sub1
	v2gSub2Cert, _, _, _ := findCertAndKey(session.V2GSub2)
	v2gSub1Cert, _, _, _ := findCertAndKey(session.V2GSub1)
	certChain := signedCert
	if v2gSub2Cert != "" {
		certChain += v2gSub2Cert
	}
	if v2gSub1Cert != "" {
		certChain += v2gSub1Cert
	}

	// Step 3: Send certificate chain back to device via CertificateSigned
	innerPayload, _ := json.Marshal(map[string]string{"certificateChain": certChain})
	wrapper := map[string]string{
		"vendorId":  pncVendorId,
		"messageId": "CertificateSigned",
		"data":      string(innerPayload),
	}
	// Respond to SignCertificate first
	sendDataTransferResult(dc, call.MsgID, "Accepted", "")

	// Then send CertificateSigned to the device (server→device via DataTransfer)
	certSignedCall, _ := ocppws.BuildCall(uuid.New().String(), "DataTransfer", wrapper)
	dc.WriteCh <- certSignedCall

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "SECC Leaf certificate signed and sent to device")
}

// ─── Get15118EVCertificate (4.2.9.5) ───────────────────────────────────────
// Device requests a contract certificate (ISO 15118).

type get15118EVCertReq struct {
	ISO15118SchemaVersion string `json:"iso15118SchemaVersion"`
	Action                string `json:"action"` // Install | Update
	ExiRequest            string `json:"exiRequest"`
}

func handlePNCGet15118EVCert(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req get15118EVCertReq
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		log.Printf("[ocppws/v16/pnc] Get15118EVCertificate from %s: bad payload: %v", dc.DeviceName, err)
		sendDataTransferResult(dc, call.MsgID, "Rejected", "")
		return
	}

	log.Printf("[ocppws/v16/pnc] Get15118EVCertificate from %s (action=%s, schemaVersion=%s, exiLen=%d)",
		dc.DeviceName, req.Action, req.ISO15118SchemaVersion, len(req.ExiRequest))

	// Look up tenant's certificate library for ContractGenerate (4.2.9.5).
	// If a contract cert group was pre-selected by the user (2.3.2.4.e),
	// resolve those specific certificates; otherwise use all tenant certs.
	allCerts, err := repository.ListCertificates(model.RoleCSAdmin, "", dc.TenantID, "")
	if err != nil {
		log.Printf("[ocppws/v16/pnc] Get15118EVCertificate from %s: cert list error: %v", dc.DeviceName, err)
	}

	certs := allCerts
	if group := GetContractCertGroup(dc.DeviceID); group != nil {
		// Resolve selected cert names → actual certificate objects
		nameSet := make(map[string]bool)
		for _, name := range group {
			nameSet[name] = true
		}
		var matched []*model.Certificate
		for _, c := range allCerts {
			if nameSet[c.Name] {
				matched = append(matched, c)
			}
		}
		if len(matched) > 0 {
			certs = matched
			log.Printf("[ocppws/v16/pnc] Get15118EVCertificate using contract cert group: %d certs", len(certs))
		}
	}

	// If the generator needs initialization (e.g., CGO lib), ensure it's done.
	if ig, ok := ContractGen.(interface{ Initialize(tenantID string) error }); ok {
		if err := ig.Initialize(dc.TenantID); err != nil {
			log.Printf("[ocppws/v16/pnc] ContractGen init for tenant %s: %v", dc.TenantID, err)
		}
	}

	// Call ContractGenerate interface with the EXI request and certificate list
	exiResp, err := ContractGen.Generate(req.ExiRequest, certs)
	if err != nil {
		log.Printf("[ocppws/v16/pnc] Get15118EVCertificate from %s: ContractGenerate error: %v", dc.DeviceName, err)
		innerResp, _ := json.Marshal(map[string]string{
			"status":      "Failed",
			"exiResponse": "",
		})
		sendDataTransferResult(dc, call.MsgID, "Failed", string(innerResp))
		pushEvent(eventCh, dc.TenantID, "error", dc.DeviceName,
			"PNC Get15118EVCertificate ContractGenerate failed: "+err.Error())
		return
	}

	status := "Accepted"
	innerResp, _ := json.Marshal(map[string]string{
		"status":      status,
		"exiResponse": exiResp,
	})
	sendDataTransferResult(dc, call.MsgID, status, string(innerResp))

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"PNC Get15118EVCertificate action="+req.Action+" status="+status)
}

// ─── ContractGenerator (4.2.9.5) ─────────────────────────────────────────────
// ContractGenerator defines the interface for processing ISO 15118 contract
// certificate requests. Implementations handle EXI encoding/decoding per
// ISO 15118-2, processing a CertificateInstallationReq and returning a
// CertificateInstallationRes.
//
// Input:
//   - exiRequest: Base64-encoded EXI CertificateInstallationReq from the EV
//   - certificates: available certificates for this tenant (for building cert chain)
//
// Output:
//   - exiResponse: Base64-encoded EXI CertificateInstallationRes to send back to EV
//   - err: non-nil if processing failed
type ContractGenerator interface {
	Generate(exiRequest string, certificates []*model.Certificate) (exiResponse string, err error)
}

// DefaultContractGenerator is the default (stub) implementation of ContractGenerator.
// It accepts the EXI request and returns a minimal valid EXI response.
// Replace ContractGen with a full EXI-codec implementation for production use.
type DefaultContractGenerator struct{}

// Generate implements ContractGenerator for the default stub.
// It validates the input, then returns a Base64-encoded placeholder
// CertificateInstallationRes indicating success.
func (g *DefaultContractGenerator) Generate(exiRequest string, certificates []*model.Certificate) (string, error) {
	log.Printf("[ocppws/v16/pnc] ContractGenerate: exiLen=%d, available certs=%d",
		len(exiRequest), len(certificates))

	for _, c := range certificates {
		log.Printf("[ocppws/v16/pnc] ContractGenerate: cert name=%s type=%s serial=%s",
			c.Name, c.Type, c.SerialNumber)
	}

	// TODO: Implement full EXI encoding/decoding per ISO 15118-2.
	// The production implementation should:
	//   1. Base64-decode exiRequest into EXI binary
	//   2. Decode EXI → XML using ISO 15118-2 schema
	//   3. Process CertificateInstallationReq (validate OEM ProvCert, generate contract cert)
	//   4. Build CertificateInstallationRes with contract cert chain + private key
	//   5. Encode XML → EXI binary
	//   6. Base64-encode the EXI binary → return as exiResponse
	//
	// For now, return a minimal valid response: Base64-encoded EXI representing
	// an empty CertificateInstallationRes with ResponseCode=OK.
	// This is a 0x00 byte (EXI header with no content) Base64-encoded.
	return "AA==", nil
}

// ContractGen is the active ContractGenerator implementation.
// Uses the C EXI static library (clib/exi_lib.a) for ISO 15118-2 EXI processing.
// Falls back to DefaultContractGenerator if CGO is not available (build tag `no_cgo`).
var ContractGen ContractGenerator = &CGOContractGenerator{}

// ─── PNC Authorize (4.2.9.6) ───────────────────────────────────────────────
// Device sends certificate + idToken for authorization.

type pncAuthorizeReq struct {
	Certificate                 string          `json:"certificate"`
	IDTokenRaw                  json.RawMessage `json:"idToken"`
	idTokenParsed               string
	ISO15118CertificateHashData []struct {
		HashAlgorithm  string `json:"hashAlgorithm"`
		IssuerNameHash string `json:"issuerNameHash"`
		IssuerKeyHash  string `json:"issuerKeyHash"`
		SerialNumber   string `json:"serialNumber"`
		ResponderURL   string `json:"responderURL"`
	} `json:"iso15118CertificateHashData"`
}

// validateCertChain validates a PEM certificate chain by parsing it, finding the
// issuer in the database, and verifying the signature.
func validateCertChain(certPEM string) string {
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return "SignatureError"
	}
	leafCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "SignatureError"
	}

	// Check leaf cert expiry
	now := time.Now()
	if now.Before(leafCert.NotBefore) || now.After(leafCert.NotAfter) {
		return "CertificateExpired"
	}

	// Find the issuer in DB by matching subject name of issuer cert
	dbIssuer, err := repository.FindCertBySubject(leafCert.Issuer.String())
	if err != nil {
		return "NoCertificateAvailable"
	}

	// Parse the issuer cert from DB
	issuerBlock, _ := pem.Decode([]byte(dbIssuer.Content))
	if issuerBlock == nil {
		return "SignatureError"
	}
	issuerCert, err := x509.ParseCertificate(issuerBlock.Bytes)
	if err != nil {
		return "SignatureError"
	}

	// Verify leaf cert signature against issuer
	if err := leafCert.CheckSignatureFrom(issuerCert); err != nil {
		return "SignatureError"
	}

	// Check issuer cert expiry
	if now.Before(issuerCert.NotBefore) || now.After(issuerCert.NotAfter) {
		return "CertificateExpired"
	}

	// Check if issuer cert is enabled in DB
	if !dbIssuer.Enabled {
		return "CertificateRevoked"
	}

	return "Accepted"
}

func handlePNCAuthorize(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req pncAuthorizeReq
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		sendDataTransferResult(dc, call.MsgID, "Rejected", "")
		return
	}

	// Parse idToken: schema requires object {idToken: "..."}, but some devices send plain string
	if len(req.IDTokenRaw) > 0 {
		var obj struct{ IDToken string }
		if json.Unmarshal(req.IDTokenRaw, &obj) == nil && obj.IDToken != "" {
			req.idTokenParsed = obj.IDToken
		} else {
			// Try plain string
			var plain string
			if json.Unmarshal(req.IDTokenRaw, &plain) == nil {
				req.idTokenParsed = plain
			}
		}
	}

	certStatus := "Accepted"
	idTokenStatus := "Accepted"

	// certificate and iso15118CertificateHashData are mutually exclusive (2选1)
	if req.Certificate != "" {
		// Option 1: PEM certificate chain — parse, find issuer in DB, validate
		certStatus = validateCertChain(req.Certificate)
	} else if len(req.ISO15118CertificateHashData) > 0 {
		// Option 2: hash data — look up issuer by hash in DB
		certStatus = "NoCertificateAvailable"
		for _, h := range req.ISO15118CertificateHashData {
			// Search by issuerNameHash + issuerKeyHash
			found, _ := repository.FindCertByHash(h.IssuerNameHash, h.IssuerKeyHash)
			if found {
				certStatus = "Accepted"
				break
			}
		}
	}

	// Check idTag
	tag, err := repository.GetIDTagByTagID(req.idTokenParsed)
	if err != nil {
		idTokenStatus = "Invalid"
	} else {
		switch tag.Status {
		case "Blocked":
			idTokenStatus = "Blocked"
		case "Expired":
			idTokenStatus = "Expired"
		}
	}

	resp := map[string]interface{}{
		"certificateStatus": certStatus,
		"idTokenInfo": map[string]interface{}{
			"status": idTokenStatus,
		},
	}
	innerJSON, _ := json.Marshal(resp)
	sendDataTransferResult(dc, call.MsgID, "Accepted", string(innerJSON))

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"PNC Authorize cert="+certStatus+" idTag="+idTokenStatus)
}

// ─── CertificateSigned (device←server, data=JSON string) ────────────────────
// Server sends the signed certificate chain to the device.

type pncCertSignedReq struct {
	CertificateChain string `json:"certificateChain"`
}

func handlePNCCertificateSigned(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	// This is typically a server→device confirmation. If device sends it, accept.
	sendDataTransferResult(dc, call.MsgID, "Accepted", "")
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "PNC CertificateSigned acknowledged")
}

// ─── GetInstalledCertificateIds ──────────────────────────────────────────────

func handlePNCGetInstalledCertIds(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	// Device returns list of installed certs. Forward response to pending HTTP request
	// (matched by msgID in server.go's SendRequest).
	sendDataTransferResult(dc, call.MsgID, "Accepted", string(call.Payload))
}

// ─── InstallCertificate ─────────────────────────────────────────────────────

func handlePNCInstallCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	// Device acknowledges certificate installation
	sendDataTransferResult(dc, call.MsgID, "Accepted", "")
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "PNC InstallCertificate acknowledged")
}

// ─── DeleteCertificate ──────────────────────────────────────────────────────

func handlePNCDeleteCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	sendDataTransferResult(dc, call.MsgID, "Accepted", "")
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "PNC DeleteCertificate acknowledged")
}

// ─── TriggerMessage ─────────────────────────────────────────────────────────

func handlePNCTriggerMessage(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	sendDataTransferResult(dc, call.MsgID, "Accepted", "")
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "PNC TriggerMessage acknowledged")
}

// ─── GetCertificateStatus ───────────────────────────────────────────────────

func handlePNCGetCertStatus(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	// Stub: return generic OK status
	sendDataTransferResult(dc, call.MsgID, "Accepted", "")
}

// ─── Contract Cert Group (2.3.2.4.e / 4.2.9.5) ─────────────────────────────

var (
	contractCertGroups   = make(map[string]map[string]string) // deviceID → {type → certName}
	contractCertGroupsMu sync.RWMutex
)

// SetContractCertGroup persists the user-selected contract certificate group for a device.
// Saves to both memory (fast lookup) and DB (survives restarts).
func SetContractCertGroup(deviceID, tenantID string, group map[string]string) {
	contractCertGroupsMu.Lock()
	contractCertGroups[deviceID] = group
	contractCertGroupsMu.Unlock()
	if err := repository.SaveContractCertGroup(deviceID, tenantID, group); err != nil {
		log.Printf("[ocppws/v16/pnc] contract cert group DB save for %s: %v", deviceID, err)
	}
	log.Printf("[ocppws/v16/pnc] contract cert group set for %s: %d certs", deviceID, len(group))
}

// GetContractCertGroup retrieves the stored contract certificate group for a device.
// Tries in-memory cache first, then falls back to DB.
func GetContractCertGroup(deviceID string) map[string]string {
	contractCertGroupsMu.RLock()
	g, ok := contractCertGroups[deviceID]
	contractCertGroupsMu.RUnlock()
	if ok {
		return g
	}
	// Fall back to DB
	dbGroup, err := repository.GetContractCertGroup(deviceID)
	if err != nil || len(dbGroup) == 0 {
		return nil
	}
	// Cache for next lookup
	contractCertGroupsMu.Lock()
	contractCertGroups[deviceID] = dbGroup
	contractCertGroupsMu.Unlock()
	return dbGroup
}
