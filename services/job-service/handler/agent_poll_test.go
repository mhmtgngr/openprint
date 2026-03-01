// Package handler provides unit tests for agent poll handlers.
package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/openprint/openprint/internal/agent"
	apperrors "github.com/openprint/openprint/internal/shared/errors"
	"github.com/openprint/openprint/services/job-service/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockAssignmentRepository is a mock implementation of AssignmentRepository.
type MockAssignmentRepository struct {
	assignments     []*repository.JobAssignment
	jobs            []*repository.PrintJob
	assignJobError  error
	findByAgentError error
	updateStatusError error
}

func (m *MockAssignmentRepository) GetJobsForAgentPolling(ctx context.Context, agentID, userEmail string, limit int) ([]*repository.PrintJob, error) {
	if m.jobs == nil {
		return []*repository.PrintJob{}, nil
	}
	if len(m.jobs) > limit {
		return m.jobs[:limit], nil
	}
	return m.jobs, nil
}

func (m *MockAssignmentRepository) AssignJob(ctx context.Context, assignment *repository.JobAssignment) error {
	if m.assignJobError != nil {
		return m.assignJobError
	}
	assignment.ID = "test-assignment-id"
	m.assignments = append(m.assignments, assignment)
	return nil
}

func (m *MockAssignmentRepository) FindByAgent(ctx context.Context, agentID string, limit int) ([]*repository.JobAssignment, error) {
	if m.findByAgentError != nil {
		return nil, m.findByAgentError
	}
	return m.assignments, nil
}

func (m *MockAssignmentRepository) UpdateStatus(ctx context.Context, assignmentID, status string) error {
	if m.updateStatusError != nil {
		return m.updateStatusError
	}
	for _, a := range m.assignments {
		if a.ID == assignmentID {
			a.Status = status
			return nil
		}
	}
	return errors.New("assignment not found")
}

func (m *MockAssignmentRepository) FindByJobAndAgent(ctx context.Context, jobID, agentID string) (*repository.JobAssignment, error) {
	for _, a := range m.assignments {
		if a.JobID == jobID && a.AgentID == agentID {
			return a, nil
		}
	}
	return nil, errors.New("assignment not found")
}

func (m *MockAssignmentRepository) UpdateHeartbeat(ctx context.Context, assignmentID string, heartbeat time.Time) error {
	return nil
}

func (m *MockAssignmentRepository) GetJobWithPrinter(ctx context.Context, jobID string) (*repository.PrintJob, map[string]interface{}, error) {
	for _, j := range m.jobs {
		if j.ID == jobID {
			printerInfo := map[string]interface{}{
				"name":   j.PrinterID,
				"status": "online",
			}
			return j, printerInfo, nil
		}
	}
	return nil, nil, errors.New("job not found")
}

func (m *MockAssignmentRepository) ResolveUserDefaultPrinter(ctx context.Context, userEmail, clientAgentID, printerType string) (string, string, error) {
	return "Default Printer", "printer-default-id", nil
}

// TestAgentPollHandler_PollJobs_Success tests successful job polling.
func TestAgentPollHandler_PollJobs_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockAssignmentRepository{
		jobs: []*repository.PrintJob{
			{
				ID:        "job-1",
				DocumentID: "doc-1",
				PrinterID: "printer-1",
				UserName:  "Test User",
				UserEmail: "test@example.com",
				Title:     "Test Document",
				Status:    "queued",
				CreatedAt: now,
			},
		},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo:     mockRepo,
		DocumentBaseURL:    "http://localhost:8004",
		AssignmentTimeout:  30 * time.Minute,
	})

	reqBody := agent.JobPollRequest{
		AgentID: "agent-123",
		Limit:   10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/agents/jobs/poll", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.PollJobs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp agent.JobPollResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp.Jobs, 1)
	assert.Equal(t, "job-1", resp.Jobs[0].JobID)
}

