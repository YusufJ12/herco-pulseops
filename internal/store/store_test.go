package store_test

import (
	"testing"

	"github.com/yusufjaelani/pulseops/internal/model"
	"github.com/yusufjaelani/pulseops/internal/store"
)

func TestStore_AddAndListTargets(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	target, err := s.AddTarget("Portfolio", "https://portfolioyusufjaelani.vercel.app")
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}

	if target.ID == 0 {
		t.Errorf("expected target ID > 0, got %d", target.ID)
	}

	targets, err := s.ListTargets()
	if err != nil {
		t.Fatalf("failed to list targets: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}

	if targets[0].Name != "Portfolio" {
		t.Errorf("expected name 'Portfolio', got %s", targets[0].Name)
	}
}

func TestStore_ProbeLogsAndSummary(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	target, err := s.AddTarget("Test API", "https://api.example.com")
	if err != nil {
		t.Fatalf("failed to add target: %v", err)
	}

	// Add 2 probes: 1 up, 1 down
	err = s.SaveProbeLog(&model.ProbeLog{
		TargetID:      target.ID,
		StatusCode:    200,
		IsUp:          true,
		LatencyMs:     100,
		SSLExpiryDays: 60,
		Headers:       `{"server":"cloudflare"}`,
	})
	if err != nil {
		t.Fatalf("failed to save probe log: %v", err)
	}

	err = s.SaveProbeLog(&model.ProbeLog{
		TargetID:      target.ID,
		StatusCode:    500,
		IsUp:          false,
		LatencyMs:     200,
		SSLExpiryDays: 60,
		ErrorMsg:      "Internal Server Error",
	})
	if err != nil {
		t.Fatalf("failed to save probe log: %v", err)
	}

	summaries, err := s.GetTargetSummaries()
	if err != nil {
		t.Fatalf("failed to get summaries: %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	sum := summaries[0]
	if sum.TotalProbes != 2 {
		t.Errorf("expected 2 probes, got %d", sum.TotalProbes)
	}

	if sum.UptimePercent != 50.0 {
		t.Errorf("expected 50%% uptime, got %f", sum.UptimePercent)
	}

	if sum.AvgLatencyMs != 150.0 {
		t.Errorf("expected avg latency 150ms, got %f", sum.AvgLatencyMs)
	}
}

func TestStore_DeleteTargetCascade(t *testing.T) {
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()

	target, _ := s.AddTarget("To Delete", "https://delete.me")
	_ = s.SaveProbeLog(&model.ProbeLog{
		TargetID:   target.ID,
		StatusCode: 200,
		IsUp:       true,
		LatencyMs:  50,
	})

	err = s.DeleteTarget(target.ID)
	if err != nil {
		t.Fatalf("failed to delete target: %v", err)
	}

	targets, _ := s.ListTargets()
	if len(targets) != 0 {
		t.Errorf("expected 0 targets after deletion, got %d", len(targets))
	}
}
