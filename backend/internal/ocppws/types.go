package ocppws

import (
	"encoding/json"
	"fmt"
)

// OCPP-J WebSocket message type constants
const (
	Call       = 2 // [2, "msgId", "Action", payload]
	CallResult = 3 // [3, "msgId", payload]
	CallError  = 4 // [4, "msgId", "ErrorCode", "ErrorDescription", details]
)

// CallMessage represents an incoming OCPP CALL [2, msgId, Action, payload].
type CallMessage struct {
	MsgID   string          `json:"-"`
	Action  string          `json:"-"`
	Payload json.RawMessage `json:"-"`
}

// CallResultMessage represents an incoming OCPP CALLRESULT [3, msgId, payload].
type CallResultMessage struct {
	MsgID   string          `json:"-"`
	Payload json.RawMessage `json:"-"`
}

// CallErrorMessage represents an incoming OCPP CALLERROR [4, msgId, code, desc, details].
type CallErrorMessage struct {
	MsgID       string          `json:"-"`
	ErrorCode   string          `json:"-"`
	Description string          `json:"-"`
	Details     json.RawMessage `json:"-"`
}

// ParseCall parses a [2, msgId, action, payload] JSON array.
func ParseCall(data []byte) (*CallMessage, error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("ocppws: invalid JSON: %w", err)
	}
	if len(raw) < 4 {
		return nil, fmt.Errorf("ocppws: CALL expects 4 elements, got %d", len(raw))
	}
	var msgType int
	if err := json.Unmarshal(raw[0], &msgType); err != nil || msgType != Call {
		return nil, fmt.Errorf("ocppws: not a CALL message (type=%d)", msgType)
	}
	var msgID, action string
	json.Unmarshal(raw[1], &msgID)
	json.Unmarshal(raw[2], &action)
	return &CallMessage{MsgID: msgID, Action: action, Payload: raw[3]}, nil
}

// ParseResult parses a [3, msgId, payload] or [4, ...] response.
func ParseResult(data []byte) (msgType int, msgID string, payload json.RawMessage, errCode string, errDesc string, err error) {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, "", nil, "", "", fmt.Errorf("ocppws: invalid JSON: %w", err)
	}
	if len(raw) < 2 {
		return 0, "", nil, "", "", fmt.Errorf("ocppws: too few elements: %d", len(raw))
	}
	var t int
	json.Unmarshal(raw[0], &t)
	json.Unmarshal(raw[1], &msgID)

	switch t {
	case CallResult:
		if len(raw) >= 3 {
			return CallResult, msgID, raw[2], "", "", nil
		}
		return CallResult, msgID, nil, "", "", nil
	case CallError:
		if len(raw) >= 5 {
			json.Unmarshal(raw[2], &errCode)
			json.Unmarshal(raw[3], &errDesc)
			return CallError, msgID, raw[3], errCode, errDesc, nil
		}
		return CallError, msgID, nil, "", "", fmt.Errorf("ocppws: malformed CALLERROR")
	default:
		return t, msgID, nil, "", "", fmt.Errorf("ocppws: unknown message type: %d", t)
	}
}

// BuildCall packs a CALL [2, msgId, action, payload] as JSON bytes.
func BuildCall(msgID, action string, payload interface{}) ([]byte, error) {
	return json.Marshal([]interface{}{Call, msgID, action, payload})
}

// BuildCallResult packs a CALLRESULT [3, msgId, payload] as JSON bytes.
func BuildCallResult(msgID string, payload interface{}) ([]byte, error) {
	return json.Marshal([]interface{}{CallResult, msgID, payload})
}

// BuildCallError packs a CALLERROR [4, msgId, code, desc, details] as JSON bytes.
func BuildCallError(msgID, code, desc string, details interface{}) ([]byte, error) {
	if details == nil {
		details = struct{}{}
	}
	return json.Marshal([]interface{}{CallError, msgID, code, desc, details})
}
