package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Collector tracks application metrics in Prometheus exposition format.
type Collector struct {
	counters   sync.Map // name -> *counter
	histograms sync.Map // name -> *histogram
	gauges     sync.Map // name -> *gauge
}

var Global = NewCollector()

func NewCollector() *Collector {
	return &Collector{}
}

// --- Counter ---

type counter struct {
	values sync.Map // labels_key -> *int64
}

func (c *Collector) IncrCounter(name string, labels map[string]string) {
	raw, _ := c.counters.LoadOrStore(name, &counter{})
	ctr := raw.(*counter)
	key := labelsKey(labels)
	raw2, _ := ctr.values.LoadOrStore(key, new(int64))
	atomic.AddInt64(raw2.(*int64), 1)
}

func (c *Collector) AddCounter(name string, labels map[string]string, delta int64) {
	raw, _ := c.counters.LoadOrStore(name, &counter{})
	ctr := raw.(*counter)
	key := labelsKey(labels)
	raw2, _ := ctr.values.LoadOrStore(key, new(int64))
	atomic.AddInt64(raw2.(*int64), delta)
}

// --- Gauge ---

type gauge struct {
	values sync.Map
}

func (c *Collector) SetGauge(name string, labels map[string]string, value float64) {
	raw, _ := c.gauges.LoadOrStore(name, &gauge{})
	g := raw.(*gauge)
	key := labelsKey(labels)
	g.values.Store(key, value)
}

// --- Histogram (simplified: tracks count, sum, buckets) ---

type histogram struct {
	mu      sync.Mutex
	buckets []float64
	values  map[string]*histData
}

type histData struct {
	count   int64
	sum     float64
	buckets []int64 // count per bucket
}

func (c *Collector) ObserveHistogram(name string, labels map[string]string, value float64) {
	buckets := []float64{50, 100, 250, 500, 1000, 2500, 5000, 10000}
	raw, _ := c.histograms.LoadOrStore(name, &histogram{
		buckets: buckets,
		values:  make(map[string]*histData),
	})
	h := raw.(*histogram)
	key := labelsKey(labels)

	h.mu.Lock()
	defer h.mu.Unlock()

	d, ok := h.values[key]
	if !ok {
		d = &histData{buckets: make([]int64, len(h.buckets))}
		h.values[key] = d
	}
	d.count++
	d.sum += value
	for i, b := range h.buckets {
		if value <= b {
			d.buckets[i]++
		}
	}
}

// --- Prometheus Handler ---

// Handler returns a gin handler that serves /metrics in Prometheus format.
func (c *Collector) Handler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var sb strings.Builder

		// Counters
		c.counters.Range(func(name, raw interface{}) bool {
			ctr := raw.(*counter)
			ctr.values.Range(func(key, val interface{}) bool {
				v := atomic.LoadInt64(val.(*int64))
				if key.(string) == "" {
					fmt.Fprintf(&sb, "%s %d\n", name, v)
				} else {
					fmt.Fprintf(&sb, "%s{%s} %d\n", name, key, v)
				}
				return true
			})
			return true
		})

		// Gauges
		c.gauges.Range(func(name, raw interface{}) bool {
			g := raw.(*gauge)
			g.values.Range(func(key, val interface{}) bool {
				if key.(string) == "" {
					fmt.Fprintf(&sb, "%s %g\n", name, val.(float64))
				} else {
					fmt.Fprintf(&sb, "%s{%s} %g\n", name, key, val.(float64))
				}
				return true
			})
			return true
		})

		// Histograms
		c.histograms.Range(func(name, raw interface{}) bool {
			h := raw.(*histogram)
			h.mu.Lock()
			defer h.mu.Unlock()
			for key, d := range h.values {
				labels := ""
				if key != "" {
					labels = key + ","
				}
				for i, b := range h.buckets {
					fmt.Fprintf(&sb, "%s_bucket{%sle=\"%g\"} %d\n", name, labels, b, d.buckets[i])
				}
				fmt.Fprintf(&sb, "%s_bucket{%sle=\"+Inf\"} %d\n", name, labels, d.count)
				fmt.Fprintf(&sb, "%s_sum{%s} %g\n", name, strings.TrimSuffix(labels, ","), d.sum)
				fmt.Fprintf(&sb, "%s_count{%s} %d\n", name, strings.TrimSuffix(labels, ","), d.count)
			}
			return true
		})

		ctx.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(sb.String()))
	}
}

// --- Middleware ---

// RequestMetrics records HTTP request metrics.
func RequestMetrics(c *Collector) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()
		duration := float64(time.Since(start).Milliseconds())

		labels := map[string]string{
			"method": ctx.Request.Method,
			"path":   ctx.FullPath(),
			"status": fmt.Sprintf("%d", ctx.Writer.Status()),
		}

		c.IncrCounter("silentpass_http_requests_total", labels)
		c.ObserveHistogram("silentpass_http_request_duration_ms", labels, duration)
	}
}

func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, labels[k]))
	}
	return strings.Join(parts, ",")
}

// --- Application Metric Helpers ---

func RecordVerification(method, country, result string, latencyMs float64) {
	labels := map[string]string{"method": method, "country": country, "result": result}
	Global.IncrCounter("silentpass_verifications_total", labels)
	Global.ObserveHistogram("silentpass_verification_duration_ms",
		map[string]string{"method": method, "country": country}, latencyMs)
}

func RecordOTP(channel, country, status string) {
	Global.IncrCounter("silentpass_otp_total",
		map[string]string{"channel": channel, "country": country, "status": status})
}

func RecordRiskCheck(checkType, verdict string) {
	Global.IncrCounter("silentpass_risk_checks_total",
		map[string]string{"type": checkType, "verdict": verdict})
}

func RecordUpstreamCall(provider, country string, success bool, latencyMs float64) {
	status := "success"
	if !success {
		status = "failure"
	}
	Global.IncrCounter("silentpass_upstream_calls_total",
		map[string]string{"provider": provider, "country": country, "status": status})
	Global.ObserveHistogram("silentpass_upstream_latency_ms",
		map[string]string{"provider": provider, "country": country}, latencyMs)
}
