package handler

import (
	"testing"
	"time"

	"github.com/openprint/openprint/services/analytics-service/processor"
)

func TestCalculatePercentChange(t *testing.T) {
	tests := []struct {
		name     string
		old      int
		new      int
		expected float64
	}{
		{"increase from 100 to 150", 100, 150, 50.0},
		{"decrease from 100 to 50", 100, 50, -50.0},
		{"no change", 100, 100, 0.0},
		{"zero to positive", 0, 10, 100.0},
		{"zero to zero", 0, 0, 0.0},
		{"double", 50, 100, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePercentChange(tt.old, tt.new)
			if got != tt.expected {
				t.Errorf("calculatePercentChange(%d, %d) = %f, want %f", tt.old, tt.new, got, tt.expected)
			}
		})
	}
}

func TestCalculateTrends_EmptyData(t *testing.T) {
	h := &Handler{}
	trends := h.calculateTrends(nil)
	if trends.JobsChangePercent != 0 {
		t.Errorf("JobsChangePercent = %f, want 0", trends.JobsChangePercent)
	}
	if trends.PagesChangePercent != 0 {
		t.Errorf("PagesChangePercent = %f, want 0", trends.PagesChangePercent)
	}
}

func TestCalculateTrends_SingleDataPoint(t *testing.T) {
	h := &Handler{}
	data := []processor.DailyJobCount{
		{Date: time.Now(), Count: 10, Pages: 50},
	}
	trends := h.calculateTrends(data)
	if trends.JobsChangePercent != 0 {
		t.Errorf("JobsChangePercent = %f, want 0", trends.JobsChangePercent)
	}
}

func TestCalculateTrends_TwoWeeksData(t *testing.T) {
	h := &Handler{}
	now := time.Now()

	data := []processor.DailyJobCount{
		// Previous week (8-14 days ago)
		{Date: now.AddDate(0, 0, -10), Count: 10, Pages: 50},
		{Date: now.AddDate(0, 0, -11), Count: 10, Pages: 50},
		// Recent week (1-7 days ago)
		{Date: now.AddDate(0, 0, -3), Count: 15, Pages: 75},
		{Date: now.AddDate(0, 0, -4), Count: 15, Pages: 75},
	}

	trends := h.calculateTrends(data)
	// Recent: 30 jobs, Previous: 20 jobs => 50% increase
	if trends.JobsChangePercent != 50.0 {
		t.Errorf("JobsChangePercent = %f, want 50.0", trends.JobsChangePercent)
	}
	if trends.PagesChangePercent != 50.0 {
		t.Errorf("PagesChangePercent = %f, want 50.0", trends.PagesChangePercent)
	}
}
