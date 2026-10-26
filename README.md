# URL Shortener

Aplikasi pemendek URL dengan Go, Vue, dan PostgreSQL. Termasuk dashboard, autentikasi, audit log, dan QR code.

## Menjalankan

Dengan Docker:

```sh
docker compose up --build
```

Buka `http://localhost:8080`.

Untuk menjalankan lokal, siapkan Go 1.26, Node.js 22, dan PostgreSQL. Jalankan `npm ci`, `npm run build`, lalu `go run ./cmd/server` dengan `DATABASE_URL` yang sesuai.

## Akun default

- Admin: `admin@example.com` / `password`
- User: `user@example.com` / `password`

Ganti kredensial sebelum deployment.

## Konfigurasi utama

| Variabel | Default |
|---|---|
| `DATABASE_URL` | `postgres://postgres:12345@localhost:5432/url_shortener?sslmode=disable` |
| `PUBLIC_BASE_URL` | `http://localhost:8080` |
| `AUTO_MIGRATE` | `true` |
| `SEED_DEFAULT_ACCOUNTS` | `true` |
| `COOKIE_SECURE` | `false` |

Set `COOKIE_SECURE=true` jika HTTPS berakhir di reverse proxy tepercaya. Untuk production, pertimbangkan menjalankan migration terpisah dan menonaktifkan seed akun default.

## Route utama

- `/login`, `/dashboard`, `/urls`, `/users`, `/audit`
- `/api/auth/*`, `/api/urls`, `/api/users`, `/api/audit-logs`
- `/{code}` untuk redirect dan `/healthz` untuk healthcheck

Request mutasi memakai proteksi CSRF. Jalankan `go test ./...` dan `npm run build` untuk pemeriksaan dasar.
