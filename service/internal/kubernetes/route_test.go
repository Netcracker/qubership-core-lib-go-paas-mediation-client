package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/stretchr/testify/require"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/api/networking/v1"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
	gatewayclientfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"
)

func getVariables() (*entity.Route, *cache.ResourcesCache) {
	resourcesCache := cache.NewTestResourcesCache()
	routeInCache := entity.Route{Metadata: entity.Metadata{Name: testIngress, Namespace: testNamespace1}}
	_, err := resourcesCache.Ingresses.Set(context.Background(), routeInCache)
	if err != nil {
		panic(err.Error())
	}
	routeToCreate := &entity.Route{Metadata: entity.Metadata{Name: testIngress, Namespace: testNamespace1},
		Spec: entity.RouteSpec{Host: "local"}}
	return routeToCreate, resourcesCache
}

func getNetworkingIngress() v1.Ingress {
	ingressJson := map[string]any{
		"metadata": map[string]string{
			"name":            testIngress,
			"namespace":       testNamespace1,
			"resourceVersion": "1"},
		"spec": map[string]any{
			"rules": []map[string]any{{
				"host": "test.host",
				"http": map[string]any{
					"paths": []map[string]any{{
						"pathType": "TYPE",
						"path":     "test-path",
						"backend": map[string]any{
							"service": map[string]any{
								"number": 80,
							},
						}}}}}}},
		"ingressClassName": &testIngressClassName,
	}
	marshaledIngress, err := json.Marshal(ingressJson)
	if err != nil {
		panic(err)
	}
	var ingress v1.Ingress
	err = json.Unmarshal(marshaledIngress, &ingress)
	if err != nil {
		panic(err)
	}
	return ingress
}

func GetIngress(ingressJson map[string]any) v1beta1.Ingress {
	marshaledIngress, err := json.Marshal(ingressJson)
	if err != nil {
		panic(err)
	}
	var ingress v1beta1.Ingress
	err = json.Unmarshal(marshaledIngress, &ingress)
	if err != nil {
		panic(err)
	}
	return ingress
}

func Test_CreateRoute_success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	routeToCreate, resourcesCache := getVariables()
	kubeClient.Cache = resourcesCache
	newRoute, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(newRoute)
}

func Test_CreateRoute_UseNetworkingV1Ingress_success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.UseNetworkingV1Ingress = true
	routeToCreate, resourcesCache := getVariables()
	kubeClient.Cache = resourcesCache
	newRoute, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(newRoute)
}

func Test_DeleteRoute_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	route := v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace1}}
	kubeClientSet := fake.NewClientset(&route)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Nil(err)
}

func Test_DeleteRoute_UseNetworkingV1Ingress_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	route := v1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace1}}
	kubeClientSet := fake.NewClientset(&route)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.UseNetworkingV1Ingress = true
	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Nil(err)
}

func Test_UpdateOrCreateRoute_Create_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	routeToCreate := &entity.Route{Metadata: entity.Metadata{Name: testIngress, Namespace: testNamespace1},
		Spec: entity.RouteSpec{Host: "local"}}
	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	route, err := kubeClient.UpdateOrCreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
}

func Test_CreateRoute_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	ingress := GetIngress(map[string]any{
		"metadata": map[string]string{
			"name":            testIngress,
			"namespace":       testNamespace1,
			"resourceVersion": "1"},
		"spec": map[string]any{
			"rules": []map[string]any{{
				"host": "test.host",
				"http": map[string]any{
					"paths": []map[string]any{{
						"path": "test-path",
						"backend": map[string]any{
							"serviceName": "name",
							"servicePort": "8080",
						}}}}}}}},
	)

	routeIngress := entity.RouteFromIngress(&ingress)
	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	route, err := kubeClient.CreateRoute(ctx, routeIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
}

func Test_UpdateOrCreateRoute_Create_UseNetworkingV1Ingress_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	ingress := getNetworkingIngress()
	routeIngress := entity.RouteFromIngressNetworkingV1(&ingress)

	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.UseNetworkingV1Ingress = true
	route, err := kubeClient.UpdateOrCreateRoute(ctx, routeIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
	assertions.Equal(ingress.ObjectMeta.Name, route.Metadata.Name)
	assertions.Equal(ingress.ObjectMeta.Namespace, route.Metadata.Namespace)
	assertions.Equal(ingress.Spec.IngressClassName, route.Spec.IngressClassName)
}

func Test_UpdateOrCreateRoute_Update_UseNetworkingV1Ingress_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	ingress := getNetworkingIngress()
	routeIngress := entity.RouteFromIngressNetworkingV1(&ingress)

	routeIngress.Spec.Port.TargetPort = int32(30)

	kubeClientSet := fake.NewClientset(&ingress)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	kubeClient.Cache = cache.NewTestResourcesCache()

	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromIngressNetworkingV1(&ingress))
	assertions.NoError(err)
	assertions.True(ok)

	kubeClient.UseNetworkingV1Ingress = true
	route, err := kubeClient.UpdateOrCreateRoute(ctx, routeIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
	assertions.Equal(ingress.ObjectMeta.Name, route.Metadata.Name)
	assertions.Equal(ingress.ObjectMeta.Namespace, route.Metadata.Namespace)
	assertions.Equal(ingress.Spec.IngressClassName, route.Spec.IngressClassName)
}

