package handler

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/yusufjaelani/pulseops/internal/prober"
	"github.com/yusufjaelani/pulseops/internal/store"
)

type Handler struct {
	store  *store.Store
	prober *prober.Prober
	webFS  fs.FS
}

func New(s *store.Store, p *prober.Prober, webFS fs.FS) *Handler {
	return &Handler{
		store:  s,
		prober: p,
		webFS:  webFS,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health & Ops endpoints
	mux.HandleFunc("GET /healthz", h.handleHealthz)
	mux.HandleFunc("GET /readyz", h.handleReadyz)
	mux.HandleFunc("GET /metrics", h.handleMetrics)

	// API endpoints
	mux.HandleFunc("GET /api/targets", h.handleListTargets)
	mux.HandleFunc("POST /api/targets", h.handleCreateTarget)
	mux.HandleFunc("DELETE /api/targets/{id}", h.handleDeleteTarget)
	mux.HandleFunc("POST /api/targets/{id}/probe", h.handleManualProbe)
	mux.HandleFunc("GET /api/targets/{id}/history", h.handleTargetHistory)

	// Web UI SPA
	if h.webFS != nil {
		fileServer := http.FileServer(http.FS(h.webFS))
		mux.Handle("/", fileServer)
	}

	return withLogging(mux)
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleReadyz(w http.ResponseWriter, r *http.Request) {
	// Verify DB is queryable
	if _, err := h.store.ListTargets(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.store.GetTargetSummaries()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	fmt.Fprintf(w, "# HELP pulseops_targets_total Total number of monitored probe targets\n")
	fmt.Fprintf(w, "# TYPE pulseops_targets_total gauge\n")
	fmt.Fprintf(w, "pulseops_targets_total %d\n\n", len(summaries))

	fmt.Fprintf(w, "# HELP pulseops_target_up Current status of the target (1 = UP, 0 = DOWN)\n")
	fmt.Fprintf(w, "# TYPE pulseops_target_up gauge\n")
	for _, s := range summaries {
		upVal := 0
		if s.LastProbe != nil && s.LastProbe.IsUp {
			upVal = 1
		}
		fmt.Fprintf(w, "pulseops_target_up{id=\"%d\",name=\"%s\",url=\"%s\"} %d\n",
			s.Target.ID, escapeLabel(s.Target.Name), escapeLabel(s.Target.URL), upVal)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP pulseops_target_latency_ms Last probe latency in milliseconds\n")
	fmt.Fprintf(w, "# TYPE pulseops_target_latency_ms gauge\n")
	for _, s := range summaries {
		var lat int64 = 0
		if s.LastProbe != nil {
			lat = s.LastProbe.LatencyMs
		}
		fmt.Fprintf(w, "pulseops_target_latency_ms{id=\"%d\",name=\"%s\",url=\"%s\"} %d\n",
			s.Target.ID, escapeLabel(s.Target.Name), escapeLabel(s.Target.URL), lat)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP pulseops_target_ssl_expiry_days Days until SSL certificate expires\n")
	fmt.Fprintf(w, "# TYPE pulseops_target_ssl_expiry_days gauge\n")
	for _, s := range summaries {
		days := 0
		if s.LastProbe != nil {
			days = s.LastProbe.SSLExpiryDays
		}
		fmt.Fprintf(w, "pulseops_target_ssl_expiry_days{id=\"%d\",name=\"%s\",url=\"%s\"} %d\n",
			s.Target.ID, escapeLabel(s.Target.Name), escapeLabel(s.Target.URL), days)
	}
	fmt.Fprintf(w, "\n")

	fmt.Fprintf(w, "# HELP pulseops_target_uptime_percent Uptime percentage over historical probes\n")
	fmt.Fprintf(w, "# TYPE pulseops_target_uptime_percent gauge\n")
	for _, s := range summaries {
		fmt.Fprintf(w, "pulseops_target_uptime_percent{id=\"%d\",name=\"%s\",url=\"%s\"} %.2f\n",
			s.Target.ID, escapeLabel(s.Target.Name), escapeLabel(s.Target.URL), s.UptimePercent)
	}
}

func (h *Handler) handleListTargets(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.store.GetTargetSummaries()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, summaries)
}

type createTargetReq struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func (h *Handler) handleCreateTarget(w http.ResponseWriter, r *http.Request) {
	var req createTargetReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.URL = strings.TrimSpace(req.URL)

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
		return
	}
	if err := prober.ValidateURL(req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	target, err := h.store.AddTarget(req.Name, req.URL)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "target with this URL already exists or failed to save"})
		return
	}

	// Trigger initial probe immediately
	lastProbe, _ := h.prober.ProbeTarget(target)

	summary := map[string]any{
		"target":         target,
		"last_probe":     lastProbe,
		"uptime_percent": 100.0,
		"avg_latency_ms": 0.0,
		"total_probes":   1,
	}
	if lastProbe != nil {
		summary["avg_latency_ms"] = float64(lastProbe.LatencyMs)
	}

	writeJSON(w, http.StatusCreated, summary)
}

func (h *Handler) handleDeleteTarget(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target id"})
		return
	}

	if err := h.store.DeleteTarget(id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "target deleted"})
}

func (h *Handler) handleManualProbe(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target id"})
		return
	}

	target, err := h.store.GetTarget(id)
	if err != nil || target == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "target not found"})
		return
	}

	log, err := h.prober.ProbeTarget(target)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, log)
}

func (h *Handler) handleTargetHistory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target id"})
		return
	}

	limit := 30
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.store.GetProbeHistory(id, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, history)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func escapeLabel(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Terse logging for non-metrics endpoints
		if r.URL.Path != "/metrics" && r.URL.Path != "/healthz" {
			slog.Debug("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start).String(),
			)
		}
	})
}
