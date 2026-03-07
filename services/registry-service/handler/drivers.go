// Package handler provides driver management HTTP handlers for the registry service.
package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// DriverHandler provides HTTP handlers for print driver management operations.
type DriverHandler struct {
	db *sql.DB
}

// NewDriverHandler creates a new DriverHandler instance.
func NewDriverHandler(db *sql.DB) *DriverHandler {
	return &DriverHandler{db: db}
}

// Driver represents a print driver package record.
type Driver struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Manufacturer   string  `json:"manufacturer"`
	ModelPattern   *string `json:"model_pattern,omitempty"`
	OS             string  `json:"os"`
	Architecture   string  `json:"architecture"`
	Version        string  `json:"version"`
	FilePath       *string `json:"file_path,omitempty"`
	FileSizeBytes  *int64  `json:"file_size_bytes,omitempty"`
	ChecksumSHA256 *string `json:"checksum_sha256,omitempty"`
	IsUniversal    bool    `json:"is_universal"`
	IsLatest       bool    `json:"is_latest"`
	ReleaseNotes   *string `json:"release_notes,omitempty"`
	UploadedAt     time.Time `json:"uploaded_at"`
	UploadedBy     *string `json:"uploaded_by,omitempty"`
}

// DriverAssignment represents a driver-printer assignment record.
type DriverAssignment struct {
	PrinterID  string    `json:"printer_id"`
	DriverID   string    `json:"driver_id"`
	AssignedAt time.Time `json:"assigned_at"`
}

// ListDrivers returns available drivers with optional filtering.
// GET /drivers
func (h *DriverHandler) ListDrivers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query filters
	manufacturer := r.URL.Query().Get("manufacturer")
	os := r.URL.Query().Get("os")
	arch := r.URL.Query().Get("architecture")
	latestOnly := r.URL.Query().Get("latest") == "true"

	// Parse pagination
	limit := 100
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		fmt.Sscanf(l, "%d", &limit)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		fmt.Sscanf(o, "%d", &offset)
	}

	// Build query dynamically
	conditions := []string{}
	args := []interface{}{}
	argIdx := 1

	if manufacturer != "" {
		conditions = append(conditions, fmt.Sprintf("manufacturer ILIKE $%d", argIdx))
		args = append(args, "%"+manufacturer+"%")
		argIdx++
	}
	if os != "" {
		conditions = append(conditions, fmt.Sprintf("os = $%d", argIdx))
		args = append(args, os)
		argIdx++
	}
	if arch != "" {
		conditions = append(conditions, fmt.Sprintf("architecture = $%d", argIdx))
		args = append(args, arch)
		argIdx++
	}
	if latestOnly {
		conditions = append(conditions, "is_latest = true")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM print_drivers %s", whereClause)
	if err := h.db.QueryRowContext(r.Context(), countQuery, args...).Scan(&total); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to count drivers", http.StatusInternalServerError))
		return
	}

	// Fetch drivers
	args = append(args, limit, offset)
	query := fmt.Sprintf(
		`SELECT id, name, manufacturer, model_pattern, os, architecture, version,
		        file_path, file_size_bytes, checksum_sha256, is_universal, is_latest,
		        release_notes, uploaded_at, uploaded_by
		 FROM print_drivers %s
		 ORDER BY manufacturer, name, version DESC
		 LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)

	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query drivers", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	drivers := make([]Driver, 0)
	for rows.Next() {
		var d Driver
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Manufacturer, &d.ModelPattern, &d.OS,
			&d.Architecture, &d.Version, &d.FilePath, &d.FileSizeBytes,
			&d.ChecksumSHA256, &d.IsUniversal, &d.IsLatest, &d.ReleaseNotes,
			&d.UploadedAt, &d.UploadedBy,
		); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan driver", http.StatusInternalServerError))
			return
		}
		drivers = append(drivers, d)
	}

	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate drivers", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"data":   drivers,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetDriver returns details for a specific driver.
// GET /drivers/{id}
func (h *DriverHandler) GetDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract driver ID from path: /drivers/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	driverID := parts[1]

	var d Driver
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, manufacturer, model_pattern, os, architecture, version,
		        file_path, file_size_bytes, checksum_sha256, is_universal, is_latest,
		        release_notes, uploaded_at, uploaded_by
		 FROM print_drivers
		 WHERE id = $1`, driverID,
	).Scan(
		&d.ID, &d.Name, &d.Manufacturer, &d.ModelPattern, &d.OS,
		&d.Architecture, &d.Version, &d.FilePath, &d.FileSizeBytes,
		&d.ChecksumSHA256, &d.IsUniversal, &d.IsLatest, &d.ReleaseNotes,
		&d.UploadedAt, &d.UploadedBy,
	)
	if err == sql.ErrNoRows {
		respondError(w, apperrors.ErrNotFound)
		return
	}
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query driver", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, d)
}