func Test_UpdateOrCreateRoute_Update_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	ingress := GetIngress(map[string]any{
		"metadata": map[string]string{
			"name":            testIngress,
			"namespace":       testNamespace1,
			"resourceVersion": "1"},
		"spec": map[string]any{
			"rules": []map[string]any{{
				"host": "test.host",
				"http": map[string]any{
					"paths": []map[string]any{{
						"path": "test-path",
						"backend": map[string]any{
							"serviceName": "name",
							"servicePort": "8080",
						}}}}}}}},
	)
	routeIngress := entity.RouteFromIngress(&ingress)

	routeIngress.Spec.Port.TargetPort = int32(30)

	kubeClientSet := fake.NewClientset(&ingress)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	kubeClient.Cache = cache.NewTestResourcesCache()
	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromIngress(&ingress))
	assertions.NoError(err)
	assertions.True(ok)

	route, err := kubeClient.UpdateOrCreateRoute(ctx, routeIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
	assertions.Equal(ingress.ObjectMeta.Name, route.Metadata.Name)
	assertions.Equal(ingress.ObjectMeta.Namespace, route.Metadata.Namespace)
	assertions.Equal(ingress.Spec.IngressClassName, route.Spec.IngressClassName)
}

func Test_GetRoute_LegacyIngress_ReadsIngress(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	ingress := GetIngress(map[string]any{
		"metadata": map[string]string{
			"name":            testIngress,
			"namespace":       testNamespace1,
			"resourceVersion": "1"},
		"spec": map[string]any{
			"rules": []map[string]any{{
				"host": "test.host",
				"http": map[string]any{
					"paths": []map[string]any{{
						"path": "test-path",
						"backend": map[string]any{
							"serviceName": "name",
							"servicePort": "8080",
						}}}}}}}},
	)
	kubeClientSet := fake.NewClientset(&ingress)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.GatewaySystem.Type = LegacyIngress

	kubeClient.Cache = cache.NewTestResourcesCache()
	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromIngress(&ingress))
	assertions.NoError(err)
	assertions.True(ok)

	route, err := kubeClient.GetRoute(ctx, testIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
	assertions.Equal(testIngress, route.Name)
}

func Test_GetRoute_GatewayApiDefault_ReadsHTTPRoute(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	httpRoute := dualModeTestRoute().ToHTTPRoute("gateway-system", "default-external-gateway")
	kubeClientSet := fake.NewClientset()
	gwClient := gatewayclientfake.NewSimpleClientset(httpRoute)
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  kubeClientSet,
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	assertions.NoError(err)

	route, err := kubeClient.GetRoute(ctx, testIngress, testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(route)
	assertions.Equal(httpRoute.Name, route.Name)
}

func Test_GetRoute_DualMode_IgnoresIngress(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	httpRoute := dualModeTestRoute().ToHTTPRoute("gateway-system", "default-external-gateway")
	ingress := GetIngress(map[string]any{
		"metadata": map[string]string{
			"name":      testIngress,
			"namespace": testNamespace1,
		},
		"spec": map[string]any{
			"rules": []map[string]any{{
				"host": "legacy-only.example.com",
				"http": map[string]any{
					"paths": []map[string]any{{
						"path": "legacy-path",
						"backend": map[string]any{
							"serviceName": "legacy-service",
							"servicePort": "9090",
						},
					}},
				},
			}},
		},
	})
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(&ingress),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gatewayclientfake.NewSimpleClientset(httpRoute),
		}).
		WithGatewaySystemType(LegacyIngress + "," + GatewayApiDefault).
		Build()
	assertions.NoError(err)

	route, err := kubeClient.GetRoute(ctx, testIngress, testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(route)
	expected := entity.RouteFromHTTPRoute(httpRoute)
	assertions.Equal(expected.Spec.Host, route.Spec.Host)
	assertions.NotEqual("legacy-only.example.com", route.Spec.Host)
}
func Test_GetRouteFromCache_UseNetworkingV1Ingress_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	ingress := getNetworkingIngress()

	kubeClientSet := fake.NewClientset()
	kubeClientSet.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "v1.23.0"}
	kubeClientSet.PrependReactor("get", "ingresses", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("test api server error"))
	})

	cert_client := &certClient.Clientset{}
	kubeClient, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	assertions.Nil(err)
	kubeClient.UseNetworkingV1Ingress = true

	kubeClient.Cache = cache.NewTestResourcesCache()
	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromIngressNetworkingV1(&ingress))
	assertions.NoError(err)
	assertions.True(ok)

	route, err := kubeClient.GetRoute(ctx, testIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
	assertions.Equal(testIngress, route.Name)
}

