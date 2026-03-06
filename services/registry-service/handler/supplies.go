// Package handler provides supply and maintenance HTTP handlers for the registry service.
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

// SupplyHandler provides HTTP handlers for printer supply and maintenance operations.
type SupplyHandler struct {
	db *sql.DB
}

// NewSupplyHandler creates a new SupplyHandler instance.
func NewSupplyHandler(db *sql.DB) *SupplyHandler {
	return &SupplyHandler{db: db}
}

// Supply represents a printer supply record.
type Supply struct {
	ID                     string     `json:"id"`
	PrinterID              string     `json:"printer_id"`
	SupplyType             string     `json:"supply_type"`
	Name                   string     `json:"name"`
	LevelPercent           int        `json:"level_percent"`
	Status                 string     `json:"status"`
	PartNumber             *string    `json:"part_number,omitempty"`
	EstimatedPagesRemaining *int      `json:"estimated_pages_remaining,omitempty"`
	LastReplacedAt         *time.Time `json:"last_replaced_at,omitempty"`
	AlertThreshold         int        `json:"alert_threshold"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

// MaintenanceTask represents a printer maintenance record.
type MaintenanceTask struct {
	ID              string     `json:"id"`
	PrinterID       string     `json:"printer_id"`
	MaintenanceType string     `json:"maintenance_type"`
	Description     *string    `json:"description,omitempty"`
	ScheduledAt     time.Time  `json:"scheduled_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	AssignedTo      *string    `json:"assigned_to,omitempty"`
	Status          string     `json:"status"`
	Notes           *string    `json:"notes,omitempty"`
	Recurrence      *string    `json:"recurrence,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// GetSupplyLevels returns all supply levels for a printer.
// GET /printers/{id}/supplies
func (h *SupplyHandler) GetSupplyLevels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from path: /printers/{id}/supplies
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, printer_id, supply_type, name, level_percent, status,
		        part_number, estimated_pages_remaining, last_replaced_at,
		        alert_threshold, updated_at
		 FROM printer_supplies
		 WHERE printer_id = $1
		 ORDER BY supply_type, name`, printerID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query supplies", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	supplies := make([]Supply, 0)
	for rows.Next() {
		var s Supply
		if err := rows.Scan(
			&s.ID, &s.PrinterID, &s.SupplyType, &s.Name, &s.LevelPercent,
			&s.Status, &s.PartNumber, &s.EstimatedPagesRemaining,
			&s.LastReplacedAt, &s.AlertThreshold, &s.UpdatedAt,
		); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan supply", http.StatusInternalServerError))
			return
		}
		supplies = append(supplies, s)
	}

	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate supplies", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"supplies":   supplies,
		"printer_id": printerID,
		"count":      len(supplies),
	})
}

