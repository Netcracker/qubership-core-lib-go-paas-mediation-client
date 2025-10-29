package openshiftV3

import (
	"context"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	kube "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	openshiftappsfake "github.com/openshift/client-go/apps/clientset/versioned/fake"
	openshiftprojectfake "github.com/openshift/client-go/project/clientset/versioned/fake"
	openshiftroutefake "github.com/openshift/client-go/route/clientset/versioned/fake"
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

	labels1 := make(map[string]string)
	labels1[entity.AppNameProp] = "app_name-1"
	labels1[entity.AppVersionProp] = "app_version-1"
	labels1[entity.NameProp] = "name-1"
	labels1[entity.FamilyNameProp] = "family-1"
	labels1[entity.VersionProp] = "version-1"
	labels1[entity.BlueGreenVersionProp] = "bluegreen_version-1"
	labels1[entity.StateProp] = "state-1"

	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-name1", Namespace: testNamespace, Labels: labels1},
		Spec:       v1.DeploymentSpec{},
	}
	kubeClientSet := fake.NewClientset(&deployment)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	labels2 := make(map[string]string)
	labels2[entity.AppNameProp] = "app_name-2"
	labels2[entity.AppVersionProp] = "app_version-2"
	labels2[entity.NameProp] = "name-2"
	labels2[entity.FamilyNameProp] = "family-1"
	labels2[entity.VersionProp] = "version-2"
	labels2[entity.BlueGreenVersionProp] = "bluegreen_version-2"
	labels2[entity.StateProp] = "state-2"

	osDeployment1 := openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-name2", Namespace: testNamespace, Labels: labels2},
	}

	labels3 := make(map[string]string)
	labels3[entity.AppNameProp] = "app_name-3"
	labels3[entity.AppVersionProp] = "app_version-3"
	labels3[entity.NameProp] = "name-3"
	labels3[entity.FamilyNameProp] = "family-2"
	labels3[entity.VersionProp] = "version-3"
	labels3[entity.BlueGreenVersionProp] = "bluegreen_version-3"
	labels3[entity.StateProp] = "state-3"

	osDeployment2 := openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-name3", Namespace: testNamespace, Labels: labels3},
	}

	appsV1Client := openshiftappsfake.NewClientset(&osDeployment1, &osDeployment2).AppsV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	// found 2
	result, err := os.GetDeploymentFamilyVersions(ctx, "family-1", testNamespace)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "deployment must be not nil")
	r.Equal(2, len(result), "expected 2 deployment")

	// found 1
	result2, err2 := os.GetDeploymentFamilyVersions(ctx, "family-2", testNamespace)
	r.Nil(err2, "unexpected error returned")
	r.NotNil(result2, "deployment must be not nil")
	r.Equal(1, len(result2), "expected 1 deployment")
	r.Equal("name-3", result2[0].Name)
	r.Equal("version-3", result2[0].Version)
	r.Equal("bluegreen_version-3", result2[0].BlueGreenVersion)

	// not found
	result3, err3 := os.GetDeploymentFamilyVersions(ctx, "family1", testNamespace)
	r.Nil(err3, "unexpected error returned")
	r.NotNil(result3, "deployment must be not nil")
	r.Equal(0, len(result3), "expected 0 deployment")
}

func Test_GetDeploymentFamilyVersions_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset()
	expectedError := fmt.Errorf("test error during list Deployment")
	appsV1Client.Fake.PrependReactor("list", "deploymentconfigs",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client.AppsV1(), kubeClient)
	_, err := os.GetDeploymentFamilyVersions(ctx, "some-family", testNamespace)
	r.Equal(expectedError, err, "unexpected error returned")
}
