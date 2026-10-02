# 📌 Job Application Tracker

Aplikasi web sederhana untuk mencatat dan melacak progres lamaran kerja (pipeline tracking). Proyek ini dibangun sebagai alat produktivitas pribadi sekaligus eksplorasi backend dasar Golang dan transaksi database relasional.

---

## 🛠️ Tech Stack & Architecture

- **Backend:** Golang (standar library net/http)
- **Database:** SQLite (driver: modernc.org/sqlite)
- **Configuration:** .env (library: github.com/joho/godotenv)
- **Frontend:** Vanilla HTML5, JavaScript (Fetch API), Tailwind CSS via CDN

---

## ✨ Fitur Utama

- **CRUD Lamaran:** Tambah lamaran baru, melihat daftar lamaran, ubah status, dan hapus lamaran.
- **Atomic Database Transaction:** Saat status lamaran diperbarui, sistem menggunakan transaksi database (tx.Begin, tx.Commit, tx.Rollback) untuk mengupdate tabel applications sekaligus mencatat riwayat perubahan status ke tabel audit status_logs secara konsisten.
- **Zero Heavy Bundler:** Frontend ringan tanpa setup build tool rumit.

---

## 💡 Catatan Pengembangan (Development Approach)

Proyek ini dikembangkan dengan pendekatan **AI pair-programming (Antigravity IDE)**:
- **Arsitektur & Spesifikasi:** Skema database relasional, aturan transaksi data atomik, dan kontrak endpoint REST API dirancang dan dispesifikasikan secara manual melalui panduan PROJECT_CONTEXT.md.
- **Scaffolding:** Implementasi kode awal dibantu oleh AI coding agent berbasis spesifikasi yang telah ditentukan.
- **Tujuan Pembelajaran:** Proyek ini difokuskan untuk memahami alur kerja backend modern, integrasi database transaksi atomik di Go, serta manajemen konfigurasi environment yang aman.

---

## 🚀 Cara Menjalankan Secara Lokal

### 1. Prasyarat
- Go 1.22 atau lebih baru terpasang di sistem.

### 2. Instalasi & Setup

Clone repositori ini:
git clone https://github.com/afakihfaj/job-tracker
cd job-tracker

Siapkan environment:
cp .env.example .env

Jalankan server:
go run main.go

Buka browser di:
http://localhost:8085

---

## 📂 Struktur Direktori

- database/            : Setup koneksi dan inisialisasi tabel SQLite
- handlers/            : REST API handler & database transactions
- static/              : Frontend (HTML & Tailwind UI)
- .env.example         : Template konfigurasi environment
- .gitignore           : Menjaga database lokal dan .env tidak ter-push
- main.go              : Entry point aplikasi
- PROJECT_CONTEXT.md   : Spesifikasi arsitektur proyek
