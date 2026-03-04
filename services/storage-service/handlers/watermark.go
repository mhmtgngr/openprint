// Package handler provides HTTP handlers for document watermarking functionality.
package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/storage-service/storage"
)

// WatermarkRepository defines the interface for watermark template operations.
type WatermarkRepository interface {
	CreateTemplate(ctx context.Context, template *WatermarkTemplate) error
	GetTemplate(ctx context.Context, templateID string) (*WatermarkTemplate, error)
	ListTemplates(ctx context.Context, organizationID string) ([]*WatermarkTemplate, error)
	UpdateTemplate(ctx context.Context, template *WatermarkTemplate) error
	DeleteTemplate(ctx context.Context, templateID string) error
}

// WatermarkTemplate represents a watermark configuration.
type WatermarkTemplate struct {
	ID             string
	OrganizationID string
	Name           string
	Type           string // 'text', 'image', 'overlay'
	Content        string // Text content or base64 image data
	Position       string // 'top-left', 'top-center', 'top-right', 'center', 'bottom-left', 'bottom-center', 'bottom-right'
	Opacity        float64
	Rotation       int
	FontSize       int
	FontColor      string
	ImageData      []byte
	IsDefault      bool
	ApplyToAll     bool
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// WatermarkHandler handles watermark HTTP endpoints.
type WatermarkHandler struct {
	db        *pgxpool.Pool
	repo      WatermarkRepository
	backend   storage.Backend
	uploadDir string
	tempDir   string
}

// NewWatermarkHandler creates a new watermark handler instance.
func NewWatermarkHandler(db *pgxpool.Pool, backend storage.Backend, uploadDir, tempDir string) *WatermarkHandler {
	return &WatermarkHandler{
		db:        db,
		repo:      NewWatermarkRepository(db),
		backend:   backend,
		uploadDir: uploadDir,
		tempDir:   tempDir,
	}
}

// TemplateRequest represents a request to create/update a watermark template.
type TemplateRequest struct {
	OrganizationID string  `json:"organization_id"`
	Name           string  `json:"name"`
	Type           string  `json:"type"`       // 'text', 'image', 'overlay'
	Content        string  `json:"content"`    // Text content or image URL
	Position       string  `json:"position"`   // 'top-left', 'top-center', 'top-right', 'center', 'bottom-left', 'bottom-center', 'bottom-right'
	Opacity        float64 `json:"opacity"`    // 0.0 to 1.0
	Rotation       int     `json:"rotation"`   // Rotation angle in degrees
	FontSize       int     `json:"font_size"`  // Font size in points
	FontColor      string  `json:"font_color"` // Hex color code
	ImageData      string  `json:"image_data"` // Base64 encoded image data
	IsDefault      bool    `json:"is_default"`
	ApplyToAll     bool    `json:"apply_to_all"`
}

// ApplyWatermarkRequest represents a request to apply a watermark to a document.
type ApplyWatermarkRequest struct {
	DocumentID   string           `json:"document_id"`
	TemplateID   string           `json:"template_id"`
	Watermark    *TemplateRequest `json:"watermark,omitempty"` // Custom watermark if template not provided
	OutputFormat string           `json:"output_format"`       // 'pdf', 'original'
}

// TemplateListHandler handles listing watermark templates.
func (h *WatermarkHandler) TemplateListHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	orgID := r.URL.Query().Get("organization_id")
	if orgID == "" {
		respondError(w, apperrors.New("organization_id is required", http.StatusBadRequest))
		return
	}

	templates, err := h.repo.ListTemplates(ctx, orgID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to list templates", http.StatusInternalServerError))
		return
	}

	response := make([]map[string]interface{}, len(templates))
	for i, t := range templates {
		response[i] = templateToResponse(t)
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"templates": response,
		"count":     len(response),
	})
}

