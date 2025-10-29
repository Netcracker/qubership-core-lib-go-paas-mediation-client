package entity

import (
	"testing"
	"time"

	v12 "github.com/openshift/api/apps/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_NewDeployment_success(t *testing.T) {
	kuberDepl := v1.Deployment{
		Status: v1.DeploymentStatus{
			Conditions: []v1.DeploymentCondition{{}},
		},
	}
	kuberDeplExpected := &Deployment{
		Metadata: Metadata{Kind: "Deployment"},
		Status: DeploymentStatus{
			Conditions: []DeploymentCondition{{}},
		},
	}

	result := NewDeployment(&kuberDepl)

	assert.Equalf(t, kuberDeplExpected, result, "Not expected Deployment for income kuber deploymnet")
}

func Test_NewDeploymentConfig_success(t *testing.T) {
	openshiftDepl := v12.DeploymentConfig{
		Status: v12.DeploymentConfigStatus{
			Conditions: []v12.DeploymentCondition{{}},
		},
	}
	openshiftDeplExpected := &Deployment{
		Metadata: Metadata{Kind: "DeploymentConfig"},
		Spec:     DeploymentSpec{Replicas: &openshiftDepl.Spec.Replicas},
		Status: DeploymentStatus{
			Conditions: []DeploymentCondition{{}},
		},
	}
	result := NewDeploymentConfig(&openshiftDepl)

	assert.Equalf(t, openshiftDeplExpected, result, "Not expected Deployment for income openshift deploymnet")
}

func Test_givenNil_getFormattedTimeString_returnNil(t *testing.T) {
	var expected *string
	result := getFormattedTimeString(nil)
	assert.Equalf(t, expected, result, "Given nil time method should return nil")
}

func Test_givenZero_getFormattedTimeString_returnNil(t *testing.T) {
	var expected *string
	result := getFormattedTimeString(&metav1.Time{})
	assert.Equalf(t, expected, result, "Given zero time method should return nil")
}

func Test_givenNow_getFormattedTimeString_success(t *testing.T) {
	now := metav1.Now()
	result := getFormattedTimeString(&now)
	assert.Equalf(t, now.Format(time.RFC3339), *result, "Given now time method should return nil")
}

func Test_Deployment_GetMetadata(t *testing.T) {
	deployment := Deployment{
		Metadata: Metadata{
			Kind:      "Deployment",
			Name:      "test-deployment",
			Namespace: "test-namespace",
			Labels: map[string]string{
				"app": "test-app",
			},
		},
	}

	metadata := deployment.GetMetadata()

	assert.Equal(t, "Deployment", metadata.Kind)
	assert.Equal(t, "test-deployment", metadata.Name)
	assert.Equal(t, "test-namespace", metadata.Namespace)
	assert.Equal(t, "test-app", metadata.Labels["app"])
}

func Test_NewDeploymentList(t *testing.T) {
	apiDeployments := []*v1.Deployment{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment1",
				Namespace: "test-namespace",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment2",
				Namespace: "test-namespace",
			},
		},
	}

	deploymentList := NewDeploymentList(apiDeployments)

	assert.NotNil(t, deploymentList)
	assert.Equal(t, 2, len(deploymentList))
	assert.Equal(t, "deployment1", deploymentList[0].Name)
	assert.Equal(t, "deployment2", deploymentList[1].Name)
}

func Test_NewDeploymentList_EmptySlice(t *testing.T) {
	apiDeployments := []*v1.Deployment{}

	deploymentList := NewDeploymentList(apiDeployments)

	// The function should return an empty slice, not nil
	assert.Empty(t, deploymentList)
	assert.Len(t, deploymentList, 0)
}
