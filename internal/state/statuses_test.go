package state

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		status JobStatus
		want   string
	}{
		{StatusQueued, "queued"},
		{StatusProcessing, "processing"},
		{StatusSucceeded, "succeeded"},
		{StatusFailed, "failed"},
		{StatusRetrying, "retrying"},
		{StatusDead, "dead"},
		{JobStatus("custom"), "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.status.String())
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name  string
		from  JobStatus
		to    JobStatus
		valid bool
	}{
		{"queued to processing", StatusQueued, StatusProcessing, true},
		{"processing to succeeded", StatusProcessing, StatusSucceeded, true},
		{"processing to failed", StatusProcessing, StatusFailed, true},
		{"failed to retrying", StatusFailed, StatusRetrying, true},
		{"retrying to processing", StatusRetrying, StatusProcessing, true},
		{"failed to dead", StatusFailed, StatusDead, true},
		{"queued to succeeded (invalid)", StatusQueued, StatusSucceeded, false},
		{"succeeded to failed (invalid)", StatusSucceeded, StatusFailed, false},
		{"dead to queued (invalid)", StatusDead, StatusQueued, false},
		{"processing to retrying (invalid)", StatusProcessing, StatusRetrying, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTransition(tt.from, tt.to)
			assert.Equal(t, tt.valid, got)
		})
	}
}

func TestAllStatuses(t *testing.T) {
	assert.Len(t, AllStatuses, 6)
	assert.Contains(t, AllStatuses, StatusQueued)
	assert.Contains(t, AllStatuses, StatusProcessing)
	assert.Contains(t, AllStatuses, StatusSucceeded)
	assert.Contains(t, AllStatuses, StatusFailed)
	assert.Contains(t, AllStatuses, StatusRetrying)
	assert.Contains(t, AllStatuses, StatusDead)
}

func TestValidTransitions(t *testing.T) {
	assert.Len(t, ValidTransitions, 6)
	for _, tr := range ValidTransitions {
		assert.True(t, IsValidTransition(tr.From, tr.To),
			"ValidTransitions should match IsValidTransition for %s -> %s", tr.From, tr.To)
	}
}
