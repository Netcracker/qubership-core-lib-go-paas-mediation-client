package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeploymentToDeploymentFamilyVersion(t *testing.T) {
	tests := []struct {
		name     string
		labels   map[string]string
		expected DeploymentFamilyVersion
	}{
		{
			name: "complete labels",
			labels: map[string]string{
				AppNameProp:          "test-app",
				AppVersionProp:       "1.0.0",
				NameProp:             "test-deployment",
				FamilyNameProp:       "test-family",
				BlueGreenVersionProp: "blue",
				VersionProp:          "v1",
				StateProp:            "active",
			},
			expected: DeploymentFamilyVersion{
				AppName:          "test-app",
				AppVersion:       "1.0.0",
				Name:             "test-deployment",
				FamilyName:       "test-family",
				BlueGreenVersion: "blue",
				Version:          "v1",
				State:            "active",
			},
		},
		{
			name:     "empty labels",
			labels:   map[string]string{},
			expected: DeploymentFamilyVersion{},
		},
		{
			name: "partial labels",
			labels: map[string]string{
				AppNameProp: "test-app",
				VersionProp: "v1",
			},
			expected: DeploymentFamilyVersion{
				AppName: "test-app",
				Version: "v1",
			},
		},
		{
			name: "labels with extra keys",
			labels: map[string]string{
				AppNameProp: "test-app",
				VersionProp: "v1",
				"extra":     "value",
			},
			expected: DeploymentFamilyVersion{
				AppName: "test-app",
				Version: "v1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeploymentToDeploymentFamilyVersion(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}
