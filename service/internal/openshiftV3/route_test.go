package openshiftV3

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	kube "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	openshift_v1 "github.com/openshift/api/route/v1"
	openshiftappsfake "github.com/openshift/client-go/apps/clientset/versioned/fake"
	openshiftprojectfake "github.com/openshift/client-go/project/clientset/versioned/fake"
	openshiftroutefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

var (
	testIngressClassName = "test-ingress-class-name"
)

//todo - cache does not support 2 sources, and will be cleaned up if resource is not found in the first source
// it will be fixed when caches will be updated on watch events
//func Test_GetRoute_usingCache_success(t *testing.T) {
//	ctx := context.Background()
//
//	kubeClientSet := fake.NewSimpleClientset()
//	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet})
//
//	routeClientset := openshiftroutefake.NewSimpleClientset()
//	routeClientset.PrependReactor("*", "*", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
//		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
//	})
//	routeV1Client := routeClientset.RouteV1()
//	projectV1Client := openshiftprojectfake.NewSimpleClientset().ProjectV1()
//
//	appsV1Client := openshiftappsfake.NewSimpleClientset().AppsV1()
//
//	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)
//
//	routeTest := openshift_v1.Route{ObjectMeta: metav1.ObjectMeta{Name: testRoute, Namespace: testNamespace}}
//
//	kubeClient.Cache = cache.NewTestResourcesCache()
//	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromOsRoute(&routeTest))
//	assert.NoError(t, err)
//	assert.True(t, ok)
//
//	route, err := os.GetRoute(ctx, testRoute, testNamespace)
//	assert.Nil(t, err)
//	assert.NotNil(t, route)
//}

func Test_GetRoute_ExistsAsIngress(t *testing.T) {
	ctx := context.Background()

	ingress := getNetworkingIngress(testRoute)
	kubeClientSet := fake.NewClientset(ingress)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.UseNetworkingV1Ingress = true

	routeClientset := openshiftroutefake.NewClientset()
	routeClientset.PrependReactor("*", "*", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
	})
	routeV1Client := routeClientset.RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	route, err := os.GetRoute(ctx, testRoute, testNamespace)
	assert.Nil(t, err)
	assert.NotNil(t, route)
	expectedRoute := entity.RouteFromIngressNetworkingV1(ingress)
	assert.Equal(t, *expectedRoute, *route)
}

func Test_ListRoutes_ExistAsBothRoutesAndIngresses(t *testing.T) {
	ctx := context.Background()

	ingress1 := getNetworkingIngress("test-route-1")
	ingress2 := getNetworkingIngress("test-route-2")

	kubeClientSet := fake.NewClientset(ingress1)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.UseNetworkingV1Ingress = true

	expectedRoute1 := entity.RouteFromIngressNetworkingV1(ingress1)
	expectedRoute2 := entity.RouteFromIngressNetworkingV1(ingress2)
	osRoute2 := expectedRoute2.ToOsRoute()

	routeClientset := openshiftroutefake.NewClientset(osRoute2)
	routeV1Client := routeClientset.RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	routes, err := os.GetRouteList(ctx, testNamespace, filter.Meta{})
	assert.Nil(t, err)
	assert.NotNil(t, routes)
	assert.Equal(t, 2, len(routes))
	assert.Equal(t, *expectedRoute1, routes[0])
	assert.Equal(t, *entity.RouteFromOsRoute(osRoute2), routes[1])
}

func Test_UpdateOrCreateRoute_UpdateFromCache_success(t *testing.T) {
	ctx := context.Background()

	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()
	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	routeTest := openshift_v1.Route{ObjectMeta: metav1.ObjectMeta{Name: testRoute, Namespace: testNamespace}}
	routeUpdate := &entity.Route{Metadata: entity.Metadata{Name: testRoute, Namespace: testNamespace},
		Spec: entity.RouteSpec{Host: "local"}}

	kubeClient.Cache = cache.NewTestResourcesCache()
	ok, err := kubeClient.Cache.Ingresses.Set(ctx, *entity.RouteFromOsRoute(&routeTest))
	assert.NoError(t, err)
	assert.True(t, ok)
	updatedRoute, err := os.UpdateOrCreateRoute(ctx, routeUpdate, testNamespace)

	assert.Nil(t, err)
	assert.NotNil(t, updatedRoute)
	assert.Equal(t, routeUpdate.Spec.Host, updatedRoute.Spec.Host)
}

func Test_UpdateOrCreateRoute_Update_success(t *testing.T) {
	ctx := context.Background()

	kubeClientSet := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})

	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()

	routeTest := openshift_v1.Route{ObjectMeta: metav1.ObjectMeta{Name: testRoute, Namespace: testNamespace}}
	routeV1Client := openshiftroutefake.NewClientset(&routeTest).RouteV1()
	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	routeUpdate := &entity.Route{Metadata: entity.Metadata{Name: testRoute, Namespace: testNamespace},
		Spec: entity.RouteSpec{Host: "local"}}

	updatedRoute, err := os.UpdateOrCreateRoute(ctx, routeUpdate, testNamespace)

	assert.Nil(t, err)
	assert.NotNil(t, updatedRoute)
	assert.Equal(t, routeUpdate.Spec.Host, updatedRoute.Spec.Host)
}

func Test_DeleteRoute_success(t *testing.T) {
	ctx := context.Background()

	ingress := getNetworkingIngress(testRoute)
	kubeClientSet := fake.NewClientset(ingress)
	cert_client := &certClient.Clientset{}
	kubeClient, _ := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: kubeClientSet, CertmanagerInterface: cert_client})
	kubeClient.UseNetworkingV1Ingress = true

	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	err := os.DeleteRoute(ctx, testRoute, testNamespace)

	assert.Nil(t, err)
}

func getNetworkingIngress(name string) *networkingV1.Ingress {
	ingressJson := map[string]any{
		"metadata": map[string]string{
			"name":            name,
			"namespace":       testNamespace,
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
	return &ingress
}
