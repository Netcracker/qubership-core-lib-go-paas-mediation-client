package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
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

func Test_dualModeRouteError_BothSucceeded_ReturnsNil(t *testing.T) {
	err := dualModeRouteError(
		routeResourceResult{status: routeStatusCreated},
		routeResourceResult{status: routeStatusCreated},
	)
	require.NoError(t, err)
}

func Test_dualModeRouteError_ShowsHintOnCreatePartialFailure(t *testing.T) {
	err := dualModeRouteError(
		routeResourceResult{status: routeStatusCreated},
		routeResourceResult{status: routeStatusCreated, err: paasErrors.NewInternalError(fmt.Errorf("ingress create failed"))},
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "try using Update endpoint")
}

func Test_CreateRoute_DualMode_BothAlreadyExists_ReturnsAlreadyExistsError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	gwClient.PrependReactor("create", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewAlreadyExists(
			schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "httproutes"}, testIngress)
	})
	kubeClientSet := fake.NewClientset()
	kubeClientSet.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewAlreadyExists(schema.GroupResource{Resource: "ingresses"}, testIngress)
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
	assertions.True(paasErrors.IsAlreadyExists(err))
	assertions.False(paasErrors.IsInternalError(err))
	assertions.Contains(err.Error(), "httproute: error:")
	assertions.Contains(err.Error(), "already exists")
	assertions.Contains(err.Error(), "ingress: error:")
	assertions.NotContains(err.Error(), "try using Update endpoint")
}

func Test_CreateRoute_DualMode_AlreadyExistsAndInternalError_ReturnsAlreadyExistsError(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	gwClient := gatewayclientfake.NewSimpleClientset()
	gwClient.PrependReactor("create", "httproutes", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewAlreadyExists(
			schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "httproutes"}, testIngress)
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
	assertions.True(paasErrors.IsAlreadyExists(err))
	assertions.Contains(err.Error(), "httproute: error:")
	assertions.Contains(err.Error(), "ingress: error:")
	assertions.Contains(err.Error(), "ingress create failed")
	assertions.NotContains(err.Error(), "try using Update endpoint")
}

func Test_CreateRoute_DualMode_PartialCreate_IngressAlreadyExists_ReturnsAlreadyExistsWithHint(t *testing.T) {
	assertions := require.New(t)
	ctx := context.Background()
	kubeClient, k8sClient, _ := newDualModeKubeClient(t)

	k8sClient.PrependReactor("create", "ingresses", func(action kube_test.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewAlreadyExists(schema.GroupResource{Resource: "ingresses"}, testIngress)
	})

	_, err := kubeClient.CreateRoute(ctx, dualModeTestRoute(), testNamespace1)
	assertions.Error(err)
	assertions.True(paasErrors.IsAlreadyExists(err))
	assertions.Contains(err.Error(), "httproute: created")
	assertions.Contains(err.Error(), "ingress: error")
	assertions.Contains(err.Error(), "try using Update endpoint")
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

func Test_dualModeRouteError_NoErrorDuplication(t *testing.T) {
	t.Parallel()
	assertions := require.New(t)

	tests := []struct {
		name           string
		httpRouteRes   routeResourceResult
		ingressRes     routeResourceResult
		wantErrContain string
		wantNoContain  string
	}{
		{
			name: "HTTPRoute success, Ingress failed - no duplication",
			httpRouteRes: routeResourceResult{
				route:  &entity.Route{Metadata: entity.Metadata{Name: "test"}},
				status: routeStatusCreated,
				err:    nil,
			},
			ingressRes: routeResourceResult{
				route:  nil,
				status: "",
				err:    fmt.Errorf("ingress creation failed"),
			},
			wantErrContain: "httproute: created, ingress: error - try using Update endpoint: ingress creation failed",
			wantNoContain:  "ingress creation failed - try using Update endpoint: ingress creation failed",
		},
		{
			name: "HTTPRoute failed, Ingress success - no duplication",
			httpRouteRes: routeResourceResult{
				route:  nil,
				status: "",
				err:    fmt.Errorf("httproute creation failed"),
			},
			ingressRes: routeResourceResult{
				route:  &entity.Route{Metadata: entity.Metadata{Name: "test"}},
				status: routeStatusCreated,
				err:    nil,
			},
			wantErrContain: "httproute: error, ingress: created - try using Update endpoint: httproute creation failed",
			wantNoContain:  "httproute creation failed - try using Update endpoint: httproute creation failed",
		},
		{
			name: "Both failed - both errors wrapped",
			httpRouteRes: routeResourceResult{
				route:  nil,
				status: "",
				err:    fmt.Errorf("httproute error"),
			},
			ingressRes: routeResourceResult{
				route:  nil,
				status: "",
				err:    fmt.Errorf("ingress error"),
			},
			wantErrContain: "httproute: error: httproute error, ingress: error: ingress error",
			wantNoContain:  "",
		},
		{
			name: "Both success - no error",
			httpRouteRes: routeResourceResult{
				route:  &entity.Route{Metadata: entity.Metadata{Name: "test"}},
				status: routeStatusCreated,
				err:    nil,
			},
			ingressRes: routeResourceResult{
				route:  &entity.Route{Metadata: entity.Metadata{Name: "test"}},
				status: routeStatusCreated,
				err:    nil,
			},
			wantErrContain: "",
			wantNoContain:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := dualModeRouteError(tt.httpRouteRes, tt.ingressRes)

			if tt.wantErrContain == "" {
				assertions.NoError(err)
				return
			}

			assertions.Error(err)
			assertions.Contains(err.Error(), tt.wantErrContain)

			if tt.wantNoContain != "" {
				assertions.NotContains(err.Error(), tt.wantNoContain, "error message should not contain duplicated error text")
			}
		})
	}
}
