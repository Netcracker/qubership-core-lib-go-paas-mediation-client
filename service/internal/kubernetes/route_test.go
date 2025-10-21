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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
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
		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
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
