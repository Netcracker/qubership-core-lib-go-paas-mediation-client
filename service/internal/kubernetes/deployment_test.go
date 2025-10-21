package kubernetes

import (
	"context"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func Test_GetDeployment_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-name",
			Namespace: testNamespace1},
		Spec: v1.DeploymentSpec{}}
	clientset := fake.NewClientset(&deployment)
	cert_client := &certClient.Clientset{}
	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	// found
	result, err := kube.GetDeployment(ctx, "test-name", testNamespace1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "deployment must be not nil")
	r.Equal("test-name", result.Metadata.Name)

	// not found
	result2, err2 := kube.GetDeployment(ctx, "test-name1", testNamespace1)
	r.Nil(err2, "unexpected error returned")
	r.Nil(result2, "deployment must be nil")
}

func Test_GetDeployment_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	clientset := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	expectedError := fmt.Errorf("test error during list Deployment")
	clientset.Fake.PrependReactor("get", "deployments",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})
	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	_, err = kube.GetDeployment(ctx, "test-name", testNamespace1)
	r.Equal(expectedError, err, "unexpected error returned")
}

// there are 3 deployments:
// deployment1 {name: "test-name1", labels: {"app-name": "app-test-name"}, annotations: {"annot.tation":"app-annotation"}}
// deployment2 {name: "test-name2", labels: {"app-name": "app-test-name"}, annotations: {"annot.tation":"app-annotation2"}}
// deployment2 {name: "test-name3", labels: {"app-name": "app-test-name3"}}
//
// filter1 {labels: {"app-name": "app-test-name"}, annotations: {"annot.tation":"app-annotation"}}
// expectation: deployment1
//
// filter2 {annotations: {"annot.tation":"app-annotation2"}}
// expectation: deployment2
func Test_GetDeploymentList_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	labels1 := make(map[string]string)
	labels1["app-name"] = "app-test-name"
	annotations1 := make(map[string]string)
	annotations1["annot.tation"] = "app-annotation"
	deployment1 := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-name1",
			Namespace:   testNamespace1,
			Labels:      labels1,
			Annotations: annotations1,
		},
		Spec: v1.DeploymentSpec{}}

	labels2 := make(map[string]string)
	labels2["app-name"] = "app-test-name"
	annotations2 := make(map[string]string)
	annotations2["annot.tation"] = "app-annotation2"
	deployment2 := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-name2",
			Namespace:   testNamespace1,
			Labels:      labels2,
			Annotations: annotations2,
		},
		Spec: v1.DeploymentSpec{}}

	labels3 := make(map[string]string)
	labels3["app-name"] = "app-test-name3"
	deployment3 := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name3",
			Namespace: testNamespace1,
			Labels:    labels3,
		},
		Spec: v1.DeploymentSpec{}}

	clientset := fake.NewClientset(&deployment1, &deployment2, &deployment3)
	cert_client := &certClient.Clientset{}
	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	filter1 := filter.Meta{
		Labels:      labels1,
		Annotations: annotations1,
	}

	deployments, err := kube.GetDeploymentList(ctx, testNamespace1, filter1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(deployments, "deployment must be not nil")
	r.Equal(1, len(deployments), "expected 1 deployment")
	r.Equal("test-name1", deployments[0].Metadata.Name, "expected first deployment")

	filter2 := filter.Meta{
		Annotations: annotations2,
	}

	deployments2, err := kube.GetDeploymentList(ctx, testNamespace1, filter2)
	r.Nil(err, "unexpected error returned")
	r.NotNil(deployments2, "deployment must be not nil")
	r.Equal(1, len(deployments2), "expected 1 deployment")
	r.Equal("test-name2", deployments2[0].Metadata.Name, "expected second deployment")
}

func Test_GetDeploymentList_Error(t *testing.T) {
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

	labels1 := make(map[string]string)
	labels1["app-name"] = "app-test-name"

	filter2 := filter.Meta{
		Labels: labels1,
	}

	_, err = kube.GetDeploymentList(ctx, testNamespace1, filter2)
	r.Equal(expectedError, err, "unexpected error returned")
}
