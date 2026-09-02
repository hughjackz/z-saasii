package ocpp

// config.go — GetConfiguration / ChangeConfiguration handlers.
// OCPP 1.6 uses GetConfiguration/ChangeConfiguration; OCPP 2.0.1 uses
// GetVariables/SetVariables (README 4.3.15). Both are exposed through the same
// HTTP endpoints and normalized to the same response shape.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/csms-backend/internal/ocppws"
)

// componentName is the component used for station-level variables
// (specification Part "Referenced Components and Variables" Chapter 2).
const componentName = "ChargingStation"

// variablesComponent builds the GetVariables component+variable descriptor.
func variablesComponent(key string) map[string]interface{} {
	return map[string]interface{}{
		"component": map[string]interface{}{"name": componentName},
		"variable":  map[string]interface{}{"name": key},
	}
}

// normalizeGetVariables converts a GetVariablesResponse into the
// configurationKey shape used by the frontend.
func normalizeGetVariables(result interface{}) map[string]interface{} {
	out := map[string]interface{}{"configurationKey": []interface{}{}}
	m, ok := result.(map[string]interface{})
	if !ok {
		return out
	}
	results, _ := m["getVariableResult"].([]interface{})
	var keys []interface{}
	for _, r := range results {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		variable, _ := rm["variable"].(map[string]interface{})
		name, _ := variable["name"].(string)
		if name == "" {
			continue
		}
		value := ""
		readonly := true
		switch rm["attributeStatus"] {
		case "Accepted", "RebootRequired":
			readonly = false
		}
		// attributeValue may be absent (NotSupported) or a typed value
		if av, exists := rm["attributeValue"]; exists {
			switch v := av.(type) {
			case string:
				value = v
			default:
				value = ""
			}
		}
		keys = append(keys, map[string]interface{}{
			"key": name, "value": value, "readonly": readonly,
		})
	}
	out["configurationKey"] = keys
	return out
}

// GET /api/ocpp/:deviceId/configuration
func GetConfiguration(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceIsOcpp2(deviceID) {
		// OCPP 2.0.1: GetVariables with an empty getVariableData list fetches
		// the component's variables where the device supports it; otherwise we
		// request the station-level component without a variable filter.
		payload := map[string]interface{}{
			"getVariableData": []interface{}{
				map[string]interface{}{
					"attributeType": "Actual",
					"component":     map[string]interface{}{"name": componentName},
					"variable":      map[string]interface{}{},
				},
			},
		}
		result, err := ocppws.Default.SendRequest(deviceID, "GetVariables", payload)
		if err != nil {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, normalizeGetVariables(result))
		return
	}
	sendToOCPP(c, deviceID, "GetConfiguration", map[string]interface{}{
		"key": []string{},
	})
}

// POST /api/ocpp/:deviceId/configuration/get
func GetConfigurationKeys(c *gin.Context) {
	var body struct {
		Keys []string `json:"keys"`
	}
	_ = c.ShouldBindJSON(&body)

	deviceID := c.Param("deviceId")
	if deviceIsOcpp2(deviceID) {
		items := make([]interface{}, 0, len(body.Keys))
		for _, k := range body.Keys {
			items = append(items, map[string]interface{}{
				"attributeType": "Actual",
				"component":     map[string]interface{}{"name": componentName},
				"variable":      map[string]interface{}{"name": k},
			})
		}
		result, err := ocppws.Default.SendRequest(deviceID, "GetVariables", map[string]interface{}{
			"getVariableData": items,
		})
		if err != nil {
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, normalizeGetVariables(result))
		return
	}
	sendToOCPP(c, deviceID, "GetConfiguration", map[string]interface{}{
		"key": body.Keys,
	})
}

// POST /api/ocpp/:deviceId/configuration/set
// README 4.3.15: for SetVariables, attributeType=Actual and attributeValue is
// the value entered by the frontend user.
func SetConfiguration(c *gin.Context) {
	var body struct {
		Configs []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"configs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	deviceID := c.Param("deviceId")
	results := make([]map[string]interface{}, 0)

	if deviceIsOcpp2(deviceID) {
		items := make([]interface{}, 0, len(body.Configs))
		for _, cfg := range body.Configs {
			items = append(items, map[string]interface{}{
				"attributeType":  "Actual",
				"attributeValue": cfg.Value,
				"component":      map[string]interface{}{"name": componentName},
				"variable":       map[string]interface{}{"name": cfg.Key},
			})
		}
		resp, err := ocppws.Default.SendRequest(deviceID, "SetVariables", map[string]interface{}{
			"setVariableData": items,
		})
		if err == nil {
			if m, ok := resp.(map[string]interface{}); ok {
				if sr, ok := m["setVariableResult"].([]interface{}); ok {
					results = append(results, map[string]interface{}{
						"results": sr,
					})
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"results": results})
		return
	}

	for _, cfg := range body.Configs {
		resp, err := ocppws.Default.SendRequest(deviceID, "ChangeConfiguration",
			map[string]string{"key": cfg.Key, "value": cfg.Value})
		if err == nil {
			results = append(results, map[string]interface{}{
				"key":    cfg.Key,
				"status": resp,
			})
		}
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
