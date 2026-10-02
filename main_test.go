package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"job-tracker/database"
	"job-tracker/handlers"
	"job-tracker/models"
)

func setupTestDB(t *testing.T) (*handlers.ApplicationHandler, func()) {
	testDBPath := "test_job_tracker.db"
	_ = os.Remove(testDBPath)

	db, err := database.InitDB(testDBPath)
	if err != nil {
		t.Fatalf("Failed to init test db: %v", err)
	}

	handler := handlers.NewApplicationHandler(db)

	cleanup := func() {
		db.Close()
		_ = os.Remove(testDBPath)
	}

	return handler, cleanup
}

func TestApplicationLifecycle(t *testing.T) {
	handler, cleanup := setupTestDB(t)
	defer cleanup()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/applications", handler.GetAllApplications)
	mux.HandleFunc("POST /api/applications", handler.CreateApplication)
	mux.HandleFunc("PUT /api/applications/{id}/status", handler.UpdateApplicationStatus)
	mux.HandleFunc("DELETE /api/applications/{id}", handler.DeleteApplication)
	mux.HandleFunc("GET /api/applications/{id}/logs", handler.GetApplicationLogs)

	// 1. Test GET /api/applications when empty
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/applications", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", rec.Code)
	}
	var apps []models.Application
	if err := json.Unmarshal(rec.Body.Bytes(), &apps); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}
	if len(apps) != 0 {
		t.Fatalf("Expected 0 applications, got %d", len(apps))
	}

	// 2. Test POST /api/applications
	createPayload := models.CreateApplicationRequest{
		Company:  "Tech Corp",
		Role:     "Golang Backend Developer",
		Platform: "LinkedIn",
		Status:   models.StatusApplied,
		Salary:   15000000,
		Notes:    "Interview ronde 1 via Google Meet",
	}
	payloadBytes, _ := json.Marshal(createPayload)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/applications", bytes.NewReader(payloadBytes))
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var created models.Application
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("Failed to parse created app: %v", err)
	}
	if created.ID <= 0 || created.Company != "Tech Corp" || created.Status != "Applied" {
		t.Fatalf("Created app fields mismatch: %+v", created)
	}

	// 3. Test PUT /api/applications/{id}/status (DB Transaction & Status Log)
	updatePayload := models.UpdateStatusRequest{
		Status: models.StatusInterview,
	}
	updateBytes, _ := json.Marshal(updatePayload)

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("PUT", "/api/applications/1/status", bytes.NewReader(updateBytes))
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on status update, got %d: %s", rec.Code, rec.Body.String())
	}

	// 4. Test GET /api/applications/{id}/logs
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/applications/1/logs", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on logs, got %d", rec.Code)
	}
	var logs []models.StatusLog
	if err := json.Unmarshal(rec.Body.Bytes(), &logs); err != nil {
		t.Fatalf("Failed to parse logs: %v", err)
	}
	// Initial creation log + status update log = at least 2 logs
	if len(logs) < 2 {
		t.Fatalf("Expected at least 2 logs, got %d", len(logs))
	}
	if logs[0].NewStatus != models.StatusInterview || logs[0].OldStatus != models.StatusApplied {
		t.Fatalf("Latest log mismatch: %+v", logs[0])
	}

	// 5. Test DELETE /api/applications/{id}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("DELETE", "/api/applications/1", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on delete, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Verify application is deleted
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/applications", nil)
	mux.ServeHTTP(rec, req)

	var remainingApps []models.Application
	_ = json.Unmarshal(rec.Body.Bytes(), &remainingApps)
	if len(remainingApps) != 0 {
		t.Fatalf("Expected 0 applications after delete, got %d", len(remainingApps))
	}
}
