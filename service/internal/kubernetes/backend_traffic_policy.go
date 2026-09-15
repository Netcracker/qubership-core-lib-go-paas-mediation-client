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

func skipOrWrapPolicyError(ctx context.Context, operation, name string, err error) error {
	if isIgnorablePolicyAbsence(err) {
		logger.WarnC(ctx, "Skipping BackendTrafficPolicy %s for %s: %v", operation, name, err)
		return nil
	}
	return fmt.Errorf("failed to %s BackendTrafficPolicy %s: %w", operation, name, err)
}

func (kube *Kubernetes) applyBackendTrafficPolicy(ctx context.Context, route *entity.Route, namespace string) error {
	client := kube.backendTrafficPolicyClient(namespace)
	if client == nil {
		logger.WarnC(ctx, "Dynamic client is not configured; skipping BackendTrafficPolicy for %s", route.Name)
		return nil
	}

	routeCopy := *route
	routeCopy.Metadata.Namespace = namespace
	policy, err := routeCopy.ToBackendTrafficPolicy(kube.HTTPRouteRequestIdleTimeout)
	if err != nil {
		return err
	}
	if policy == nil {
		return kube.deleteOwnedBackendTrafficPolicy(ctx, route.Name, namespace)
	}

	existing, getErr := client.Get(ctx, route.Name, metav1.GetOptions{})
	if getErr != nil {
		if paasErrors.IsNotFound(getErr) || isIgnorablePolicyAbsence(getErr) {
			_, createErr := client.Create(ctx, policy, metav1.CreateOptions{})
			if createErr != nil {
				return skipOrWrapPolicyError(ctx, "create", route.Name, createErr)
			}
			logger.InfoC(ctx, "BackendTrafficPolicy created: %s", route.Name)
			return nil
		}
		return fmt.Errorf("failed to get BackendTrafficPolicy %s: %w", route.Name, getErr)
	}

	policy.SetResourceVersion(existing.GetResourceVersion())
	_, updateErr := client.Update(ctx, policy, metav1.UpdateOptions{})
	if updateErr != nil {
		return skipOrWrapPolicyError(ctx, "update", route.Name, updateErr)
	}
	logger.InfoC(ctx, "BackendTrafficPolicy updated: %s", route.Name)
	return nil
}

func (kube *Kubernetes) deleteOwnedBackendTrafficPolicy(ctx context.Context, name, namespace string) error {
	client := kube.backendTrafficPolicyClient(namespace)
	if client == nil {
		return nil
	}

	if err := client.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return skipOrWrapPolicyError(ctx, "delete", name, err)
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
