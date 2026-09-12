# Panduan Kontribusi PulseOps 🛠️

Terima kasih telah tertarik berkontribusi pada proyek PulseOps! Panduan ini dirancang agar engineer baru dapat memahami arsitektur sistem, alur kerja development, dan standar kualitas kode yang diterapkan.

---

## 1. Prinsip & Standar Pengembangan

PulseOps dibangun dengan prinsip **Cloud-Native SRE** dan **Vibe Coding**:
- **YAGNI & Minim Dependensi:** Utamakan pustaka standar Go (`net/http`, `crypto/tls`, `log/slog`) untuk performa maksimal dan binary yang ramping.
- **Portabilitas Pure-Go:** Menggunakan driver `modernc.org/sqlite` agar kompilasi binary tidak bergantung pada CGO atau compiler GCC eksternal.
- **Standar Kode Bersih (Zero-Warning):** Kode Go dan frontend harus lolos linter ketat (`gofmt`, SonarQube/SonarLint) dengan 0 issue/code smell.
- **Commit Atomik:** Gunakan format standar [Conventional Commits](https://www.conventionalcommits.org/):
  - `feat: ...` untuk penambahan fitur baru
  - `fix: ...` untuk perbaikan bug
  - `refactor: ...` untuk restrukturisasi kode tanpa mengubah fungsionalitas
  - `docs: ...` untuk pembaruan dokumentasi
  - `test: ...` untuk penambahan atau pembaruan pengujian otomatis

---

## 2. Setup Lingkungan Lokal & Alur Kerja

### Prasyarat
- [Docker & Docker Compose](https://www.docker.com/) (Sangat direkomendasikan)
- *Opsional:* [Go 1.23+](https://go.dev/) jika ingin menjalankan langsung tanpa container

### Perintah Cepat dengan Makefile
```bash
# Menampilkan daftar perintah yang tersedia
make help

# Menjalankan seluruh pengujian unit dan integrasi
make test

# Menjalankan seluruh stack layanan (PulseOps, Prometheus, Grafana)
make docker-compose-up

# Menghentikan seluruh container
make docker-compose-down
```

### Menjalankan Langsung dengan Go (Tanpa Docker)
Jika ingin mengembangkan secara native:
```bash
# 1. Unduh dependensi modul
go mod download

# 2. Jalankan test otomatis
go test -v ./...

# 3. Jalankan aplikasi
# Database SQLite otomatis dibuat di ./pulseops.db
DB_PATH=./pulseops.db PORT=8080 go run ./cmd/server
```

---

## 3. Panduan Menambahkan Fitur Baru

### Menambahkan Metrik Telemetri Baru
1. **Model:** Tambahkan kolom baru pada struct `ProbeLog` di `internal/model/target.go`.
2. **Migrasi Database:** Perbarui skema tabel SQL di `internal/store/store.go` (pada fungsi `InitDB`).
3. **Logika Prober:** Pada `internal/prober/prober.go`, tambahkan pengukuran metrik baru saat `ProbeTarget` dijalankan (contoh: durasi DNS lookup, TTFB, ukuran body).
4. **Ekspor Prometheus:** Pada `internal/handler/handler.go` (`handleMetrics`), daftarkan metrik tersebut sebagai Prometheus Gauge.
5. **Dashboard Web:** Jika diperlukan, tampilkan data baru di `web/app.js` dan `web/index.html`.

### Menambahkan Notifikasi Alert Webhook (Contoh: Discord / Telegram / Slack)
1. Daftarkan variabel `ALERT_WEBHOOK_URL` di `internal/config/config.go`.
2. Di `internal/prober/prober.go`, deteksi perubahan status target (misalnya status sebelumnya `UP`, status saat ini menjadi `DOWN`).
3. Kirim payload HTTP POST ke webhook URL secara asinkron menggunakan goroutine non-blocking.

---

## 4. Checklist Sebelum Mengajukan Pull Request

Sebelum melakukan push atau membuka Pull Request:
- [ ] Pengujian otomatis berhasil 100%: `make test`
- [ ] Format kode rapi: `go fmt ./...`
- [ ] Tidak ada hardcoded credential, secret, atau tautan localhost di tampilan produksi.
- [ ] Build Docker berhasil tanpa error: `make docker-build`
- [ ] SonarLint menunjukkan 0 peringatan (cognitive complexity < 15, aksesibilitas HTML valid).
