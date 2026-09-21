package kubernetes

import (
	"context"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
	gatewayclientfake "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/fake"
)

func newTestBackendAPI(dyn dynamic.Interface) *backend.KubernetesApi {
	return &backend.KubernetesApi{
		KubernetesInterface:  k8sfake.NewClientset(),
		CertmanagerInterface: &certClient.Clientset{},
		GatewayInterface:     gatewayclientfake.NewSimpleClientset(),
		DynamicInterface:     dyn,
	}
}

func newDynamicFake() *fake.FakeDynamicClient {
	return fake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		backendTrafficPolicyGVR: "BackendTrafficPolicyList",
	})
}

func newKubeWithDynamic(t *testing.T, dyn *fake.FakeDynamicClient, idleTimeout string) *Kubernetes {
	t.Helper()
	kube, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(newTestBackendAPI(dyn)).
		WithHTTPRouteRequestIdleTimeout(idleTimeout).
		WithGatewaySystemType(GatewayApiDefault).
		Build()
	require.NoError(t, err)
	return kube
}

func btpRoute(name string, annotations map[string]string) *entity.Route {
	return &entity.Route{
		Metadata: entity.Metadata{
			Name:        name,
			Namespace:   testNamespace1,
			Annotations: annotations,
			Labels:      map[string]string{"app": "demo"},
		},
		Spec: entity.RouteSpec{
			Host:    "example.com",
			Path:    "/",
			Service: entity.Target{Name: "svc"},
		},
	}
}

func applyPolicyFromRoute(kube *Kubernetes, ctx context.Context, route *entity.Route, namespace string) error {
	policy, err := kube.backendTrafficPolicyFromRoute(route, namespace)
	if err != nil {
		return err
	}
	_, err = kube.applyBackendTrafficPolicy(ctx, policy, route.Name, namespace)
	return err
}

func TestApplyBackendTrafficPolicy_SkipsWithoutDynamicClient(t *testing.T) {
	kube, err := NewTestKubernetesClient(testNamespace1, newTestBackendAPI(nil))
	require.NoError(t, err)

	err = applyPolicyFromRoute(kube, context.Background(), btpRoute("r1", map[string]string{
		entity.AnnotationProxyReadTimeout: "60",
	}), testNamespace1)
	assert.NoError(t, err)
}