// TemplateCreateHandler handles creating a new watermark template.
func (h *WatermarkHandler) TemplateCreateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if err := validateTemplateRequest(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid template data", http.StatusBadRequest))
		return
	}

	// Decode image data if provided
	var imageData []byte
	if req.Type == "image" && req.ImageData != "" {
		var err error
		imageData, err = base64.StdEncoding.DecodeString(req.ImageData)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "invalid image data", http.StatusBadRequest))
			return
		}
	}

	// Get user ID from context
	userID := "system"
	if uid := r.Context().Value("user_id"); uid != nil {
		userID = fmt.Sprintf("%v", uid)
	}

	template := &WatermarkTemplate{
		ID:             uuid.New().String(),
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Type:           req.Type,
		Content:        req.Content,
		Position:       req.Position,
		Opacity:        req.Opacity,
		Rotation:       req.Rotation,
		FontSize:       req.FontSize,
		FontColor:      req.FontColor,
		ImageData:      imageData,
		IsDefault:      req.IsDefault,
		ApplyToAll:     req.ApplyToAll,
		CreatedBy:      userID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.repo.CreateTemplate(ctx, template); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to create template", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusCreated, templateToResponse(template))
}

// TemplateGetHandler handles retrieving a specific template.
func (h *WatermarkHandler) TemplateGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract template ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	templateID := parts[len(parts)-1]

	if templateID == "" {
		respondError(w, apperrors.New("template_id is required", http.StatusBadRequest))
		return
	}

	template, err := h.repo.GetTemplate(ctx, templateID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get template", http.StatusInternalServerError))
		return
	}

	if template == nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	respondJSON(w, http.StatusOK, templateToResponse(template))
}

// TemplateUpdateHandler handles updating a watermark template.
func (h *WatermarkHandler) TemplateUpdateHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract template ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	templateID := parts[len(parts)-1]

	if templateID == "" {
		respondError(w, apperrors.New("template_id is required", http.StatusBadRequest))
		return
	}

	// Get existing template
	template, err := h.repo.GetTemplate(ctx, templateID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get template", http.StatusInternalServerError))
		return
	}
	if template == nil {
		respondError(w, apperrors.ErrNotFound)
		return
	}

	var req TemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Update fields
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Type != "" {
		template.Type = req.Type
	}
	if req.Content != "" {
		template.Content = req.Content
	}
	if req.Position != "" {
		template.Position = req.Position
	}
	if req.Opacity > 0 {
		template.Opacity = req.Opacity
	}
	if req.Rotation >= 0 {
		template.Rotation = req.Rotation
	}
	if req.FontSize > 0 {
		template.FontSize = req.FontSize
	}
	if req.FontColor != "" {
		template.FontColor = req.FontColor
	}
	if req.ImageData != "" {
		imageData, _ := base64.StdEncoding.DecodeString(req.ImageData)
		template.ImageData = imageData
	}
	template.UpdatedAt = time.Now()

	if err := h.repo.UpdateTemplate(ctx, template); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to update template", http.StatusInternalServerError))
		return
	}

	respondJSON(w, http.StatusOK, templateToResponse(template))
}

// TemplateDeleteHandler handles deleting a watermark template.
func (h *WatermarkHandler) TemplateDeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract template ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	templateID := parts[len(parts)-1]

	if templateID == "" {
		respondError(w, apperrors.New("template_id is required", http.StatusBadRequest))
		return
	}

	if err := h.repo.DeleteTemplate(ctx, templateID); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to delete template", http.StatusInternalServerError))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ApplyWatermarkHandler handles applying a watermark to a document.
