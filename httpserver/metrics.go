package httpserver

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks server metrics for Prometheus exposition.
type Metrics struct {
	mu sync.RWMutex

	// Request counters by endpoint and status
	requestCount map[string]*atomic.Int64
	errorCount   map[string]*atomic.Int64

	// Latency tracking (histogram buckets in ms)
	latencyBuckets []float64
	latencyCounts  map[string][]atomic.Int64
	latencySum     map[string]*atomic.Int64

	// Query-specific metrics
	queryCount       atomic.Int64
	queryFromCache   atomic.Int64
	queryFromAPI     atomic.Int64
	queryParseErrors atomic.Int64
	queryExecErrors  atomic.Int64

	// Sync metrics
	syncCount      atomic.Int64
	syncErrors     atomic.Int64
	syncDurationMs atomic.Int64
	lastSyncTime   atomic.Int64

	// Cache metrics
	cacheHits   atomic.Int64
	cacheMisses atomic.Int64

	// Graph metrics
	graphQueryCount  atomic.Int64
	graphQueryErrors atomic.Int64

	// Start time for uptime calculation
	startTime time.Time
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	buckets := []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	m := &Metrics{
		requestCount:   make(map[string]*atomic.Int64),
		errorCount:     make(map[string]*atomic.Int64),
		latencyBuckets: buckets,
		latencyCounts:  make(map[string][]atomic.Int64),
		latencySum:     make(map[string]*atomic.Int64),
		startTime:      time.Now(),
	}
	return m
}

// RecordRequest records a request for an endpoint.
func (m *Metrics) RecordRequest(endpoint string, status int, durationMs float64) {
	m.mu.Lock()
	if m.requestCount[endpoint] == nil {
		m.requestCount[endpoint] = &atomic.Int64{}
		m.errorCount[endpoint] = &atomic.Int64{}
		m.latencyCounts[endpoint] = make([]atomic.Int64, len(m.latencyBuckets)+1)
		m.latencySum[endpoint] = &atomic.Int64{}
	}
	m.mu.Unlock()

	m.requestCount[endpoint].Add(1)
	if status >= 400 {
		m.errorCount[endpoint].Add(1)
	}

	// Record latency histogram
	m.latencySum[endpoint].Add(int64(durationMs * 1000)) // store as microseconds for precision
	bucketIdx := len(m.latencyBuckets)
	for i, bound := range m.latencyBuckets {
		if durationMs <= bound {
			bucketIdx = i
			break
		}
	}
	m.latencyCounts[endpoint][bucketIdx].Add(1)
}

// RecordQuery records a query execution.
func (m *Metrics) RecordQuery(fromCache bool, parseError, execError bool) {
	m.queryCount.Add(1)
	if fromCache {
		m.queryFromCache.Add(1)
	} else {
		m.queryFromAPI.Add(1)
	}
	if parseError {
		m.queryParseErrors.Add(1)
	}
	if execError {
		m.queryExecErrors.Add(1)
	}
}

// RecordCacheHit records a cache hit.
func (m *Metrics) RecordCacheHit() {
	m.cacheHits.Add(1)
}

// RecordCacheMiss records a cache miss.
func (m *Metrics) RecordCacheMiss() {
	m.cacheMisses.Add(1)
}

// RecordSync records a sync operation.
func (m *Metrics) RecordSync(durationMs int64, err error) {
	m.syncCount.Add(1)
	m.syncDurationMs.Store(durationMs)
	m.lastSyncTime.Store(time.Now().Unix())
	if err != nil {
		m.syncErrors.Add(1)
	}
}

// RecordGraphQuery records a graph query.
func (m *Metrics) RecordGraphQuery(err error) {
	m.graphQueryCount.Add(1)
	if err != nil {
		m.graphQueryErrors.Add(1)
	}
}

