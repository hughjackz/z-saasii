package v16

// auth.go — Authorization handler.
// Handles Authorize requests from charge points.

import (
	"encoding/json"
	"time"

	"github.com/yourorg/csms-backend/internal/model"
	"github.com/yourorg/csms-backend/internal/ocppws"
	"github.com/yourorg/csms-backend/internal/repository"
)

// ─── Authorize ───────────────────────────────────────────────────────────────
// Schema: Authorize.json / AuthorizeResponse.json

type authorizeReq struct {
	IDTag string `json:"idTag"`
}

func handleAuthorize(dc *ocppws.DeviceConnection, call *ocppws.CallMessage, eventCh chan<- *model.Event) {
	var req authorizeReq
	_ = json.Unmarshal(call.Payload, &req)

	idTagInfo := map[string]interface{}{"status": "Accepted"}

	tag, err := repository.GetIDTagByTagID(req.IDTag)
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
	sendResult(dc, call.MsgID, resp)
}
