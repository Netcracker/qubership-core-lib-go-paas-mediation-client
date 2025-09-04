package kubernetes

import (
	"errors"
	"fmt"
	"sync"
	"testing"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	cmfake "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes/mock"
	. "github.com/smarty/assertions"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	gatewayclientfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"
)

const (
	testDeploymentName = "test-deployment"

	testNamespace1 = "test-namespace-1"
	testNamespace2 = "test-namespace-2"

	testReplicaSet     = "test-rs"
	testIngress        = "test-ingress"
	testSecret         = "test-secret"
	testService        = "test-service"
	testPod            = "test-pod"
	testServiceAccount = "test-service-account"
	testConfigMap      = "test-configmap"
	testCertificate    = "test-certificate"
	labelKey1          = "test-label-1"
	labelKey2          = "test-label-2"

	labelValue1 = "value-1"
	labelValue2 = "value-2"

	annotationKey1 = "test-annotation-1"
	annotationKey2 = "test-annotation-2"
	annotationKey3 = "test-annotation-3"

	annotationValue1 = "value-1"
	annotationValue2 = "value-2"
	annotationValue3 = "value-3"

	kubernetesVersion = "1.23.1"
)

var (
	testIngressClassName = "test-ingress-class-name"
)

func TestGetServiceListWithLabelsAndAnnotations(t *testing.T) {
	testGetResourceListWithLabelsAndAnnotations(t, "service")
}

func TestGetPodListWithLabelsAndAnnotations(t *testing.T) {
	testGetResourceListWithLabelsAndAnnotations(t, "pod")
}

func TestGetSecretListWithLabelsAndAnnotations(t *testing.T) {
	testGetResourceListWithLabelsAndAnnotations(t, "secret")
}

func TestGetConfigMapListWithLabelsAndAnnotations(t *testing.T) {
	testGetResourceListWithLabelsAndAnnotations(t, "configmap")
}

func TestGetRouteListWithLabelsAndAnnotations(t *testing.T) {
	testGetResourceListWithLabelsAndAnnotations(t, "route")
}

func TestGetCertificateListWithLabelsAndAnnotations(t *testing.T) {
	testGetResourceListWithLabelsAndAnnotations(t, "certificate")
}

