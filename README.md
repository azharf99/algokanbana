# ⚡ AlgoKanbana — High-Performance Algonova Tutor Portal (Go Single-Binary)

Remake berkecepatan tinggi untuk portal administrasi tutor Algonova Indonesia menggunakan **Golang 1.26 + SQLite WAL (Pure-Go tanpa CGO) + Svelte 5 & Tailwind CSS v4** yang ditanam langsung ke dalam satu binary tunggal (`//go:embed`) di atas Docker `FROM scratch`.

## Keunggulan Utama vs Portal Lama (`nano-kanbana.vercel.app`)
- **Kecepatan Ekstrem**: Pembukaan halaman Jadwal & Dasbor turun dari **`11.834–12.175 ms`** menjadi **`< 2 ms`** (SQLite WAL lokal + *background sync*).
- **Docker Image Super Kecil**: Hanya **`~18 MB`** (`FROM scratch`, tanpa Node.js / PHP-FPM di kontainer produksi).
- **Normalisasi Zona Waktu Otomatis**: Timestamp mentah server LMS (`UTC+03:00` Moskow) otomatis dikonversi ke **WIB (`UTC+07:00`)**.
- **100% Aman di Mode Lokal (`SAFE_READONLY_UPSTREAM=true`)**: Proteksi tingkat *HTTP Transport* memastikan tidak ada request `POST`/`PUT`/`PATCH`/`DELETE` kritikal yang terkirim ke server produksi Algonova.
- **5 Bug Aplikasi Lama Diperbaiki**: Validasi grup numerik LMS pada *Laporan ke TL*, input nama/ID fleksibel di *Form Bantuan*, bebas *double-fetch* `router.refresh()`, filter siswa instan di memori, dan deduplikasi artikel SOP.

---

## 1. Menjalankan di Lokal (Tanpa Docker)

```powershell
# 1. Build aset frontend Svelte 5 ke web/dist
cd web
npm install
npm run build
cd ..

# 2. Kompilasi single-binary Golang & jalankan
go build -trimpath -ldflags="-s -w" -o algokanbana.exe ./cmd/server
.\algokanbana.exe
```

Buka **`http://localhost:8090`** di browser.

---

## 2. Build & Jalankan dengan Docker (Lokal maupun VPS)

```powershell
# Build image ultra-kecil FROM scratch
docker build -t algokanbana:latest .

# Cek ukuran image (~18 MB!)
docker images algokanbana:latest

# Jalankan menggunakan Docker Compose (data tersimpan di volume persisten)
docker compose up -d
```