// UploadDriver registers a new driver package.
// POST /drivers
func (h *DriverHandler) UploadDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name           string  `json:"name"`
		Manufacturer   string  `json:"manufacturer"`
		ModelPattern   *string `json:"model_pattern,omitempty"`
		OS             string  `json:"os"`
		Architecture   string  `json:"architecture"`
		Version        string  `json:"version"`
		FilePath       *string `json:"file_path,omitempty"`
		FileSizeBytes  *int64  `json:"file_size_bytes,omitempty"`
		ChecksumSHA256 *string `json:"checksum_sha256,omitempty"`
		IsUniversal    bool    `json:"is_universal"`
		ReleaseNotes   *string `json:"release_notes,omitempty"`
		UploadedBy     *string `json:"uploaded_by,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.Name == "" || req.Manufacturer == "" || req.OS == "" || req.Version == "" {
		respondError(w, apperrors.New("name, manufacturer, os, and version are required", http.StatusBadRequest))
		return
	}

	if req.Architecture == "" {
		req.Architecture = "x64"
	}

	// Mark previous versions as not latest for the same manufacturer/model/os/arch
	_, _ = h.db.ExecContext(r.Context(),
		`UPDATE print_drivers SET is_latest = false
		 WHERE manufacturer = $1 AND os = $2 AND architecture = $3
		   AND COALESCE(model_pattern, '') = COALESCE($4, '')
		   AND is_latest = true`,
		req.Manufacturer, req.OS, req.Architecture, req.ModelPattern)

	var driverID string
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO print_drivers (name, manufacturer, model_pattern, os, architecture,
		                            version, file_path, file_size_bytes, checksum_sha256,
		                            is_universal, is_latest, release_notes, uploaded_by)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, $11, $12)
		 RETURNING id`,
		req.Name, req.Manufacturer, req.ModelPattern, req.OS, req.Architecture,
		req.Version, req.FilePath, req.FileSizeBytes, req.ChecksumSHA256,
		req.IsUniversal, req.ReleaseNotes, req.UploadedBy,
	).Scan(&driverID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create driver", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":           driverID,
		"name":         req.Name,
		"manufacturer": req.Manufacturer,
		"os":           req.OS,
		"architecture": req.Architecture,
		"version":      req.Version,
		"is_universal": req.IsUniversal,
		"is_latest":    true,
		"uploaded_at":  time.Now().Format(time.RFC3339),
	})
}

