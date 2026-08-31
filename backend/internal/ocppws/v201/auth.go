package v201

// auth.go — Authorization handler (README 4.3.1).
// Handles Authorize requests from charge points.

import (
	"encoding/json"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── Authorize ───────────────────────────────────────────────────────────────
// Schema: AuthorizeRequest.json / AuthorizeResponse.json
// Response requires idTokenInfo; certificateStatus is optional and returned
// when the device provides ISO 15118 certificate hash data.

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

	// idToken lookup (mirrors v16 auth.go)
	idTagInfo := map[string]interface{}{"status": "Accepted"}
	tag, err := repository.GetIDTagByTagID(req.IDToken.IDToken)
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

	resp := map[string]interface{}{
		"idTagInfo": idTagInfo,
	}

	// ISO 15118 certificate hash data: accept when a matching issuer exists in DB
	if len(req.ISO15118CertificateHashData) > 0 {
		certStatus := "NoCertificateAvailable"
		for _, h := range req.ISO15118CertificateHashData {
			found, _ := repository.FindCertByHash(h.IssuerNameHash, h.IssuerKeyHash)
			if found {
				certStatus = "Accepted"
				break
			}
		}
		resp["certificateStatus"] = certStatus
	}

	sendResult(dc, call.MsgID, resp)
}
