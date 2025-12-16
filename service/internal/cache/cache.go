package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	k8sResource "k8s.io/apimachinery/pkg/api/resource"
)

var (
	logger logging.Logger
)

func init() {
	logger = logging.GetLogger("cache")
}

type ResourcesCache struct {
	Certificates *ResourceCache[entity.Certificate]
	ConfigMaps   *ResourceCache[entity.ConfigMap]
	Ingresses    *ResourceCache[entity.Route]
	HTTPRoute    *ResourceCache[entity.HttpRoute]
	GRPCRoute    *ResourceCache[entity.GrpcRoute]
	Secrets      *ResourceCache[entity.Secret]
	Services     *ResourceCache[entity.Service]
	Namespaces   *ResourceCache[entity.Namespace] // todo for backward compatibility, delete later
}

// /go:generate mockgen -source=cache.go -destination=cache_mock.go -package=cache //todo generics in interfaces not supported yet by mockgen
type LimitedCache[T entity.HasMetadata] interface {
	Set(kind string, namespace string, name string, value T) bool
	Get(kind string, namespace string, name string) *T
	//List(kind string, namespace string) []T //todo list must take into acount evictions on Set()
	Del(kind string, namespace string, name string)
}

// ResourceCache - generic proxy for specific entity type stored in the single shared cache
type ResourceCache[T entity.HasMetadata] struct {
	Kind  string
	Cache LimitedCache[T]
}

func NewResourcesCache(numItems int64, maxSizeInBytes int64, maxItemSizeInBytes int64, ttl time.Duration, cacheToEnable ...CacheName) (*ResourcesCache, error) {
	ristrettoCache, err := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: numItems * 10,
		MaxCost:     maxSizeInBytes,
		BufferItems: 64,
		OnEvict: func(item *ristretto.Item[[]byte]) {
			var objMap map[string]any
			json.Unmarshal(item.Value, &objMap)
			metadata := objMap["metadata"].(map[string]any)
			kind := metadata["kind"]
			namespace := metadata["namespace"]
			name := metadata["name"]
			logger.Debug("Item was evicted from ResourceCache: kind: '%s', namespace: '%s', name: '%s', cost: %s",
				kind, namespace, name, k8sResource.NewQuantity(item.Cost, k8sResource.BinarySI).String())
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate ristretto cache: %w", err)
	}
	adapter := &ristrettoCacheAdapter{cache: ristrettoCache, ttl: ttl, maxItemSize: maxItemSizeInBytes}
	resourceCache := &ResourcesCache{}
	if len(cacheToEnable) == 0 {
		cacheToEnable = append(cacheToEnable, BasicCaches...)
	}
	for _, cacheName := range cacheToEnable {
		if cacheName == CertificateCache {
			logger.Warn("Certificate cache is requested, but not supported yet") // todo figure out how to initate watch for certificates
			//resourceCache.Certificates = NewResourceCache(&entity.Certificate{}, newEntityAdapter[entity.Certificate](adapter))
		}
		if cacheName == ConfigMapCache {
			resourceCache.ConfigMaps = NewResourceCache(&entity.ConfigMap{}, newEntityAdapter[entity.ConfigMap](adapter))
		}
		if cacheName == RouteCache {
			resourceCache.Ingresses = NewResourceCache(&entity.Route{}, newEntityAdapter[entity.Route](adapter))
		}
		if cacheName == SecretCache {
			resourceCache.Secrets = NewResourceCache(&entity.Secret{}, newEntityAdapter[entity.Secret](adapter))
		}
		if cacheName == ServiceCache {
			resourceCache.Services = NewResourceCache(&entity.Service{}, newEntityAdapter[entity.Service](adapter))
		}
		if cacheName == NamespaceCache {
			resourceCache.Namespaces = NewResourceCache(&entity.Namespace{}, newEntityAdapter[entity.Namespace](adapter))
		}
		if cacheName == HttpRouteCache {
			resourceCache.HTTPRoute = NewResourceCache(&entity.HttpRoute{}, newEntityAdapter[entity.HttpRoute](adapter))
		}
		if cacheName == GrpcRouteCache {
			resourceCache.GRPCRoute = NewResourceCache(&entity.GrpcRoute{}, newEntityAdapter[entity.GrpcRoute](adapter))
		}
	}
	return resourceCache, nil
}

func NewTestResourcesCache(cacheToEnable ...CacheName) *ResourcesCache {
	cache, err := NewResourcesCache(int64(1000), int64(10000), int64(1024), 0, cacheToEnable...)
	if err != nil {
		panic(err.Error())
	}
	return cache
}