// UpdateSupplyLevel updates a supply level for a printer (from agent reports).
// PUT /printers/{id}/supplies
func (h *SupplyHandler) UpdateSupplyLevel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from path: /printers/{id}/supplies
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	var req struct {
		SupplyType              string  `json:"supply_type"`
		Name                    string  `json:"name"`
		LevelPercent            int     `json:"level_percent"`
		Status                  string  `json:"status"`
		PartNumber              *string `json:"part_number,omitempty"`
		EstimatedPagesRemaining *int    `json:"estimated_pages_remaining,omitempty"`
		AlertThreshold          *int    `json:"alert_threshold,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.SupplyType == "" || req.Name == "" {
		respondError(w, apperrors.New("supply_type and name are required", http.StatusBadRequest))
		return
	}

	if req.LevelPercent < 0 || req.LevelPercent > 100 {
		respondError(w, apperrors.New("level_percent must be between 0 and 100", http.StatusBadRequest))
		return
	}

	if req.Status == "" {
		req.Status = "ok"
		if req.LevelPercent <= 15 {
			req.Status = "low"
		}
		if req.LevelPercent == 0 {
			req.Status = "empty"
		}
	}

	alertThreshold := 15
	if req.AlertThreshold != nil {
		alertThreshold = *req.AlertThreshold
	}

	// Upsert: update if exists for this printer + supply_type + name, else insert
	var supplyID string
	err := h.db.QueryRowContext(r.Context(),
		`INSERT INTO printer_supplies (printer_id, supply_type, name, level_percent, status,
		                               part_number, estimated_pages_remaining, alert_threshold, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		 ON CONFLICT (id) DO UPDATE SET
		     level_percent = EXCLUDED.level_percent,
		     status = EXCLUDED.status,
		     part_number = EXCLUDED.part_number,
		     estimated_pages_remaining = EXCLUDED.estimated_pages_remaining,
		     alert_threshold = EXCLUDED.alert_threshold,
		     updated_at = NOW()
		 RETURNING id`,
		printerID, req.SupplyType, req.Name, req.LevelPercent, req.Status,
		req.PartNumber, req.EstimatedPagesRemaining, alertThreshold,
	).Scan(&supplyID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to upsert supply", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":            supplyID,
		"printer_id":    printerID,
		"supply_type":   req.SupplyType,
		"name":          req.Name,
		"level_percent": req.LevelPercent,
		"status":        req.Status,
		"updated_at":    time.Now().Format(time.RFC3339),
	})
}

// GetLowSupplyAlerts returns printers with supplies below their alert threshold.
// GET /supplies/alerts
func (h *SupplyHandler) GetLowSupplyAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := h.db.QueryContext(r.Context(),
		`SELECT id, printer_id, supply_type, name, level_percent, status,
		        part_number, estimated_pages_remaining, last_replaced_at,
		        alert_threshold, updated_at
		 FROM printer_supplies
		 WHERE level_percent <= alert_threshold
		 ORDER BY level_percent ASC, updated_at DESC`)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to query low supplies", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	alerts := make([]Supply, 0)
	for rows.Next() {
		var s Supply
		if err := rows.Scan(
			&s.ID, &s.PrinterID, &s.SupplyType, &s.Name, &s.LevelPercent,
			&s.Status, &s.PartNumber, &s.EstimatedPagesRemaining,
			&s.LastReplacedAt, &s.AlertThreshold, &s.UpdatedAt,
		); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan supply alert", http.StatusInternalServerError))
			return
		}
		alerts = append(alerts, s)
	}

	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate supply alerts", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
	})
}

// CreateMaintenanceTask schedules a new maintenance task for a printer.
// POST /printers/{id}/maintenance
func (h *SupplyHandler) CreateMaintenanceTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from path: /printers/{id}/maintenance
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	var req struct {
		MaintenanceType string  `json:"maintenance_type"`
		Description     *string `json:"description,omitempty"`
		ScheduledAt     string  `json:"scheduled_at"`
		AssignedTo      *string `json:"assigned_to,omitempty"`
		Notes           *string `json:"notes,omitempty"`
		Recurrence      *string `json:"recurrence,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.MaintenanceType == "" {
		respondError(w, apperrors.New("maintenance_type is required", http.StatusBadRequest))
		return
	}

	if req.ScheduledAt == "" {
		respondError(w, apperrors.New("scheduled_at is required", http.StatusBadRequest))
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		respondError(w, apperrors.New("scheduled_at must be in RFC3339 format", http.StatusBadRequest))
		return
	}

	var taskID string
	err = h.db.QueryRowContext(r.Context(),
		`INSERT INTO printer_maintenance (printer_id, maintenance_type, description,
		                                  scheduled_at, assigned_to, notes, recurrence)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id`,
		printerID, req.MaintenanceType, req.Description,
		scheduledAt, req.AssignedTo, req.Notes, req.Recurrence,
	).Scan(&taskID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create maintenance task", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"id":               taskID,
		"printer_id":       printerID,
		"maintenance_type": req.MaintenanceType,
		"scheduled_at":     scheduledAt.Format(time.RFC3339),
		"status":           "scheduled",
		"created_at":       time.Now().Format(time.RFC3339),
	})
}