func TestApplyBackendTrafficPolicy_CreateUpdateDelete(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()
	route := btpRoute("route-btp", map[string]string{
		entity.AnnotationProxyReadTimeout: "1800",
	})

	require.NoError(t, applyPolicyFromRoute(kube, ctx, route, testNamespace1))

	created, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-btp", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, entity.ManagedByPaasMediation, created.GetLabels()[entity.ManagedByLabel])
	timeout, found, err := unstructured.NestedString(created.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1800s", timeout)

	route.Metadata.Annotations[entity.AnnotationProxyReadTimeout] = "900"
	require.NoError(t, applyPolicyFromRoute(kube, ctx, route, testNamespace1))
	updated, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-btp", metav1.GetOptions{})
	require.NoError(t, err)
	timeout, _, err = unstructured.NestedString(updated.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.Equal(t, "900s", timeout)

	_, err = kube.deleteBackendTrafficPolicy(ctx, "route-btp", testNamespace1)
	require.NoError(t, err)
	_, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-btp", metav1.GetOptions{})
	assert.True(t, err != nil)
}

func TestApplyBackendTrafficPolicy_UsesDefaultIdleTimeout(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "30m")
	ctx := context.Background()

	require.NoError(t, applyPolicyFromRoute(kube, ctx, btpRoute("route-default", nil), testNamespace1))
	created, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-default", metav1.GetOptions{})
	require.NoError(t, err)
	timeout, found, err := unstructured.NestedString(created.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "30m", timeout)
}

func TestApplyBackendTrafficPolicy_DeletesWhenNoLongerNeeded(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()
	route := btpRoute("route-clear", map[string]string{
		entity.AnnotationProxyReadTimeout: "60",
	})
	require.NoError(t, applyPolicyFromRoute(kube, ctx, route, testNamespace1))

	route.Metadata.Annotations = nil
	require.NoError(t, applyPolicyFromRoute(kube, ctx, route, testNamespace1))
	_, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-clear", metav1.GetOptions{})
	assert.Error(t, err)
}

func TestApplyBackendTrafficPolicy_DoesNotUpdateOrDeleteUnmanagedPolicy(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()

	existing := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.envoyproxy.io/v1alpha1",
		"kind":       "BackendTrafficPolicy",
		"metadata": map[string]interface{}{
			"name":      "companion",
			"namespace": testNamespace1,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "saasDeployer",
			},
		},
		"spec": map[string]interface{}{
			"timeout": map[string]interface{}{
				"http": map[string]interface{}{
					"streamIdleTimeout": "1800s",
				},
			},
		},
	}}
	_, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Create(ctx, existing, metav1.CreateOptions{})
	require.NoError(t, err)

	route := btpRoute("companion", nil)
	route.Spec.StreamIdleTimeout = "111s"
	policy, err := kube.backendTrafficPolicyFromRoute(route, testNamespace1)
	require.NoError(t, err)
	stored, err := kube.applyBackendTrafficPolicy(ctx, policy, route.Name, testNamespace1)
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "saasDeployer", stored.GetLabels()[entity.ManagedByLabel])
	timeout, found, err := unstructured.NestedString(stored.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1800s", timeout)

	got, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "companion", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "saasDeployer", got.GetLabels()[entity.ManagedByLabel])
	timeout, found, err = unstructured.NestedString(got.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1800s", timeout)

	skipped, err := kube.deleteBackendTrafficPolicy(ctx, "companion", testNamespace1)
	require.NoError(t, err)
	require.NotNil(t, skipped)
	assert.Equal(t, "saasDeployer", skipped.GetLabels()[entity.ManagedByLabel])
	got, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "companion", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, "saasDeployer", got.GetLabels()[entity.ManagedByLabel])
}

func TestBackendTrafficPolicyFromRoute_InvalidTimeout(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	route := btpRoute("bad", nil)
	route.Spec.StreamIdleTimeout = "not-a-duration"
	_, err := kube.backendTrafficPolicyFromRoute(route, testNamespace1)
	assert.Error(t, err)
	assert.True(t, paasErrors.IsBadRequest(err))
}

func TestBackendTrafficPolicyFromRoute_RejectsUnsupportedAnnotations(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	route := btpRoute("bad-ann", map[string]string{
		entity.AnnotationProxyReadTimeout: "60",
		AnnotationConfigSnippet:           "something",
	})
	_, err := kube.backendTrafficPolicyFromRoute(route, testNamespace1)
	assert.Error(t, err)
	assert.True(t, paasErrors.IsInvalid(err))
}

func TestWithHTTPRouteRequestIdleTimeout_InvalidRejectedAtBuild(t *testing.T) {
	_, err := NewKubernetesClientBuilder().
		WithNamespace(testNamespace1).
		WithClient(newTestBackendAPI(nil)).
		WithHTTPRouteRequestIdleTimeout("bad").
		Build()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), HTTPRouteRequestIdleTimeoutProperty)
}

func TestSkipOrWrapPolicyError_IgnorableAndReal(t *testing.T) {
	ctx := context.Background()
	assert.NoError(t, skipOrWrapPolicyError(ctx, "create", "x",
		paasErrors.NewMethodNotSupported(schema.GroupResource{Resource: "backendtrafficpolicies"}, "create")))
	err := skipOrWrapPolicyError(ctx, "update", "x", assert.AnError)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update BackendTrafficPolicy")
}

func TestApplyBackendTrafficPolicy_CreateIgnorableAbsence(t *testing.T) {
	dyn := newDynamicFake()
	dyn.PrependReactor("create", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, paasErrors.NewMethodNotSupported(
			schema.GroupResource{Resource: "backendtrafficpolicies"}, "create")
	})
	kube := newKubeWithDynamic(t, dyn, "")
	err := applyPolicyFromRoute(kube, context.Background(), btpRoute("route-no-crd", map[string]string{
		entity.AnnotationProxyReadTimeout: "60",
	}), testNamespace1)
	assert.NoError(t, err)
}

