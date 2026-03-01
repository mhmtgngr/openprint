// Package main is the entry point for the OpenPrint Compliance Service.
// This service handles FedRAMP, HIPAA, GDPR, and SOC2 compliance tracking and reporting.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
)

// ComplianceFramework represents the compliance framework type.
type ComplianceFramework string

const (
	FrameworkFedRAMP ComplianceFramework = "fedramp"
	FrameworkHIPAA  ComplianceFramework = "hipaa"
	FrameworkGDPR   ComplianceFramework = "gdpr"
	FrameworkSOC2   ComplianceFramework = "soc2"
)

// ComplianceStatus represents the status of compliance requirements.
type ComplianceStatus string

const (
	StatusCompliant     ComplianceStatus = "compliant"
	StatusNonCompliant  ComplianceStatus = "non_compliant"
	StatusPending       ComplianceStatus = "pending"
	StatusNotApplicable ComplianceStatus = "not_applicable"
	StatusUnknown       ComplianceStatus = "unknown"
)

// Control represents a compliance control requirement.
type Control struct {
	ID               string               `json:"id"`
	Framework        ComplianceFramework  `json:"framework"`
	Family           string               `json:"family"`
	Title            string               `json:"title"`
	Description      string               `json:"description"`
	Implementation   string               `json:"implementation"`
	Status           ComplianceStatus     `json:"status"`
	LastAssessed     *time.Time           `json:"last_assessed"`
	NextReview       *time.Time           `json:"next_review"`
	EvidenceCount    int                  `json:"evidence_count"`
	Policies         []string             `json:"policies"`
	ResponsibleTeam  string               `json:"responsible_team"`
	RiskLevel        string               `json:"risk_level"`
	Remediation      *RemediationPlan     `json:"remediation,omitempty"`
}

// RemediationPlan represents a plan to address compliance gaps.
type RemediationPlan struct {
	ID          string    `json:"id"`
	ControlID   string    `json:"control_id"`
	Description string    `json:"description"`
	Priority    string    `json:"priority"`
	TargetDate  time.Time `json:"target_date"`
	Assignee    string    `json:"assignee"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditEvent represents a compliance-related audit event.
type AuditEvent struct {
	ID            string            `json:"id"`
	Timestamp     time.Time         `json:"timestamp"`
	EventType     string            `json:"event_type"`
	Category      string            `json:"category"`
	UserID        string            `json:"user_id"`
	UserName      string            `json:"user_name"`
	ResourceID    string            `json:"resource_id"`
	ResourceType  string            `json:"resource_type"`
	Action        string            `json:"action"`
	Outcome       string            `json:"outcome"`
	IPAddress     string            `json:"ip_address"`
	UserAgent     string            `json:"user_agent"`
	Metadata      map[string]string `json:"metadata"`
	RetentionDate *time.Time        `json:"retention_date"`
}

// DataBreach represents a data breach incident for tracking.
type DataBreach struct {
	ID                string    `json:"id"`
	DiscoveredAt      time.Time `json:"discovered_at"`
	ReportedAt        time.Time `json:"reported_at"`
	Severity          string    `json:"severity"`
	AffectedRecords   int       `json:"affected_records"`
	DataTypes         []string  `json:"data_types"`
	Description       string    `json:"description"`
	ContainmentStatus string    `json:"containment_status"`
	NotificationSent  bool      `json:"notification_sent"`
	ResolvedAt        *time.Time `json:"resolved_at"`
	LessonsLearned    string    `json:"lessons_learned"`
}

// ComplianceReport represents a generated compliance report.
type ComplianceReport struct {
	ID              string                 `json:"id"`
	Framework       ComplianceFramework    `json:"framework"`
	PeriodStart     time.Time              `json:"period_start"`
	PeriodEnd       time.Time              `json:"period_end"`
	OverallStatus   ComplianceStatus       `json:"overall_status"`
	CompliantCount  int                    `json:"compliant_count"`
	NonCompliant    int                    `json:"non_compliant_count"`
	PendingCount    int                    `json:"pending_count"`
	TotalControls   int                    `json:"total_controls"`
	HighRiskCount   int                    `json:"high_risk_count"`
	Findings        []Finding              `json:"findings"`
	GeneratedAt     time.Time              `json:"generated_at"`
	GeneratedBy     string                 `json:"generated_by"`
	ReportHash      string                 `json:"report_hash"`
	Signature       string                 `json:"signature,omitempty"`
}

// Finding represents a compliance finding or issue.
type Finding struct {
	ID            string         `json:"id"`
	ControlID     string         `json:"control_id"`
	Severity      string         `json:"severity"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Evidence      []EvidenceItem `json:"evidence"`
	Recommendation string        `json:"recommendation"`
	Status        string         `json:"status"`
	OpenedAt      time.Time      `json:"opened_at"`
	ClosedAt      *time.Time     `json:"closed_at,omitempty"`
}