func testGetResourceListWithLabelsAndAnnotations(t *testing.T, resType string) {
	r := require.New(t)
	testWatchExecutor := newFakeWatchExecutor()

	// ----------------- #1
	resourceLabelsMap1 := make(map[string]string)
	resourceLabelsMap1[labelKey1] = labelValue1

	resourceAnnotationMap1 := make(map[string]string)
	resourceAnnotationMap1[annotationKey1] = annotationValue1
	resourceAnnotationMap1[annotationKey2] = annotationValue2

	resourceName1 := "test-resource-1"
	resource1 := createTestResource(resType, resourceName1, testNamespace1, resourceLabelsMap1, resourceAnnotationMap1)

	// ----------------- #2
	resourceLabelsMap2 := make(map[string]string)
	resourceLabelsMap2[labelKey2] = labelValue2

	resourceAnnotationMap2 := make(map[string]string)
	resourceAnnotationMap2[annotationKey2] = annotationValue2
	resourceAnnotationMap2[annotationKey3] = annotationValue3

	resourceName2 := "test-resource-2"
	service2 := createTestResource(resType, resourceName2, testNamespace1, resourceLabelsMap2, resourceAnnotationMap2)

	var clientset = createTestClient(resType, resource1, service2)
	var badClientset = &backend.KubernetesApi{
		KubernetesInterface:  &mock.KubeClientset{},
		CertmanagerInterface: &mock.CmClientset{},
	}
	var badResources = BadResources{NewBadRoutes()}
	cache := cache.NewTestResourcesCache()
	kubeClient := &Kubernetes{client: clientset, WatchExecutor: testWatchExecutor, namespace: testNamespace1,
		Cache: cache, BadResources: &badResources}

	var annotationMap = make(map[string]string)
	annotationMap[annotationKey1] = annotationValue1
	annotationMap[annotationKey2] = annotationValue2

	ctx := context.Background()
	switch resType {
	case "service":
		foundResources, err := kubeClient.GetServiceList(ctx, testNamespace1, filter.Meta{Annotations: annotationMap})
		r.True(So(err, ShouldBeNil))
		r.Equal(1, len(foundResources), "expect 1 service to be found")
		r.Equal(foundResources[0].Name, resourceName1, "invalid service name")

		//Broke client and test cache
		kubeClient.client = badClientset
		ok, err := cache.Services.Set(ctx, *entity.NewService(resource1.(*v1.Service)))
		r.True(ok)
		r.NoError(err)
		foundResourceInCache, err := kubeClient.GetService(ctx, resourceName1, testNamespace1)
		r.Nil(err)
		r.Equal(foundResourceInCache.Name, resourceName1, "invalid service name")
	case "pod":
		foundResources, err := kubeClient.GetPodList(ctx, testNamespace1, filter.Meta{Annotations: annotationMap})
		r.True(So(err, ShouldBeNil))
		r.Equal(1, len(foundResources), "expect 1 pod to be found")
		r.Equal(foundResources[0].Name, resourceName1, "invalid pod name")
		//Broke client and get error
		kubeClient.client = badClientset
		_, err = kubeClient.GetPodList(ctx, testNamespace1, filter.Meta{Annotations: annotationMap})
		r.NotNil(err)
	case "secret":
		foundResources, err := kubeClient.GetSecretList(ctx, testNamespace1, filter.Meta{Annotations: annotationMap})
		r.Nil(err)
		r.Equal(1, len(foundResources), "expect 1 secret to be found")
		r.Equal(foundResources[0].Name, resourceName1, "invalid service name")
		//Broke client and test cache
		kubeClient.client = badClientset
		ok, err := cache.Secrets.Set(ctx, *entity.NewSecret(resource1.(*v1.Secret)))
		r.True(ok)
		r.NoError(err)
		foundResourceInCache, err := kubeClient.GetSecret(ctx, resourceName1, testNamespace1)
		r.Nil(err)
		r.Equal(foundResourceInCache.Name, resourceName1, "invalid service name")
	case "configmap":
		foundResources, err := kubeClient.GetConfigMapList(ctx, testNamespace1,
			filter.Meta{Annotations: annotationMap})
		r.Nil(err)
		r.Equal(1, len(foundResources), "expect 1 configmap to be found")
		r.Equal(foundResources[0].Name, resourceName1, "invalid service name")
		//Broke client and test cache
		kubeClient.client = badClientset
		ok, err := cache.ConfigMaps.Set(ctx, *entity.NewConfigMap(resource1.(*v1.ConfigMap)))
		r.True(ok)
		r.NoError(err)
		foundResourceInCache, err := kubeClient.GetConfigMap(ctx, resourceName1, testNamespace1)
		r.Nil(err)
		r.Equal(foundResourceInCache.Name, resourceName1, "invalid service name")
	case "route":
		foundResources, err := kubeClient.GetRouteList(ctx, testNamespace1, filter.Meta{Annotations: annotationMap})
		r.Nil(err)
		r.Equal(1, len(foundResources), "expect 1 route to be found")
		r.Equal(foundResources[0].Name, resourceName1, "invalid service name")
		//Broke client and test cache
		kubeClient.client = badClientset
		ok, err := cache.Ingresses.Set(ctx, *entity.RouteFromIngress(resource1.(*v1beta1.Ingress)))
		r.True(ok)
		r.NoError(err)
		foundResourceInCache, err := kubeClient.GetRoute(ctx, resourceName1, testNamespace1)
		r.Nil(err)
		r.Equal(foundResourceInCache.Name, resourceName1, "invalid service name")
	case "certificate":
		foundResources, err := kubeClient.GetCertificateList(ctx, testNamespace1,
			filter.Meta{Annotations: annotationMap})
		r.Nil(err)
		r.Equal(1, len(foundResources), "expect 1 certificate to be found")
		r.Equal(foundResources[0].Name, resourceName1, "invalid certificate name")
		//Broke client and test cache
		kubeClient.client = badClientset
		//todo certificats cache not supported yet
		//ok, err := cache.Certificates.Set(ctx, *entity.NewCertificate(resource1.(*cmv1.Certificate)))
		//r.True(ok)
		//r.NoError(err)
		//foundResourceInCache, err := kubeClient.GetCertificate(ctx, resourceName1, testNamespace1)
		//r.Nil(err)
		//r.Equal(foundResourceInCache.Name, resourceName1, "invalid certificate name")
		_, err = kubeClient.GetCertificate(ctx, resourceName1, testNamespace1)
		r.NotNil(err)
	default:
		r.False(false, "unsupported type "+resType)
	}
}