// DeleteDriver removes a driver package.
// DELETE /drivers/{id}
func (h *DriverHandler) DeleteDriver(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract driver ID from path: /drivers/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	driverID := parts[1]

	result, err := h.db.ExecContext(r.Context(),
		`DELETE FROM print_drivers WHERE id = $1`, driverID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete driver", http.StatusInternalServerError))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AssignDriverToPrinter assigns a driver to a printer.
// POST /printers/{id}/drivers
func (h *DriverHandler) AssignDriverToPrinter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from path: /printers/{id}/drivers
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	var req struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.DriverID == "" {
		respondError(w, apperrors.New("driver_id is required", http.StatusBadRequest))
		return
	}

	// Verify driver exists
	var driverExists bool
	err := h.db.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM print_drivers WHERE id = $1)`, req.DriverID).Scan(&driverExists)
	if err != nil || !driverExists {
		respondError(w, apperrors.New("driver not found", http.StatusNotFound))
		return
	}

	_, err = h.db.ExecContext(r.Context(),
		`INSERT INTO printer_driver_assignments (printer_id, driver_id)
		 VALUES ($1, $2)
		 ON CONFLICT (printer_id, driver_id) DO NOTHING`,
		printerID, req.DriverID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to assign driver", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"printer_id":  printerID,
		"driver_id":   req.DriverID,
		"assigned_at": time.Now().Format(time.RFC3339),
	})
}

// GetDriversForPrinter returns all drivers assigned to a specific printer.
// GET /printers/{id}/drivers
func (h *DriverHandler) GetDriversForPrinter(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from path: /printers/{id}/drivers
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT d.id, d.name, d.manufacturer, d.model_pattern, d.os, d.architecture,
		        d.version, d.file_path, d.file_size_bytes, d.checksum_sha256,
		        d.is_universal, d.is_latest, d.release_notes, d.uploaded_at, d.uploaded_by
		 FROM print_drivers d
		 INNER JOIN printer_driver_assignments pda ON d.id = pda.driver_id
		 WHERE pda.printer_id = $1
		 ORDER BY d.manufacturer, d.name`, printerID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query printer drivers", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	drivers := make([]Driver, 0)
	for rows.Next() {
		var d Driver
		if err := rows.Scan(
			&d.ID, &d.Name, &d.Manufacturer, &d.ModelPattern, &d.OS,
			&d.Architecture, &d.Version, &d.FilePath, &d.FileSizeBytes,
			&d.ChecksumSHA256, &d.IsUniversal, &d.IsLatest, &d.ReleaseNotes,
			&d.UploadedAt, &d.UploadedBy,
		); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan driver", http.StatusInternalServerError))
			return
		}
		drivers = append(drivers, d)
	}

	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate drivers", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"drivers":    drivers,
		"printer_id": printerID,
		"count":      len(drivers),
	})
}

// GetLatestDriverForModel returns the latest driver for a given printer model and OS.
// GET /drivers/latest?manufacturer=...&model=...&os=...&architecture=...
func (h *DriverHandler) GetLatestDriverForModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	manufacturer := r.URL.Query().Get("manufacturer")
	model := r.URL.Query().Get("model")
	osParam := r.URL.Query().Get("os")
	arch := r.URL.Query().Get("architecture")

	if manufacturer == "" || osParam == "" {
		respondError(w, apperrors.New("manufacturer and os query parameters are required", http.StatusBadRequest))
		return
	}

	if arch == "" {
		arch = "x64"
	}

	// Look for a specific model match first, then fall back to universal drivers
	var d Driver
	err := h.db.QueryRowContext(r.Context(),
		`SELECT id, name, manufacturer, model_pattern, os, architecture, version,
		        file_path, file_size_bytes, checksum_sha256, is_universal, is_latest,
		        release_notes, uploaded_at, uploaded_by
		 FROM print_drivers
		 WHERE manufacturer ILIKE $1 AND os = $2 AND architecture = $3
		   AND is_latest = true
		   AND (model_pattern IS NULL OR $4 ILIKE model_pattern OR is_universal = true)
		 ORDER BY
		   CASE WHEN model_pattern IS NOT NULL AND $4 ILIKE model_pattern THEN 0
		        WHEN is_universal THEN 1
		        ELSE 2 END,
		   uploaded_at DESC
		 LIMIT 1`,
		manufacturer, osParam, arch, model,
	).Scan(
		&d.ID, &d.Name, &d.Manufacturer, &d.ModelPattern, &d.OS,
		&d.Architecture, &d.Version, &d.FilePath, &d.FileSizeBytes,
		&d.ChecksumSHA256, &d.IsUniversal, &d.IsLatest, &d.ReleaseNotes,
		&d.UploadedAt, &d.UploadedBy,
	)
	if err == sql.ErrNoRows {
		respondError(w, apperrors.New("no matching driver found", http.StatusNotFound))
		return
	}
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query latest driver", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, d)
}
