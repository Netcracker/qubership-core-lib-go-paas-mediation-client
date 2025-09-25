package kubernetes

import (
	"context"
	"fmt"
	"reflect"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	coreV1 "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var retryWatchTimeout = 5 * time.Second

var (
	loggerCacheAdapter logging.Logger
)

func init() {
	loggerCacheAdapter = logging.GetLogger("cache_adapter")
}

type CacheAdapters struct {
	Certificates          *CacheAdapter[*cmv1.Certificate, entity.Certificate]
	ConfigMaps            *CacheAdapter[*coreV1.ConfigMap, entity.ConfigMap]
	Services              *CacheAdapter[*coreV1.Service, entity.Service]
	Secrets               *CacheAdapter[*coreV1.Secret, entity.Secret]
	IngressesNetworkingV1 *CacheAdapter[*networkingV1.Ingress, entity.Route]
	HTTPRouteV1           *CacheAdapter[*gatewayv1.HTTPRoute, entity.HttpRoute]
	GRPCRouteV1           *CacheAdapter[*gatewayv1.GRPCRoute, entity.GrpcRoute]
}

func NewCacheAdapters(ctx context.Context, namespace string, cache *cache.ResourcesCache, watchHandlers *SharedWatchHandlers) (*CacheAdapters, error) {
	adapters := &CacheAdapters{}
	var err error
	if cache.Certificates != nil && watchHandlers.Certificates != nil {
		if adapters.Certificates, err = newCacheAdapter(ctx, namespace, cache.Certificates, watchHandlers.Certificates); err != nil {
			return nil, err
		}
	}
	if cache.ConfigMaps != nil && watchHandlers.ConfigMaps != nil {
		if adapters.ConfigMaps, err = newCacheAdapter(ctx, namespace, cache.ConfigMaps, watchHandlers.ConfigMaps); err != nil {
			return nil, err
		}
	}
	if cache.Secrets != nil && watchHandlers.Secrets != nil {
		if adapters.Secrets, err = newCacheAdapter(ctx, namespace, cache.Secrets, watchHandlers.Secrets); err != nil {
			return nil, err
		}
	}
	if cache.Services != nil && watchHandlers.Services != nil {
		if adapters.Services, err = newCacheAdapter(ctx, namespace, cache.Services, watchHandlers.Services); err != nil {
			return nil, err
		}
	}
	if cache.Ingresses != nil && watchHandlers.IngressesNetworkingV1 != nil {
		if adapters.IngressesNetworkingV1, err = newCacheAdapter(ctx, namespace, cache.Ingresses, watchHandlers.IngressesNetworkingV1); err != nil {
			return nil, err
		}
	}

	if cache.HTTPRoute != nil && watchHandlers.HTTPRouteV1 != nil {
		if adapters.HTTPRouteV1, err = newCacheAdapter(ctx, namespace, cache.HTTPRoute, watchHandlers.HTTPRouteV1); err != nil {
			return nil, err
		}
	}

	if cache.GRPCRoute != nil && watchHandlers.GRPCRouteV1 != nil {
		if adapters.GRPCRouteV1, err = newCacheAdapter(ctx, namespace, cache.GRPCRoute, watchHandlers.GRPCRouteV1); err != nil {
			return nil, err
		}
	}

	return adapters, nil
}

type CacheAdapter[F runtime.Object, T entity.HasMetadata] struct {
	ResourceType   string
	Cache          *cache.ResourceCache[T]
	WatcherHandler *SharedWatchHandler[F, T]
	logger         logging.Logger
}

func newCacheAdapter[F runtime.Object, T entity.HasMetadata](ctx context.Context, namespace string,
	cache *cache.ResourceCache[T], watchHandler *SharedWatchHandler[F, T]) (*CacheAdapter[F, T], error) {
	var resource T
	resourceName := reflect.TypeOf(resource).Name()
	adapter := &CacheAdapter[F, T]{ResourceType: resourceName, Cache: cache, WatcherHandler: watchHandler, logger: loggerCacheAdapter}
	err := adapter.startWatching(ctx, namespace)
	return adapter, err
}

func (a *CacheAdapter[F, T]) startWatching(ctx context.Context, namespace string) error {
	errChan := make(chan error)
	go a.watchloop(ctx, namespace, errChan)
	err := <-errChan
	if err != nil {
		return fmt.Errorf("faield to start watch handler for cache: %w", err)
	}
	a.logger.InfoC(ctx, "Started '%s' cache adapter", a.ResourceType)
	return nil
}

func (a *CacheAdapter[F, T]) watchloop(ctx context.Context, namespace string, errChan chan error) {
	handler, err := a.WatcherHandler.Watch(ctx, namespace, filter.Meta{})
	errChan <- err
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-handler.Channel:
			if !ok {
				time.Sleep(retryWatchTimeout)
				handler, err = a.WatcherHandler.Watch(ctx, namespace, filter.Meta{})
				if err != nil {
					a.logger.ErrorC(ctx, "failed to restart watch handler: %s", err.Error())
				}
			} else {
				a.OnWatchEvent(ctx, event.Type, *event.Object.(*T))
			}
		}
	}
}

func (a *CacheAdapter[F, T]) OnWatchEvent(ctx context.Context, event string, resource T) {
	switch event {
	case string(watch.Added):
		namespace, name := getNamespaceAndName(resource)
		if ok, err := a.Cache.Set(ctx, resource); err != nil {
			a.logger.WarnC(ctx, "failed to add to the cache resource [kind: '%s', '%s@%s'], err: %s", a.ResourceType, name, namespace, err.Error())
		} else {
			a.logger.DebugC(ctx, "attempted to add to the cache resource [kind: '%s', '%s@%s'], added: %v", a.ResourceType, name, namespace, ok)
		}
	case string(watch.Modified):
		namespace, name := getNamespaceAndName(resource)
		// replace existing in cache (if any) entity with received over watch api one
		a.Cache.Delete(ctx, namespace, name)
		if ok, err := a.Cache.Set(ctx, resource); err != nil {
			a.logger.ErrorC(ctx, "failed to update in the cache resource [kind: '%s', '%s@%s'], err: %s", a.ResourceType, name, namespace, err.Error())
		} else {
			a.logger.DebugC(ctx, "attempted to update in the cache resource [kind: '%s', '%s@%s'], added: %v", a.ResourceType, name, namespace, ok)
		}
	case string(watch.Deleted):
		namespace, name := getNamespaceAndName(resource)
		a.Cache.Delete(ctx, namespace, name)
		a.logger.DebugC(ctx, "deleted from the cache resource [kind: '%s', '%s@%s']", a.ResourceType, name, namespace)
	}
}

func getNamespaceAndName[T entity.HasMetadata](resource T) (namespace string, name string) {
	metadata := resource.GetMetadata()
	namespace = metadata.Namespace
	name = metadata.Name
	return
}
