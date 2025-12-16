package service

import (
	"time"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/exec"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
)

type PlatformClientBuilder struct {
	platformType            *types.PlatformType
	caches                  map[cache.CacheName]struct{}
	cacheNumItems           int64
	cacheMaxSizeInBytes     int64
	cacheMaxItemSizeInBytes int64
	cacheItemTTL            time.Duration
	kubernetesApi           *backend.KubernetesApi
	openshiftApi            *backend.OpenshiftApi
	namespace               *string
	watchExecutor           watch.Executor
	watchClientTimeout      *time.Duration
	badResources            *kubernetes.BadResources
	rolloutParallelism      *int
	rolloutExecutor         exec.RolloutExecutor
	consulEnabled           *bool
	consulUrl               *string
	consulToken             *string
}

func NewPlatformClientBuilder() PlatformClientBuilder {
	return PlatformClientBuilder{
		caches:                  map[cache.CacheName]struct{}{},
		cacheNumItems:           1000,
		cacheMaxItemSizeInBytes: 4096,             //4KB
		cacheMaxSizeInBytes:     10 * 1024 * 1024, //10MB
		cacheItemTTL:            time.Hour,
	}
}

func (builder PlatformClientBuilder) WithPlatformType(platformType types.PlatformType) PlatformClientBuilder {
	builder.platformType = &platformType
	return builder
}

func (builder PlatformClientBuilder) WithCacheSettings(numItems int64, maxSizeInBytes int64, maxItemSizeInBytes int64, ttl time.Duration) PlatformClientBuilder {
	builder.cacheNumItems = numItems
	builder.cacheMaxSizeInBytes = maxSizeInBytes
	builder.cacheMaxItemSizeInBytes = maxItemSizeInBytes
	builder.cacheItemTTL = ttl
	return builder
}

// WithAllCaches enables all Kubernetes resource caches including Certificate, ConfigMap, Namespace, Route, Secret, Service, HttpRoute, and GrpcRoute caches.
// Use carefully, HTTPRoute and GRPCRoute watching requires special RBAC permissions.
func (builder PlatformClientBuilder) WithAllCaches() PlatformClientBuilder {
	for _, cacheName := range cache.BasicCaches {
		builder.caches[cacheName] = struct{}{}
	}
	for _, cacheName := range cache.GatewayApiRoutesCaches {
		builder.caches[cacheName] = struct{}{}
	}

	return builder
}

// WithBasicCaches enables the basic Kubernetes resource caches including Certificate, ConfigMap, Namespace, Route, Secret, and Service caches.
func (builder PlatformClientBuilder) WithBasicCaches() PlatformClientBuilder {
	for _, cacheName := range cache.BasicCaches {
		builder.caches[cacheName] = struct{}{}
	}
	return builder
}

// WithGatewayApiRoutesCaches enables the Gateway API route caches including HttpRoute and GrpcRoute caches.
// Use carefully, HTTPRoute and GRPCRoute watching requires special RBAC permissions.
func (builder PlatformClientBuilder) WithGatewayApiRoutesCaches() PlatformClientBuilder {
	for _, cacheName := range cache.GatewayApiRoutesCaches {
		builder.caches[cacheName] = struct{}{}
	}
	return builder
}

func (builder PlatformClientBuilder) WithRouteCache() PlatformClientBuilder {
	builder.caches[cache.RouteCache] = struct{}{}
	return builder
}

func (builder PlatformClientBuilder) WithSecretCache() PlatformClientBuilder {
	builder.caches[cache.SecretCache] = struct{}{}
	return builder
}

func (builder PlatformClientBuilder) WithConfigMapCache() PlatformClientBuilder {
	builder.caches[cache.ConfigMapCache] = struct{}{}
	return builder
}

func (builder PlatformClientBuilder) WithNamespaceCache() PlatformClientBuilder {
	builder.caches[cache.NamespaceCache] = struct{}{}
	return builder
}

func (builder PlatformClientBuilder) WithServiceCache() PlatformClientBuilder {
	builder.caches[cache.ServiceCache] = struct{}{}
	return builder
}

func (builder PlatformClientBuilder) WithClients(kubernetesApi *backend.KubernetesApi, openshiftApi *backend.OpenshiftApi) PlatformClientBuilder {
	builder.kubernetesApi = kubernetesApi
	builder.openshiftApi = openshiftApi
	return builder
}

func (builder PlatformClientBuilder) WithNamespace(namespace string) PlatformClientBuilder {
	builder.namespace = &namespace
	return builder
}

func (builder PlatformClientBuilder) WithRolloutParallelism(parallelism int) PlatformClientBuilder {
	builder.rolloutParallelism = &parallelism
	return builder
}

func (builder PlatformClientBuilder) WithWatchExecutor(executor watch.Executor) PlatformClientBuilder {
	builder.watchExecutor = executor
	return builder
}

func (builder PlatformClientBuilder) WithWatchClientTimeout(watchClientTimeout time.Duration) PlatformClientBuilder {
	if watchClientTimeout <= 0 {
		panic("watchClientTimeout must be greater than 0")
	}
	builder.watchClientTimeout = &watchClientTimeout
	return builder
}

func (builder PlatformClientBuilder) WithRolloutExecutor(rolloutExecutor exec.RolloutExecutor) PlatformClientBuilder {
	builder.rolloutExecutor = rolloutExecutor
	return builder
}

func (builder PlatformClientBuilder) WithConsul(consulEnabled bool, consulUrl string, consulToken ...string) PlatformClientBuilder {
	builder.consulEnabled = &consulEnabled
	builder.consulUrl = &consulUrl
	if len(consulToken) > 0 {
		builder.consulToken = &consulToken[0]
	} else {
		s := ""
		builder.consulToken = &s
	}
	return builder
}

func (builder PlatformClientBuilder) Build() (PlatformService, error) {
	return createPlatformService(builder)
}
