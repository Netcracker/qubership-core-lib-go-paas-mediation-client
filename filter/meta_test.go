package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeta_GetAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expected    map[string]string
	}{
		{
			name:        "nil annotations",
			annotations: nil,
			expected:    nil,
		},
		{
			name:        "empty annotations",
			annotations: map[string]string{},
			expected:    map[string]string{},
		},
		{
			name: "single annotation",
			annotations: map[string]string{
				"key1": "value1",
			},
			expected: map[string]string{
				"key1": "value1",
			},
		},
		{
			name: "multiple annotations",
			annotations: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			expected: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := Meta{
				Annotations: tt.annotations,
			}

			result := meta.GetAnnotations()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMeta_GetLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected map[string]string
	}{
		{
			name:     "nil labels",
			labels:   nil,
			expected: nil,
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: map[string]string{},
		},
		{
			name: "single label",
			labels: map[string]string{
				"app": "test-app",
			},
			expected: map[string]string{
				"app": "test-app",
			},
		},
		{
			name: "multiple labels",
			labels: map[string]string{
				"app":     "test-app",
				"version": "v1.0.0",
				"env":     "production",
			},
			expected: map[string]string{
				"app":     "test-app",
				"version": "v1.0.0",
				"env":     "production",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := Meta{
				Labels: tt.labels,
			}

			result := meta.GetLabels()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMeta_Complete(t *testing.T) {
	meta := Meta{
		Labels: map[string]string{
			"app":     "test-app",
			"version": "v1.0.0",
		},
		Annotations: map[string]string{
			"description": "test resource",
			"owner":       "team-a",
		},
		WatchBookmark: true,
	}

	// Test labels
	labels := meta.GetLabels()
	assert.Equal(t, "test-app", labels["app"])
	assert.Equal(t, "v1.0.0", labels["version"])

	// Test annotations
	annotations := meta.GetAnnotations()
	assert.Equal(t, "test resource", annotations["description"])
	assert.Equal(t, "team-a", annotations["owner"])

	// Test WatchBookmark
	assert.True(t, meta.WatchBookmark)
}

func TestMeta_ZeroValue(t *testing.T) {
	var meta Meta

	// Test zero value behavior
	labels := meta.GetLabels()
	assert.Nil(t, labels)

	annotations := meta.GetAnnotations()
	assert.Nil(t, annotations)

	assert.False(t, meta.WatchBookmark)
}