// EvidenceItem represents evidence for a compliance finding.
type EvidenceItem struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	FilePath    string    `json:"file_path"`
	CollectedAt time.Time `json:"collected_at"`
	CollectedBy string    `json:"collected_by"`
	Hash        string    `json:"hash"`
}

// Service provides compliance tracking functionality.
type Service struct {
	db *pgxpool.Pool
}

// Config holds service configuration.
type Config struct {
	DB *pgxpool.Pool
}

// New creates a new compliance service.
func New(cfg Config) *Service {
	return &Service{
		db: cfg.DB,
	}
}

// Repository handles compliance data operations.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new compliance repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetControl retrieves a compliance control by ID.
func (r *Repository) GetControl(ctx context.Context, controlID string) (*Control, error) {
	query := `
		SELECT id, framework, family, title, description, implementation,
		       status, last_assessed, next_review, evidence_count,
		       policies, responsible_team, risk_level
		FROM compliance_controls
		WHERE id = $1
	`

	var control Control
	var policiesJSON []byte

	err := r.db.QueryRow(ctx, query, controlID).Scan(
		&control.ID,
		&control.Framework,
		&control.Family,
		&control.Title,
		&control.Description,
		&control.Implementation,
		&control.Status,
		&control.LastAssessed,
		&control.NextReview,
		&control.EvidenceCount,
		&policiesJSON,
		&control.ResponsibleTeam,
		&control.RiskLevel,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, apperrors.ErrNotFound
		}
		return nil, fmt.Errorf("get control: %w", err)
	}

	if len(policiesJSON) > 0 {
		json.Unmarshal(policiesJSON, &control.Policies)
	}

	return &control, nil
}

// ListControls retrieves all controls for a framework with optional filtering.
func (r *Repository) ListControls(ctx context.Context, framework ComplianceFramework, status ComplianceStatus, limit, offset int) ([]*Control, int, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if framework != "" {
		whereClause += fmt.Sprintf(" AND framework = $%d", argIdx)
		args = append(args, framework)
		argIdx++
	}

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM compliance_controls " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count controls: %w", err)
	}

	// Get controls
	query := `
		SELECT id, framework, family, title, description, implementation,
		       status, last_assessed, next_review, evidence_count,
		       policies, responsible_team, risk_level
		FROM compliance_controls
	` + whereClause + `
		ORDER BY family, id
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list controls: %w", err)
	}
	defer rows.Close()

	var controls []*Control
	for rows.Next() {
		var control Control
		var policiesJSON []byte

		if err := rows.Scan(
			&control.ID,
			&control.Framework,
			&control.Family,
			&control.Title,
			&control.Description,
			&control.Implementation,
			&control.Status,
			&control.LastAssessed,
			&control.NextReview,
			&control.EvidenceCount,
			&policiesJSON,
			&control.ResponsibleTeam,
			&control.RiskLevel,
		); err != nil {
			return nil, 0, err
		}

		if len(policiesJSON) > 0 {
			json.Unmarshal(policiesJSON, &control.Policies)
		}

		controls = append(controls, &control)
	}

	return controls, total, rows.Err()
}

// UpdateControlStatus updates the status of a compliance control.
func (r *Repository) UpdateControlStatus(ctx context.Context, controlID string, status ComplianceStatus, lastAssessed time.Time, nextReview time.Time) error {
	query := `
		UPDATE compliance_controls
		SET status = $2, last_assessed = $3, next_review = $4, updated_at = $5
		WHERE id = $1
	`

	cmdTag, err := r.db.Exec(ctx, query, controlID, status, lastAssessed, nextReview, time.Now())
	if err != nil {
		return fmt.Errorf("update control status: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}

// CreateAuditEvent creates a new audit event.
func (r *Repository) CreateAuditEvent(ctx context.Context, event *AuditEvent) error {
	event.ID = uuid.New().String()
	event.Timestamp = time.Now()

	// Calculate retention date (7 years for HIPAA, other frameworks vary)
	retentionDate := time.Now().AddDate(7, 0, 0)
	event.RetentionDate = &retentionDate

	metadataJSON, _ := json.Marshal(event.Metadata)

	query := `
		INSERT INTO audit_log
		(id, timestamp, event_type, category, user_id, user_name, resource_id,
		 resource_type, action, outcome, ip_address, user_agent, metadata,
		 retention_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err := r.db.Exec(ctx, query,
		event.ID, event.Timestamp, event.EventType, event.Category,
		event.UserID, event.UserName, event.ResourceID, event.ResourceType,
		event.Action, event.Outcome, event.IPAddress, event.UserAgent,
		metadataJSON, event.RetentionDate,
	)

	if err != nil {
		return fmt.Errorf("create audit event: %w", err)
	}

	return nil
}