func Test_CreateRouteBG2_Enabled(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	ingress := GetIngress(map[string]any{
		"metadata": map[string]string{
			"name":            testIngress,
			"namespace":       testNamespace1,
			"resourceVersion": "1"},
		"spec": map[string]any{
			"rules": []map[string]any{{
				"host": "test.host",
				"http": map[string]any{
					"paths": []map[string]any{{
						"path": "test-path",
						"backend": map[string]any{
							"serviceName": "name",
							"servicePort": "8080",
						}}}}}}}},
	)

	routeIngress := entity.RouteFromIngress(&ingress)
	kubeClientSet := fake.NewClientset()
	certManager := &certClient.Clientset{}
	kubeClient, err := NewKubernetesClientBuilder().
		WithClient(&backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certManager}).
		WithNamespace(testNamespace1).
		WithBG2Enabled(func() bool {
			return true
		}).Build()

	assertions.Nil(err)
	route, err := kubeClient.CreateRoute(ctx, routeIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
}

func Test_CreateRoute_GatewayAPIOnly_Error(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	routeToCreate := &entity.Route{
		Metadata: entity.Metadata{Name: testIngress, Namespace: testNamespace1},
		Spec:     entity.RouteSpec{Host: "local"},
	}

	kubeClientSet := fake.NewClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})
	kubeClient.GatewaySystem.Type = "invalid-type"

	_, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "does not allow any Route creation")
}

func Test_DeleteRoute_InvalidType_NoOp(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	kubeClientSet := fake.NewClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})
	kubeClient.GatewaySystem.Type = "invalid-type"

	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Nil(err)
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_Error(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	routeToCreate := &entity.Route{
		Metadata: entity.Metadata{Name: testIngress, Namespace: testNamespace1},
		Spec:     entity.RouteSpec{Host: "local"},
	}

	kubeClientSet := fake.NewClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})
	kubeClient.GatewaySystem.Type = "invalid-type"

	_, err := kubeClient.UpdateOrCreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.NotNil(err)
	assertions.Contains(err.Error(), "does not allow any Route update")
}

func newGatewayAPIOnlyKubeClient(t *testing.T) (*Kubernetes, *gatewayclientfake.Clientset) {
	t.Helper()
	gwClient := gatewayclientfake.NewSimpleClientset()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)
	return kubeClient, gwClient
}

func Test_CreateRoute_GatewayAPIOnly_Success(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	route, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(route)

	list, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(list.Items, 1)
}

func Test_CreateRoute_GatewayAPIOnly_InvalidTimeout_DoesNotCreateHTTPRoute(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	route := dualModeTestRoute()
	route.Spec.StreamIdleTimeout = "not-a-duration"
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsBadRequest(err))

	list, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Empty(list.Items)
}

func Test_CreateRoute_GatewayAPIOnly_UnsupportedAnnotation_DoesNotCreateHTTPRoute(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	route := dualModeTestRoute()
	route.Metadata.Annotations = map[string]string{AnnotationConfigSnippet: "something"}
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInvalid(err))

	list, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Empty(list.Items)
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_InvalidTimeout_LeavesHTTPRouteUnchanged(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	original, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)

	route.Spec.Host = "changed.example.com"
	route.Spec.StreamIdleTimeout = "not-a-duration"
	_, err = kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsBadRequest(err))

	got, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)
	assertions.Equal(original.Spec.Hostnames, got.Spec.Hostnames)
	assertions.Equal(original.ResourceVersion, got.ResourceVersion)
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_UnsupportedAnnotation_LeavesHTTPRouteUnchanged(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	original, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)

	route.Spec.Host = "changed.example.com"
	route.Metadata.Annotations = map[string]string{AnnotationConfigSnippet: "something"}
	_, err = kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInvalid(err))

	got, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)
	assertions.Equal(original.Spec.Hostnames, got.Spec.Hostnames)
	assertions.Equal(original.ResourceVersion, got.ResourceVersion)
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_UpdatesHTTPRoute(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	route.Spec.Port.TargetPort = 7070
	updated, err := kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(updated)

	httpRoute, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)
	assertions.NotEmpty(httpRoute.Spec.Rules[0].BackendRefs)
}

func Test_DeleteRoute_GatewayAPIOnly_DeletesHTTPRoute(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.NoError(err)

	_, err = gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))
}

func dualModeTestRoute() *entity.Route {
	return &entity.Route{
		Metadata: entity.Metadata{
			Name:      testIngress,
			Namespace: testNamespace1,
		},
		Spec: entity.RouteSpec{
			Host:    "test.example.com",
			Path:    "/test",
			Service: entity.Target{Name: "test-service"},
			Port:    entity.RoutePort{TargetPort: 8080},
		},
	}
}

func newDualModeKubeClient(t *testing.T) (*Kubernetes, *fake.Clientset, *gatewayclientfake.Clientset) {
	t.Helper()
	return newRouteKubeClient(t, LegacyIngress+","+GatewayApiDefault)
}