func createTestResource(resType string, name string, namespace string, labels map[string]string,
	annotations map[string]string) runtime.Object {
	switch resType {
	case "service":
		return createTestService(name, namespace, labels, annotations)
	case "pod":
		return createTestPod(name, namespace, labels, annotations)
	case "route":
		return createTestRoute(name, namespace, labels, annotations)
	case "configmap":
		return createTestConfigMap(name, namespace, labels, annotations)
	case "secret":
		return createTestSecret(name, namespace, labels, annotations)
	case "certificate":
		return createTestCertificate(name, namespace, labels, annotations)
	default:
		panic(errors.New("Unknown resource type " + resType))
	}
}

func createTestClient(resType string, objects ...runtime.Object) *backend.KubernetesApi {
	switch resType {
	case "service", "pod", "route", "configmap", "secret":
		return &backend.KubernetesApi{
			KubernetesInterface: fake.NewSimpleClientset(objects...),
		}
	case "certificate":
		return &backend.KubernetesApi{
			CertmanagerInterface: cmfake.NewSimpleClientset(objects...),
		}
	default:
		panic(errors.New("Unknown resource type " + resType))
	}
}

func createTestService(name string, namespace string, labels map[string]string,
	annotations map[string]string) *v1.Service {
	return &v1.Service{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels,
		Annotations: annotations}}
}

func createTestPod(name string, namespace string, labels map[string]string,
	annotations map[string]string) *v1.Pod {
	return &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels,
		Annotations: annotations}}
}

func createTestSecret(name string, namespace string, labels map[string]string,
	annotations map[string]string) *v1.Secret {
	return &v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels,
		Annotations: annotations}}
}

func createTestConfigMap(name string, namespace string, labels map[string]string,
	annotations map[string]string) *v1.ConfigMap {
	return &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels,
		Annotations: annotations}}
}

func createTestRoute(name string, namespace string, labels map[string]string,
	annotations map[string]string) *v1beta1.Ingress {
	backend := v1beta1.IngressBackend{ServiceName: "test-service-name-for-route-" + name}
	rule := v1beta1.IngressRule{}
	rule.HTTP = &v1beta1.HTTPIngressRuleValue{Paths: append([]v1beta1.HTTPIngressPath{},
		v1beta1.HTTPIngressPath{Backend: backend})}
	rule.HTTP.Paths = append([]v1beta1.HTTPIngressPath{}, v1beta1.HTTPIngressPath{Path: "/test-path-for-route-" + name})
	rule.Host = "test-host-for-route-" + name
	ingress := &v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels,
		Annotations: annotations},
		Spec: v1beta1.IngressSpec{Rules: append([]v1beta1.IngressRule{}, rule)}}
	return ingress
}

func createTestCertificate(name string, namespace string, labels map[string]string,
	annotations map[string]string) *cmv1.Certificate {
	return &cmv1.Certificate{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace, Labels: labels,
		Annotations: annotations}}
}

func Test_getKubernetesClientVersion_appsV1Client_success(t *testing.T) {
	r := require.New(t)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	clientset := fake.NewSimpleClientset(&namespace)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	res, err := kube.getKubernetesClientVersion(testNamespace1)
	r.Nil(err)
	r.Equal("appsV1Client", res)
}

func Test_getKubernetesVersion_success(t *testing.T) {
	r := require.New(t)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	fakeVersion := version.Info{GitVersion: kubernetesVersion}
	clientset := fake.NewSimpleClientset(&namespace)
	cert_client := &certClient.Clientset{}
	fakeDiscoveryClient := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscoveryClient.FakedServerVersion = &fakeVersion
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	res, err := kube.GetKubernetesVersion()
	r.Nil(err)
	r.Equal(kubernetesVersion, res)
}

func Test_BadRoutes(t *testing.T) {
	r := require.New(t)
	badRoutes := NewBadRoutes()
	badRoutes.Add("test-ns", "test-route")
	r.Equal(map[string][]string{"test-ns": {"test-route"}}, badRoutes.ToSliceMap())
	badRoutes.Delete("test-ns", "test-route")
	r.Equal(map[string][]string{}, badRoutes.ToSliceMap())
}

