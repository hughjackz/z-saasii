package v201

// certs.go — ISO 15118 certificate-related handlers (README 4.3.8-4.3.13).
// Reuses the v16 package's exported PNC helpers (SECC session, CSR signing,
// contract-certificate EXI generator).

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/ocppws/v16"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── CertificateSigned ──────────────────────────────────────────────────────
// Schema: CertificateSignedResponse.json — requires status.

func handleCertificateSigned(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	sendResult(dc, call.MsgID, map[string]string{"status": "Accepted"})
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "CertificateSigned acknowledged (2.0.1)")
}

// ─── DeleteCertificate ──────────────────────────────────────────────────────
// Schema: DeleteCertificateResponse.json — requires status.

func handleDeleteCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	sendResult(dc, call.MsgID, map[string]string{"status": "Accepted"})
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "DeleteCertificate acknowledged (2.0.1)")
}

// ─── InstallCertificate ─────────────────────────────────────────────────────
// Schema: InstallCertificateResponse.json — requires status.

func handleInstallCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	sendResult(dc, call.MsgID, map[string]string{"status": "Accepted"})
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "InstallCertificate acknowledged (2.0.1)")
}

// ─── GetInstalledCertificateIds ─────────────────────────────────────────────
// Device asks which certificates it should install. Responds with the
// tenant's root certificates (V2G root / MO root) from the certificate DB.
// Schema: GetInstalledCertificateIdsResponse.json — requires status;
// certificateHashDataChain items require certificateType + certificateHashData.

func handleGetInstalledCertificateIds(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	certs, err := repository.ListCertificates(model.RoleCSAdmin, "", dc.TenantID, "")
	if err != nil {
		log.Printf("[ocppws/v201] GetInstalledCertificateIds from %s: cert list error: %v", dc.DeviceName, err)
		sendResult(dc, call.MsgID, map[string]string{"status": "Failed"})
		return
	}

	chain := make([]map[string]interface{}, 0, len(certs))
	for _, c := range certs {
		certType := ""
		switch c.Type {
		case "V2G-root-cert":
			certType = "V2GRootCertificate"
		case "MO-root-cert":
			certType = "MORootCertificate"
		default:
			continue // only root certificates are offered for installation
		}
		if c.SerialNumber == "" || c.HashAlgorithm == "" || c.IssuerNameHash == "" || c.IssuerKeyHash == "" {
			continue
		}
		chain = append(chain, map[string]interface{}{
			"certificateType": certType,
			"certificateHashData": map[string]string{
				"hashAlgorithm":  c.HashAlgorithm,
				"issuerNameHash": c.IssuerNameHash,
				"issuerKeyHash":  c.IssuerKeyHash,
				"serialNumber":   c.SerialNumber,
			},
		})
	}

	sendResult(dc, call.MsgID, map[string]interface{}{
		"status":                   "Accepted",
		"certificateHashDataChain": chain,
	})
}

// ─── Get15118EVCertificate ──────────────────────────────────────────────────
// Device requests a contract certificate for the EV. Mirrors the v16 PNC
// handler (v16/pnc.go Get15118EVCertificate) minus the DataTransfer wrapper.
// Schema: Get15118EVCertificateResponse.json — requires status + exiResponse.

type get15118EVCertReq struct {
	ISO15118SchemaVersion string `json:"iso15118SchemaVersion"`
	Action                string `json:"action"` // Install | Update
	ExiRequest            string `json:"exiRequest"`
}