// newRouteKubeClient builds a client for gatewaySystemType over fake Kubernetes and gateway API backends, with
// networking/v1 ingresses. Both backends are returned so that a test can inject an error into either one and read
// back what was created.
func newRouteKubeClient(t *testing.T, gatewaySystemType string) (*Kubernetes, *fake.Clientset, *gatewayclientfake.Clientset) {
	t.Helper()
	kubeClientSet := fake.NewClientset()
	gwClient := gatewayclientfake.NewSimpleClientset()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  kubeClientSet,
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(gatewaySystemType).
		Build()
	require.NoError(t, err)
	kubeClient.UseNetworkingV1Ingress = true
	return kubeClient, kubeClientSet, gwClient
}

// failingReactor builds a reactor that returns err instead of letting the fake client's object tracker handle the
// action.
func failingReactor(err error) kube_test.ReactionFunc {
	return func(kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, err
	}
}

func httpRoutesResource() schema.GroupResource {
	return schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "httproutes"}
}

func ingressesResource() schema.GroupResource {
	return schema.GroupResource{Resource: "ingresses"}
}

func Test_CreateRoute_DualMode_HTTPRouteCreated_IngressFailed_ReturnsPartialCreateError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	k8sClient.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress create failed"))
	})

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: created")
	assertions.Contains(err.Error(), "ingress: error")
	assertions.Contains(err.Error(), "try using Update endpoint")

	httpRouteList, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(httpRouteList.Items, 1)

	ingressList, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Empty(ingressList.Items)
}

func Test_CreateRoute_DualMode_BothFailed_ReturnsFullStatusError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	gwClient.PrependReactor("create", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("httproute create failed"))
	})
	kubeClientSet := fake.NewClientset()
	kubeClientSet.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress create failed"))
	})
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  kubeClientSet,
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(LegacyIngress + "," + GatewayApiDefault).
		Build()
	require.NoError(t, err)
	kubeClient.UseNetworkingV1Ingress = true

	_, err = kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: error:")
	assertions.Contains(err.Error(), "httproute create failed")
	assertions.Contains(err.Error(), "ingress: error:")
	assertions.Contains(err.Error(), "ingress create failed")
	assertions.NotContains(err.Error(), "try using Update endpoint")
}

// Test_CreateRoute_ExistingRoute_ReturnsAlreadyExistsInEveryGatewayMode pins the reason a caller matches with
// paasErrors.IsAlreadyExists, whichever resources the gateway system type asks for. A dual-mode create used to
// report every failure as an internal error of its own, so a caller could no longer tell a route that is already
// there from an API server that is broken.
func Test_CreateRoute_ExistingRoute_ReturnsAlreadyExistsInEveryGatewayMode(t *testing.T) {
	gatewayModes := []struct {
		name              string
		gatewaySystemType string
	}{
		{"legacy ingress only", LegacyIngress},
		{"gateway api only", GatewayApiDefault},
		{"dual mode", LegacyIngress + "," + GatewayApiDefault},
	}

	for _, mode := range gatewayModes {
		t.Run(mode.name, func(t *testing.T) {
			ctx := context.Background()
			kubeClient, k8sClient, gwClient := newRouteKubeClient(t, mode.gatewaySystemType)
			gwClient.PrependReactor("create", "httproutes",
				failingReactor(paasErrors.NewAlreadyExists(httpRoutesResource(), testIngress)))
			k8sClient.PrependReactor("create", "ingresses",
				failingReactor(paasErrors.NewAlreadyExists(ingressesResource(), testIngress)))

			_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)

			require.Equal(t, metav1.StatusReasonAlreadyExists, paasErrors.ReasonForError(err),
				"reason of the CreateRoute error %v", err)
		})
	}
}

// Test_CreateRoute_DualMode_IngressAlreadyExists_ReturnsAlreadyExistsWithUpdateHint covers a create that got as far
// as the HTTPRoute: the error the API server returned for the ingress reaches the caller in the chain, and the hint
// names the endpoint that finishes the job.
func Test_CreateRoute_DualMode_IngressAlreadyExists_ReturnsAlreadyExistsWithUpdateHint(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, _ := newDualModeKubeClient(t)
	ingressExists := paasErrors.NewAlreadyExists(ingressesResource(), testIngress)
	k8sClient.PrependReactor("create", "ingresses", failingReactor(ingressExists))

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)

	assertions.ErrorIs(err, ingressExists)
	assertions.Equal(metav1.StatusReasonAlreadyExists, paasErrors.ReasonForError(err),
		"reason of the CreateRoute error %v", err)
	assertions.Contains(err.Error(), "try using Update endpoint")
}

// Test_CreateRoute_DualMode_IngressCreateFailed_ReportsTheIngressErrorOnce guards the summary that the message
// carries beside the wrapped error. A summary built from the full per-resource status would print the ingress error
// a second time, which is what a caller used to read.
func Test_CreateRoute_DualMode_IngressCreateFailed_ReportsTheIngressErrorOnce(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, _ := newDualModeKubeClient(t)
	k8sClient.PrependReactor("create", "ingresses",
		failingReactor(paasErrors.NewInternalError(fmt.Errorf("ingress create failed"))))

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)

	assertions.Error(err)
	assertions.Equal(1, strings.Count(err.Error(), "ingress create failed"),
		"occurrences of the ingress error in %q", err)
}

