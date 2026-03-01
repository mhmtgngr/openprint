// Package handler provides HTTP handlers for mobile app support.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// MobileDevice represents a registered mobile device.
type MobileDevice struct {
	ID              string
	UserID          string
	DeviceName      string
	DeviceType      string // 'ios', 'android'
	DeviceToken     string // Push notification token
	AppVersion      string
	OSVersion       string
	IsActive        bool
	LastSeen        time.Time
	PairingCode     string
	PairedPrinterID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PushNotification represents a push notification to be sent.
type PushNotification struct {
	ID          string
	DeviceID    string
	Title       string
	Body        string
	Data        map[string]interface{}
	Priority    int // 0-10
	TTL         time.Duration
	ScheduledAt time.Time
	SentAt      *time.Time
	FailedAt    *time.Time
	Error       string
	CreatedAt   time.Time
}

// MobileHandler handles mobile app HTTP endpoints.
type MobileHandler struct {
	db *pgxpool.Pool
}

// NewMobileHandler creates a new mobile handler instance.
func NewMobileHandler(db *pgxpool.Pool) *MobileHandler {
	return &MobileHandler{db: db}
}

// RegisterDeviceRequest represents a request to register a mobile device.
type RegisterDeviceRequest struct {
	UserID        string `json:"user_id"`
	DeviceName    string `json:"device_name"`
	DeviceType    string `json:"device_type"`
	DeviceToken   string `json:"device_token"`
	AppVersion    string `json:"app_version"`
	OSVersion     string `json:"os_version"`
}

// PairPrinterRequest represents a request to pair a printer with a mobile device.
type PairPrinterRequest struct {
	PairingCode string `json:"pairing_code"`
	DeviceID    string `json:"device_id"`
}

// SendNotificationRequest represents a request to send a push notification.
type SendNotificationRequest struct {
	UserID  string                 `json:"user_id"`
	DeviceID string                 `json:"device_id,omitempty"`
	Title   string                 `json:"title"`
	Body    string                 `json:"body"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// RegisterDeviceHandler handles mobile device registration.
func (h *MobileHandler) RegisterDeviceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.UserID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}
	if req.DeviceName == "" {
		respondError(w, apperrors.New("device_name is required", http.StatusBadRequest))
		return
	}
	if req.DeviceType == "" {
		req.DeviceType = "unknown"
	}

	// Generate unique pairing code
	pairingCode, _ := generatePairingCode()

	device := &MobileDevice{
		ID:           uuid.New().String(),
		UserID:       req.UserID,
		DeviceName:   req.DeviceName,
		DeviceType:   req.DeviceType,
		DeviceToken:  req.DeviceToken,
		AppVersion:   req.AppVersion,
		OSVersion:    req.OSVersion,
		IsActive:     true,
		LastSeen:     time.Now(),
		PairingCode:  pairingCode,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Store device
	if err := h.storeDevice(ctx, device); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to register device", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"device_id":    device.ID,
		"pairing_code": device.PairingCode,
		"created_at":   device.CreatedAt.Format(time.RFC3339),
	})
}

// ListDevicesHandler handles listing mobile devices for a user.
func (h *MobileHandler) ListDevicesHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, apperrors.New("user_id is required", http.StatusBadRequest))
		return
	}

	devices, err := h.listDevices(ctx, userID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list devices", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(devices))
	for i, d := range devices {
		response[i] = map[string]interface{}{
			"device_id":     d.ID,
			"device_name":   d.DeviceName,
			"device_type":   d.DeviceType,
			"app_version":   d.AppVersion,
			"os_version":    d.OSVersion,
			"is_active":     d.IsActive,
			"last_seen":     d.LastSeen.Format(time.RFC3339),
			"paired_printer": d.PairedPrinterID,
			"created_at":    d.CreatedAt.Format(time.RFC3339),
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"devices": response,
		"count":   len(response),
	})
}

// DeviceHandler handles individual device operations.
func (h *MobileHandler) DeviceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract device ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid device path", http.StatusBadRequest))
		return
	}
	deviceID := parts[2]

	switch r.Method {
	case http.MethodGet:
		h.getDevice(w, r, ctx, deviceID)
	case http.MethodPut:
		h.updateDevice(w, r, ctx, deviceID)
	case http.MethodDelete:
		h.deleteDevice(w, r, ctx, deviceID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *MobileHandler) getDevice(w http.ResponseWriter, r *http.Request, ctx context.Context, deviceID string) {
	device, err := h.getDeviceByID(ctx, deviceID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get device", http.StatusInternalServerError))
		return
	}
	if device == nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":      device.ID,
		"user_id":        device.UserID,
		"device_name":    device.DeviceName,
		"device_type":    device.DeviceType,
		"app_version":    device.AppVersion,
		"os_version":     device.OSVersion,
		"is_active":      device.IsActive,
		"last_seen":      device.LastSeen.Format(time.RFC3339),
		"pairing_code":   device.PairingCode,
		"paired_printer": device.PairedPrinterID,
		"created_at":     device.CreatedAt.Format(time.RFC3339),
		"updated_at":     device.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *MobileHandler) updateDevice(w http.ResponseWriter, r *http.Request, ctx context.Context, deviceID string) {
	device, err := h.getDeviceByID(ctx, deviceID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get device", http.StatusInternalServerError))
		return
	}
	if device == nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	var req RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update fields
	if req.DeviceName != "" {
		device.DeviceName = req.DeviceName
	}
	if req.DeviceToken != "" {
		device.DeviceToken = req.DeviceToken
	}
	if req.AppVersion != "" {
		device.AppVersion = req.AppVersion
	}
	if req.OSVersion != "" {
		device.OSVersion = req.OSVersion
	}
	device.UpdatedAt = time.Now()
	device.LastSeen = time.Now()

	if err := h.updateDeviceStore(ctx, device); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update device", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"device_id":  device.ID,
		"updated_at": device.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *MobileHandler) deleteDevice(w http.ResponseWriter, r *http.Request, ctx context.Context, deviceID string) {
	if err := h.deleteDeviceStore(ctx, deviceID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete device", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GeneratePairingCodeHandler handles generating a new pairing code.
func (h *MobileHandler) GeneratePairingCodeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract device ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid device path", http.StatusBadRequest))
		return
	}
	deviceID := parts[3]

	// Generate new pairing code
	pairingCode, _ := generatePairingCode()

	// Update device
	query := `
		UPDATE mobile_devices
		SET pairing_code = $1,
		    updated_at = NOW()
		WHERE id = $2::uuid
	`
	_, err := h.db.Exec(ctx, query, pairingCode, deviceID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to generate pairing code", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pairing_code": pairingCode,
		"expires_at":   time.Now().Add(5 * time.Minute).Format(time.RFC3339),
	})
}

// PairPrinterHandler handles pairing a mobile device with a printer.
func (h *MobileHandler) PairPrinterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PairPrinterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.PairingCode == "" {
		respondError(w, apperrors.New("pairing_code is required", http.StatusBadRequest))
		return
	}
	if req.DeviceID == "" {
		respondError(w, apperrors.New("device_id is required", http.StatusBadRequest))
		return
	}

	// Find device by ID
	device, err := h.getDeviceByID(ctx, req.DeviceID)
	if err != nil || device == nil {
		respondError(w, apperrors.New("device not found", http.StatusNotFound))
		return
	}

	// Find printer by pairing code (from agent)
	var printerID, printerName string
	query := `
		SELECT id, name
		FROM printers
		WHERE pairing_code = $1
		  AND pairing_code_expires > NOW()
		LIMIT 1
	`
	err = h.db.QueryRow(ctx, query, req.PairingCode).Scan(&printerID, &printerName)
	if err != nil {
		respondError(w, apperrors.New("invalid or expired pairing code", http.StatusNotFound))
		return
	}

	// Update device with paired printer
	device.PairedPrinterID = printerID
	device.UpdatedAt = time.Now()
	if err := h.updateDeviceStore(ctx, device); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to pair printer", http.StatusInternalServerError))
		return
	}

	// Send push notification to device
	_ = h.sendPushNotification(ctx, device.ID, "Printer Paired", fmt.Sprintf("Successfully paired with %s", printerName), nil)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printer_id":   printerID,
		"printer_name": printerName,
		"paired_at":    time.Now().Format(time.RFC3339),
	})
}

// UnpairPrinterHandler handles unpairing a printer from a mobile device.
func (h *MobileHandler) UnpairPrinterHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract device ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 4 {
		respondError(w, apperrors.New("invalid device path", http.StatusBadRequest))
		return
	}
	deviceID := parts[3]

	// Get device
	device, err := h.getDeviceByID(ctx, deviceID)
	if err != nil || device == nil {
		respondError(w, apperrors.New("device not found", http.StatusNotFound))
		return
	}

	// Clear paired printer
	device.PairedPrinterID = ""
	device.UpdatedAt = time.Now()
	if err := h.updateDeviceStore(ctx, device); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to unpair printer", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// NearbyPrintersHandler handles listing nearby printers for mobile discovery.
func (h *MobileHandler) NearbyPrintersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get location parameters
	latitude := r.URL.Query().Get("lat")
	longitude := r.URL.Query().Get("lng")
	radius := r.URL.Query().Get("radius") // in meters

	// For now, return all active printers
	// In production, use actual geolocation queries
	query := `
		SELECT p.id, p.name, p.location, p.status, p.capabilities,
		       p.organization_id, o.name as organization_name
		FROM printers p
		LEFT JOIN organizations o ON o.id = p.organization_id
		WHERE p.status = 'online'
		ORDER BY p.name
		LIMIT 100
	`

	rows, err := h.db.Query(ctx, query)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to find nearby printers", http.StatusInternalServerError))
		return
	}
	defer rows.Close()

	printers := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, name, location, status, capabilities, orgID, orgName string
		if err := rows.Scan(&id, &name, &location, &status, &capabilities, &orgID, &orgName); err != nil {
			continue
		}

		printers = append(printers, map[string]interface{}{
			"printer_id":       id,
			"name":             name,
			"location":         location,
			"status":           status,
			"capabilities":     capabilities,
			"organization_id":  orgID,
			"organization_name": orgName,
			"distance":         nil, // Would contain actual distance if using geolocation
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"printers": printers,
		"count":    len(printers),
		"location": map[string]interface{}{
			"lat":    latitude,
			"lng":    longitude,
			"radius": radius,
		},
	})
}

// SendPushNotificationHandler handles sending push notifications.
func (h *MobileHandler) SendPushNotificationHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SendNotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.UserID == "" && req.DeviceID == "" {
		respondError(w, apperrors.New("either user_id or device_id is required", http.StatusBadRequest))
		return
	}
	if req.Title == "" {
		respondError(w, apperrors.New("title is required", http.StatusBadRequest))
		return
	}

	// Get devices to notify
	var devices []*MobileDevice

	if req.DeviceID != "" {
		device, _ := h.getDeviceByID(ctx, req.DeviceID)
		if device != nil {
			devices = []*MobileDevice{device}
		}
	} else {
		devices, _ = h.listDevices(ctx, req.UserID)
	}

	sentCount := 0
	for _, device := range devices {
		if err := h.sendPushNotification(ctx, device.ID, req.Title, req.Body, req.Data); err == nil {
			sentCount++
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"sent_count":  sentCount,
		"total_count": len(devices),
	})
}

// DeviceHeartbeatHandler handles mobile device heartbeat updates.
func (h *MobileHandler) DeviceHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract device ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		respondError(w, apperrors.New("invalid device path", http.StatusBadRequest))
		return
	}
	deviceID := parts[2]

	// Update last seen
	query := `
		UPDATE mobile_devices
		SET last_seen = NOW(),
		    is_active = true,
		    updated_at = NOW()
		WHERE id = $1::uuid
	`

	_, err := h.db.Exec(ctx, query, deviceID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update heartbeat", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Helper functions

func (h *MobileHandler) storeDevice(ctx context.Context, device *MobileDevice) error {
	// Create table if not exists
	initQuery := `
		CREATE TABLE IF NOT EXISTS mobile_devices (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			device_name VARCHAR(255) NOT NULL,
			device_type VARCHAR(50) DEFAULT 'unknown',
			device_token TEXT,
			app_version VARCHAR(50),
			os_version VARCHAR(50),
			is_active BOOLEAN DEFAULT true,
			last_seen TIMESTAMPTZ DEFAULT NOW(),
			pairing_code VARCHAR(20) UNIQUE,
			paired_printer_id UUID REFERENCES printers(id) ON DELETE SET NULL,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_mobile_devices_user ON mobile_devices(user_id);
		CREATE INDEX IF NOT EXISTS idx_mobile_devices_pairing ON mobile_devices(pairing_code);
		CREATE TRIGGER update_mobile_devices_updated_at BEFORE UPDATE ON mobile_devices
		    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
	`
	h.db.Exec(ctx, initQuery)

	query := `
		INSERT INTO mobile_devices (
			id, user_id, device_name, device_type, device_token,
			app_version, os_version, is_active, last_seen, pairing_code,
			paired_printer_id, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11::uuid, $12, $13
		)
	`

	_, err := h.db.Exec(ctx, query,
		device.ID, device.UserID, device.DeviceName, device.DeviceType, device.DeviceToken,
		device.AppVersion, device.OSVersion, device.IsActive, device.LastSeen, device.PairingCode,
		nullIfEmpty(device.PairedPrinterID), device.CreatedAt, device.UpdatedAt,
	)

	return err
}

func (h *MobileHandler) getDeviceByID(ctx context.Context, deviceID string) (*MobileDevice, error) {
	query := `
		SELECT id, user_id, device_name, device_type, device_token,
		       app_version, os_version, is_active, last_seen, pairing_code,
		       paired_printer_id, created_at, updated_at
		FROM mobile_devices
		WHERE id = $1::uuid
	`

	var device MobileDevice
	err := h.db.QueryRow(ctx, query, deviceID).Scan(
		&device.ID, &device.UserID, &device.DeviceName, &device.DeviceType, &device.DeviceToken,
		&device.AppVersion, &device.OSVersion, &device.IsActive, &device.LastSeen, &device.PairingCode,
		&device.PairedPrinterID, &device.CreatedAt, &device.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &device, nil
}

func (h *MobileHandler) listDevices(ctx context.Context, userID string) ([]*MobileDevice, error) {
	query := `
		SELECT id, user_id, device_name, device_type, device_token,
		       app_version, os_version, is_active, last_seen, pairing_code,
		       paired_printer_id, created_at, updated_at
		FROM mobile_devices
		WHERE user_id = $1::uuid
		ORDER BY created_at DESC
	`

	rows, err := h.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []*MobileDevice
	for rows.Next() {
		var device MobileDevice
		if err := rows.Scan(
			&device.ID, &device.UserID, &device.DeviceName, &device.DeviceType, &device.DeviceToken,
			&device.AppVersion, &device.OSVersion, &device.IsActive, &device.LastSeen, &device.PairingCode,
			&device.PairedPrinterID, &device.CreatedAt, &device.UpdatedAt,
		); err != nil {
			return nil, err
		}
		devices = append(devices, &device)
	}

	return devices, nil
}

func (h *MobileHandler) updateDeviceStore(ctx context.Context, device *MobileDevice) error {
	query := `
		UPDATE mobile_devices
		SET device_name = $2, device_token = $3, app_version = $4, os_version = $5,
		    is_active = $6, last_seen = $7, pairing_code = $8, paired_printer_id = $9::uuid,
		    updated_at = $10
		WHERE id = $1::uuid
	`

	_, err := h.db.Exec(ctx, query,
		device.ID, device.DeviceName, device.DeviceToken, device.AppVersion, device.OSVersion,
		device.IsActive, device.LastSeen, device.PairingCode, nullIfEmpty(device.PairedPrinterID),
		device.UpdatedAt,
	)

	return err
}

func (h *MobileHandler) deleteDeviceStore(ctx context.Context, deviceID string) error {
	query := `DELETE FROM mobile_devices WHERE id = $1::uuid`

	_, err := h.db.Exec(ctx, query, deviceID)
	return err
}

func (h *MobileHandler) sendPushNotification(ctx context.Context, deviceID, title, body string, data map[string]interface{}) error {
	// Store notification for sending
	// In production, integrate with APNs (iOS) and FCM (Android)
	query := `
		INSERT INTO push_notifications (
			id, device_id, title, body, data, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5::jsonb, NOW()
		)
	`

	id := uuid.New().String()
	dataJSON, _ := json.Marshal(data)

	_, err := h.db.Exec(ctx, query, id, deviceID, title, body, dataJSON)

	// Create table if not exists
	initQuery := `
		CREATE TABLE IF NOT EXISTS push_notifications (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			device_id UUID NOT NULL REFERENCES mobile_devices(id) ON DELETE CASCADE,
			title VARCHAR(255) NOT NULL,
			body TEXT,
			data JSONB,
			priority INTEGER DEFAULT 5,
			ttl INTERVAL DEFAULT '1 day',
			scheduled_at TIMESTAMPTZ DEFAULT NOW(),
			sent_at TIMESTAMPTZ,
			failed_at TIMESTAMPTZ,
			error TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_push_notifications_device ON push_notifications(device_id);
	`
	h.db.Exec(ctx, initQuery)

	return err
}

func generatePairingCode() (string, error) {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return strings.ToUpper(hex.EncodeToString(bytes)), nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if json.Unmarshal([]byte(fmt.Sprintf("%v", err)), &appErr) == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.StatusCode)
		json.NewEncoder(w).Encode(apperrors.ToJSON(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "An internal error occurred",
	})
}
