package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"job-tracker/models"
)

type ApplicationHandler struct {
	DB *sql.DB
}

func NewApplicationHandler(db *sql.DB) *ApplicationHandler {
	return &ApplicationHandler{DB: db}
}

// GET /api/applications
func (h *ApplicationHandler) GetAllApplications(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	searchQuery := r.URL.Query().Get("search")

	query := `SELECT id, company, role, platform, status, salary, notes, applied_at, updated_at 
	          FROM applications WHERE 1=1`
	var args []any

	if statusFilter != "" {
		query += " AND status = ?"
		args = append(args, statusFilter)
	}

	if searchQuery != "" {
		query += " AND (company LIKE ? OR role LIKE ? OR platform LIKE ?)"
		likePattern := "%" + searchQuery + "%"
		args = append(args, likePattern, likePattern, likePattern)
	}

	query += " ORDER BY applied_at DESC, id DESC"

	rows, err := h.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal mengambil data lamaran: "+err.Error())
		return
	}
	defer rows.Close()

	applications := make([]models.Application, 0)
	for rows.Next() {
		var app models.Application
		var appliedAtStr, updatedAtStr string

		err := rows.Scan(
			&app.ID,
			&app.Company,
			&app.Role,
			&app.Platform,
			&app.Status,
			&app.Salary,
			&app.Notes,
			&appliedAtStr,
			&updatedAtStr,
		)
		if err != nil {
			sendError(w, http.StatusInternalServerError, "Gagal memproses data lamaran: "+err.Error())
			return
		}

		app.AppliedAt = parseTimeFlexible(appliedAtStr)
		app.UpdatedAt = parseTimeFlexible(updatedAtStr)
		applications = append(applications, app)
	}

	if err := rows.Err(); err != nil {
		sendError(w, http.StatusInternalServerError, "Error saat membaca data: "+err.Error())
		return
	}

	sendJSON(w, http.StatusOK, applications)
}