// Test_CreateRoute_DualMode_BothCreatesFailed_WrapsBothErrors covers a create that produced neither resource: both
// API errors stay in the chain, and the hint stays out because there is nothing for an update to finish.
func Test_CreateRoute_DualMode_BothCreatesFailed_WrapsBothErrors(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)
	httpRouteForbidden := paasErrors.NewForbidden(httpRoutesResource(), testIngress,
		fmt.Errorf("httproutes are read-only"))
	ingressExists := paasErrors.NewAlreadyExists(ingressesResource(), testIngress)
	gwClient.PrependReactor("create", "httproutes", failingReactor(httpRouteForbidden))
	k8sClient.PrependReactor("create", "ingresses", failingReactor(ingressExists))

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)

	assertions.ErrorIs(err, httpRouteForbidden)
	assertions.ErrorIs(err, ingressExists)
	assertions.NotContains(err.Error(), "try using Update endpoint")
}

func Test_CreateRoute_DualMode_HTTPRouteFailed_IngressCreated_ReturnsPartialCreateError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	gwClient.PrependReactor("create", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("httproute create failed"))
	})
	kubeClientSet := fake.NewClientset()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  kubeClientSet,
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(LegacyIngress + "," + GatewayApiDefault).
		Build()
	require.NoError(t, err)
	kubeClient.UseNetworkingV1Ingress = true

	_, err = kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: error")
	assertions.Contains(err.Error(), "ingress: created")
	assertions.Contains(err.Error(), "try using Update endpoint")

	ingressList, err := kubeClientSet.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
}

func Test_CreateRoute_DualMode_CreatesHTTPRouteAndIngress(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	route, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(route)

	httpRouteList, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(httpRouteList.Items, 1)
	assertions.Equal(testIngress, httpRouteList.Items[0].Name)

	ingressList, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
	assertions.Equal("true", ingressList.Items[0].Annotations[IgnoreApiConverterAnnotation])
}

func Test_UpdateOrCreateRoute_DualMode_AfterPartialCreate_CreatesMissingIngress(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	var ingressCreateAttempts int
	k8sClient.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		ingressCreateAttempts++
		if ingressCreateAttempts == 1 {
			return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress create failed"))
		}
		return false, nil, nil
	})
	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "httproute: created")
	assertions.Contains(err.Error(), "ingress: error")

	httpRouteList, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(httpRouteList.Items, 1)

	updated, err := kubeClient.UpdateOrCreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(updated)

	ingressList, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
}

func Test_UpdateOrCreateRoute_DualMode_HTTPRouteExists_CreatesMissingIngress(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	route := dualModeTestRoute()
	httpRoute := route.ToHTTPRoute(kubeClient.GatewaySystem.Namespace, kubeClient.GatewaySystem.Name)
	_, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Create(ctx, httpRoute, metav1.CreateOptions{})
	assertions.NoError(err)

	updated, err := kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(updated)

	ingressList, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
	assertions.Equal("true", ingressList.Items[0].Annotations[IgnoreApiConverterAnnotation])
}

func Test_UpdateOrCreateRoute_DualMode_HTTPRouteUpdated_IngressUpdateError_ReturnsFullStatusError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	k8sClient.PrependReactor("update", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress update failed"))
	})

	_, err = kubeClient.UpdateOrCreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: updated")
	assertions.Contains(err.Error(), "ingress: error")
	assertions.Contains(err.Error(), "ingress update failed")
	assertions.Contains(err.Error(), "try using Update endpoint")

	httpRouteList, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(httpRouteList.Items, 1)

	ingressList, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
}

func Test_UpdateOrCreateRoute_DualMode_UpdatesBoth(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	route.Spec.Port.TargetPort = 9090
	updated, err := kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(updated)

	httpRoute, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)
	assertions.NotEmpty(httpRoute.Spec.Rules[0].BackendRefs)

	ingress, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)
	assertions.Equal(int32(9090), ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number)
}

func Test_DeleteRoute_DualMode_HTTPRouteDeleted_IngressNotFound_NoError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	err = k8sClient.NetworkingV1().Ingresses(testNamespace1).Delete(ctx, testIngress, metav1.DeleteOptions{})
	assertions.NoError(err)

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.NoError(err)

	_, err = gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_DualMode_HTTPRouteDeleted_IngressDeleteError_ReturnsFullStatusError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	k8sClient.PrependReactor("delete", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress delete failed"))
	})

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "ingress: error:")
	assertions.Contains(err.Error(), "ingress delete failed")

	_, err = gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_DualMode_HTTPRouteDeleteError_IngressDeleted_ReturnsFullStatusError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	gwClient.PrependReactor("delete", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("httproute delete failed"))
	})

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: error:")
	assertions.Contains(err.Error(), "httproute delete failed")

	_, err = k8sClient.NetworkingV1().Ingresses(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_DualMode_BothDeleteFailed_ReturnsFullStatusError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	gwClient.PrependReactor("delete", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("httproute delete failed"))
	})
	k8sClient.PrependReactor("delete", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress delete failed"))
	})

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: error:")
	assertions.Contains(err.Error(), "httproute delete failed")
	assertions.Contains(err.Error(), "ingress: error:")
	assertions.Contains(err.Error(), "ingress delete failed")
}