func handleGet15118EVCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req get15118EVCertReq
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		log.Printf("[ocppws/v201] Get15118EVCertificate from %s: bad payload: %v", dc.DeviceName, err)
		sendResult(dc, call.MsgID, map[string]string{"status": "Failed", "exiResponse": ""})
		return
	}

	log.Printf("[ocppws/v201] Get15118EVCertificate from %s (action=%s, schemaVersion=%s, exiLen=%d)",
		dc.DeviceName, req.Action, req.ISO15118SchemaVersion, len(req.ExiRequest))

	// Tenant certificate library; narrow to the user-selected contract cert
	// group when present (2.3.2.4.e / 4.2.9.5).
	allCerts, err := repository.ListCertificates(model.RoleCSAdmin, "", dc.TenantID, "")
	if err != nil {
		log.Printf("[ocppws/v201] Get15118EVCertificate from %s: cert list error: %v", dc.DeviceName, err)
	}

	certs := allCerts
	if group := v16.GetContractCertGroup(dc.DeviceID); group != nil {
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
		}
	}

	if ig, ok := v16.ContractGen.(interface{ Initialize(tenantID string) error }); ok {
		if err := ig.Initialize(dc.TenantID); err != nil {
			log.Printf("[ocppws/v201] ContractGen init for tenant %s: %v", dc.TenantID, err)
		}
	}

	exiResp, err := v16.ContractGen.Generate(req.ExiRequest, certs)
	if err != nil {
		log.Printf("[ocppws/v201] Get15118EVCertificate from %s: ContractGenerate error: %v", dc.DeviceName, err)
		sendResult(dc, call.MsgID, map[string]string{"status": "Failed", "exiResponse": ""})
		pushEvent(eventCh, dc.TenantID, "error", dc.DeviceName,
			"Get15118EVCertificate ContractGenerate failed: "+err.Error())
		return
	}

	sendResult(dc, call.MsgID, map[string]interface{}{
		"status":                "Accepted",
		"exiResponse":           exiResp,
		"iso15118SchemaVersion": req.ISO15118SchemaVersion,
	})
	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"Get15118EVCertificate action="+req.Action+" status=Accepted")
}

// ─── SignCertificate ────────────────────────────────────────────────────────
// Device sends a CSR for SECC Leaf signing (mirrors the v16 PNC flow).
// Schema: SignCertificateResponse.json — requires status (Accepted/Rejected).

type signCertificateReq struct {
	CSR             string `json:"csr"`
	CertificateType string `json:"certificateType"`
}

func handleSignCertificate(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req signCertificateReq
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		log.Printf("[ocppws/v201] SignCertificate from %s cannot unmarshal payload: %v", dc.DeviceName, err)
		sendResult(dc, call.MsgID, map[string]string{"status": "Rejected"})
		return
	}

	session := v16.GetSECCSession(dc.DeviceID)
	if session == nil {
		log.Printf("[ocppws/v201] SignCertificate from %s but no pending SECC session", dc.DeviceName)
		sendResult(dc, call.MsgID, map[string]string{"status": "Rejected"})
		return
	}

	log.Printf("[ocppws/v201] Signing SECC Leaf for %s with V2G Sub2=%s", dc.DeviceName, session.V2GSub2)

	signedCert, err := v16.SignCSR(req.CSR, dc.DeviceName, dc.TenantID, session)
	if err != nil {
		log.Printf("[ocppws/v201] SECC signing failed for %s: %v", dc.DeviceName, err)
		sendResult(dc, call.MsgID, map[string]string{"status": "Rejected"})
		pushEvent(eventCh, dc.TenantID, "error", dc.DeviceName, "SECC Leaf signing failed: "+err.Error())
		return
	}

	// Build certificate chain: SECC Leaf + V2G Sub2 + V2G Sub1
	certChain := signedCert
	if c, _, err := findCertContent(session.V2GSub2); err == nil {
		certChain += c
	}
	if c, _, err := findCertContent(session.V2GSub1); err == nil {
		certChain += c
	}

	// Respond to SignCertificate first, then push CertificateSigned to the device
	sendResult(dc, call.MsgID, map[string]string{"status": "Accepted"})

	certSignedCall, _ := ocppws.BuildCall(uuid.New().String(), "CertificateSigned",
		map[string]string{"certificateChain": certChain})
	dc.WriteCh <- certSignedCall

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName, "SECC Leaf certificate signed and sent to device (2.0.1)")
}

// findCertContent looks up a certificate's PEM content from the DB by name.
func findCertContent(name string) (content string, keyPass string, err error) {
	certs, err := repository.ListCertificates(model.RoleCSAdmin, "", "", "")
	if err != nil {
		return "", "", err
	}
	for _, c := range certs {
		if c.Name == name {
			cnt, _, pass, e := repository.GetCertKeyAndPassphrase(c.ID)
			return cnt, pass, e
		}
	}
	return "", "", fmt.Errorf("certificate %q not found", name)
}
