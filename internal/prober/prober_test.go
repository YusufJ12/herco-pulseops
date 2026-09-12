package prober_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yusufjaelani/pulseops/internal/prober"
	"github.com/yusufjaelani/pulseops/internal/store"
)

func TestProber_ProbeSuccessAndFailure(t *testing.T) {
	// 1. Mock server that returns 200 OK
	server200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-vercel-cache", "HIT")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server200.Close()

	// 2. Mock server that returns 500 Error
	server500 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error"))
	}))
	defer server500.Close()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	targetOk, err := s.AddTarget("Mock 200", server200.URL)
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}

	targetErr, err := s.AddTarget("Mock 500", server500.URL)
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}

	p := prober.New(s, 1*time.Minute)

	// Test 200 probe
	resOk, err := p.ProbeTarget(targetOk)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if !resOk.IsUp {
		t.Errorf("expected target to be UP, got DOWN")
	}
	if resOk.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resOk.StatusCode)
	}

	// Test 500 probe
	resErr, err := p.ProbeTarget(targetErr)
	if err != nil {
		t.Fatalf("probe error: %v", err)
	}
	if resErr.IsUp {
		t.Errorf("expected target to be DOWN, got UP")
	}
	if resErr.StatusCode != 500 {
		t.Errorf("expected 500, got %d", resErr.StatusCode)
	}
}

func TestValidateURL(t *testing.T) {
	if err := prober.ValidateURL("https://portfolioyusufjaelani.vercel.app"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := prober.ValidateURL("http://localhost:8080"); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}
	if err := prober.ValidateURL("ftp://not-supported"); err == nil {
		t.Errorf("expected invalid, got nil error")
	}
	if err := prober.ValidateURL("google.com"); err == nil {
		t.Errorf("expected invalid without scheme, got nil error")
	}
}
