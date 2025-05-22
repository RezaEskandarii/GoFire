package state

import (
	"testing"
)

func TestJobStatus_String(t *testing.T) {
	tests := []struct {
		name     string
		status   JobStatus
		expected string
	}{
		{
			name:     "Queued status",
			status:   StatusQueued,
			expected: "queued",
		},
		{
			name:     "Processing status",
			status:   StatusProcessing,
			expected: "processing",
		},
		{
			name:     "Succeeded status",
			status:   StatusSucceeded,
			expected: "succeeded",
		},
		{
			name:     "Failed status",
			status:   StatusFailed,
			expected: "failed",
		},
		{
			name:     "Retrying status",
			status:   StatusRetrying,
			expected: "retrying",
		},
		{
			name:     "Dead status",
			status:   StatusDead,
			expected: "dead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsValidTransition(t *testing.T) {
	tests := []struct {
		name     string
		from     JobStatus
		to       JobStatus
		expected bool
	}{
		{
			name:     "Valid: Queued to Processing",
			from:     StatusQueued,
			to:       StatusProcessing,
			expected: true,
		},
		{
			name:     "Valid: Processing to Succeeded",
			from:     StatusProcessing,
			to:       StatusSucceeded,
			expected: true,
		},
		{
			name:     "Valid: Processing to Failed",
			from:     StatusProcessing,
			to:       StatusFailed,
			expected: true,
		},
		{
			name:     "Valid: Failed to Retrying",
			from:     StatusFailed,
			to:       StatusRetrying,
			expected: true,
		},
		{
			name:     "Valid: Retrying to Processing",
			from:     StatusRetrying,
			to:       StatusProcessing,
			expected: true,
		},
		{
			name:     "Valid: Failed to Dead",
			from:     StatusFailed,
			to:       StatusDead,
			expected: true,
		},
		{
			name:     "Invalid: Queued to Succeeded",
			from:     StatusQueued,
			to:       StatusSucceeded,
			expected: false,
		},
		{
			name:     "Invalid: Succeeded to Failed",
			from:     StatusSucceeded,
			to:       StatusFailed,
			expected: false,
		},
		{
			name:     "Invalid: Dead to Processing",
			from:     StatusDead,
			to:       StatusProcessing,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidTransition(tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("IsValidTransition() = %v, want %v", result, tt.expected)
			}
		})
	}
}