func NewResourceCache[T entity.HasMetadata](entity *T, cache LimitedCache[T]) *ResourceCache[T] {
	kind := reflect.TypeOf(entity).Elem().Name()
	return &ResourceCache[T]{Kind: kind, Cache: cache}
}

func newEntityAdapter[T entity.HasMetadata](sharedAdapter *ristrettoCacheAdapter) LimitedCache[T] {
	return &ristrettoPerEntityCacheAdapter[T]{sharedAdapter: sharedAdapter, maxItemSize: sharedAdapter.maxItemSize}
}

func (cache *ResourceCache[T]) Set(ctx context.Context, resource T) (bool, error) {
	metadata := resource.GetMetadata()
	ok := cache.Cache.Set(cache.Kind, metadata.Namespace, metadata.Name, resource)
	return ok, nil
}

func (cache *ResourceCache[T]) Delete(ctx context.Context, namespace string, name string) {
	cache.Cache.Del(cache.Kind, namespace, name)
}

func (cache *ResourceCache[T]) Get(ctx context.Context, namespace string, name string) *T {
	return cache.Cache.Get(cache.Kind, namespace, name)
}

//func (cache *ResourceCache[T]) List(ctx context.Context, namespace string) []T {
//	return cache.Cache.List(cache.Kind, namespace)
//}

type cacheKey struct {
	kind      string
	namespace string
	name      string
}

func toCacheKey(kind string, namespace string, name string) cacheKey {
	return cacheKey{
		kind:      kind,
		namespace: namespace,
		name:      name,
	}
}

func fromCacheKey(key cacheKey) (kind string, namespace string, name string) {
	return key.kind, key.namespace, key.name
}

func (key cacheKey) String() string {
	return fmt.Sprintf("%s/%s/%s", key.kind, key.namespace, key.name)
}

type ristrettoCacheAdapter struct {
	cache       *ristretto.Cache[string, []byte]
	maxItemSize int64
	ttl         time.Duration
}

func (adapter *ristrettoCacheAdapter) Set(kind string, namespace string, name string, value []byte, cost int64) bool {
	key := toCacheKey(kind, namespace, name)
	ok := adapter.cache.SetWithTTL(key.String(), value, cost, adapter.ttl)
	if ok {
		adapter.cache.Wait()
	}
	return ok
}

func (adapter *ristrettoCacheAdapter) Get(kind string, namespace string, name string) ([]byte, bool) {
	key := toCacheKey(kind, namespace, name)
	v, ok := adapter.cache.Get(key.String())
	if ok {
		return v, ok
	}
	return nil, ok
}

func (adapter *ristrettoCacheAdapter) Del(kind string, namespace string, name string) {
	key := toCacheKey(kind, namespace, name)
	adapter.cache.Del(key.String())
}

type ristrettoPerEntityCacheAdapter[T entity.HasMetadata] struct {
	maxItemSize   int64
	sharedAdapter *ristrettoCacheAdapter
}

func (adapter *ristrettoPerEntityCacheAdapter[T]) Set(kind string, namespace string, name string, value T) bool {
	bytes, err := json.Marshal(value)
	if err != nil {
		logger.Error("Failed to marshall value: %s", err.Error())
		return false
	}
	cost := int64(len(bytes)) + 16
	if adapter.maxItemSize > 0 && cost > adapter.maxItemSize {
		// skip too large items
		logger.Warn("Resource of type: '%s' with namespace: '%s', name: '%s' is too large (%s > max (%s)) and NOT placed into the cache", kind, namespace, name,
			k8sResource.NewQuantity(cost, k8sResource.BinarySI).String(), k8sResource.NewQuantity(adapter.maxItemSize, k8sResource.BinarySI).String())
		return false
	}
	ok := adapter.sharedAdapter.Set(kind, namespace, name, bytes, cost)
	logger.Debug("Resource of type: '%s' with namespace: '%s', name: '%s', cost: %v placed into the cache: %v",
		kind, namespace, name, k8sResource.NewQuantity(cost, k8sResource.BinarySI).String(), ok)
	return ok
}

func (adapter *ristrettoPerEntityCacheAdapter[T]) Get(kind string, namespace string, name string) *T {
	bytes, ok := adapter.sharedAdapter.Get(kind, namespace, name)
	if ok {
		var r T
		json.Unmarshal(bytes, &r)
		logger.Debug("Resource of type: '%s' with namespace: '%s', name: '%s' retrieved from the cache", kind, namespace, name)
		return &r
	} else {
		return nil
	}
}

func (adapter *ristrettoPerEntityCacheAdapter[T]) Del(kind string, namespace string, name string) {
	adapter.sharedAdapter.Del(kind, namespace, name)
	logger.Debug("Resource of type: '%s' with namespace: '%s', name: '%s' deleted from the cache", kind, namespace, name)
}
