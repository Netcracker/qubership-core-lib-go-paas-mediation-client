package exec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFixedRolloutExecutor(t *testing.T) {
	tests := []struct {
		name        string
		parallelism int
		bufferSize  int
		expectError bool
	}{
		{
			name:        "valid parameters",
			parallelism: 5,
			bufferSize:  10,
			expectError: false,
		},
		{
			name:        "zero parallelism",
			parallelism: 0,
			bufferSize:  10,
			expectError: false,
		},
		{
			name:        "zero buffer size",
			parallelism: 5,
			bufferSize:  0,
			expectError: false,
		},
		{
			name:        "negative parallelism",
			parallelism: -1,
			bufferSize:  10,
			expectError: false, // NewFixedPool should handle this
		},
		{
			name:        "negative buffer size",
			parallelism: 5,
			bufferSize:  -1,
			expectError: true, // This should panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				assert.Panics(t, func() {
					NewFixedRolloutExecutor(tt.parallelism, tt.bufferSize)
				})
			} else {
				executor := NewFixedRolloutExecutor(tt.parallelism, tt.bufferSize)
				assert.NotNil(t, executor)
			}
		})
	}
}