// TestAgentPollHandler_PollJobs_MissingAgentID tests request without agent_id.
func TestAgentPollHandler_PollJobs_MissingAgentID(t *testing.T) {
	mockRepo := &MockAssignmentRepository{}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	reqBody := agent.JobPollRequest{
		Limit: 10,
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/agents/jobs/poll", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.PollJobs(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&errResp)
	assert.Contains(t, errResp["message"], "agent_id")
}

// TestAgentPollHandler_PollJobs_InvalidLimit tests limit enforcement.
func TestAgentPollHandler_PollJobs_InvalidLimit(t *testing.T) {
	mockRepo := &MockAssignmentRepository{
		jobs: make([]*repository.PrintJob, 200), // Create many jobs
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	reqBody := agent.JobPollRequest{
		AgentID: "agent-123",
		Limit:   200, // Request more than max
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest("POST", "/agents/jobs/poll", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.PollJobs(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp agent.JobPollResponse
	json.NewDecoder(w.Body).Decode(&resp)
	// Should be capped at 100
	assert.LessOrEqual(t, len(resp.Jobs), 100)
}

// TestAgentPollHandler_UpdateJobStatus_Success tests successful status update.
func TestAgentPollHandler_UpdateJobStatus_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockAssignmentRepository{
		assignments: []*repository.JobAssignment{
			{
				ID:      "assignment-1",
				JobID:   "job-1",
				AgentID: "agent-123",
				Status:  "assigned",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	update := agent.JobStatusUpdate{
		JobID:    "job-1",
		AgentID:  "agent-123",
		Status:   "in_progress",
		Message:  "Processing",
		Timestamp: time.Now(),
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/agents/agent-123/jobs/job-1/status", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.UpdateJobStatus(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "in_progress", resp["status"])
}

// TestAgentPollHandler_UpdateJobStatus_NotFound tests updating non-existent job.
func TestAgentPollHandler_UpdateJobStatus_NotFound(t *testing.T) {
	mockRepo := &MockAssignmentRepository{
		assignments: []*repository.JobAssignment{},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	update := agent.JobStatusUpdate{
		JobID:    "non-existent",
		AgentID:  "agent-123",
		Status:   "in_progress",
		Timestamp: time.Now(),
	}
	body, _ := json.Marshal(update)

	req := httptest.NewRequest("PUT", "/agents/agent-123/jobs/non-existent/status", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.UpdateJobStatus(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestAgentPollHandler_CompleteJob_Success tests completing a job.
func TestAgentPollHandler_CompleteJob_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockAssignmentRepository{
		assignments: []*repository.JobAssignment{
			{
				ID:      "assignment-1",
				JobID:   "job-1",
				AgentID: "agent-123",
				Status:  "in_progress",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	completeReq := struct {
		PagesPrinted int    `json:"pages_printed"`
		Message      string `json:"message"`
	}{
		PagesPrinted: 5,
		Message:      "Successfully printed",
	}
	body, _ := json.Marshal(completeReq)

	req := httptest.NewRequest("POST", "/agents/agent-123/jobs/job-1/complete", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.CompleteJob(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "completed", resp["status"])
	assert.Equal(t, float64(5), resp["pages_printed"])
}

// TestAgentPollHandler_FailJob_Success tests failing a job.
func TestAgentPollHandler_FailJob_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockAssignmentRepository{
		assignments: []*repository.JobAssignment{
			{
				ID:         "assignment-1",
				JobID:      "job-1",
				AgentID:    "agent-123",
				Status:     "in_progress",
				RetryCount: 0,
				CreatedAt:  now,
				UpdatedAt:  now,
			},
		},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	failReq := struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"message"`
		Retry     bool   `json:"retry"`
	}{
		ErrorCode: "PRINTER_ERROR",
		Message:   "Printer offline",
		Retry:     true,
	}
	body, _ := json.Marshal(failReq)

	req := httptest.NewRequest("POST", "/agents/agent-123/jobs/job-1/fail", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler.FailJob(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "failed", resp["status"])
	assert.Equal(t, "PRINTER_ERROR", resp["error_code"])
	assert.Equal(t, true, resp["retry_eligible"])
}

// TestAgentPollHandler_JobHeartbeat_Success tests sending job heartbeat.
func TestAgentPollHandler_JobHeartbeat_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockAssignmentRepository{
		assignments: []*repository.JobAssignment{
			{
				ID:      "assignment-1",
				JobID:   "job-1",
				AgentID: "agent-123",
				Status:  "in_progress",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo: mockRepo,
	})

	req := httptest.NewRequest("POST", "/agents/agent-123/jobs/job-1/heartbeat", nil)
	w := httptest.NewRecorder()

	handler.JobHeartbeat(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "assignment-1", resp["assignment_id"])
}

// TestAgentPollHandler_GetJobDetails_Success tests getting job details.
func TestAgentPollHandler_GetJobDetails_Success(t *testing.T) {
	now := time.Now()
	mockRepo := &MockAssignmentRepository{
		jobs: []*repository.PrintJob{
			{
				ID:        "job-1",
				DocumentID: "doc-1",
				PrinterID: "printer-1",
				UserName:  "Test User",
				UserEmail: "test@example.com",
				Title:     "Test Document",
				Status:    "queued",
				CreatedAt: now,
			},
		},
	}

	handler := NewAgentPollHandler(AgentPollConfig{
		AssignmentRepo:  mockRepo,
		DocumentBaseURL: "http://localhost:8004",
	})

	req := httptest.NewRequest("GET", "/agents/agent-123/jobs/job-1", nil)
	w := httptest.NewRecorder()

	handler.GetJobDetails(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	assert.Equal(t, "job-1", resp["job_id"])
	assert.Equal(t, "printer-1", resp["printer_id"])
}

// TestRespondPollError tests the error response helper.
func TestRespondPollError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "AppError",
			err:        apperrors.New("test error", http.StatusBadRequest),
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "NotFoundError",
			err:        apperrors.ErrNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "NOT_FOUND",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondPollError(w, tt.err)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp map[string]interface{}
			json.NewDecoder(w.Body).Decode(&resp)
			assert.Equal(t, tt.wantCode, resp["code"])
		})
	}
}