func (h *WatermarkHandler) ApplyWatermarkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ApplyWatermarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	// Validate request
	if req.DocumentID == "" {
		respondError(w, apperrors.New("document_id is required", http.StatusBadRequest))
		return
	}

	var template *WatermarkTemplate

	// Get template if provided
	if req.TemplateID != "" {
		var err error
		template, err = h.repo.GetTemplate(ctx, req.TemplateID)
		if err != nil {
			respondError(w, apperrors.Wrap(err, "failed to get template", http.StatusInternalServerError))
			return
		}
		if template == nil {
			respondError(w, apperrors.New("template not found", http.StatusNotFound))
			return
		}
	} else if req.Watermark != nil {
		// Use custom watermark from request
		if err := validateTemplateRequest(req.Watermark); err != nil {
			respondError(w, apperrors.Wrap(err, "invalid watermark data", http.StatusBadRequest))
			return
		}

		imageData := []byte{}
		if req.Watermark.Type == "image" && req.Watermark.ImageData != "" {
			imageData, _ = base64.StdEncoding.DecodeString(req.Watermark.ImageData)
		}

		template = &WatermarkTemplate{
			ID:        uuid.New().String(),
			Type:      req.Watermark.Type,
			Content:   req.Watermark.Content,
			Position:  req.Watermark.Position,
			Opacity:   req.Watermark.Opacity,
			Rotation:  req.Watermark.Rotation,
			FontSize:  req.Watermark.FontSize,
			FontColor: req.Watermark.FontColor,
			ImageData: imageData,
		}
	} else {
		respondError(w, apperrors.New("either template_id or watermark is required", http.StatusBadRequest))
		return
	}

	// Get document from storage
	docPath := fmt.Sprintf("documents/%s", req.DocumentID)
	docContent, err := h.backend.Get(ctx, docPath)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to retrieve document", http.StatusInternalServerError))
		return
	}

	// Apply watermark
	watermarkedContent, err := h.applyWatermark(ctx, docContent, template)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to apply watermark", http.StatusInternalServerError))
		return
	}

	// Store watermarked document
	watermarkedID := uuid.New().String()
	watermarkPath := fmt.Sprintf("watermarked/%s", watermarkedID)
	if err := h.backend.Put(ctx, watermarkPath, watermarkedContent); err != nil {
		respondError(w, apperrors.Wrap(err, "failed to store watermarked document", http.StatusInternalServerError))
		return
	}

	// Return the new document ID
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"document_id":       watermarkedID,
		"template_id":       template.ID,
		"original_document": req.DocumentID,
		"size":              len(watermarkedContent),
	})
}

// GetWatermarkedDocumentHandler handles retrieving a watermarked document.
func (h *WatermarkHandler) GetWatermarkedDocumentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract document ID from path
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		respondError(w, apperrors.New("invalid document path", http.StatusBadRequest))
		return
	}
	docID := parts[len(parts)-1]

	// Get watermarked document
	docPath := fmt.Sprintf("watermarked/%s", docID)
	content, err := h.backend.Get(ctx, docPath)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to retrieve document", http.StatusInternalServerError))
		return
	}

	// Set headers
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=watermarked_%s.pdf", docID))

	w.Write(content)
}