// ListMaintenanceTasks lists maintenance tasks for a printer.
// GET /printers/{id}/maintenance
func (h *SupplyHandler) ListMaintenanceTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract printer ID from path: /printers/{id}/maintenance
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	printerID := parts[1]

	// Parse optional status filter
	statusFilter := r.URL.Query().Get("status")

	var rows *sql.Rows
	var queryErr error

	if statusFilter != "" {
		rows, queryErr = h.db.QueryContext(r.Context(),
			`SELECT id, printer_id, maintenance_type, description, scheduled_at,
			        completed_at, assigned_to, status, notes, recurrence,
			        created_at, updated_at
			 FROM printer_maintenance
			 WHERE printer_id = $1 AND status = $2
			 ORDER BY scheduled_at ASC`, printerID, statusFilter)
	} else {
		rows, queryErr = h.db.QueryContext(r.Context(),
			`SELECT id, printer_id, maintenance_type, description, scheduled_at,
			        completed_at, assigned_to, status, notes, recurrence,
			        created_at, updated_at
			 FROM printer_maintenance
			 WHERE printer_id = $1
			 ORDER BY scheduled_at ASC`, printerID)
	}

	if queryErr != nil {
		respondError(w, apperrors.Wrap(queryErr, "failed to query maintenance tasks", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	tasks := make([]MaintenanceTask, 0)
	for rows.Next() {
		var t MaintenanceTask
		if err := rows.Scan(
			&t.ID, &t.PrinterID, &t.MaintenanceType, &t.Description,
			&t.ScheduledAt, &t.CompletedAt, &t.AssignedTo, &t.Status,
			&t.Notes, &t.Recurrence, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			respondError(w, apperrors.Wrap(err, "failed to scan maintenance task", http.StatusInternalServerError))
			return
		}
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to iterate maintenance tasks", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"tasks":      tasks,
		"printer_id": printerID,
		"count":      len(tasks),
	})
}

// UpdateMaintenanceTask updates or completes a maintenance task.
// PUT /maintenance/{id}
func (h *SupplyHandler) UpdateMaintenanceTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract task ID from path: /maintenance/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	taskID := parts[1]

	var req struct {
		Status      *string `json:"status,omitempty"`
		AssignedTo  *string `json:"assigned_to,omitempty"`
		Notes       *string `json:"notes,omitempty"`
		ScheduledAt *string `json:"scheduled_at,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Verify task exists
	var exists bool
	err := h.db.QueryRowContext(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM printer_maintenance WHERE id = $1)`, taskID).Scan(&exists)
	if err != nil || !exists {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	// Build update dynamically
	setClauses := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		validStatuses := map[string]bool{
			"scheduled": true, "in_progress": true, "completed": true, "cancelled": true,
		}
		if !validStatuses[*req.Status] {
			respondError(w, apperrors.New("invalid status value", http.StatusBadRequest))
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *req.Status)
		argIdx++

		if *req.Status == "completed" {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}

	if req.AssignedTo != nil {
		setClauses = append(setClauses, fmt.Sprintf("assigned_to = $%d", argIdx))
		args = append(args, *req.AssignedTo)
		argIdx++
	}

	if req.Notes != nil {
		setClauses = append(setClauses, fmt.Sprintf("notes = $%d", argIdx))
		args = append(args, *req.Notes)
		argIdx++
	}

	if req.ScheduledAt != nil {
		scheduledAt, parseErr := time.Parse(time.RFC3339, *req.ScheduledAt)
		if parseErr != nil {
			respondError(w, apperrors.New("scheduled_at must be in RFC3339 format", http.StatusBadRequest))
			return
		}
		setClauses = append(setClauses, fmt.Sprintf("scheduled_at = $%d", argIdx))
		args = append(args, scheduledAt)
		argIdx++
	}

	args = append(args, taskID)
	query := fmt.Sprintf(
		`UPDATE printer_maintenance SET %s WHERE id = $%d
		 RETURNING id, printer_id, maintenance_type, description, scheduled_at,
		           completed_at, assigned_to, status, notes, recurrence, created_at, updated_at`,
		strings.Join(setClauses, ", "), argIdx,
	)

	var t MaintenanceTask
	err = h.db.QueryRowContext(r.Context(), query, args...).Scan(
		&t.ID, &t.PrinterID, &t.MaintenanceType, &t.Description,
		&t.ScheduledAt, &t.CompletedAt, &t.AssignedTo, &t.Status,
		&t.Notes, &t.Recurrence, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update maintenance task", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, t)
}

// DeleteMaintenanceTask cancels/deletes a maintenance task.
// DELETE /maintenance/{id}
func (h *SupplyHandler) DeleteMaintenanceTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract task ID from path: /maintenance/{id}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	taskID := parts[1]

	result, err := h.db.ExecContext(r.Context(),
		`DELETE FROM printer_maintenance WHERE id = $1`, taskID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete maintenance task", http.StatusInternalServerError))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
