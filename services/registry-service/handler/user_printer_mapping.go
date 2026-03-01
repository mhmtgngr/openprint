// Package handler provides HTTP handlers for user-printer mapping management.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/registry-service/repository"
)

// UserPrinterMappingHandler handles user-printer mapping HTTP requests.
type UserPrinterMappingHandler struct {
	mappingRepo *repository.UserPrinterMappingRepository
}

// NewUserPrinterMappingHandler creates a new handler.
func NewUserPrinterMappingHandler(repo *repository.UserPrinterMappingRepository) *UserPrinterMappingHandler {
	return &UserPrinterMappingHandler{mappingRepo: repo}
}

// CreateMappingRequest represents a request to create a user-printer mapping.
type CreateMappingRequest struct {
	OrganizationID    string `json:"organization_id,omitempty"`
	UserEmail         string `json:"user_email"`
	UserName          string `json:"user_name,omitempty"`
	ClientAgentID     string `json:"client_agent_id"`
	TargetPrinterID   string `json:"target_printer_id,omitempty"`
	TargetPrinterName string `json:"target_printer_name,omitempty"`
	ServerAgentID     string `json:"server_agent_id,omitempty"`
	IsDefault         bool   `json:"is_default"`
}

// MappingsHandler handles list and create operations.
// GET /user-printer-mappings?user_email=...&organization_id=...&client_agent_id=...
// POST /user-printer-mappings
func (h *UserPrinterMappingHandler) MappingsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		h.listMappings(w, r)
	case http.MethodPost:
		h.createMapping(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
	_ = ctx
}

func (h *UserPrinterMappingHandler) createMapping(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateMappingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondMappingError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if req.UserEmail == "" {
		respondMappingError(w, apperrors.New("user_email is required", http.StatusBadRequest))
		return
	}
	if req.ClientAgentID == "" {
		respondMappingError(w, apperrors.New("client_agent_id is required", http.StatusBadRequest))
		return
	}

	mapping := &repository.UserPrinterMapping{
		OrganizationID:    req.OrganizationID,
		UserEmail:         req.UserEmail,
		UserName:          req.UserName,
		ClientAgentID:     req.ClientAgentID,
		TargetPrinterID:   req.TargetPrinterID,
		TargetPrinterName: req.TargetPrinterName,
		ServerAgentID:     req.ServerAgentID,
		IsActive:          true,
		IsDefault:         req.IsDefault,
	}

	if err := h.mappingRepo.Create(ctx, mapping); err != nil {
		respondMappingError(w, apperrors.Wrap(err, "failed to create mapping", http.StatusInternalServerError))
		return
	}

	respondMappingJSON(w, http.StatusCreated, mappingToResponse(mapping))
}

