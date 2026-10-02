package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"job-tracker/database"
	"job-tracker/handlers"
)

func main() {
	// Muat file .env jika tersedia
	if err := godotenv.Load(); err != nil {
		log.Println("Info: File .env tidak ditemukan, menggunakan environment variable bawaan / default")
	}

	port := os.Getenv("PORT")
	if strings.TrimSpace(port) == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if strings.TrimSpace(dbPath) == "" {
		dbPath = "job_tracker.db"
	}

	// Inisialisasi Database SQLite
	db, err := database.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Fatal: Gagal menginisialisasi database (%s): %v", dbPath, err)
	}
	defer db.Close()
	log.Printf("Database SQLite berhasil terhubung: %s", dbPath)

	handler := handlers.NewApplicationHandler(db)

	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("GET /api/applications", handler.GetAllApplications)
	mux.HandleFunc("POST /api/applications", handler.CreateApplication)
	mux.HandleFunc("PUT /api/applications/{id}/status", handler.UpdateApplicationStatus)
	mux.HandleFunc("DELETE /api/applications/{id}", handler.DeleteApplication)
	mux.HandleFunc("GET /api/applications/{id}/logs", handler.GetApplicationLogs)

	// Static frontend
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	// Root path menyajikan static/index.html
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./static/index.html")
	})

	// Logging & CORS Middleware
	loggedMux := loggingMiddleware(corsMiddleware(mux))

	serverAddr := ":" + port
	log.Printf("==================================================")
	log.Printf(" Job Application Tracker Server Aktif!")
	log.Printf(" Buka di browser: http://localhost:%s", port)
	log.Printf(" Database: %s", dbPath)
	log.Printf("==================================================")

	if err := http.ListenAndServe(serverAddr, loggedMux); err != nil {
		log.Fatalf("Server berhenti dengan error: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// Logging untuk API requests
		if strings.HasPrefix(r.URL.Path, "/api/") {
			log.Printf("[%s] %s %s took %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
		}
	})
}