// POST /api/applications
func (h *ApplicationHandler) CreateApplication(w http.ResponseWriter, r *http.Request) {
	var req models.CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Format JSON tidak valid: "+err.Error())
		return
	}

	req.Company = strings.TrimSpace(req.Company)
	req.Role = strings.TrimSpace(req.Role)
	req.Platform = strings.TrimSpace(req.Platform)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.Company == "" {
		sendError(w, http.StatusBadRequest, "Nama perusahaan (company) wajib diisi")
		return
	}
	if req.Role == "" {
		sendError(w, http.StatusBadRequest, "Posisi/jabatan (role) wajib diisi")
		return
	}

	if req.Status == "" {
		req.Status = models.StatusApplied
	} else if !models.IsValidStatus(req.Status) {
		sendError(w, http.StatusBadRequest, "Status tidak valid. Pilihan: 'Applied', 'Interview', 'Offering', 'TTD Kontrak', 'Rejected'")
		return
	}

	now := time.Now().UTC()
	appliedAt := now
	if req.AppliedAt != "" {
		parsed := parseTimeFlexible(req.AppliedAt)
		if !parsed.IsZero() {
			appliedAt = parsed
		}
	}

	// Begin transaction to create application and initial status log
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal memulai transaksi: "+err.Error())
		return
	}
	defer tx.Rollback()

	query := `INSERT INTO applications (company, role, platform, status, salary, notes, applied_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.ExecContext(r.Context(), query,
		req.Company,
		req.Role,
		req.Platform,
		req.Status,
		req.Salary,
		req.Notes,
		appliedAt.Format(time.RFC3339),
		now.Format(time.RFC3339),
	)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal menyimpan lamaran: "+err.Error())
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal mendapatkan ID lamaran baru: "+err.Error())
		return
	}

	// Insert initial status log
	logQuery := `INSERT INTO status_logs (application_id, old_status, new_status, changed_at) VALUES (?, ?, ?, ?)`
	if _, err := tx.ExecContext(r.Context(), logQuery, id, "-", req.Status, now.Format(time.RFC3339)); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal mencatat log status awal: "+err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal melakukan commit transaksi: "+err.Error())
		return
	}

	createdApp := models.Application{
		ID:        id,
		Company:   req.Company,
		Role:      req.Role,
		Platform:  req.Platform,
		Status:    req.Status,
		Salary:    req.Salary,
		Notes:     req.Notes,
		AppliedAt: appliedAt,
		UpdatedAt: now,
	}

	sendJSON(w, http.StatusCreated, createdApp)
}

// PUT /api/applications/{id}/status
// Wajib menggunakan Database Transaction (tx.Begin(), tx.Commit(), tx.Rollback())
func (h *ApplicationHandler) UpdateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		sendError(w, http.StatusBadRequest, "ID lamaran tidak valid")
		return
	}

	var req models.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Format JSON tidak valid: "+err.Error())
		return
	}

	req.Status = strings.TrimSpace(req.Status)
	if !models.IsValidStatus(req.Status) {
		sendError(w, http.StatusBadRequest, "Status tidak valid. Pilihan: 'Applied', 'Interview', 'Offering', 'TTD Kontrak', 'Rejected'")
		return
	}

	// Database Transaction
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal memulai transaksi: "+err.Error())
		return
	}
	defer tx.Rollback()

	var oldStatus string
	checkQuery := `SELECT status FROM applications WHERE id = ?`
	err = tx.QueryRowContext(r.Context(), checkQuery, id).Scan(&oldStatus)
	if errors.Is(err, sql.ErrNoRows) {
		sendError(w, http.StatusNotFound, "Data lamaran dengan ID tersebut tidak ditemukan")
		return
	} else if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal memeriksa data lamaran: "+err.Error())
		return
	}

	now := time.Now().UTC()

	// Update applications table
	updateQuery := `UPDATE applications SET status = ?, updated_at = ? WHERE id = ?`
	if _, err := tx.ExecContext(r.Context(), updateQuery, req.Status, now.Format(time.RFC3339), id); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal memperbarui status lamaran: "+err.Error())
		return
	}

	// Insert into status_logs table
	logQuery := `INSERT INTO status_logs (application_id, old_status, new_status, changed_at) VALUES (?, ?, ?, ?)`
	if _, err := tx.ExecContext(r.Context(), logQuery, id, oldStatus, req.Status, now.Format(time.RFC3339)); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal mencatat log perubahan status: "+err.Error())
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal commit perubahan status: "+err.Error())
		return
	}

	// Query updated record to return
	var updatedApp models.Application
	var appliedAtStr, updatedAtStr string
	err = h.DB.QueryRowContext(r.Context(),
		`SELECT id, company, role, platform, status, salary, notes, applied_at, updated_at FROM applications WHERE id = ?`,
		id,
	).Scan(
		&updatedApp.ID,
		&updatedApp.Company,
		&updatedApp.Role,
		&updatedApp.Platform,
		&updatedApp.Status,
		&updatedApp.Salary,
		&updatedApp.Notes,
		&appliedAtStr,
		&updatedAtStr,
	)
	if err == nil {
		updatedApp.AppliedAt = parseTimeFlexible(appliedAtStr)
		updatedApp.UpdatedAt = parseTimeFlexible(updatedAtStr)
	}

	sendJSON(w, http.StatusOK, map[string]any{
		"message":     "Status lamaran berhasil diperbarui",
		"old_status":  oldStatus,
		"new_status":  req.Status,
		"application": updatedApp,
	})
}

// DELETE /api/applications/{id}
func (h *ApplicationHandler) DeleteApplication(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		sendError(w, http.StatusBadRequest, "ID lamaran tidak valid")
		return
	}

	// Begin transaction to ensure status_logs and application are cleaned up
	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal memulai transaksi: "+err.Error())
		return
	}
	defer tx.Rollback()

	// Check existence
	var exists bool
	err = tx.QueryRowContext(r.Context(), "SELECT EXISTS(SELECT 1 FROM applications WHERE id = ?)", id).Scan(&exists)
	if err != nil || !exists {
		sendError(w, http.StatusNotFound, "Data lamaran tidak ditemukan")
		return
	}

	// Delete from status_logs
	if _, err := tx.ExecContext(r.Context(), "DELETE FROM status_logs WHERE application_id = ?", id); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal menghapus log riwayat status: "+err.Error())
		return
	}

	// Delete from applications
	res, err := tx.ExecContext(r.Context(), "DELETE FROM applications WHERE id = ?", id)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal menghapus data lamaran: "+err.Error())
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		sendError(w, http.StatusNotFound, "Data lamaran tidak ditemukan")
		return
	}

	if err := tx.Commit(); err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal commit penghapusan data: "+err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]any{
		"message": "Data lamaran berhasil dihapus",
		"id":      id,
	})
}

// GET /api/applications/{id}/logs
func (h *ApplicationHandler) GetApplicationLogs(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		sendError(w, http.StatusBadRequest, "ID lamaran tidak valid")
		return
	}

	query := `SELECT id, application_id, old_status, new_status, changed_at 
	          FROM status_logs 
	          WHERE application_id = ? 
	          ORDER BY changed_at DESC, id DESC`

	rows, err := h.DB.QueryContext(r.Context(), query, id)
	if err != nil {
		sendError(w, http.StatusInternalServerError, "Gagal mengambil log status: "+err.Error())
		return
	}
	defer rows.Close()

	logs := make([]models.StatusLog, 0)
	for rows.Next() {
		var log models.StatusLog
		var changedAtStr string
		if err := rows.Scan(&log.ID, &log.ApplicationID, &log.OldStatus, &log.NewStatus, &changedAtStr); err != nil {
			sendError(w, http.StatusInternalServerError, "Gagal membaca log: "+err.Error())
			return
		}
		log.ChangedAt = parseTimeFlexible(changedAtStr)
		logs = append(logs, log)
	}

	sendJSON(w, http.StatusOK, logs)
}

func parseTimeFlexible(value string) time.Time {
	if value == "" {
		return time.Time{}
	}

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t
		}
	}

	return time.Time{}
}

func sendJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{
		"error": message,
	})
}