// Test_DeleteRoute_DualMode_IngressDeleteRefused_ReturnsForbidden pins the reason a caller matches with
// paasErrors.IsForbidden. A dual-mode delete used to report every failure as an internal error of its own.
func Test_DeleteRoute_DualMode_IngressDeleteRefused_ReturnsForbidden(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, _ := newDualModeKubeClient(t)
	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	ingressForbidden := paasErrors.NewForbidden(ingressesResource(), testIngress,
		fmt.Errorf("ingresses are read-only"))
	k8sClient.PrependReactor("delete", "ingresses", failingReactor(ingressForbidden))

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)

	assertions.ErrorIs(err, ingressForbidden)
	assertions.Equal(metav1.StatusReasonForbidden, paasErrors.ReasonForError(err),
		"reason of the DeleteRoute error %v", err)
}

func Test_DeleteRoute_DualMode_HTTPRouteDeleteRefused_ReturnsForbidden(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, _, gwClient := newDualModeKubeClient(t)
	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	httpRouteForbidden := paasErrors.NewForbidden(httpRoutesResource(), testIngress,
		fmt.Errorf("httproutes are read-only"))
	gwClient.PrependReactor("delete", "httproutes", failingReactor(httpRouteForbidden))

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)

	assertions.ErrorIs(err, httpRouteForbidden)
	assertions.Equal(metav1.StatusReasonForbidden, paasErrors.ReasonForError(err),
		"reason of the DeleteRoute error %v", err)
}

// Test_DeleteRoute_DualMode_BothDeletesRefused_WrapsBothErrors covers the branch that reports two failures: each
// API error stays in the chain, so a caller can match either one.
func Test_DeleteRoute_DualMode_BothDeletesRefused_WrapsBothErrors(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)
	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	httpRouteForbidden := paasErrors.NewForbidden(httpRoutesResource(), testIngress,
		fmt.Errorf("httproutes are read-only"))
	ingressConflict := paasErrors.NewConflict(ingressesResource(), testIngress,
		fmt.Errorf("the ingress was modified"))
	gwClient.PrependReactor("delete", "httproutes", failingReactor(httpRouteForbidden))
	k8sClient.PrependReactor("delete", "ingresses", failingReactor(ingressConflict))

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)

	assertions.ErrorIs(err, httpRouteForbidden)
	assertions.ErrorIs(err, ingressConflict)
}

func Test_DeleteRoute_DualMode_BothNotFound_ReturnsNotFound(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, _, _ := newDualModeKubeClient(t)

	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.True(paasErrors.IsNotFound(err))
	assertions.Regexp(`httproute: .* not found,`, err.Error())
	assertions.Regexp(`, ingress: .* not found`, err.Error())
}

func Test_DeleteRoute_DualMode_DeletesBoth(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.NoError(err)

	_, err = gwClient.GatewayV1().HTTPRoutes(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))

	_, err = k8sClient.NetworkingV1().Ingresses(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_CreateRoute_LegacyIngressOnly_NoIgnoreAnnotation(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	routeToCreate := &entity.Route{
		Metadata: entity.Metadata{
			Name:      testIngress,
			Namespace: testNamespace1,
		},
		Spec: entity.RouteSpec{
			Host:    "test.example.com",
			Path:    "/test",
			Service: entity.Target{Name: "test-service"},
			Port:    entity.RoutePort{TargetPort: 8080},
		},
	}

	kubeClientSet := fake.NewClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})
	kubeClient.UseNetworkingV1Ingress = true
	kubeClient.GatewaySystem.Type = LegacyIngress

	route, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)

	ingressList, err := kubeClientSet.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.Nil(err)
	assertions.Equal(1, len(ingressList.Items))
	assertions.Empty(ingressList.Items[0].Annotations[IgnoreApiConverterAnnotation])
}

func Test_ValidateAnnotationsForGatewayAPI_AllowedAnnotations(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	routeToCreate := &entity.Route{
		Metadata: entity.Metadata{
			Name:      testIngress,
			Namespace: testNamespace1,
			Annotations: map[string]string{
				entity.AnnotationAffinity:          "cookie",
				entity.AnnotationSessionCookieName: "my-cookie",
				entity.AnnotationProxyReadTimeout:  "1800",
			},
		},
		Spec: entity.RouteSpec{
			Host:    "test.example.com",
			Service: entity.Target{Name: "test-service"},
			Port:    entity.RoutePort{TargetPort: 8080},
		},
	}

	kubeClientSet := fake.NewClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})
	kubeClient.GatewaySystem.Type = LegacyIngress

	route, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
}