// QueryAuditEvents retrieves audit events with filtering.
func (r *Repository) QueryAuditEvents(ctx context.Context, filter AuditFilter) ([]*AuditEvent, int, error) {
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if !filter.StartTime.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp >= $%d", argIdx)
		args = append(args, filter.StartTime)
		argIdx++
	}

	if !filter.EndTime.IsZero() {
		whereClause += fmt.Sprintf(" AND timestamp <= $%d", argIdx)
		args = append(args, filter.EndTime)
		argIdx++
	}

	if filter.UserID != "" {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, filter.UserID)
		argIdx++
	}

	if filter.EventType != "" {
		whereClause += fmt.Sprintf(" AND event_type = $%d", argIdx)
		args = append(args, filter.EventType)
		argIdx++
	}

	if filter.Category != "" {
		whereClause += fmt.Sprintf(" AND category = $%d", argIdx)
		args = append(args, filter.Category)
		argIdx++
	}

	// Get total count
	var total int
	countQuery := "SELECT COUNT(*) FROM audit_log " + whereClause
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}

	// Get events
	query := `
		SELECT id, timestamp, event_type, category, user_id, user_name,
		       resource_id, resource_type, action, outcome, ip_address,
		       user_agent, metadata, retention_date
		FROM audit_log
	` + whereClause + `
		ORDER BY timestamp DESC
		LIMIT $` + fmt.Sprintf("%d", argIdx) + ` OFFSET $` + fmt.Sprintf("%d", argIdx+1)

	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	var events []*AuditEvent
	for rows.Next() {
		var event AuditEvent
		var metadataJSON []byte

		if err := rows.Scan(
			&event.ID, &event.Timestamp, &event.EventType, &event.Category,
			&event.UserID, &event.UserName, &event.ResourceID, &event.ResourceType,
			&event.Action, &event.Outcome, &event.IPAddress, &event.UserAgent,
			&metadataJSON, &event.RetentionDate,
		); err != nil {
			return nil, 0, err
		}

		if len(metadataJSON) > 0 {
			json.Unmarshal(metadataJSON, &event.Metadata)
		}

		events = append(events, &event)
	}

	return events, total, rows.Err()
}

// AuditFilter represents filters for querying audit events.
type AuditFilter struct {
	StartTime time.Time
	EndTime   time.Time
	UserID    string
	EventType string
	Category  string
	Limit     int
	Offset    int
}

// GenerateReport generates a compliance report for the given framework.
func (s *Service) GenerateReport(ctx context.Context, framework ComplianceFramework, periodStart, periodEnd time.Time, generatedBy string) (*ComplianceReport, error) {
	repo := NewRepository(s.db)

	// Get all controls for the framework
	controls, total, err := repo.ListControls(ctx, framework, "", 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("get controls: %w", err)
	}

	// Calculate statistics
	report := &ComplianceReport{
		ID:            uuid.New().String(),
		Framework:     framework,
		PeriodStart:   periodStart,
		PeriodEnd:     periodEnd,
		TotalControls: total,
		GeneratedAt:   time.Now(),
		GeneratedBy:   generatedBy,
		Findings:      []Finding{},
	}

	compliantCount := 0
	nonCompliantCount := 0
	pendingCount := 0
	highRiskCount := 0

	for _, control := range controls {
		switch control.Status {
		case StatusCompliant:
			compliantCount++
		case StatusNonCompliant:
			nonCompliantCount++
			if control.RiskLevel == "high" {
				highRiskCount++
			}
			// Add finding for non-compliant control
			report.Findings = append(report.Findings, Finding{
				ID:          uuid.New().String(),
				ControlID:   control.ID,
				Severity:    control.RiskLevel,
				Title:       control.Title + " - Non-Compliant",
				Description: control.Description,
				Recommendation: "Update implementation to meet " + string(framework) + " requirements",
				Status:      "open",
				OpenedAt:    time.Now(),
			})
		case StatusPending:
			pendingCount++
		}
	}

	report.CompliantCount = compliantCount
	report.NonCompliant = nonCompliantCount
	report.PendingCount = pendingCount
	report.HighRiskCount = highRiskCount

	// Determine overall status
	if nonCompliantCount > 0 {
		report.OverallStatus = StatusNonCompliant
	} else if pendingCount > 0 {
		report.OverallStatus = StatusPending
	} else {
		report.OverallStatus = StatusCompliant
	}

	return report, nil
}