func TestDeleteBackendTrafficPolicy_SkipsWithoutDynamic(t *testing.T) {
	kube, err := NewTestKubernetesClient(testNamespace1, newTestBackendAPI(nil))
	require.NoError(t, err)
	_, err = kube.deleteBackendTrafficPolicy(context.Background(), "missing", testNamespace1)
	assert.NoError(t, err)
}

func TestIsIgnorablePolicyAbsence(t *testing.T) {
	assert.True(t, isIgnorablePolicyAbsence(nil))
	assert.True(t, isIgnorablePolicyAbsence(paasErrors.NewNotFound(schema.GroupResource{Resource: "backendtrafficpolicies"}, "x")))
	assert.True(t, isIgnorablePolicyAbsence(&meta.NoKindMatchError{GroupKind: schema.GroupKind{Kind: "BackendTrafficPolicy"}}))
	assert.True(t, isIgnorablePolicyAbsence(&meta.NoResourceMatchError{PartialResource: schema.GroupVersionResource{Resource: "backendtrafficpolicies"}}))
	assert.False(t, isIgnorablePolicyAbsence(assert.AnError))
}

func TestRouteRoundTrip_RestoresTimeoutAndGrpcFromManagedPolicy(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "30m")
	ctx := context.Background()

	route := btpRoute("route-roundtrip", map[string]string{
		entity.AnnotationBackendProtocol: "GRPC",
	})
	route.Spec.StreamIdleTimeout = "3600s"
	route.Spec.Path = "/"
	route.Spec.PathType = "Prefix"
	route.Spec.Port = entity.RoutePort{TargetPort: 8080}

	created, err := kube.CreateRoute(ctx, route, testNamespace1)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, "3600s", created.Spec.StreamIdleTimeout)
	assert.Equal(t, "GRPC", created.Metadata.Annotations[entity.AnnotationBackendProtocol])

	got, err := kube.GetRoute(ctx, "route-roundtrip", testNamespace1)
	require.NoError(t, err)
	assert.Equal(t, "3600s", got.Spec.StreamIdleTimeout)
	assert.Equal(t, "GRPC", got.Metadata.Annotations[entity.AnnotationBackendProtocol])

	listed, err := kube.GetRouteList(ctx, testNamespace1, filter.Meta{})
	require.NoError(t, err)
	require.Len(t, listed, 1)
	assert.Equal(t, "3600s", listed[0].Spec.StreamIdleTimeout)
	assert.Equal(t, "GRPC", listed[0].Metadata.Annotations[entity.AnnotationBackendProtocol])

	updated, err := kube.UpdateOrCreateRoute(ctx, got, testNamespace1)
	require.NoError(t, err)
	assert.Equal(t, "3600s", updated.Spec.StreamIdleTimeout)
	assert.Equal(t, "GRPC", updated.Metadata.Annotations[entity.AnnotationBackendProtocol])

	policy, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-roundtrip", metav1.GetOptions{})
	require.NoError(t, err)
	timeout, found, err := unstructured.NestedString(policy.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "3600s", timeout)
	useClient, found, err := unstructured.NestedBool(policy.Object, "spec", "useClientProtocol")
	require.NoError(t, err)
	assert.True(t, found)
	assert.True(t, useClient)
}