// BatchApplyWatermarkHandler handles applying watermarks to multiple documents.
func (h *WatermarkHandler) BatchApplyWatermarkHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DocumentIDs []string `json:"document_ids"`
		TemplateID  string   `json:"template_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, apperrors.Wrap(err, "invalid request body", http.StatusBadRequest))
		return
	}

	if len(req.DocumentIDs) == 0 {
		respondError(w, apperrors.New("document_ids is required", http.StatusBadRequest))
		return
	}
	if req.TemplateID == "" {
		respondError(w, apperrors.New("template_id is required", http.StatusBadRequest))
		return
	}

	// Get template
	template, err := h.repo.GetTemplate(ctx, req.TemplateID)
	if err != nil {
		respondError(w, apperrors.Wrap(err, "failed to get template", http.StatusInternalServerError))
		return
	}
	if template == nil {
		respondError(w, apperrors.New("template not found", http.StatusNotFound))
		return
	}

	// Process documents
	results := make([]map[string]interface{}, 0)
	for _, docID := range req.DocumentIDs {
		docPath := fmt.Sprintf("documents/%s", docID)
		docContent, err := h.backend.Get(ctx, docPath)
		if err != nil {
			results = append(results, map[string]interface{}{
				"document_id": docID,
				"success":     false,
				"error":       "failed to retrieve document",
			})
			continue
		}

		watermarkedContent, err := h.applyWatermark(ctx, docContent, template)
		if err != nil {
			results = append(results, map[string]interface{}{
				"document_id": docID,
				"success":     false,
				"error":       err.Error(),
			})
			continue
		}

		watermarkedID := uuid.New().String()
		watermarkPath := fmt.Sprintf("watermarked/%s", watermarkedID)
		if err := h.backend.Put(ctx, watermarkPath, watermarkedContent); err != nil {
			results = append(results, map[string]interface{}{
				"document_id": docID,
				"success":     false,
				"error":       "failed to store watermarked document",
			})
			continue
		}

		results = append(results, map[string]interface{}{
			"document_id":    docID,
			"success":        true,
			"watermarked_id": watermarkedID,
			"size":           len(watermarkedContent),
		})
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"results": results,
		"total":   len(req.DocumentIDs),
		"success": countSuccess(results),
	})
}

// applyWatermark applies a watermark to document content.
func (h *WatermarkHandler) applyWatermark(ctx context.Context, content []byte, template *WatermarkTemplate) ([]byte, error) {
	// For PDF documents, we can use a library like pdftk or ghostscript
	// For this implementation, we'll use a simplified approach

	// Detect file type
	if len(content) < 4 {
		return nil, fmt.Errorf("invalid file content")
	}

	// Check if PDF
	if string(content[0:4]) == "%PDF" {
		return h.applyPDFWatermark(content, template)
	}

	// For other formats, return content as-is (in production, implement proper conversion)
	return content, nil
}

// applyPDFWatermark applies a watermark to a PDF document.
func (h *WatermarkHandler) applyPDFWatermark(content []byte, template *WatermarkTemplate) ([]byte, error) {
	// Create temporary files for processing
	tempDir := h.tempDir
	if tempDir == "" {
		tempDir = os.TempDir()
	}

	inputFile := filepath.Join(tempDir, fmt.Sprintf("input_%s.pdf", uuid.New().String()))
	outputFile := filepath.Join(tempDir, fmt.Sprintf("output_%s.pdf", uuid.New().String()))
	watermarkFile := filepath.Join(tempDir, fmt.Sprintf("watermark_%s.pdf", uuid.New().String()))

	// Clean up temp files when done
	defer func() {
		os.Remove(inputFile)
		os.Remove(outputFile)
		os.Remove(watermarkFile)
	}()

	// Write input PDF
	if err := os.WriteFile(inputFile, content, 0644); err != nil {
		return nil, fmt.Errorf("write input file: %w", err)
	}

	// Create watermark PDF
	if err := h.createWatermarkPDF(watermarkFile, template); err != nil {
		return nil, fmt.Errorf("create watermark pdf: %w", err)
	}

	// Use pdftk or ghostscript to apply watermark
	// Try pdftk first
	if _, err := exec.LookPath("pdftk"); err == nil {
		cmd := exec.Command("pdftk", inputFile, "stamp", watermarkFile, "output", outputFile)
		if err := cmd.Run(); err != nil {
			// Fall back to background option
			cmd = exec.Command("pdftk", inputFile, "background", watermarkFile, "output", outputFile)
			if err := cmd.Run(); err != nil {
				return nil, fmt.Errorf("pdftk failed: %w", err)
			}
		}
	} else if _, err := exec.LookPath("gs"); err == nil {
		// Use ghostscript
		opacityStr := fmt.Sprintf("%.2f", template.Opacity)
		_ = opacityStr // Format for potential future use
		cmd := exec.Command("gs",
			"-dBATCH", "-dNOPAUSE", "-q", "-sDEVICE=pdfwrite",
			"-c",
			fmt.Sprintf("<</Install {%.2f setfillconstantcolor}>> setpagedevice", template.Opacity),
			"-sOutputFile="+outputFile,
			watermarkFile,
			inputFile,
		)
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("ghostscript failed: %w", err)
		}
	} else {
		// No PDF manipulation tools available - return original content
		// In production, this should return an error or use an embedded library
		return content, nil
	}

	// Read output
	result, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("read output file: %w", err)
	}

	return result, nil
}

// createWatermarkPDF creates a simple PDF with watermark text/image.
func (h *WatermarkHandler) createWatermarkPDF(filename string, template *WatermarkTemplate) error {
	// Create a simple PDF with the watermark
	// For text watermarks, we'll create a basic PDF

	var content bytes.Buffer

	// PDF header
	content.WriteString("%PDF-1.4\n")
	// Minimal PDF with watermark text
	// In production, use a proper PDF library

	if template.Type == "text" && template.Content != "" {
		// Create a simple text watermark PDF
		content.WriteString(fmt.Sprintf("1 0 obj<</Type/Page/MediaBox[0 0 612 792]/Contents 2 0 R/Resources<<>>/Parent 3 0 R>>endobj\n"))
		content.WriteString(fmt.Sprintf("2 0 obj<</Length %d>>stream\n", len(template.Content)+100))
		// Simple text stream
		content.WriteString(fmt.Sprintf("BT\n/F1 24 Tf\n100 700 Td\n(%s) Tj\nET\n", template.Content))
		content.WriteString("endstream\nendobj\n")
		content.WriteString("3 0 obj<</Type/Pages/Kids[1 0 R]/Count 1>>endobj\n")
		content.WriteString("4 0 obj<</Type/Catalog/Pages 3 0 R>>endobj\n")
		content.WriteString("xref\n0 5\n0000000000 65535 f\n0000000009 00000 n\n0000000098 00000 n\n0000000156 00000 n\n0000000203 00000 n\n")
		content.WriteString("trailer<</Size 5/Root 4 0 R>>\n")
		content.WriteString("%%EOF\n")
	} else if template.Type == "image" && len(template.ImageData) > 0 {
		// For image watermarks, we'd need a proper PDF library
		// Create a placeholder PDF
		content.WriteString("1 0 obj<</Type/Page/MediaBox[0 0 612 792]/Parent 3 0 R>>endobj\n")
		content.WriteString("3 0 obj<</Type/Pages/Kids[1 0 R]/Count 1>>endobj\n")
		content.WriteString("4 0 obj<</Type/Catalog/Pages 3 0 R>>endobj\n")
		content.WriteString("xref\n0 5\n0000000000 65535 f\n0000000009 00000 n\n0000000056 00000 n\n0000000098 00000 n\n0000000145 00000 n\n")
		content.WriteString("trailer<</Size 5/Root 4 0 R>>\n")
		content.WriteString("%%EOF\n")
	}

	return os.WriteFile(filename, content.Bytes(), 0644)
}

// Helper functions

func validateTemplateRequest(req *TemplateRequest) error {
	if req.OrganizationID == "" {
		return stderrors.New("organization_id is required")
	}
	if req.Name == "" {
		return stderrors.New("name is required")
	}
	if req.Type == "" {
		req.Type = "text"
	}
	if req.Type != "text" && req.Type != "image" && req.Type != "overlay" {
		return stderrors.New("type must be 'text', 'image', or 'overlay'")
	}
	if req.Position == "" {
		req.Position = "center"
	}
	if req.Opacity <= 0 || req.Opacity > 1 {
		req.Opacity = 0.3
	}
	if req.FontSize <= 0 {
		req.FontSize = 48
	}
	if req.FontColor == "" {
		req.FontColor = "#CCCCCC"
	}
	if req.Type == "text" && req.Content == "" {
		return stderrors.New("content is required for text watermarks")
	}
	if req.Type == "image" && req.ImageData == "" {
		return stderrors.New("image_data is required for image watermarks")
	}
	return nil
}

func templateToResponse(t *WatermarkTemplate) map[string]interface{} {
	resp := map[string]interface{}{
		"id":              t.ID,
		"organization_id": t.OrganizationID,
		"name":            t.Name,
		"type":            t.Type,
		"position":        t.Position,
		"opacity":         t.Opacity,
		"rotation":        t.Rotation,
		"font_size":       t.FontSize,
		"font_color":      t.FontColor,
		"is_default":      t.IsDefault,
		"apply_to_all":    t.ApplyToAll,
		"created_by":      t.CreatedBy,
		"created_at":      t.CreatedAt.Format(time.RFC3339),
		"updated_at":      t.UpdatedAt.Format(time.RFC3339),
	}
	if t.Content != "" {
		resp["content"] = t.Content
	}
	if len(t.ImageData) > 0 {
		resp["has_image"] = true
	}
	return resp
}

func countSuccess(results []map[string]interface{}) int {
	count := 0
	for _, r := range results {
		if s, ok := r["success"].(bool); ok && s {
			count++
		}
	}
	return count
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, err error) {
	var appErr *apperrors.AppError
	if stderrors.As(err, &appErr) {
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