// GetComplianceSummary returns a summary of compliance status across all frameworks.
func (s *Service) GetComplianceSummary(ctx context.Context) (map[ComplianceFramework]ComplianceStatus, error) {
	repo := NewRepository(s.db)

	summary := make(map[ComplianceFramework]ComplianceStatus)

	frameworks := []ComplianceFramework{FrameworkFedRAMP, FrameworkHIPAA, FrameworkGDPR, FrameworkSOC2}

	for _, fw := range frameworks {
		controls, _, err := repo.ListControls(ctx, fw, "", 1000, 0)
		if err != nil {
			return nil, fmt.Errorf("list controls for %s: %w", fw, err)
		}

		hasNonCompliant := false
		hasPending := false
		hasCompliant := false

		for _, c := range controls {
			switch c.Status {
			case StatusNonCompliant:
				hasNonCompliant = true
			case StatusPending:
				hasPending = true
			case StatusCompliant:
				hasCompliant = true
			}
		}

		if hasNonCompliant {
			summary[fw] = StatusNonCompliant
		} else if hasPending {
			summary[fw] = StatusPending
		} else if hasCompliant {
			summary[fw] = StatusCompliant
		} else {
			summary[fw] = StatusUnknown
		}
	}

	return summary, nil
}

// ExportAuditLogs exports audit logs in the specified format (CSV, JSON).
func (s *Service) ExportAuditLogs(ctx context.Context, filter AuditFilter, format string) ([]byte, error) {
	events, _, err := NewRepository(s.db).QueryAuditEvents(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}

	switch format {
	case "json":
		return json.MarshalIndent(events, "", "  ")
	case "csv":
		return s.exportToCSV(events)
	default:
		return nil, apperrors.New("unsupported export format", http.StatusBadRequest)
	}
}

// exportToCSV converts audit events to CSV format.
func (s *Service) exportToCSV(events []*AuditEvent) ([]byte, error) {
	// CSV header
	csv := "Timestamp,EventType,Category,UserID,UserName,ResourceID,ResourceType,Action,Outcome,IPAddress\n"

	for _, e := range events {
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			e.Timestamp.Format(time.RFC3339),
			e.EventType, e.Category, e.UserID, e.UserName,
			e.ResourceID, e.ResourceType, e.Action, e.Outcome, e.IPAddress)
	}

	return []byte(csv), nil
}

// RecordDataBreach records a data breach incident.
func (r *Repository) RecordDataBreach(ctx context.Context, breach *DataBreach) error {
	breach.ID = uuid.New().String()
	breach.ReportedAt = time.Now()

	dataTypesJSON, _ := json.Marshal(breach.DataTypes)

	query := `
		INSERT INTO data_breaches
		(id, discovered_at, reported_at, severity, affected_records, data_types,
		 description, containment_status, notification_sent, resolved_at, lessons_learned)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		breach.ID, breach.DiscoveredAt, breach.ReportedAt, breach.Severity,
		breach.AffectedRecords, dataTypesJSON, breach.Description,
		breach.ContainmentStatus, breach.NotificationSent, breach.ResolvedAt,
		breach.LessonsLearned,
	)

	if err != nil {
		return fmt.Errorf("record data breach: %w", err)
	}

	return nil
}

// GetPendingReviews returns controls with upcoming or overdue reviews.
func (r *Repository) GetPendingReviews(ctx context.Context, within time.Duration) ([]*Control, error) {
	query := `
		SELECT id, framework, family, title, description, implementation,
		       status, last_assessed, next_review, evidence_count,
		       policies, responsible_team, risk_level
		FROM compliance_controls
		WHERE next_review <= $1
		ORDER BY next_review ASC
	`

	rows, err := r.db.Query(ctx, query, time.Now().Add(within))
	if err != nil {
		return nil, fmt.Errorf("get pending reviews: %w", err)
	}
	defer rows.Close()

	var controls []*Control
	for rows.Next() {
		var control Control
		var policiesJSON []byte

		if err := rows.Scan(
			&control.ID, &control.Framework, &control.Family, &control.Title,
			&control.Description, &control.Implementation, &control.Status,
			&control.LastAssessed, &control.NextReview, &control.EvidenceCount,
			&policiesJSON, &control.ResponsibleTeam, &control.RiskLevel,
		); err != nil {
			return nil, err
		}

		if len(policiesJSON) > 0 {
			json.Unmarshal(policiesJSON, &control.Policies)
		}

		controls = append(controls, &control)
	}

	return controls, rows.Err()
}