func Test_BadRoutesAsync(t *testing.T) {
	r := require.New(t)
	badRoutes := NewBadRoutes()
	count := 100
	var wg sync.WaitGroup
	wg.Add(count)
	for i := 0; i < count; i++ {
		go func(i int) {
			badRoutes.Add(testNamespace1, fmt.Sprintf("test-route-%d", i))
			wg.Done()
		}(i)
	}
	wg.Wait()
	result := badRoutes.ToSliceMap()
	r.Equal(count, len(result[testNamespace1]))
}

func Test_WithoutCache(t *testing.T) {
	assertions := require.New(t)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	fakeVersion := version.Info{GitVersion: kubernetesVersion}
	clientset := fake.NewSimpleClientset(&namespace)
	cert_client := &certClient.Clientset{}
	fakeDiscoveryClient := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscoveryClient.FakedServerVersion = &fakeVersion
	kubernetes, err := NewKubernetesClientBuilder().WithNamespace(testNamespace1).WithClient(&backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}).Build()
	assertions.NoError(err)
	assertions.NotNil(kubernetes)
	assertions.NotNil(kubernetes.Cache)
}

func Test_BuildEnablesGatewayHTTPRouteWatchHandlers(t *testing.T) {
	assertions := require.New(t)
	k8sClient := fake.NewClientset()
	// advertise gateway kinds in discovery
	fakeDisc := k8sClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Resources = []*metav1.APIResourceList{{
		GroupVersion: "gateway.networking.k8s.io/v1",
		APIResources: []metav1.APIResource{{Kind: "HTTPRoute"}},
	}}
	// gateway client (fake)
	gwClient := gatewayclientfake.NewSimpleClientset()
	kube, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{KubernetesInterface: k8sClient, CertmanagerInterface: &certClient.Clientset{}, GatewayInterface: gwClient}).
		Build()
	assertions.NoError(err)
	assertions.NotNil(kube.WatchHandlers.HTTPRouteV1)
}

func Test_BuildEnablesGatewayGRPCRouteWatchHandlers(t *testing.T) {
	assertions := require.New(t)
	k8sClient := fake.NewClientset()
	fakeDisc := k8sClient.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Resources = []*metav1.APIResourceList{{
		GroupVersion: "gateway.networking.k8s.io/v1",
		APIResources: []metav1.APIResource{{Kind: "GRPCRoute"}},
	}}
	gwClient := gatewayclientfake.NewSimpleClientset()
	kube, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{KubernetesInterface: k8sClient, CertmanagerInterface: &certClient.Clientset{}, GatewayInterface: gwClient}).
		Build()
	assertions.NoError(err)
	assertions.NotNil(kube.WatchHandlers.GRPCRouteV1)
}

func TestHasKindGatewayApi_HTTPRouteSupported(t *testing.T) {
	r := require.New(t)
	clientset := fake.NewClientset()
	fakeDisc := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "gateway.networking.k8s.io/v1",
			APIResources: []metav1.APIResource{{Kind: "HTTPRoute"}},
		},
	}
	r.True(hasKindGatewayApi("HTTPRoute", fakeDisc))
}

func TestHasKindGatewayApi_GRPCRouteSupported(t *testing.T) {
	r := require.New(t)
	clientset := fake.NewClientset()
	fakeDisc := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "gateway.networking.k8s.io/v1",
			APIResources: []metav1.APIResource{{Kind: "GRPCRoute"}},
		},
	}
	r.True(hasKindGatewayApi("GRPCRoute", fakeDisc))
}

func TestHasKindGatewayApi_KindNotSupported(t *testing.T) {
	r := require.New(t)
	clientset := fake.NewClientset()
	fakeDisc := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDisc.Resources = []*metav1.APIResourceList{
		{
			GroupVersion: "gateway.networking.k8s.io/v1",
			APIResources: []metav1.APIResource{{Kind: "SomeOtherKind"}},
		},
	}
	r.False(hasKindGatewayApi("GRPCRoute", fakeDisc))
	r.False(hasKindGatewayApi("HTTPRoute", fakeDisc))
}