func (h *UserPrinterMappingHandler) listMappings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userEmail := r.URL.Query().Get("user_email")
	clientAgentID := r.URL.Query().Get("client_agent_id")
	orgID := r.URL.Query().Get("organization_id")

	var mappings []*repository.UserPrinterMapping
	var err error

	switch {
	case userEmail != "":
		mappings, err = h.mappingRepo.FindByUserEmail(ctx, userEmail)
	case clientAgentID != "":
		mappings, err = h.mappingRepo.FindByClientAgent(ctx, clientAgentID)
	case orgID != "":
		limit := 100
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" {
			fmt.Sscanf(l, "%d", &limit)
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			fmt.Sscanf(o, "%d", &offset)
		}
		var total int
		mappings, total, err = h.mappingRepo.FindByOrganization(ctx, orgID, limit, offset)
		if err == nil {
			response := make([]map[string]interface{}, len(mappings))
			for i, m := range mappings {
				response[i] = mappingToResponse(m)
			}
			respondMappingJSON(w, http.StatusOK, map[string]interface{}{
				"mappings": response,
				"total":    total,
				"limit":    limit,
				"offset":   offset,
			})
			return
		}
	default:
		respondMappingError(w, apperrors.New("user_email, client_agent_id, or organization_id query parameter is required", http.StatusBadRequest))
		return
	}

	if err != nil {
		respondMappingError(w, apperrors.Wrap(err, "failed to list mappings", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(mappings))
	for i, m := range mappings {
		response[i] = mappingToResponse(m)
	}

	respondMappingJSON(w, http.StatusOK, map[string]interface{}{
		"mappings": response,
		"count":    len(response),
	})
}

// MappingHandler handles individual mapping operations.
// GET/PUT/DELETE /user-printer-mappings/{id}
func (h *UserPrinterMappingHandler) MappingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Extract mapping ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondMappingError(w, apperrors.New("invalid path", http.StatusBadRequest))
		return
	}
	mappingID := parts[len(parts)-1]

	// Check for /resolve sub-path
	if mappingID == "resolve" {
		h.resolveUsername(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		mapping, err := h.mappingRepo.FindByID(ctx, mappingID)
		if err != nil {
			respondMappingError(w, apperrors.ErrNotFound)
			return
		}
		respondMappingJSON(w, http.StatusOK, mappingToResponse(mapping))

	case http.MethodPut:
		var req CreateMappingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondMappingError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
			return
		}

		mapping, err := h.mappingRepo.FindByID(ctx, mappingID)
		if err != nil {
			respondMappingError(w, apperrors.ErrNotFound)
			return
		}

		if req.UserName != "" {
			mapping.UserName = req.UserName
		}
		if req.ClientAgentID != "" {
			mapping.ClientAgentID = req.ClientAgentID
		}
		if req.TargetPrinterID != "" {
			mapping.TargetPrinterID = req.TargetPrinterID
		}
		if req.TargetPrinterName != "" {
			mapping.TargetPrinterName = req.TargetPrinterName
		}
		if req.ServerAgentID != "" {
			mapping.ServerAgentID = req.ServerAgentID
		}
		mapping.IsDefault = req.IsDefault

		if err := h.mappingRepo.Update(ctx, mapping); err != nil {
			respondMappingError(w, apperrors.Wrap(err, "failed to update mapping", http.StatusInternalServerError))
			return
		}

		respondMappingJSON(w, http.StatusOK, mappingToResponse(mapping))

	case http.MethodDelete:
		if err := h.mappingRepo.Delete(ctx, mappingID); err != nil {
			respondMappingError(w, apperrors.Wrap(err, "failed to delete mapping", http.StatusInternalServerError))
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// resolveUsername resolves a Windows username to an email address.
// GET /user-printer-mappings/resolve?username=DOMAIN\user
func (h *UserPrinterMappingHandler) resolveUsername(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	username := r.URL.Query().Get("username")
	if username == "" {
		respondMappingError(w, apperrors.New("username query parameter is required", http.StatusBadRequest))
		return
	}

	email, err := h.mappingRepo.ResolveUsername(r.Context(), username)
	if err != nil {
		respondMappingError(w, apperrors.ErrNotFound)
		return
	}

	respondMappingJSON(w, http.StatusOK, map[string]interface{}{
		"username":   username,
		"user_email": email,
	})
}

func mappingToResponse(m *repository.UserPrinterMapping) map[string]interface{} {
	resp := map[string]interface{}{
		"id":                  m.ID,
		"user_email":          m.UserEmail,
		"user_name":           m.UserName,
		"client_agent_id":     m.ClientAgentID,
		"target_printer_name": m.TargetPrinterName,
		"is_active":           m.IsActive,
		"is_default":          m.IsDefault,
		"created_at":          m.CreatedAt.Format(time.RFC3339),
		"updated_at":          m.UpdatedAt.Format(time.RFC3339),
	}
	if m.OrganizationID != "" {
		resp["organization_id"] = m.OrganizationID
	}
	if m.TargetPrinterID != "" {
		resp["target_printer_id"] = m.TargetPrinterID
	}
	if m.ServerAgentID != "" {
		resp["server_agent_id"] = m.ServerAgentID
	}
	return resp
}

func respondMappingJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondMappingError(w http.ResponseWriter, err error) {
	respondError(w, err)
}
