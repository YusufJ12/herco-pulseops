package prober

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/yusufjaelani/pulseops/internal/model"
	"github.com/yusufjaelani/pulseops/internal/store"
)

type Prober struct {
	store      *store.Store
	httpClient *http.Client
	interval   time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

func New(s *store.Store, interval time.Duration) *Prober {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
		ResponseHeaderTimeout: 10 * time.Second,
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("stopped after 5 redirects")
			}
			return nil
		},
	}

	return &Prober{
		store:      s,
		httpClient: client,
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

func (p *Prober) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		slog.Info("Prober daemon started", "interval", p.interval)

		// Initial probe tick immediately
		p.probeAll()

		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.probeAll()
			case <-p.stopCh:
				slog.Info("Prober daemon stopping")
				return
			}
		}
	}()
}

func (p *Prober) Stop() {
	close(p.stopCh)
	p.wg.Wait()
}

func (p *Prober) probeAll() {
	targets, err := p.store.ListTargets()
	if err != nil {
		slog.Error("Failed to list targets for probing", "error", err)
		return
	}

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t model.Target) {
			defer wg.Done()
			_, err := p.ProbeTarget(&t)
			if err != nil {
				slog.Warn("Probe failed", "target", t.URL, "error", err)
			}
		}(target)
	}
	wg.Wait()
}

func (p *Prober) ProbeTarget(target *model.Target) (*model.ProbeLog, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "PulseOps-ServiceProbe/1.0 (+https://github.com/yusufjaelani/pulseops)")

	start := time.Now()
	resp, err := p.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()

	probe := &model.ProbeLog{
		TargetID:   target.ID,
		LatencyMs:  latency,
		IsUp:       false,
		StatusCode: 0,
	}

	if err != nil {
		probe.ErrorMsg = err.Error()
		if saveErr := p.store.SaveProbeLog(probe); saveErr != nil {
			slog.Error("Failed to save failed probe log", "error", saveErr)
		}
		return probe, nil
	}
	defer resp.Body.Close()

	probe.StatusCode = resp.StatusCode
	probe.IsUp = resp.StatusCode >= 200 && resp.StatusCode < 400

	// Extract interesting headers
	headersMap := make(map[string]string)
	interestingHeaders := []string{
		"server",
		"content-type",
		"x-vercel-cache",
		"x-vercel-id",
		"cf-ray",
		"strict-transport-security",
	}
	for _, h := range interestingHeaders {
		if val := resp.Header.Get(h); val != "" {
			headersMap[h] = val
		}
	}
	if len(headersMap) > 0 {
		if b, err := json.Marshal(headersMap); err == nil {
			probe.Headers = string(b)
		}
	}

	// Calculate SSL certificate expiry if HTTPS
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		remaining := time.Until(cert.NotAfter)
		probe.SSLExpiryDays = int(remaining.Hours() / 24)
	}

	if !probe.IsUp {
		probe.ErrorMsg = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if err := p.store.SaveProbeLog(probe); err != nil {
		slog.Error("Failed to save probe log", "error", err)
		return nil, err
	}

	slog.Info("Probe finished",
		"target", target.Name,
		"url", target.URL,
		"status", probe.StatusCode,
		"latency_ms", probe.LatencyMs,
		"ssl_days", probe.SSLExpiryDays,
		"is_up", probe.IsUp,
	)

	return probe, nil
}

// Helper to sanitize/validate URL format
func ValidateURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "https://") {
		return fmt.Errorf("url must start with http:// or https://")
	}
	return nil
}
