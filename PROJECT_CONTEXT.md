# Project Context: Job Application Tracker

## Tech Stack & Environment
- Language: Golang (Go 1.22+, net/http standar)
- Database: SQLite (library: modernc.org/sqlite)
- Environment Variables: .env (library: github.com/joho/godotenv)
- Frontend: HTML5, Vanilla JavaScript, Tailwind CSS via CDN (tanpa framework/bundler rumit)
- Port: Default 8080 (diatur via .env)

## Database Schema & Rules
1. Table `applications`:
   - id (INTEGER PRIMARY KEY AUTOINCREMENT)
   - company (TEXT)
   - role (TEXT)
   - platform (TEXT)
   - status (TEXT) -> 'Applied', 'Interview', 'Offering', 'TTD Kontrak', 'Rejected'
   - salary (INTEGER)
   - notes (TEXT)
   - applied_at (DATETIME)
   - updated_at (DATETIME)

2. Table `status_logs`:
   - id (INTEGER PRIMARY KEY AUTOINCREMENT)
   - application_id (INTEGER, FK ke applications.id)
   - old_status (TEXT)
   - new_status (TEXT)
   - changed_at (DATETIME)

## Business Logic & Transactions
- Wajib menggunakan Database Transaction (`tx.Begin()`, `tx.Commit()`, `tx.Rollback()`) saat memperbarui status lamaran di tabel `applications` sekaligus mencatat entri log perubahan status di `status_logs`.

## API Endpoints (CRUD)
- GET /api/applications (Read all)
- POST /api/applications (Create new)
- PUT /api/applications/{id}/status (Update status + Log via DB Transaction)
- DELETE /api/applications/{id} (Delete application)

## Frontend Scope
- Single-page application di `static/index.html`.
- Form untuk input lamaran baru.
- Tampilan list lamaran dengan visual badge status, tombol update status, dan tombol hapus.