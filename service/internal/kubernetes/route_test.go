package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/stretchr/testify/require"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/networking/v1"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

func getNetworkingIngress() networkingV1.Ingress {
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
	var ingress networkingV1.Ingress
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

func Test_GetRoute_Success(t *testing.T) {
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

	kubeClient.Cache = cache.NewTestResourcesCache()
	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromIngress(&ingress))
	assertions.NoError(err)
	assertions.True(ok)

	route, err := kubeClient.GetRoute(ctx, testIngress, testNamespace1)
	assertions.Nil(err)
	assertions.NotNil(route)
	assertions.Equal(testIngress, route.Name)
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

func Test_ShouldUseGatewayAPI(t *testing.T) {
	tests := []struct {
		name              string
		gatewaySystemType string
		expected          bool
	}{
		{"Empty string", "", false},
		{"Legacy ingress only", "legacy-ingress", false},
		{"Gateway API default only", "gateway-api-default", true},
		{"Both modes", "legacy-ingress,gateway-api-default", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := GatewaySystem{Type: tt.gatewaySystemType}
			result := gs.ShouldUseGatewayAPI()
			require.Equal(t, tt.expected, result)
		})
	}
}

func Test_ShouldCreateLegacyIngress(t *testing.T) {
	tests := []struct {
		name              string
		gatewaySystemType string
		expected          bool
	}{
		{"Empty string", "", true},
		{"Legacy ingress only", "legacy-ingress", true},
		{"Gateway API default only", "gateway-api-default", false},
		{"Both modes", "legacy-ingress,gateway-api-default", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := GatewaySystem{Type: tt.gatewaySystemType}
			result := gs.ShouldCreateLegacyIngress()
			require.Equal(t, tt.expected, result)
		})
	}
}

func Test_ShouldIgnoreIngressForConverter(t *testing.T) {
	tests := []struct {
		name              string
		gatewaySystemType string
		expected          bool
	}{
		{"Empty string", "", false},
		{"Legacy ingress only", "legacy-ingress", false},
		{"Gateway API default only", "gateway-api-default", false},
		{"Both modes", "legacy-ingress,gateway-api-default", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := GatewaySystem{Type: tt.gatewaySystemType}
			result := gs.ShouldIgnoreIngressForConverter()
			require.Equal(t, tt.expected, result)
		})
	}
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
	kubeClientSet := fake.NewClientset()
	gwClient := gatewayclientfake.NewSimpleClientset()
	certClientSet := &certClient.Clientset{}
	kubeClient, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(&backend.KubernetesApi{
			KubernetesInterface:  kubeClientSet,
			CertmanagerInterface: certClientSet,
			GatewayInterface:     gwClient,
		}).
		WithGatewaySystemType(LegacyIngress + "," + GatewayApiDefault).
		Build()
	require.NoError(t, err)
	kubeClient.UseNetworkingV1Ingress = true
	return kubeClient, kubeClientSet, gwClient
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

func Test_ValidateAnnotationsForGatewayAPI_BackendProtocol(t *testing.T) {
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
	kubeClient.GatewaySystem.Type = GatewayApiDefault

	_, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.NotNil(err)
	assertions.True(paasErrors.IsInvalid(err))
	var statusErr *paasErrors.StatusError
	assertions.ErrorAs(err, &statusErr)
	assertions.Equal(http.StatusUnprocessableEntity, int(statusErr.Status().Code))
	assertions.Contains(statusErr.Status().Message, "backend-protocol")
}

func Test_ValidateAnnotationsForGatewayAPI_SSLPassthrough(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()

	routeToCreate := &entity.Route{
		Metadata: entity.Metadata{
			Name:      testIngress,
			Namespace: testNamespace1,
			Annotations: map[string]string{
				AnnotationSSLPassthrough: "true",
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
	kubeClient.GatewaySystem.Type = GatewayApiDefault

	_, err := kubeClient.CreateRoute(ctx, routeToCreate, testNamespace1)
	assertions.NotNil(err)
	assertions.True(paasErrors.IsInvalid(err))
	var statusErr *paasErrors.StatusError
	assertions.ErrorAs(err, &statusErr)
	assertions.Equal(http.StatusUnprocessableEntity, int(statusErr.Status().Code))
	assertions.Contains(statusErr.Status().Message, "ssl-passthrough")
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
