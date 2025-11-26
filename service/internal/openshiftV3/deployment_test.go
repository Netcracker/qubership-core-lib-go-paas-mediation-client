package openshiftV3

import (
	"context"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	fake_cert "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
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

func Test_GetDeployment_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment", Namespace: testNamespace},
		Spec:       v1.DeploymentSpec{},
	}
	kubeClientSet := fake.NewClientset(&deployment)
	certClientSet := fake_cert.NewSimpleClientset()
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	osDeployment := openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deploymentConfig", Namespace: testNamespace},
	}
	appsV1Client := openshiftappsfake.NewClientset(&osDeployment).AppsV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	result, err := os.GetDeployment(ctx, "test-deployment", testNamespace)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "deployment must be not nil")
	r.Equal("test-deployment", result.Metadata.Name, "expected deployment with name test-deployment")

	result2, err2 := os.GetDeployment(ctx, "test-deploymentConfig", testNamespace)
	r.Nil(err2, "unexpected error returned")
	r.NotNil(result2, "deployment must be not nil")
	r.Equal("test-deploymentConfig", result2.Metadata.Name, "expected deployment with name test-deploymentConfig")

	result3, err3 := os.GetDeployment(ctx, "test-name3", testNamespace)
	r.Nil(err3, "unexpected error returned")
	r.Nil(result3, "deployment must be nil")
}

func Test_GetDeployment_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset()
	expectedError := fmt.Errorf("test error during list Deployment")
	appsV1Client.Fake.PrependReactor("get", "deploymentconfigs",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client.AppsV1(), kubeClient)
	_, err := os.GetDeployment(ctx, "test-name2", testNamespace)
	r.Equal(expectedError, err, "unexpected error returned")
}

// there are 4 deployments:
// kubeDepl-1 {name: "test-name1", labels: {"app-name":"name1", "label1":"value1"}}
// kubeDepl-2 {name: "test-name2", labels: {"app-name":"name2"}}
// osDepl-1 {name: "test-name3", labels: {"app-name":"name3", "label1":"value1"}}
// osDepl-2 {name: "test-name4", annotations: {"annot.ta":"ann1"}}
//
// filter1 {labels: {"label1":"value1"}}
// expectation: kubeDepl-1 and osDepl-1
//
// filter1 {labels: {"annot.ta":"ann1"}}
// expectation: osDepl-2
func Test_GetDeploymentList_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	kubeDeployment1Labels := make(map[string]string)
	kubeDeployment1Labels["app-name"] = "name1"
	kubeDeployment1Labels["label1"] = "value1"
	kubeDeployment1 := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name1",
			Namespace: testNamespace,
			Labels:    kubeDeployment1Labels,
		},
		Spec: v1.DeploymentSpec{},
	}

	kubeDeployment2Labels := make(map[string]string)
	kubeDeployment2Labels["app-name"] = "name2"
	kubeDeployment2 := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name2",
			Namespace: testNamespace,
			Labels:    kubeDeployment2Labels,
		},
		Spec: v1.DeploymentSpec{},
	}

	kubeClientSet := fake.NewClientset(&kubeDeployment1, &kubeDeployment2)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	osDeployment1Labels := make(map[string]string)
	osDeployment1Labels["app-name"] = "name3"
	osDeployment1Labels["label1"] = "value1"

	osDeployment1 := openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name3",
			Namespace: testNamespace,
			Labels:    osDeployment1Labels,
		},
	}

	osDeployment2Labels := make(map[string]string)
	osDeployment2Annotations := make(map[string]string)
	osDeployment2Annotations["annot.ta"] = "ann1"

	osDeployment2 := openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-name4",
			Namespace:   testNamespace,
			Labels:      osDeployment2Labels,
			Annotations: osDeployment2Annotations,
		},
	}

	appsV1Client := openshiftappsfake.NewClientset(&osDeployment1, &osDeployment2).AppsV1()
	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	filter1Labels := make(map[string]string)
	filter1Labels["label1"] = "value1"
	filter1 := filter.Meta{
		Labels: filter1Labels,
	}

	deployments1, err1 := os.GetDeploymentList(ctx, testNamespace, filter1)
	r.Nil(err1, "unexpected error returned")
	r.NotNil(deployments1, "deployment must be not nil")
	r.Equal(2, len(deployments1), "expected 2 deployments")

	filter2Annotations := make(map[string]string)
	filter2Annotations["annot.ta"] = "ann1"
	filter2 := filter.Meta{
		Annotations: filter2Annotations,
	}

	deployments2, err2 := os.GetDeploymentList(ctx, testNamespace, filter2)
	r.Nil(err2, "unexpected error returned")
	r.NotNil(deployments2, "deployment must be not nil")
	r.Equal(1, len(deployments2), "expected 1 deployment")
	r.Equal("test-name4", deployments2[0].Metadata.Name, "expected deployment with name test-name4")
}

func Test_GetDeploymentList_Error(t *testing.T) {
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
	_, err := os.GetDeploymentList(ctx, testNamespace, filter.Meta{})
	r.Equal(expectedError, err, "unexpected error returned")
}
