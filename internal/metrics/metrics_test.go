package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCounter(t *testing.T) {
	c := NewCollector()
	c.IncrCounter("test_total", map[string]string{"method": "GET"})
	c.IncrCounter("test_total", map[string]string{"method": "GET"})
	c.IncrCounter("test_total", map[string]string{"method": "POST"})

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", c.Handler())

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	r.ServeHTTP(w, req)

	body := w.Body.String()
	if !strings.Contains(body, `test_total{method="GET"} 2`) {
		t.Fatalf("missing GET counter in: %s", body)
	}
	if !strings.Contains(body, `test_total{method="POST"} 1`) {
		t.Fatalf("missing POST counter in: %s", body)
	}
}

func TestGauge(t *testing.T) {
	c := NewCollector()
	c.SetGauge("active_sessions", nil, 42)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", c.Handler())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))

	if !strings.Contains(w.Body.String(), "active_sessions 42") {
		t.Fatalf("missing gauge: %s", w.Body.String())
	}
}

func TestHistogram(t *testing.T) {
	c := NewCollector()
	c.ObserveHistogram("latency", map[string]string{"path": "/api"}, 150)
	c.ObserveHistogram("latency", map[string]string{"path": "/api"}, 500)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", c.Handler())

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/metrics", nil))

	body := w.Body.String()
	if !strings.Contains(body, "latency_count") {
		t.Fatalf("missing histogram count: %s", body)
	}
	if !strings.Contains(body, "latency_sum") {
		t.Fatalf("missing histogram sum: %s", body)
	}
}

func TestRecordVerification(t *testing.T) {
	RecordVerification("silent", "ID", "verified", 350)
	RecordOTP("sms", "TH", "sent")
	RecordRiskCheck("sim_swap", "block")
	RecordUpstreamCall("sandbox", "ID", true, 200)
	RecordUpstreamCall("sandbox", "ID", false, 5000)
	// No panic = pass
}
