package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yusufjaelani/pulseops/internal/handler"
	"github.com/yusufjaelani/pulseops/internal/prober"
	"github.com/yusufjaelani/pulseops/internal/store"
)

func setupTestApp(t *testing.T) (*handler.Handler, *store.Store) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed store init: %v", err)
	}
	p := prober.New(s, 1*time.Minute)
	h := handler.New(s, p, nil)
	return h, s
}

func TestHandler_HealthzAndReadyz(t *testing.T) {
	h, s := setupTestApp(t)
	defer s.Close()

	routes := h.Routes()

	// Healthz
	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("unexpected body: %s", rec.Body.String())
	}

	// Readyz
	req2 := httptest.NewRequest("GET", "/readyz", nil)
	rec2 := httptest.NewRecorder()
	routes.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), `"status":"ready"`) {
		t.Errorf("unexpected body: %s", rec2.Body.String())
	}
}

func TestHandler_CreateAndListTargets(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockServer.Close()

	h, s := setupTestApp(t)
	defer s.Close()

	routes := h.Routes()

	// 1. Create target
	body, _ := json.Marshal(map[string]string{
		"name": "Mock Test",
		"url":  mockServer.URL,
	})
	req := httptest.NewRequest("POST", "/api/targets", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. List targets
	reqList := httptest.NewRequest("GET", "/api/targets", nil)
	recList := httptest.NewRecorder()
	routes.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recList.Code)
	}
	if !strings.Contains(recList.Body.String(), "Mock Test") {
		t.Errorf("target not found in response: %s", recList.Body.String())
	}

	// 3. Prometheus metrics output check
	reqMetrics := httptest.NewRequest("GET", "/metrics", nil)
	recMetrics := httptest.NewRecorder()
	routes.ServeHTTP(recMetrics, reqMetrics)

	if recMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /metrics, got %d", recMetrics.Code)
	}
	if !strings.Contains(recMetrics.Body.String(), "pulseops_target_up") {
		t.Errorf("metrics missing pulseops_target_up: %s", recMetrics.Body.String())
	}
}
