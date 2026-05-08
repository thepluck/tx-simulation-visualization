package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

func debugHTTPEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("TXSIM_DEBUG_HTTP"))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func debugHTTP(next http.Handler) http.Handler {
	if !debugHTTPEnabled() {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var requestBody []byte
		if r.Body != nil {
			requestBody, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(requestBody))
		}
		slog.Info("http request", "method", r.Method, "path", r.URL.RequestURI(), "body", logBody(requestBody))

		recorder := &debugResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		slog.Info(
			"http response",
			"method", r.Method,
			"path", r.URL.RequestURI(),
			"status", recorder.status,
			"duration", time.Since(start),
			"body", logBody(recorder.body.Bytes()),
		)
	})
}

type debugResponseWriter struct {
	http.ResponseWriter
	body   bytes.Buffer
	status int
}

func (w *debugResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *debugResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func logBody(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "<empty>"
	}
	redacted, err := redactLogBody([]byte(trimmed))
	if err != nil {
		return trimmed
	}
	return redacted
}

func redactLogBody(body []byte) (string, error) {
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		return "", err
	}
	redactJSONValue(value)
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	return strings.TrimSpace(output.String()), nil
}

func redactJSONValue(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if strings.EqualFold(key, "etherscanApiKey") {
				typed[key] = "<redacted>"
				continue
			}
			redactJSONValue(child)
		}
	case []any:
		for _, child := range typed {
			redactJSONValue(child)
		}
	}
}
