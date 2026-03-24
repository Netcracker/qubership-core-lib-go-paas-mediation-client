package kubernetes

import (
	"context"
	"errors"
	"net/http"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const anyValue = "*"

func GetWrapper[F any, T entity.HasMetadata](ctx context.Context, name string, namespace string,
	getFromApiServer func(ctx context.Context, name string, opts v1.GetOptions) (*F, error),
	cache *cache.ResourceCache[T],
	convert func(obj *F) *T) (*T, error) {
	logger.DebugC(ctx, "Getting resource name: '%v' from api server", name)
	obj, err := getFromApiServer(ctx, name, v1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, nil
		}
		logger.ErrorC(ctx, "Get resource with namespace: '%s', name: '%s' API server returned err: %s", namespace, name, err.Error())
		if cache != nil {
			logger.InfoC(ctx, "Retrieving resource with namespace: '%s', name: '%s' from cache", namespace, name)
			if cached := cache.Get(ctx, namespace, name); cached != nil {
				return cached, nil
			} else {
				logger.WarnC(ctx, "Resource with namespace: '%s', name: '%s' does not exist in cache", namespace, name)
				return nil, err
			}
		}
		return nil, err
	}
	return convert(obj), err
}

func ListWrapper[T entity.HasMetadata, L v1.ListInterface](ctx context.Context, metaFilter filter.Meta,
	listFromApiServer func(ctx context.Context, opts v1.ListOptions) (L, error),
	cache *cache.ResourceCache[T],
	convert func(listObj L) []T) ([]T, error) {
	// todo, I thought limit would make things better, but somehow requests with continue take more time, and its processing consumes more memory...
	//var listOptions = v1.ListOptions{Limit: 10}
	var listOptions = v1.ListOptions{}
	if metaFilter.GetLabels() != nil {
		listOptions.LabelSelector = labels.Set(metaFilter.GetLabels()).String()
		logger.DebugC(ctx, "Applying label selector: %s", listOptions.LabelSelector)
	}
	max410GoneAttempts := 3
	attempt := 0
	var result []T
	var err error
	for attempt < max410GoneAttempts {
		attempt++
		if result, err = listLimited(ctx, listOptions, metaFilter, listFromApiServer, convert); err != nil {
			var statusErr *k8sErrors.StatusError
			if ok := errors.As(err, &statusErr); ok && statusErr.Status().Code == http.StatusGone {
				logger.WarnC(ctx, "Received 410 Gone error from api server on attempt: #%d", attempt)
				continue
			}
		}
		break
	}
	if err != nil {
		//todo get items from the cache if it contains all items received via watch handler (no item was skipped because it was too large)
	}
	return result, err
}

func listLimited[T entity.HasMetadata, L v1.ListInterface](ctx context.Context,
	listOptions v1.ListOptions, metaFilter filter.Meta, listFromApiServer func(ctx context.Context, opts v1.ListOptions) (L, error),
	convert func(listObj L) []T) ([]T, error) {
	var result []T
	for {
		apiServerList, err := listFromApiServer(ctx, listOptions)
		if err != nil {
			return nil, err
		} else {
			continueValue := apiServerList.GetContinue()
			partialResult := convert(apiServerList)
			// filter out the result according to the provided annotations if any (labels are already handled by k8s api server)
			if partialResult != nil && len(metaFilter.GetAnnotations()) > 0 {
				partialResult = filterBy(ctx, "annotations", partialResult,
					func(resource T) map[string]string {
						return resource.GetMetadata().Annotations
					}, metaFilter.GetAnnotations())
			}
			result = append(result, partialResult...)
			if continueValue != "" {
				listOptions.Continue = continueValue
				logger.DebugC(ctx, "Continue limited List with listOptions: '%+v'", listOptions)
			} else {
				logger.DebugC(ctx, "Received last part for limited List request. Returning result")
				break
			}
		}
	}
	return result, nil
}

func DeleteWrapper(ctx context.Context, name string, namespace string,
	deleteFromApiServer func(ctx context.Context, name string, opts v1.DeleteOptions) error) error {
	logger.DebugC(ctx, "Deleting resource namespace: '%s', name: '%s' from api server", namespace, name)
	err := deleteFromApiServer(ctx, name, v1.DeleteOptions{})
	if err != nil {
		logger.ErrorC(ctx, "failed to delete the resource with namespace: '%s, name: '%s' from kubernetes: %s", namespace, name, err.Error())
		return err
	}
	return nil
}

func CreateWrapper[F any, T entity.HasMetadata](ctx context.Context, resource T,
	createInApiServer func(ctx context.Context, resource *F, opts v1.CreateOptions) (*F, error),
	convertTo func() *F,
	convertFrom func(obj *F) *T) (*T, error) {
	metadata := resource.GetMetadata()
	namespace := metadata.Namespace
	name := metadata.Name
	logger.DebugC(ctx, "Creating resource with namespace: '%s', name: '%s' in the api server", namespace, name)
	obj, err := createInApiServer(ctx, convertTo(), v1.CreateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Failed to create resource with namespace: '%s', name: '%s', API server returned err: %s", namespace, name, err.Error())
		return nil, err
	}
	convertedObj := convertFrom(obj)
	return convertedObj, err
}

func UpdateOrCreateWrapper[F any, T entity.HasMetadata](ctx context.Context, resource T,
	getFromApiServer func(ctx context.Context, name string, opts v1.GetOptions) (*F, error),
	createInApiServer func(ctx context.Context, resource *F, opts v1.CreateOptions) (*F, error),
	updateInApiServer func(ctx context.Context, resource *F, opts v1.UpdateOptions) (*F, error),
	convertTo func() *F,
	convertFrom func(obj *F) *T) (*T, error) {
	metadata := resource.GetMetadata()
	namespace := metadata.Namespace
	name := metadata.Name
	logger.DebugC(ctx, "Updating or Creating resource with namespace: '%s', name: '%s' in the api server", namespace, name)
	_, err := getFromApiServer(ctx, name, v1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return CreateWrapper(ctx, resource, createInApiServer, convertTo, convertFrom)
		}
		logger.ErrorC(ctx, "Error while get the resource with namespace: '%s', name: '%s' from the api server before update: %v", namespace, name, err.Error())
		return nil, err
	} else {
		logger.DebugC(ctx, "Updating resource with namespace: '%s', name: '%s' in the api server", namespace, name)
		updatedObj, err := updateInApiServer(ctx, convertTo(), v1.UpdateOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Failed to create resource with namespace: '%s', name: '%s', API server returned err: %s", namespace, name, err.Error())
			return nil, err
		}
		return convertFrom(updatedObj), nil
	}
}

func ToPointersSlice[T any](items []T) (result []*T) {
	for _, i := range items {
		item := i
		result = append(result, &item)
	}
	return
}

func filterBy[T any](ctx context.Context, metaType string, resources []T,
	getMeta func(resource T) map[string]string,
	metaFilter map[string]string) (result []T) {
	logger.DebugC(ctx, "Applying %s selector: %v", metaType, metaFilter)
	for _, resource := range resources {
		if containsAll(getMeta(resource), metaFilter) {
			result = append(result, resource)
		}
	}
	logger.DebugC(ctx, "Found %d resources after applying %s filter", len(result), metaType)
	return
}

func containsAll(resourceAnns map[string]string, requiredAnns map[string]string) bool {
	for requiredAnnName, requiredAnnValue := range requiredAnns {
		if annValue, found := resourceAnns[requiredAnnName]; !found || requiredAnnValue != anyValue && annValue != requiredAnnValue {
			return false
		}
	}
	return true
}
