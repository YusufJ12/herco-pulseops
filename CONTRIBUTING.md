# Contributing to PulseOps 🛠️

Thank you for contributing to PulseOps! This guide provides everything you need to know to get started, understand our architecture, and maintain our high standards for code quality and reliability.

---

## 1. Development Principles & Standards

PulseOps is designed according to **Cloud-Native SRE** and **Vibe Coding** best practices:
- **YAGNI & Zero-Bloat:** Rely on Go's standard library (`net/http`, `crypto/tls`, `log/slog`) whenever possible.
- **Pure-Go Portability:** We use `modernc.org/sqlite` so the binary compiles without CGO or external GCC dependencies.
- **Zero-Warning Code Quality:** All Go and frontend code must satisfy strict linters (`gofmt`, SonarQube/SonarLint) with 0 errors/code smells.
- **Atomic Commits:** Follow [Conventional Commits](https://www.conventionalcommits.org/):
  - `feat: ...` for new features
  - `fix: ...` for bug fixes
  - `refactor: ...` for code improvements without behavior changes
  - `docs: ...` for documentation updates
  - `test: ...` for test additions

---

## 2. Local Setup & Workflow

### Prerequisites
- [Docker & Docker Compose](https://www.docker.com/) (Recommended)
- *Optional:* [Go 1.23+](https://go.dev/) if running outside Docker

### Fast Track with Makefile
```bash
# Display all available commands
make help

# Run all unit and integration tests
make test

# Start the full stack (PulseOps, Prometheus, Grafana)
make docker-compose-up

# Stop all containers
make docker-compose-down
```

### Running Natively with Go
If you prefer developing natively without containers:
```bash
# 1. Download dependencies
go mod download

# 2. Run automated tests
go test -v ./...

# 3. Run the application
# Default DB will be created at ./pulseops.db
DB_PATH=./pulseops.db PORT=8080 go run ./cmd/server
```

---

## 3. How to Extend the Application

### Adding a New Telemetry Metric
1. **Model:** Add new fields to `internal/model/target.go` (in `ProbeLog` struct).
2. **Store Migration:** Update SQL table schema in `internal/store/store.go` (`InitDB` method).
3. **Prober Logic:** In `internal/prober/prober.go`, measure the metric during `ProbeTarget` (e.g. DNS lookup time, TTFB, body size).
4. **Prometheus Metrics:** In `internal/handler/handler.go` (`handleMetrics`), expose the new metric as a Prometheus Gauge.
5. **Dashboard:** If needed, add the metric to `web/app.js` and `web/index.html`.

### Adding Alerting Webhooks (e.g., Discord / Slack)
1. Inject `ALERT_WEBHOOK_URL` in `internal/config/config.go`.
2. In `internal/prober/prober.go`, detect state changes (e.g. previous status was UP, current status is DOWN).
3. Send an asynchronous HTTP POST payload using a non-blocking goroutine.

---

## 4. Code Quality & Verification Checklist

Before submitting a Pull Request:
- [ ] Automated tests pass: `make test`
- [ ] Code is formatted: `go fmt ./...`
- [ ] No hardcoded secrets or environment-specific localhost links in production views.
- [ ] Docker image builds cleanly: `make docker-build`
- [ ] SonarLint reports 0 cognitive complexity or accessibility warnings.
