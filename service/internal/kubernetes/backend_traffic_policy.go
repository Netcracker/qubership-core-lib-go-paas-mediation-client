package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var backendTrafficPolicyGVR = schema.GroupVersionResource{
	Group:    "gateway.envoyproxy.io",
	Version:  "v1alpha1",
	Resource: "backendtrafficpolicies",
}

func (kube *Kubernetes) dynamicClient() dynamic.Interface {
	if kube.client == nil {
		return nil
	}
	dyn := kube.client.DynamicInterface
	if dyn == nil {
		return nil
	}
	// Guard against typed-nil pointers stored in the interface (common with fakes).
	v := reflect.ValueOf(dyn)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	return dyn
}

func (kube *Kubernetes) backendTrafficPolicyClient(namespace string) dynamic.ResourceInterface {
	dyn := kube.dynamicClient()
	if dyn == nil {
		return nil
	}
	return dyn.Resource(backendTrafficPolicyGVR).Namespace(namespace)
}

func (kube *Kubernetes) getBackendTrafficPolicy(ctx context.Context, name, namespace string) (*unstructured.Unstructured, error) {
	client := kube.backendTrafficPolicyClient(namespace)
	if client == nil {
		return nil, nil
	}
	policy, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if isIgnorablePolicyAbsence(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get BackendTrafficPolicy %s: %w", name, err)
	}
	return policy, nil
}

func (kube *Kubernetes) listBackendTrafficPoliciesByName(ctx context.Context, namespace string) (map[string]*unstructured.Unstructured, error) {
	client := kube.backendTrafficPolicyClient(namespace)
	if client == nil {
		return nil, nil
	}
	list, err := client.List(ctx, metav1.ListOptions{})
	if err != nil {
		if isIgnorablePolicyAbsence(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list BackendTrafficPolicies in %s: %w", namespace, err)
	}
	out := make(map[string]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		out[item.GetName()] = item
	}
	return out, nil
}

func (kube *Kubernetes) backendTrafficPolicyClientOrSkip(ctx context.Context, namespace, name string) dynamic.ResourceInterface {
	client := kube.backendTrafficPolicyClient(namespace)
	if client == nil {
		logger.WarnC(ctx, "Dynamic client is not configured; skipping BackendTrafficPolicy for %s", name)
	}
	return client
}

func isManagedBackendTrafficPolicy(policy metav1.Object) bool {
	return policy.GetLabels()[entity.ManagedByLabel] == entity.ManagedByPaasMediation
}

func skipOrWrapPolicyError(ctx context.Context, operation, name string, err error) error {
	if isIgnorablePolicyAbsence(err) {
		logger.WarnC(ctx, "Skipping BackendTrafficPolicy %s for %s: %v", operation, name, err)
		return nil
	}
	return fmt.Errorf("failed to %s BackendTrafficPolicy %s: %w", operation, name, err)
}

func (kube *Kubernetes) backendTrafficPolicyFromRoute(route *entity.Route, namespace string) (*unstructured.Unstructured, error) {
	if err := kube.validateAnnotationsForGatewayAPI(route.Metadata.Annotations); err != nil {
		return nil, err
	}
	routeCopy := *route
	routeCopy.Metadata.Namespace = namespace
	policy, err := routeCopy.ToBackendTrafficPolicy(kube.HTTPRouteRequestIdleTimeout)
	if err != nil {
		return nil, paasErrors.NewBadRequest(err.Error())
	}
	return policy, nil
}

func (kube *Kubernetes) applyBackendTrafficPolicy(ctx context.Context, policy *unstructured.Unstructured, name, namespace string) (*unstructured.Unstructured, error) {
	client := kube.backendTrafficPolicyClientOrSkip(ctx, namespace, name)
	if client == nil {
		return nil, nil
	}
	if policy == nil {
		return kube.deleteBackendTrafficPolicy(ctx, name, namespace)
	}

	existing, getErr := client.Get(ctx, name, metav1.GetOptions{})
	if getErr != nil {
		if paasErrors.IsNotFound(getErr) || isIgnorablePolicyAbsence(getErr) {
			created, createErr := client.Create(ctx, policy, metav1.CreateOptions{})
			if createErr != nil {
				return nil, skipOrWrapPolicyError(ctx, "create", name, createErr)
			}
			logger.InfoC(ctx, "BackendTrafficPolicy created: %s", name)
			if created != nil {
				return created, nil
			}
			return policy, nil
		}
		return nil, fmt.Errorf("failed to get BackendTrafficPolicy %s: %w", name, getErr)
	}

	if !isManagedBackendTrafficPolicy(existing) {
		logger.WarnC(ctx, "Skipping BackendTrafficPolicy update for %s: not managed by %s",
			name, entity.ManagedByPaasMediation)
		return existing, nil
	}

	policy.SetResourceVersion(existing.GetResourceVersion())
	updated, updateErr := client.Update(ctx, policy, metav1.UpdateOptions{})
	if updateErr != nil {
		return nil, skipOrWrapPolicyError(ctx, "update", name, updateErr)
	}
	logger.InfoC(ctx, "BackendTrafficPolicy updated: %s", name)
	if updated != nil {
		return updated, nil
	}
	return policy, nil
}

func (kube *Kubernetes) deleteBackendTrafficPolicy(ctx context.Context, name, namespace string) (*unstructured.Unstructured, error) {
	client := kube.backendTrafficPolicyClientOrSkip(ctx, namespace, name)
	if client == nil {
		return nil, nil
	}

	existing, err := client.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if isIgnorablePolicyAbsence(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get BackendTrafficPolicy %s before delete: %w", name, err)
	}
	if !isManagedBackendTrafficPolicy(existing) {
		logger.WarnC(ctx, "Skipping BackendTrafficPolicy delete for %s: not managed by %s",
			name, entity.ManagedByPaasMediation)
		return existing, nil
	}

	if err := client.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, skipOrWrapPolicyError(ctx, "delete", name, err)
	}
	logger.InfoC(ctx, "BackendTrafficPolicy deleted: %s", name)
	return nil, nil
}

func isIgnorablePolicyAbsence(err error) bool {
	if err == nil {
		return true
	}
	if paasErrors.IsNotFound(err) || paasErrors.IsMethodNotSupported(err) {
		return true
	}
	var noKind *meta.NoKindMatchError
	if errors.As(err, &noKind) {
		return true
	}
	var noResource *meta.NoResourceMatchError
	return errors.As(err, &noResource)
}