func TestCreateRoute_DoesNotReportSkippedUnmanagedPolicy(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()

	existing := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.envoyproxy.io/v1alpha1",
		"kind":       "BackendTrafficPolicy",
		"metadata": map[string]interface{}{
			"name":      "companion-create",
			"namespace": testNamespace1,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "saasDeployer",
			},
		},
		"spec": map[string]interface{}{
			"timeout": map[string]interface{}{
				"http": map[string]interface{}{
					"streamIdleTimeout": "1800s",
				},
			},
		},
	}}
	_, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Create(ctx, existing, metav1.CreateOptions{})
	require.NoError(t, err)

	route := btpRoute("companion-create", nil)
	route.Spec.StreamIdleTimeout = "111s"
	created, err := kube.CreateRoute(ctx, route, testNamespace1)
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Empty(t, created.Spec.StreamIdleTimeout)

	updated := btpRoute("companion-create", nil)
	updated.Spec.StreamIdleTimeout = "222s"
	got, err := kube.UpdateOrCreateRoute(ctx, updated, testNamespace1)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Empty(t, got.Spec.StreamIdleTimeout)

	stored, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "companion-create", metav1.GetOptions{})
	require.NoError(t, err)
	timeout, found, err := unstructured.NestedString(stored.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1800s", timeout)
}

func TestGetRoute_DoesNotRestoreUnmanagedPolicy(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()

	_, err := kube.CreateRoute(ctx, btpRoute("companion-get", nil), testNamespace1)
	require.NoError(t, err)

	existing := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "gateway.envoyproxy.io/v1alpha1",
		"kind":       "BackendTrafficPolicy",
		"metadata": map[string]interface{}{
			"name":      "companion-get",
			"namespace": testNamespace1,
			"labels": map[string]interface{}{
				"app.kubernetes.io/managed-by": "saasDeployer",
			},
		},
		"spec": map[string]interface{}{
			"useClientProtocol": true,
			"timeout": map[string]interface{}{
				"http": map[string]interface{}{
					"streamIdleTimeout": "1800s",
				},
			},
		},
	}}
	_, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Create(ctx, existing, metav1.CreateOptions{})
	require.NoError(t, err)

	got, err := kube.GetRoute(ctx, "companion-get", testNamespace1)
	require.NoError(t, err)
	assert.Empty(t, got.Spec.StreamIdleTimeout)
	assert.NotEqual(t, "GRPC", got.Metadata.Annotations[entity.AnnotationBackendProtocol])
}

func TestRouteFromWatchedHTTPRoute_RestoresManagedPolicy(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "30m")
	ctx := context.Background()

	route := btpRoute("watch-route", map[string]string{
		entity.AnnotationBackendProtocol: "GRPC",
	})
	route.Spec.StreamIdleTimeout = "3600s"
	created, err := kube.CreateRoute(ctx, route, testNamespace1)
	require.NoError(t, err)

	httpRoute, err := kube.getGatewayV1Client().HTTPRoutes(testNamespace1).Get(ctx, created.Name, metav1.GetOptions{})
	require.NoError(t, err)

	got := kube.routeFromWatchedHTTPRoute(httpRoute)
	require.NotNil(t, got)
	assert.Equal(t, "3600s", got.Spec.StreamIdleTimeout)
	assert.Equal(t, "GRPC", got.Metadata.Annotations[entity.AnnotationBackendProtocol])
	assert.Nil(t, kube.routeFromWatchedHTTPRoute(nil))
}

func TestGetRoute_PropagatesBackendTrafficPolicyGetError(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()

	_, err := kube.CreateRoute(ctx, btpRoute("route-get-err", nil), testNamespace1)
	require.NoError(t, err)

	dyn.PrependReactor("get", "backendtrafficpolicies", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})

	got, err := kube.GetRoute(ctx, "route-get-err", testNamespace1)
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorContains(t, err, "failed to get BackendTrafficPolicy")
}

func TestGetRouteList_PropagatesBackendTrafficPolicyListError(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	ctx := context.Background()

	_, err := kube.CreateRoute(ctx, btpRoute("route-list-err", nil), testNamespace1)
	require.NoError(t, err)

	dyn.PrependReactor("list", "backendtrafficpolicies", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, assert.AnError
	})

	got, err := kube.GetRouteList(ctx, testNamespace1, filter.Meta{})
	assert.Error(t, err)
	assert.Nil(t, got)
	assert.ErrorContains(t, err, "failed to list BackendTrafficPolicies")
}