func Test_ValidateAnnotationsForGatewayAPI_LegacyIngressAllowsCritical(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	routeToCreate := &entity.Route{
		Metadata: entity.Metadata{
			Name:      testIngress,
			Namespace: testNamespace1,
			Annotations: map[string]string{
				AnnotationBackendProtocol: "HTTPS",
			},
		},
		Spec: entity.RouteSpec{
			Host:    "test.example.com",
			Service: entity.Target{Name: "test-service"},
			Port:    entity.RoutePort{TargetPort: 8080},
		},
	}

	kubeClientSet := fake.NewClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: certClientSet})
	kubeClient.UseNetworkingV1Ingress = true
	kubeClient.GatewaySystem.Type = LegacyIngress

	route, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.Nil(err) // No error because legacy-ingress doesn't validate
	assertions.NotNil(route)
}

func Test_validateAnnotationsForGatewayAPI_ReferencesAllConstants(t *testing.T) {
	assertions := require.New(t)
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  fake.NewClientset(),
		CertmanagerInterface: &certClient.Clientset{},
	})

	cases := map[string]string{
		AnnotationBackendProtocol:   BackendTlsOrTrafficWarning,
		AnnotationSecureBackends:    BackendTLSWarning,
		AnnotationAuthType:          SecurityPolicyWarning,
		AnnotationSSLPassthrough:    TlsRouteWarning,
		AnnotationConfigSnippet:     ConfigSnippetWarning,
		AnnotationUpstreamVhost:     EnvoyExtensionWarning,
		AnnotationProxyRedirectFrom: EnvoyExtensionWarning,
		AnnotationProxyRedirectTo:   EnvoyExtensionWarning,
	}
	for annotation, warning := range cases {
		err := kubeClient.validateAnnotationsForGatewayAPI(map[string]string{annotation: "test-value"})
		assertions.Error(err, "annotation %s", annotation)
		assertions.True(paasErrors.IsInvalid(err))
		var statusErr *paasErrors.StatusError
		assertions.ErrorAs(err, &statusErr)
		assertions.Contains(statusErr.Status().Message, warning)
	}
	assertions.Equal("gateway-api-converter.netcracker.com/ignore", IgnoreApiConverterAnnotation)
}

func newTinyRouteCache(t *testing.T, caches ...cache.CacheName) *cache.ResourcesCache {
	t.Helper()
	resourcesCache, err := cache.NewResourcesCache(2, 100, 1, 0, caches...)
	require.NoError(t, err)
	return resourcesCache
}

func Test_CreateRoute_GatewayAPIOnly_ReturnsHTTPRouteWithoutIngress(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	gwClient := gatewayclientfake.NewSimpleClientset()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  kubeClientSet,
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)

	route, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(route)
	assertions.Equal(testIngress, route.Name)

	ingressList, err := kubeClientSet.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Empty(ingressList.Items)
}

func Test_CreateRoute_GatewayAPIOnly_PlacesHTTPRouteInCache(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, _ := newGatewayAPIOnlyKubeClient(t)
	kubeClient.Cache = cache.NewTestResourcesCache(cache.HttpRouteCache)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)

	cached := kubeClient.Cache.HTTPRoute.Get(ctx, testNamespace1, testIngress)
	assertions.NotNil(cached)
	assertions.Equal(testIngress, cached.Name)
}

func Test_CreateRoute_GatewayAPIOnly_HTTPRouteCacheSetError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, gwClient := newGatewayAPIOnlyKubeClient(t)
	kubeClient.Cache = newTinyRouteCache(t, cache.HttpRouteCache)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "failed to place HTTPRoute into cache")

	list, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(list.Items, 1)
}

func Test_CreateRoute_GatewayAPIOnly_HTTPRouteCreateError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	gwClient.PrependReactor("create", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("httproute create failed")
	})
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)

	_, err = kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "httproute: error:")
	assertions.Contains(err.Error(), "httproute create failed")
}

func Test_CreateRoute_NetworkingV1_IngressCreateError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClientSet.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress create failed"))
	})
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.UseNetworkingV1Ingress = true
	kubeClient.GatewaySystem.Type = LegacyIngress

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "ingress create failed")
}

func Test_CreateRoute_V1beta1_IngressCreateError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClientSet.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("extensions ingress create failed"))
	})
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.GatewaySystem.Type = LegacyIngress

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "extensions ingress create failed")
}

func Test_CreateRoute_NetworkingV1_IngressCacheSetError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.UseNetworkingV1Ingress = true
	kubeClient.GatewaySystem.Type = LegacyIngress
	kubeClient.Cache = newTinyRouteCache(t, cache.RouteCache)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "failed to place ingress into cache")

	ingressList, err := kubeClientSet.NetworkingV1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
}

func Test_CreateRoute_V1beta1_IngressCacheSetError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.GatewaySystem.Type = LegacyIngress
	kubeClient.Cache = newTinyRouteCache(t, cache.RouteCache)

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "failed to place ingress into cache")

	ingressList, err := kubeClientSet.ExtensionsV1beta1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
}

func newDualModeV1beta1KubeClient(t *testing.T) (*Kubernetes, *fake.Clientset, *gatewayclientfake.Clientset) {
	t.Helper()
	kubeClient, k8sClient, gwClient := newDualModeKubeClient(t)
	kubeClient.UseNetworkingV1Ingress = false
	return kubeClient, k8sClient, gwClient
}