// WritePrometheus writes metrics in Prometheus exposition format.
func (m *Metrics) WritePrometheus(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Uptime
	uptime := time.Since(m.startTime).Seconds()
	fmt.Fprintf(w, "# HELP aha_studio_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE aha_studio_uptime_seconds gauge\n")
	fmt.Fprintf(w, "aha_studio_uptime_seconds %.3f\n\n", uptime)

	// Request counts
	m.mu.RLock()
	endpoints := make([]string, 0, len(m.requestCount))
	for ep := range m.requestCount {
		endpoints = append(endpoints, ep)
	}
	m.mu.RUnlock()
	sort.Strings(endpoints)

	if len(endpoints) > 0 {
		fmt.Fprintf(w, "# HELP aha_studio_http_requests_total Total HTTP requests by endpoint\n")
		fmt.Fprintf(w, "# TYPE aha_studio_http_requests_total counter\n")
		for _, ep := range endpoints {
			m.mu.RLock()
			count := m.requestCount[ep].Load()
			m.mu.RUnlock()
			fmt.Fprintf(w, "aha_studio_http_requests_total{endpoint=%q} %d\n", ep, count)
		}
		fmt.Fprintln(w)

		fmt.Fprintf(w, "# HELP aha_studio_http_errors_total Total HTTP errors by endpoint\n")
		fmt.Fprintf(w, "# TYPE aha_studio_http_errors_total counter\n")
		for _, ep := range endpoints {
			m.mu.RLock()
			count := m.errorCount[ep].Load()
			m.mu.RUnlock()
			fmt.Fprintf(w, "aha_studio_http_errors_total{endpoint=%q} %d\n", ep, count)
		}
		fmt.Fprintln(w)

		// Latency histograms
		fmt.Fprintf(w, "# HELP aha_studio_http_request_duration_ms HTTP request duration in milliseconds\n")
		fmt.Fprintf(w, "# TYPE aha_studio_http_request_duration_ms histogram\n")
		for _, ep := range endpoints {
			m.mu.RLock()
			counts := m.latencyCounts[ep]
			sum := m.latencySum[ep].Load()
			m.mu.RUnlock()

			cumulative := int64(0)
			for i, bound := range m.latencyBuckets {
				cumulative += counts[i].Load()
				fmt.Fprintf(w, "aha_studio_http_request_duration_ms_bucket{endpoint=%q,le=\"%.0f\"} %d\n", ep, bound, cumulative)
			}
			cumulative += counts[len(m.latencyBuckets)].Load()
			fmt.Fprintf(w, "aha_studio_http_request_duration_ms_bucket{endpoint=%q,le=\"+Inf\"} %d\n", ep, cumulative)
			fmt.Fprintf(w, "aha_studio_http_request_duration_ms_sum{endpoint=%q} %.3f\n", ep, float64(sum)/1000)
			fmt.Fprintf(w, "aha_studio_http_request_duration_ms_count{endpoint=%q} %d\n", ep, cumulative)
		}
		fmt.Fprintln(w)
	}

	// Query metrics
	fmt.Fprintf(w, "# HELP aha_studio_queries_total Total AQL queries executed\n")
	fmt.Fprintf(w, "# TYPE aha_studio_queries_total counter\n")
	fmt.Fprintf(w, "aha_studio_queries_total %d\n\n", m.queryCount.Load())

	fmt.Fprintf(w, "# HELP aha_studio_queries_from_cache Queries served from cache\n")
	fmt.Fprintf(w, "# TYPE aha_studio_queries_from_cache counter\n")
	fmt.Fprintf(w, "aha_studio_queries_from_cache %d\n\n", m.queryFromCache.Load())

	fmt.Fprintf(w, "# HELP aha_studio_queries_from_api Queries served from API\n")
	fmt.Fprintf(w, "# TYPE aha_studio_queries_from_api counter\n")
	fmt.Fprintf(w, "aha_studio_queries_from_api %d\n\n", m.queryFromAPI.Load())

	fmt.Fprintf(w, "# HELP aha_studio_query_parse_errors AQL parse errors\n")
	fmt.Fprintf(w, "# TYPE aha_studio_query_parse_errors counter\n")
	fmt.Fprintf(w, "aha_studio_query_parse_errors %d\n\n", m.queryParseErrors.Load())

	fmt.Fprintf(w, "# HELP aha_studio_query_exec_errors Query execution errors\n")
	fmt.Fprintf(w, "# TYPE aha_studio_query_exec_errors counter\n")
	fmt.Fprintf(w, "aha_studio_query_exec_errors %d\n\n", m.queryExecErrors.Load())

	// Cache metrics
	fmt.Fprintf(w, "# HELP aha_studio_cache_hits Cache hits\n")
	fmt.Fprintf(w, "# TYPE aha_studio_cache_hits counter\n")
	fmt.Fprintf(w, "aha_studio_cache_hits %d\n\n", m.cacheHits.Load())

	fmt.Fprintf(w, "# HELP aha_studio_cache_misses Cache misses\n")
	fmt.Fprintf(w, "# TYPE aha_studio_cache_misses counter\n")
	fmt.Fprintf(w, "aha_studio_cache_misses %d\n\n", m.cacheMisses.Load())

	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()
	total := hits + misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	fmt.Fprintf(w, "# HELP aha_studio_cache_hit_rate Cache hit rate (0-1)\n")
	fmt.Fprintf(w, "# TYPE aha_studio_cache_hit_rate gauge\n")
	fmt.Fprintf(w, "aha_studio_cache_hit_rate %.4f\n\n", hitRate)

	// Sync metrics
	fmt.Fprintf(w, "# HELP aha_studio_sync_total Total sync operations\n")
	fmt.Fprintf(w, "# TYPE aha_studio_sync_total counter\n")
	fmt.Fprintf(w, "aha_studio_sync_total %d\n\n", m.syncCount.Load())

	fmt.Fprintf(w, "# HELP aha_studio_sync_errors Sync errors\n")
	fmt.Fprintf(w, "# TYPE aha_studio_sync_errors counter\n")
	fmt.Fprintf(w, "aha_studio_sync_errors %d\n\n", m.syncErrors.Load())

	fmt.Fprintf(w, "# HELP aha_studio_sync_duration_ms Last sync duration in milliseconds\n")
	fmt.Fprintf(w, "# TYPE aha_studio_sync_duration_ms gauge\n")
	fmt.Fprintf(w, "aha_studio_sync_duration_ms %d\n\n", m.syncDurationMs.Load())

	lastSync := m.lastSyncTime.Load()
	fmt.Fprintf(w, "# HELP aha_studio_last_sync_timestamp_seconds Unix timestamp of last sync\n")
	fmt.Fprintf(w, "# TYPE aha_studio_last_sync_timestamp_seconds gauge\n")
	fmt.Fprintf(w, "aha_studio_last_sync_timestamp_seconds %d\n\n", lastSync)

	// Graph metrics
	fmt.Fprintf(w, "# HELP aha_studio_graph_queries_total Total graph queries\n")
	fmt.Fprintf(w, "# TYPE aha_studio_graph_queries_total counter\n")
	fmt.Fprintf(w, "aha_studio_graph_queries_total %d\n\n", m.graphQueryCount.Load())

	fmt.Fprintf(w, "# HELP aha_studio_graph_query_errors Graph query errors\n")
	fmt.Fprintf(w, "# TYPE aha_studio_graph_query_errors counter\n")
	fmt.Fprintf(w, "aha_studio_graph_query_errors %d\n", m.graphQueryErrors.Load())
}

// handleMetrics handles GET /metrics.
func (s *Server) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	if s.metrics == nil {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintln(w, "# Metrics not enabled")
		return
	}
	s.metrics.WritePrometheus(w)
}
