package v201

// auth.go — Authorize handler (README 4.3.1).
// OCPP 2.0.1 merges plain idTag authorization and PnC contract-certificate +
// eMAID authorization into one message. Contract-certificate validation is
// optional and processed only when idToken.type = "eMAID"; the certificate or
// iso15118CertificateHashData is then validated against the PKI certificate
// library. Plain idTags are validated against the idtag table.

import (
	"encoding/json"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/ocppws/v16"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── Authorize ───────────────────────────────────────────────────────────────
// Schema: AuthorizeRequest.json / AuthorizeResponse.json
// Response requires idTokenInfo; certificateStatus is optional and returned
// when the device provides ISO 15118 certificate data (eMAID).

type authorizeReq struct {
	IDToken struct {
		IDToken string `json:"idToken"`
		Type    string `json:"type"`
	} `json:"idToken"`
	Certificate                 string            `json:"certificate"`
	ISO15118CertificateHashData []ocspRequestData `json:"iso15118CertificateHashData"`
}

type ocspRequestData struct {
	HashAlgorithm  string `json:"hashAlgorithm"`
	IssuerNameHash string `json:"issuerNameHash"`
	IssuerKeyHash  string `json:"issuerKeyHash"`
	SerialNumber   string `json:"serialNumber"`
	ResponderURL   string `json:"responderURL"`
}

func handleAuthorize(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req authorizeReq
	_ = json.Unmarshal(call.Payload, &req)

	idTagInfo := authorizeIDTag(req.IDToken.IDToken)

	resp := map[string]interface{}{
		"idTagInfo": idTagInfo,
	}

	// Contract-certificate authorization: optional, only when the idToken is an
	// eMAID and certificate data is provided (README 4.3.1).
	if req.IDToken.Type == "eMAID" && (req.Certificate != "" || len(req.ISO15118CertificateHashData) > 0) {
		resp["certificateStatus"] = authorizeContractCertificate(req)
	}

	sendResult(dc, call.MsgID, resp)

	pushEvent(eventCh, dc.TenantID, "info", dc.DeviceName,
		"Authorize idTag="+req.IDToken.IDToken+" type="+req.IDToken.Type+
			" status="+idTagInfo["status"].(string))
}

// authorizeIDTag validates a plain idTag against the idtag table.
func authorizeIDTag(tagID string) map[string]interface{} {
	idTagInfo := map[string]interface{}{"status": "Accepted"}
	tag, err := repository.GetIDTagByTagID(tagID)
	if err == nil {
		switch tag.Status {
		case "Blocked":
			idTagInfo["status"] = "Blocked"
		case "Expired":
			idTagInfo["status"] = "Expired"
		default:
			idTagInfo["status"] = "Accepted"
		}
		if tag.ExpiryTime != nil {
			idTagInfo["expiryDate"] = tag.ExpiryTime.Format(time.RFC3339)
		}
		if tag.ParentTagID != nil && *tag.ParentTagID != "" {
			idTagInfo["parentIdTag"] = *tag.ParentTagID
		}
	} else {
		idTagInfo["status"] = "Invalid"
	}
	return idTagInfo
}

// authorizeContractCertificate validates contract-certificate data against the
// PKI certificate library (README 4.3.1):
//   - certificate (PEM chain): parse, find issuer in DB, verify signature
//   - iso15118CertificateHashData: match issuerNameHash+issuerKeyHash in DB
func authorizeContractCertificate(req authorizeReq) string {
	if req.Certificate != "" {
		return v16.ValidateCertChain(req.Certificate)
	}
	if len(req.ISO15118CertificateHashData) > 0 {
		for _, h := range req.ISO15118CertificateHashData {
			found, _ := repository.FindCertByHash(h.IssuerNameHash, h.IssuerKeyHash)
			if found {
				return "Accepted"
			}
		}
	}
	return "NoCertificateAvailable"
}
