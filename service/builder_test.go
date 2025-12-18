package service

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"testing"

	fakecertmanager "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	"github.com/golang/mock/gomock"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/openshiftV3"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	openshiftappsfake "github.com/openshift/client-go/apps/clientset/versioned/fake"
	openshiftprojectfake "github.com/openshift/client-go/project/clientset/versioned/fake"
	openshiftroutefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apimachinery/pkg/watch"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

var testNamespace = "test-ns"

func prepareFakeClients() (*backend.KubernetesApi, *backend.OpenshiftApi) {
	fakeVersion := version.Info{GitVersion: "v1.16.3"}
	testNamespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace}}
	kubernetesClientSet := fake.NewClientset(&testNamespace)
	certManagerClientSet := fakecertmanager.NewSimpleClientset()
	fakeDiscoveryClient := kubernetesClientSet.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscoveryClient.FakedServerVersion = &fakeVersion
	fakeKubernetesClient := &backend.KubernetesApi{KubernetesInterface: kubernetesClientSet, CertmanagerInterface: certManagerClientSet}

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()
	appsV1Client := openshiftappsfake.NewClientset().AppsV1()
	fakeOpenshiftClient := &backend.OpenshiftApi{
		RouteV1Interface:   routeV1Client,
		ProjectV1Interface: projectV1Client,
		AppsV1Interface:    appsV1Client,
	}
	return fakeKubernetesClient, fakeOpenshiftClient
}

func TestPlatformClientBuilder_WithAllCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()

	testWatchExecutor := getTestWatchExecutor(t)

	client, err := NewPlatformClientBuilder().WithAllCaches().WithClients(prepareFakeClients()).WithWatchExecutor(testWatchExecutor).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testAllK8sCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithAllCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()

	testWatchExecutor := getTestWatchExecutor(t)

	client, err := NewPlatformClientBuilder().WithAllCaches().WithClients(prepareFakeClients()).WithWatchExecutor(testWatchExecutor).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testAllOpenshiftCache(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithAllCachesTwice_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()

	testWatchExecutor := getTestWatchExecutor(t)

	firstClient, err := NewPlatformClientBuilder().WithAllCaches().WithClients(prepareFakeClients()).WithWatchExecutor(testWatchExecutor).Build()
	r.Nil(err)
	r.NotNil(firstClient)
	realClientFirstClient := getK8sEntityFromClient(firstClient)
	r.NotNil(realClientFirstClient)
	testAllK8sCache(t, realClientFirstClient.Cache)
	secondClient, err := NewPlatformClientBuilder().WithAllCaches().WithClients(prepareFakeClients()).WithWatchExecutor(testWatchExecutor).Build()
	r.Nil(err)
	r.NotNil(secondClient)
	realClientSecondClient := getK8sEntityFromClient(secondClient)
	r.NotNil(realClientSecondClient)
	testAllK8sCache(t, realClientSecondClient.Cache)
	r.NotEqual(&realClientFirstClient, &realClientSecondClient)
	r.False(reflect.DeepEqual(realClientFirstClient, realClientSecondClient))
}

func TestPlatformClientBuilder_WithAllCachesTwice_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()

	testWatchExecutor := getTestWatchExecutor(t)

	firstClient, err := NewPlatformClientBuilder().WithAllCaches().WithClients(prepareFakeClients()).WithWatchExecutor(testWatchExecutor).Build()
	r.Nil(err)
	r.NotNil(firstClient)
	realClientFirstClient := getOs311EntityFromClient(firstClient)
	r.NotNil(realClientFirstClient)
	testAllOpenshiftCache(t, realClientFirstClient.Cache)
	secondClient, err := NewPlatformClientBuilder().WithAllCaches().WithClients(prepareFakeClients()).WithWatchExecutor(testWatchExecutor).Build()
	r.Nil(err)
	r.NotNil(secondClient)
	realClientSecondClient := getOs311EntityFromClient(secondClient)
	r.NotNil(realClientSecondClient)
	testAllOpenshiftCache(t, realClientSecondClient.Cache)
	r.NotEqual(&realClientFirstClient, &realClientSecondClient)
	r.False(reflect.DeepEqual(realClientFirstClient, realClientSecondClient))
}

func TestPlatformClientBuilder_WithNamespaceCache_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithNamespaceCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testNamespaceCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithNamespaceCache_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithNamespaceCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testNamespaceCache(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithRouteCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithRouteCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testIngressCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithRouteCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithRouteCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testIngressCache(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithSecretCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithSecretCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testSecretCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithSecretCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithSecretCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testSecretCache(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithConfigMapCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithConfigMapCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testConfigMapCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithConfigMapCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithConfigMapCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testConfigMapCache(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithServiceCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithServiceCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testServiceCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithServiceCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithServiceCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testServiceCache(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithServiceCachesTwice_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithServiceCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testServiceCache(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithServiceCachesTwice_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithServiceCache().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testServiceCache(t, realClient.Kubernetes.Cache)
}

func testAllOpenshiftCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	//r.NotNil(cache.Certificates) //todo Certificates cache not supported yet
	r.NotNil(cache.ConfigMaps)
	r.NotNil(cache.Ingresses)
	r.NotNil(cache.Secrets)
	r.NotNil(cache.Services)
}

func testAllK8sCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	//r.NotNil(cache.Certificates) //todo Certificates cache not supported yet
	r.NotNil(cache.ConfigMaps)
	r.NotNil(cache.Ingresses)
	r.NotNil(cache.Secrets)
	r.NotNil(cache.Services)
	r.NotNil(cache.Namespaces)
}

func testNamespaceCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	r.NotNil(cache.Namespaces)
	r.Nil(cache.Certificates)
	r.Nil(cache.ConfigMaps)
	r.Nil(cache.Ingresses)
	r.Nil(cache.Secrets)
	r.Nil(cache.Services)
}

func testIngressCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	r.NotNil(cache.Ingresses)
	r.Nil(cache.Certificates)
	r.Nil(cache.ConfigMaps)
	r.Nil(cache.Namespaces)
	r.Nil(cache.Secrets)
	r.Nil(cache.Services)
}

func testSecretCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	r.NotNil(cache.Secrets)
	r.Nil(cache.Certificates)
	r.Nil(cache.ConfigMaps)
	r.Nil(cache.Namespaces)
	r.Nil(cache.Ingresses)
	r.Nil(cache.Services)
}

func testConfigMapCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	r.NotNil(cache.ConfigMaps)
	r.Nil(cache.Certificates)
	r.Nil(cache.Secrets)
	r.Nil(cache.Namespaces)
	r.Nil(cache.Ingresses)
	r.Nil(cache.Services)
}

func testServiceCache(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	r.NotNil(cache.Services)
	r.Nil(cache.Certificates)
	r.Nil(cache.Secrets)
	r.Nil(cache.Namespaces)
	r.Nil(cache.Ingresses)
	r.Nil(cache.ConfigMaps)
}

func TestPlatformClientBuilder_WithBasicCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithBasicCaches().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testBasicCaches(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithBasicCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithBasicCaches().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testBasicCaches(t, realClient.Kubernetes.Cache)
}

func TestPlatformClientBuilder_WithGatewayApiRoutesCaches_k8s(t *testing.T) {
	r := require.New(t)
	prepareEnv("kubernetes")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithGatewayApiRoutesCaches().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getK8sEntityFromClient(client)
	r.NotNil(realClient)
	testGatewayApiRoutesCaches(t, realClient.Cache)
}

func TestPlatformClientBuilder_WithGatewayApiRoutesCaches_Openshift311(t *testing.T) {
	r := require.New(t)
	prepareEnv("openshift")
	defer cleanUpEnv()
	client, err := NewPlatformClientBuilder().WithGatewayApiRoutesCaches().WithClients(prepareFakeClients()).WithWatchExecutor(getTestWatchExecutor(t)).Build()
	r.Nil(err)
	r.NotNil(client)
	realClient := getOs311EntityFromClient(client)
	r.NotNil(realClient)
	r.NotNil(realClient.Kubernetes)
	testGatewayApiRoutesCaches(t, realClient.Kubernetes.Cache)
}

func testBasicCaches(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	// Basic caches should include: Certificate, ConfigMap, Namespace, Route, Secret, Service
	// Certificate cache is not supported yet, so it's nil
	r.Nil(cache.Certificates) // todo Certificates cache not supported yet
	r.NotNil(cache.ConfigMaps)
	r.NotNil(cache.Namespaces)
	r.NotNil(cache.Ingresses)
	r.NotNil(cache.Secrets)
	r.NotNil(cache.Services)
	// Gateway API route caches should not be enabled
	r.Nil(cache.HTTPRoute)
	r.Nil(cache.GRPCRoute)
}

func testGatewayApiRoutesCaches(t *testing.T, cache *cache.ResourcesCache) {
	r := require.New(t)
	r.NotNil(cache)
	// Gateway API route caches should include: HttpRoute, GrpcRoute
	r.NotNil(cache.HTTPRoute)
	r.NotNil(cache.GRPCRoute)
	// Basic caches should not be enabled
	r.Nil(cache.Certificates)
	r.Nil(cache.ConfigMaps)
	r.Nil(cache.Namespaces)
	r.Nil(cache.Ingresses)
	r.Nil(cache.Secrets)
	r.Nil(cache.Services)
}

func prepareEnv(platform string) {
	setPaasEnv(platform)
	configloader.Init([]*configloader.PropertySource{configloader.EnvPropertySource()}...)
}

func cleanUpEnv() {
	unsetPaasEnv()
}

func setPaasEnv(platform string) {
	os.Setenv("PAAS_PLATFORM", platform)
	os.Setenv("MICROSERVICE_NAMESPACE", testNamespace)
}

func unsetPaasEnv() {
	os.Unsetenv("PAAS_PLATFORM")
	os.Unsetenv("MICROSERVICE_NAMESPACE")
}

func getK8sEntityFromClient(client PlatformService) *kubernetes.Kubernetes {
	return client.(*kubernetes.Kubernetes)
}

func getOs311EntityFromClient(client PlatformService) *openshiftV3.OpenshiftV3Client {
	return client.(*openshiftV3.OpenshiftV3Client)
}

func getTestWatchExecutor(t *testing.T) pmWatch.Executor {
	testWatchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
	testWatchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *metav1.ListOptions) *rest.Request {
			return rest.NewRequestWithClient(&url.URL{}, "test", rest.ClientContentConfig{}, http.DefaultClient)
		}).AnyTimes()
	testWatchExecutor.EXPECT().Watch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
			return watch.NewFake(), nil
		}).AnyTimes()
	return testWatchExecutor
}
