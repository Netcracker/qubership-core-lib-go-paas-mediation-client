package kubernetes

import (
	"context"
	"errors"
	"fmt"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	return kube.client.DynamicInterface
}

func (kube *Kubernetes) applyBackendTrafficPolicy(ctx context.Context, route *entity.Route, namespace string) error {
	dyn := kube.dynamicClient()
	if dyn == nil {
		logger.WarnC(ctx, "Dynamic client is not configured; skipping BackendTrafficPolicy for %s", route.Name)
		return nil
	}

	routeCopy := *route
	routeCopy.Metadata.Namespace = namespace
	policy, err := routeCopy.ToBackendTrafficPolicy()
	if err != nil {
		return err
	}

	if policy == nil {
		return kube.deleteOwnedBackendTrafficPolicy(ctx, route.Name, namespace)
	}

	existing, getErr := dyn.Resource(backendTrafficPolicyGVR).Namespace(namespace).Get(ctx, route.Name, metav1.GetOptions{})
	if getErr != nil {
		if paasErrors.IsNotFound(getErr) || isIgnorablePolicyAbsence(getErr) {
			_, createErr := dyn.Resource(backendTrafficPolicyGVR).Namespace(namespace).Create(ctx, policy, metav1.CreateOptions{})
			if createErr != nil {
				if isIgnorablePolicyAbsence(createErr) {
					logger.WarnC(ctx, "Skipping BackendTrafficPolicy create for %s: %v", route.Name, createErr)
					return nil
				}
				return fmt.Errorf("failed to create BackendTrafficPolicy %s: %w", route.Name, createErr)
			}
			logger.InfoC(ctx, "BackendTrafficPolicy created: %s", route.Name)
			return nil
		}
		return fmt.Errorf("failed to get BackendTrafficPolicy %s: %w", route.Name, getErr)
	}

	if existing.GetLabels()[entity.ManagedByLabel] != entity.ManagedByPaasMediation {
		logger.WarnC(ctx, "Skipping BackendTrafficPolicy update for %s: not managed by %s",
			route.Name, entity.ManagedByPaasMediation)
		return nil
	}

	policy.SetResourceVersion(existing.GetResourceVersion())
	_, updateErr := dyn.Resource(backendTrafficPolicyGVR).Namespace(namespace).Update(ctx, policy, metav1.UpdateOptions{})
	if updateErr != nil {
		if isIgnorablePolicyAbsence(updateErr) {
			logger.WarnC(ctx, "Skipping BackendTrafficPolicy update for %s: %v", route.Name, updateErr)
			return nil
		}
		return fmt.Errorf("failed to update BackendTrafficPolicy %s: %w", route.Name, updateErr)
	}
	logger.InfoC(ctx, "BackendTrafficPolicy updated: %s", route.Name)
	return nil
}

func (kube *Kubernetes) deleteOwnedBackendTrafficPolicy(ctx context.Context, name, namespace string) error {
	dyn := kube.dynamicClient()
	if dyn == nil {
		return nil
	}

	existing, err := dyn.Resource(backendTrafficPolicyGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if isIgnorablePolicyAbsence(err) {
			return nil
		}
		return fmt.Errorf("failed to get BackendTrafficPolicy %s: %w", name, err)
	}

	if existing.GetLabels()[entity.ManagedByLabel] != entity.ManagedByPaasMediation {
		logger.WarnC(ctx, "Skipping BackendTrafficPolicy delete for %s: not managed by %s",
			name, entity.ManagedByPaasMediation)
		return nil
	}

	if err := dyn.Resource(backendTrafficPolicyGVR).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if isIgnorablePolicyAbsence(err) {
			logger.WarnC(ctx, "Skipping BackendTrafficPolicy delete for %s: %v", name, err)
			return nil
		}
		return fmt.Errorf("failed to delete BackendTrafficPolicy %s: %w", name, err)
	}
	logger.InfoC(ctx, "BackendTrafficPolicy deleted: %s", name)
	return nil
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
