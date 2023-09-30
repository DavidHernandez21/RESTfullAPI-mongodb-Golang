package observability

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

var TotalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Number of get requests.",
	},
	[]string{"path", "method"},
)

var ResponseStatus = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "response_status",
		Help: "Status of HTTP response",
	},
	[]string{"status", "method"},
)

var HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "http_response_time_seconds",
	Help: "Duration of HTTP requests.",
}, []string{"path", "method"})

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		method, err := route.GetMethods()

		if err != nil {
			method = []string{"unknown"}
		}

		path, err := route.GetPathTemplate()

		// log.Printf("path: %s\n", path)
		if path == os.Getenv("METRICS_ENDPOINT") {
			next.ServeHTTP(w, r)
			return
		}

		if err != nil {
			path = "unknown"
		}

		timer := prometheus.NewTimer(HTTPDuration.WithLabelValues(path, method[0]))
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)

		statusCode := rw.statusCode

		ResponseStatus.WithLabelValues(strconv.Itoa(statusCode), method[0]).Inc()
		TotalRequests.WithLabelValues(path, method[0]).Inc()

		timer.ObserveDuration()
	})
}
