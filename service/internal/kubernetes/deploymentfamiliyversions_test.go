package kubernetes

import (
	"context"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func Test_GetDeploymentFamilyVersions_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	labels := make(map[string]string)
	labels[entity.AppNameProp] = "app_name-1"
	labels[entity.AppVersionProp] = "app_version-1"
	labels[entity.NameProp] = "name-1"
	labels[entity.FamilyNameProp] = "family-1"
	labels[entity.VersionProp] = "version-1"
	labels[entity.BlueGreenVersionProp] = "bluegreen_version-1"
	labels[entity.StateProp] = "state-1"

	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name",
			Namespace: testNamespace1,
			Labels:    labels,
		},
		Spec: v1.DeploymentSpec{}}

	clientset := fake.NewClientset(&deployment)
	cert_client := &certClient.Clientset{}
	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	// found
	result, err := kube.GetDeploymentFamilyVersions(ctx, "family-1", testNamespace1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "deployment must be not nil")
	r.Equal(1, len(result), "expected 1 deployment")
	r.Equal("app_name-1", result[0].AppName)
	r.Equal("app_version-1", result[0].AppVersion)
	r.Equal("state-1", result[0].State)

	// not found
	result2, err2 := kube.GetDeploymentFamilyVersions(ctx, "family1", testNamespace1)
	r.Nil(err2, "unexpected error returned")
	r.NotNil(result2, "deployment must be not nil")
	r.Equal(0, len(result2), "expected 0 deployment")
}

func Test_GetDeploymentFamilyVersions_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	clientset := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	expectedError := fmt.Errorf("test error during list Deployment")
	clientset.Fake.PrependReactor("list", "deployments",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})
	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	_, err = kube.GetDeploymentFamilyVersions(ctx, "some-family", testNamespace1)
	r.Equal(expectedError, err, "unexpected error returned")
}
