package openshiftV3

import (
	"context"
	"fmt"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	kube "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	osV1 "github.com/openshift/api/project/v1"
	openshiftappsfake "github.com/openshift/client-go/apps/clientset/versioned/fake"
	openshiftprojectfake "github.com/openshift/client-go/project/clientset/versioned/fake"
	openshiftroutefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	kube_test "k8s.io/client-go/testing"

	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func getClients() (routev1client.RouteV1Interface, *openshiftappsfake.Clientset, *kube.Kubernetes) {
	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	appsV1Client := openshiftappsfake.NewClientset()
	return routeV1Client, appsV1Client, kubeClient
}

func getTestProjects() (*osV1.Project, *osV1.Project) {
	project1 := osV1.Project{ObjectMeta: metav1.ObjectMeta{Name: "test-project1", Namespace: testNamespace}}
	project2 := osV1.Project{ObjectMeta: metav1.ObjectMeta{Name: "test-project2", Namespace: testNamespace + "2"}}
	return &project1, &project2
}

func Test_GetNamespaces_cache_nil_success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	routeV1Client, appsV1Client, kubeClient := getClients()
	projectV1Client := openshiftprojectfake.NewClientset(getTestProjects()).ProjectV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client.AppsV1(), kubeClient)
	namespaces, err := os.GetNamespaces(ctx, filter.Meta{})
	r.Nil(err)
	r.NotNil(namespaces, "Namespaces must not be nit")
	r.Equal(2, len(namespaces), "2 namespaces should be returned")
}

func Test_GetNamespaces_cache_nil_failure(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	routeV1Client, appsV1Client, kubeClient := getClients()

	projectV1Client := openshiftprojectfake.NewClientset()
	expectedError := fmt.Errorf("test error during list Projects")
	projectV1Client.Fake.PrependReactor("list", "projects",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client.ProjectV1(), appsV1Client.AppsV1(), kubeClient)
	namespaces, err := os.GetNamespaces(ctx, filter.Meta{})
	r.Empty(namespaces)
	r.NotNil(err)
}

func Test_GetNamespaces_cache_and_projects_notNil_success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	routeV1Client, appsV1Client, kubeClient := getClients()
	kubeClient.Cache = cache.NewTestResourcesCache()

	projectV1Client := openshiftprojectfake.NewClientset(getTestProjects()).ProjectV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client.AppsV1(), kubeClient)
	namespaces, err := os.GetNamespaces(ctx, filter.Meta{})
	r.Nil(err)
	r.NotNil(namespaces, "Namespaces must not be nit")
	r.Equal(2, len(namespaces), "2 namespaces should be returned")
}

func Test_GetNamespaces_cache_NotNil_projects_Nil_error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	routeV1Client, appsV1Client, kubeClient := getClients()

	kubeClient.Cache = cache.NewTestResourcesCache()

	projectV1Client := openshiftprojectfake.NewClientset()

	expectedError := fmt.Errorf("test error during list Projects")
	projectV1Client.Fake.PrependReactor("list", "projects",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client.ProjectV1(), appsV1Client.AppsV1(), kubeClient)
	namespaces, err := os.GetNamespaces(ctx, filter.Meta{})
	r.NotNil(err)
	r.Nil(namespaces)
}
