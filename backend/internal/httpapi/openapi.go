package httpapi

import "net/http"

func (s *Server) handleOpenAPI(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, openAPISpec())
}

func (s *Server) handleSwaggerUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(swaggerHTML))
}

const swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Tx Simulation API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>
    body { margin: 0; background: #f7f7f7; }
    .swagger-ui .topbar { display: none; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
        dom_id: "#swagger-ui",
        deepLinking: true,
        displayRequestDuration: true
      });
    };
  </script>
</body>
</html>`

func openAPISpec() map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   "Tx Simulation API",
			"version": "0.1.0",
		},
		"paths": map[string]any{
			"/health": map[string]any{
				"get": map[string]any{
					"summary": "Backend health and configured chain count",
					"responses": map[string]any{
						"200": jsonResponse("#/components/schemas/HealthResponse"),
					},
				},
			},
			"/chains": map[string]any{
				"get": map[string]any{
					"summary": "List configured chain names",
					"responses": map[string]any{
						"200": jsonResponse("#/components/schemas/ChainsResponse"),
					},
				},
			},
			"/simulate": map[string]any{
				"post": map[string]any{
					"summary": "Run a Forge script simulation and return raw plus structured trace",
					"requestBody": map[string]any{
						"required": true,
						"content": map[string]any{
							"application/json": map[string]any{
								"schema": map[string]any{"$ref": "#/components/schemas/SimulateRequest"},
							},
						},
					},
					"responses": map[string]any{
						"200": jsonResponse("#/components/schemas/SimulateResponse"),
						"400": jsonResponse("#/components/schemas/ErrorResponse"),
						"429": jsonResponse("#/components/schemas/SimulateResponse"),
						"500": jsonResponse("#/components/schemas/SimulateResponse"),
						"504": jsonResponse("#/components/schemas/SimulateResponse"),
					},
				},
			},
		},
		"components": map[string]any{
			"schemas": map[string]any{
				"HealthResponse": map[string]any{
					"type":     "object",
					"required": []string{"ok", "chains", "maxConcurrentRuns"},
					"properties": map[string]any{
						"ok":                map[string]any{"type": "boolean"},
						"chains":            map[string]any{"type": "integer"},
						"maxConcurrentRuns": map[string]any{"type": "integer"},
					},
				},
				"ChainsResponse": map[string]any{
					"type":     "object",
					"required": []string{"chains"},
					"properties": map[string]any{
						"chains": map[string]any{
							"type":  "array",
							"items": map[string]any{"type": "string"},
						},
					},
				},
				"SimulateRequest": map[string]any{
					"type":     "object",
					"required": []string{"chain", "blockNumber", "sender", "target", "data"},
					"properties": map[string]any{
						"chain":                  map[string]any{"type": "string", "example": "mainnet"},
						"blockNumber":            uintStringSchema("23000000"),
						"labelOverrides":         arraySchema("#/components/schemas/LabelOverride"),
						"erc20BalanceOverrides":  arraySchema("#/components/schemas/ERC20BalanceOverride"),
						"erc20ApprovalOverrides": arraySchema("#/components/schemas/ERC20ApprovalOverride"),
						"erc721ApprovalOverrides": arraySchema(
							"#/components/schemas/ERC721ApprovalOverride",
						),
						"stateOverride":             map[string]any{"$ref": "#/components/schemas/StateOverride"},
						"stateOverrideCode":         map[string]any{"type": "string", "deprecated": true},
						"stateOverrideContractName": map[string]any{"type": "string", "deprecated": true},
						"compiler":                  map[string]any{"$ref": "#/components/schemas/CompilerConfig"},
						"sender":                    addressSchema("0x0000000000000000000000000000000000000001"),
						"target":                    addressSchema("0x0000000000000000000000000000000000000002"),
						"data":                      bytesSchema("0x"),
					},
				},
				"LabelOverride": map[string]any{
					"type":     "object",
					"required": []string{"account", "label"},
					"properties": map[string]any{
						"account": addressSchema("0x0000000000000000000000000000000000000001"),
						"label":   map[string]any{"type": "string", "example": "WETHOwner"},
					},
				},
				"ERC20BalanceOverride": map[string]any{
					"type":     "object",
					"required": []string{"token", "account", "balance"},
					"properties": map[string]any{
						"token":   addressSchema("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
						"account": addressSchema("0x0000000000000000000000000000000000000001"),
						"balance": uintStringSchema("1000000000000000000"),
					},
				},
				"ERC20ApprovalOverride": map[string]any{
					"type":     "object",
					"required": []string{"token", "owner", "spender", "amount"},
					"properties": map[string]any{
						"token":   addressSchema("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"),
						"owner":   addressSchema("0x0000000000000000000000000000000000000001"),
						"spender": addressSchema("0x0000000000000000000000000000000000000002"),
						"amount":  uintStringSchema("1000000000000000000"),
					},
				},
				"ERC721ApprovalOverride": map[string]any{
					"type":     "object",
					"required": []string{"token", "owner", "spender", "tokenId"},
					"properties": map[string]any{
						"token":   addressSchema("0xBC4CA0EdA7647A8aB7C2061c2E118A18a936f13D"),
						"owner":   addressSchema("0x0000000000000000000000000000000000000001"),
						"spender": addressSchema("0x0000000000000000000000000000000000000002"),
						"tokenId": uintStringSchema("1"),
					},
				},
				"StateOverride": map[string]any{
					"type":     "object",
					"required": []string{"source"},
					"properties": map[string]any{
						"contractName": map[string]any{"type": "string", "example": "MyStateOverride"},
						"source":       map[string]any{"type": "string"},
					},
				},
				"CompilerConfig": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"use": map[string]any{
							"type":        "string",
							"description": "Maps to forge script --use <SOLC_VERSION>.",
							"example":     "0.8.30",
						},
						"offline": map[string]any{
							"type":        "boolean",
							"description": "Maps to --offline.",
						},
						"noAutoDetect": map[string]any{
							"type":        "boolean",
							"description": "Maps to --no-auto-detect.",
						},
						"viaIR": map[string]any{
							"type":        "boolean",
							"default":     true,
							"description": "Maps to --via-ir. Defaults to true for this backend.",
						},
						"useLiteralContent": map[string]any{
							"type":        "boolean",
							"description": "Maps to --use-literal-content.",
						},
						"noMetadata": map[string]any{
							"type":        "boolean",
							"description": "Maps to --no-metadata.",
						},
						"evmVersion": map[string]any{
							"type":        "string",
							"description": "Maps to --evm-version <VERSION>.",
							"example":     "cancun",
						},
						"optimize": map[string]any{
							"type":        "boolean",
							"default":     true,
							"description": "Maps to --optimize. Defaults to true for this backend.",
						},
						"optimizerRuns": map[string]any{
							"type":        "integer",
							"minimum":     0,
							"maximum":     4294967295,
							"description": "Maps to --optimizer-runs <RUNS>.",
							"example":     200,
						},
						"revertStrings": map[string]any{
							"type":        "string",
							"enum":        []string{"default", "strip", "debug", "verboseDebug"},
							"description": "Maps to --revert-strings <REVERT>.",
							"example":     "default",
						},
					},
				},
				"SimulateResponse": map[string]any{
					"type":     "object",
					"required": []string{"id", "success", "exitCode", "durationMillis", "trace"},
					"properties": map[string]any{
						"id":             map[string]any{"type": "string"},
						"success":        map[string]any{"type": "boolean"},
						"exitCode":       map[string]any{"type": "integer"},
						"durationMillis": map[string]any{"type": "integer"},
						"trace":          map[string]any{"type": "string"},
						"structuredTrace": map[string]any{
							"type":  "array",
							"items": map[string]any{"$ref": "#/components/schemas/TraceNode"},
						},
						"error": map[string]any{"type": "string"},
					},
				},
				"TraceNode": map[string]any{
					"type":     "object",
					"required": []string{"raw", "kind"},
					"properties": map[string]any{
						"raw":        map[string]any{"type": "string"},
						"kind":       map[string]any{"type": "string", "enum": []string{"call", "return", "revert", "event", "error", "result", "unknown"}},
						"gas":        map[string]any{"type": "integer"},
						"target":     map[string]any{"type": "string"},
						"function":   map[string]any{"type": "string"},
						"arguments":  map[string]any{"type": "string"},
						"callType":   map[string]any{"type": "string"},
						"resultType": map[string]any{"type": "string"},
						"value":      map[string]any{"type": "string"},
						"children": map[string]any{
							"type":  "array",
							"items": map[string]any{"$ref": "#/components/schemas/TraceNode"},
						},
					},
				},
				"ErrorResponse": map[string]any{
					"type":     "object",
					"required": []string{"error"},
					"properties": map[string]any{
						"error": map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func jsonResponse(schemaRef string) map[string]any {
	return map[string]any{
		"description": "OK",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": schemaRef},
			},
		},
	}
}

func arraySchema(itemRef string) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": map[string]any{"$ref": itemRef},
	}
}

func addressSchema(example string) map[string]any {
	return map[string]any{
		"type":    "string",
		"pattern": "^0x[0-9a-fA-F]{40}$",
		"example": example,
	}
}

func bytesSchema(example string) map[string]any {
	return map[string]any{
		"type":    "string",
		"pattern": "^0x([0-9a-fA-F]{2})*$",
		"example": example,
	}
}

func uintStringSchema(example string) map[string]any {
	return map[string]any{
		"type":    "string",
		"pattern": "^(0x[0-9a-fA-F]+|[0-9]+)$",
		"example": example,
	}
}
