package kubernetes

import (
	"context"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
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

func TestApplyBackendTrafficPolicy_SkipsWithoutDynamicClient(t *testing.T) {
	kube, err := NewTestKubernetesClient(testNamespace1, newTestBackendAPI(nil))
	require.NoError(t, err)

	err = kube.applyBackendTrafficPolicy(context.Background(), btpRoute("r1", map[string]string{
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

	require.NoError(t, kube.applyBackendTrafficPolicy(ctx, route, testNamespace1))

	created, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-btp", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, entity.ManagedByPaasMediation, created.GetLabels()[entity.ManagedByLabel])
	timeout, found, err := unstructured.NestedString(created.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "1800s", timeout)

	route.Metadata.Annotations[entity.AnnotationProxyReadTimeout] = "900"
	require.NoError(t, kube.applyBackendTrafficPolicy(ctx, route, testNamespace1))
	updated, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-btp", metav1.GetOptions{})
	require.NoError(t, err)
	timeout, _, err = unstructured.NestedString(updated.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.Equal(t, "900s", timeout)

	require.NoError(t, kube.deleteBackendTrafficPolicy(ctx, "route-btp", testNamespace1))
	_, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-btp", metav1.GetOptions{})
	assert.True(t, err != nil)
}

func TestApplyBackendTrafficPolicy_UsesDefaultIdleTimeout(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "30m")
	ctx := context.Background()

	require.NoError(t, kube.applyBackendTrafficPolicy(ctx, btpRoute("route-default", nil), testNamespace1))
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
	require.NoError(t, kube.applyBackendTrafficPolicy(ctx, route, testNamespace1))

	route.Metadata.Annotations = nil
	require.NoError(t, kube.applyBackendTrafficPolicy(ctx, route, testNamespace1))
	_, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "route-clear", metav1.GetOptions{})
	assert.Error(t, err)
}

func TestApplyBackendTrafficPolicy_UpdatesAndDeletesCompanionPolicy(t *testing.T) {
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
	require.NoError(t, kube.applyBackendTrafficPolicy(ctx, route, testNamespace1))
	got, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "companion", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Equal(t, entity.ManagedByPaasMediation, got.GetLabels()[entity.ManagedByLabel])
	timeout, found, err := unstructured.NestedString(got.Object, "spec", "timeout", "http", "streamIdleTimeout")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, "111s", timeout)

	require.NoError(t, kube.deleteBackendTrafficPolicy(ctx, "companion", testNamespace1))
	_, err = dyn.Resource(backendTrafficPolicyGVR).Namespace(testNamespace1).Get(ctx, "companion", metav1.GetOptions{})
	assert.True(t, paasErrors.IsNotFound(err))
}

func TestApplyBackendTrafficPolicy_InvalidTimeout(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	route := btpRoute("bad", nil)
	route.Spec.StreamIdleTimeout = "not-a-duration"
	err := kube.applyBackendTrafficPolicy(context.Background(), route, testNamespace1)
	assert.Error(t, err)
	assert.True(t, paasErrors.IsBadRequest(err))
}

func TestApplyBackendTrafficPolicy_RejectsUnsupportedAnnotations(t *testing.T) {
	dyn := newDynamicFake()
	kube := newKubeWithDynamic(t, dyn, "")
	route := btpRoute("bad-ann", map[string]string{
		entity.AnnotationProxyReadTimeout: "60",
		AnnotationConfigSnippet:           "something",
	})
	err := kube.applyBackendTrafficPolicy(context.Background(), route, testNamespace1)
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
	err := kube.applyBackendTrafficPolicy(context.Background(), btpRoute("route-no-crd", map[string]string{
		entity.AnnotationProxyReadTimeout: "60",
	}), testNamespace1)
	assert.NoError(t, err)
}

func TestDeleteBackendTrafficPolicy_SkipsWithoutDynamic(t *testing.T) {
	kube, err := NewTestKubernetesClient(testNamespace1, newTestBackendAPI(nil))
	require.NoError(t, err)
	assert.NoError(t, kube.deleteBackendTrafficPolicy(context.Background(), "missing", testNamespace1))
}

func TestIsIgnorablePolicyAbsence(t *testing.T) {
	assert.True(t, isIgnorablePolicyAbsence(nil))
	assert.True(t, isIgnorablePolicyAbsence(paasErrors.NewNotFound(schema.GroupResource{Resource: "backendtrafficpolicies"}, "x")))
	assert.True(t, isIgnorablePolicyAbsence(&meta.NoKindMatchError{GroupKind: schema.GroupKind{Kind: "BackendTrafficPolicy"}}))
	assert.True(t, isIgnorablePolicyAbsence(&meta.NoResourceMatchError{PartialResource: schema.GroupVersionResource{Resource: "backendtrafficpolicies"}}))
	assert.False(t, isIgnorablePolicyAbsence(assert.AnError))
}