func Test_CreateRoute_DualMode_V1beta1_SetsIgnoreAnnotationOnNilAnnotations(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, gwClient := newDualModeV1beta1KubeClient(t)

	route, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.NoError(err)
	assertions.NotNil(route)

	httpRouteList, err := gwClient.GatewayV1().HTTPRoutes(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(httpRouteList.Items, 1)

	ingressList, err := k8sClient.ExtensionsV1beta1().Ingresses(testNamespace1).List(ctx, metav1.ListOptions{})
	assertions.NoError(err)
	assertions.Len(ingressList.Items, 1)
	assertions.Equal("true", ingressList.Items[0].Annotations[IgnoreApiConverterAnnotation])
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_HTTPRouteUpdateError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)

	route := dualModeTestRoute()
	_, err = kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	gwClient.PrependReactor("update", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("httproute update failed")
	})

	_, err = kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "httproute update failed")
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_HTTPRouteGetError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	gwClient.PrependReactor("get", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("get httproute failed"))
	})
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)

	_, err = kubeClient.UpdateOrCreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "get httproute failed")
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_HTTPRouteCacheSetError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, _ := newGatewayAPIOnlyKubeClient(t)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	kubeClient.Cache = newTinyRouteCache(t, cache.HttpRouteCache)
	route.Spec.Port.TargetPort = 7070

	_, err = kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "failed to place HTTPRoute into cache")
}

func Test_UpdateOrCreateRoute_GatewayAPIOnly_PlacesHTTPRouteInCache(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, _ := newGatewayAPIOnlyKubeClient(t)
	kubeClient.Cache = cache.NewTestResourcesCache(cache.HttpRouteCache)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	route.Spec.Port.TargetPort = 7070
	_, err = kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	cached := kubeClient.Cache.HTTPRoute.Get(ctx, testNamespace1, testIngress)
	assertions.NotNil(cached)
}

func Test_DeleteRoute_GatewayAPIOnly_HTTPRouteNotFound_ReturnsNotFound(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, _ := newGatewayAPIOnlyKubeClient(t)

	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_GatewayAPIOnly_HTTPRouteNotFound_DeletesBackendTrafficPolicy(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	dyn := newDynamicFake()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
			DynamicInterface:     dyn,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	assertions.NoError(err)

	route := dualModeTestRoute()
	route.Spec.StreamIdleTimeout = "60s"
	policy, err := kubeClient.backendTrafficPolicyFromRoute(route, testNamespace1)
	assertions.NoError(err)
	_, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).
		Create(ctx, policy, metav1.CreateOptions{})
	assertions.NoError(err)

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.True(paasErrors.IsNotFound(err))
	_, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).
		Get(ctx, testIngress, metav1.GetOptions{})
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_GatewayAPIOnly_HTTPRouteDeleteError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  fake.NewClientset(),
			CertmanagerInterface: &certClient.Clientset{},
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)

	_, err = kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	require.NoError(t, err)

	gwClient.PrependReactor("delete", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("httproute delete failed"))
	})

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "httproute delete failed")
}

func Test_DeleteRoute_LegacyIngress_NetworkingV1_IngressNotFound_ReturnsNotFound(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.UseNetworkingV1Ingress = true
	kubeClient.GatewaySystem.Type = LegacyIngress

	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_LegacyIngress_NetworkingV1_IngressDeleteError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.UseNetworkingV1Ingress = true
	kubeClient.GatewaySystem.Type = LegacyIngress

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	require.NoError(t, err)

	kubeClientSet.PrependReactor("delete", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("ingress delete failed"))
	})

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "ingress delete failed")
}

func Test_DeleteRoute_LegacyIngress_V1beta1_IngressNotFound_ReturnsNotFound(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.GatewaySystem.Type = LegacyIngress

	err := kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.True(paasErrors.IsNotFound(err))
}

func Test_DeleteRoute_LegacyIngress_V1beta1_IngressDeleteError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClientSet := fake.NewClientset()
	kubeClient, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{
		KubernetesInterface:  kubeClientSet,
		CertmanagerInterface: &certClient.Clientset{},
	})
	kubeClient.GatewaySystem.Type = LegacyIngress

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	require.NoError(t, err)

	kubeClientSet.PrependReactor("delete", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewInternalError(fmt.Errorf("extensions ingress delete failed"))
	})

	err = kubeClient.DeleteRoute(ctx, testIngress, testNamespace1)
	assertions.Error(err)
	assertions.Contains(err.Error(), "extensions ingress delete failed")
}

func Test_UpdateOrCreateRoute_DualMode_SetsIgnoreAnnotationOnIngressUpdate(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, _ := newDualModeKubeClient(t)

	route := dualModeTestRoute()
	_, err := kubeClient.CreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	route.Spec.Port.TargetPort = 6060
	_, err = kubeClient.UpdateOrCreateRoute(ctx, route, testNamespace1)
	assertions.NoError(err)

	ingress, err := k8sClient.NetworkingV1().Ingresses(testNamespace1).Get(ctx, testIngress, metav1.GetOptions{})
	assertions.NoError(err)
	assertions.Equal("true", ingress.Annotations[IgnoreApiConverterAnnotation])
}
