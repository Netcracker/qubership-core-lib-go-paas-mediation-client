package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/exec"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"golang.org/x/mod/semver"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	appsv1_client "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	extensionV1Client = "extensionsV1Client"
	appsV1Client      = "appsV1Client"
)

var (
	logger        logging.Logger
	isLocal       bool
	resourceAlias = map[string]string{
		"routes": "ingresses",
	}
	extensionResources = []string{"ingresses"}
)

func init() {
	logger = logging.GetLogger("kubernetes_service")
}

type Kubernetes struct {
	initCacheOnce          sync.Once
	client                 *backend.KubernetesApi
	WatchExecutor          pmWatch.Executor
	WatchHandlers          *SharedWatchHandlers
	namespace              string
	Cache                  *cache.ResourcesCache
	CacheAdapters          *CacheAdapters
	BadResources           *BadResources // todo remove it in the nex major release
	UseNetworkingV1Ingress bool          // todo remove it in the nex major release if we don't support k8s 1.22 anymore
	RolloutExecutor        exec.RolloutExecutor
	BG2Enabled             func() bool
}

// todo delete this in next major release!
type BadResources struct {
	Routes *BadRoutes
}

type BadRoutes struct {
	routesMap map[string]map[string]struct{}
	lock      *sync.RWMutex
}

func NewBadRoutes() *BadRoutes {
	return &BadRoutes{
		routesMap: make(map[string]map[string]struct{}),
		lock:      &sync.RWMutex{},
	}
}

func (r *BadRoutes) Add(namespace, routeName string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.routesMap[namespace] == nil {
		r.routesMap[namespace] = make(map[string]struct{})
	}
	r.routesMap[namespace][routeName] = struct{}{}
}

func (r *BadRoutes) Delete(namespace, routeName string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.routesMap[namespace] != nil {
		delete(r.routesMap[namespace], routeName)
		if len(r.routesMap[namespace]) == 0 {
			delete(r.routesMap, namespace)
		}
	}
}

func (r *BadRoutes) Exists(namespace, routeName string) (found bool) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.routesMap[namespace] != nil {
		_, found = r.routesMap[namespace][routeName]
	}
	return
}

func (r *BadRoutes) ToSliceMap() (result map[string][]string) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	result = make(map[string][]string)
	for namespace, routesMap := range r.routesMap {
		routes := make([]string, 0)
		for name := range routesMap {
			routes = append(routes, name)
		}
		result[namespace] = routes
	}
	return
}

type KubernetesClientBuilder struct {
	namespace          string
	client             *backend.KubernetesApi
	watchExecutor      pmWatch.Executor
	watchClientTimeout time.Duration
	cache              *cache.ResourcesCache
	badResources       *BadResources
	rolloutExecutor    exec.RolloutExecutor
	bg2Enabled         func() bool
}

func NewKubernetesClientBuilder() *KubernetesClientBuilder {
	return &KubernetesClientBuilder{}
}

func (b *KubernetesClientBuilder) WithNamespace(namespace string) *KubernetesClientBuilder {
	b.namespace = namespace
	return b
}

func (b *KubernetesClientBuilder) WithClient(client *backend.KubernetesApi) *KubernetesClientBuilder {
	b.client = client
	return b
}

func (b *KubernetesClientBuilder) WithWatchExecutor(executor pmWatch.Executor) *KubernetesClientBuilder {
	b.watchExecutor = executor
	return b
}

func (b *KubernetesClientBuilder) WithWatchClientTimeout(watchClientTimeout time.Duration) *KubernetesClientBuilder {
	b.watchClientTimeout = watchClientTimeout
	return b
}

func (b *KubernetesClientBuilder) WithCache(cache *cache.ResourcesCache) *KubernetesClientBuilder {
	b.cache = cache
	return b
}

func (b *KubernetesClientBuilder) WithBadResources(badResources *BadResources) *KubernetesClientBuilder {
	b.badResources = badResources
	return b
}

func (b *KubernetesClientBuilder) WithRolloutExecutor(rolloutExecutor exec.RolloutExecutor) *KubernetesClientBuilder {
	b.rolloutExecutor = rolloutExecutor
	return b
}

func (b *KubernetesClientBuilder) WithBG2Enabled(enabled func() bool) *KubernetesClientBuilder {
	b.bg2Enabled = enabled
	return b
}

func (b *KubernetesClientBuilder) Build() (*Kubernetes, error) {
	if b.namespace == "" {
		return nil, fmt.Errorf("namespace cannot be empty")
	}
	if b.client == nil {
		return nil, fmt.Errorf("client cannot be nil")
	}
	if b.watchExecutor == nil {
		b.watchExecutor = &DefaultWatchExecutor{}
	}
	if b.badResources == nil {
		badResources := BadResources{
			Routes: NewBadRoutes(),
		}
		b.badResources = &badResources
	}
	if b.rolloutExecutor == nil {
		b.rolloutExecutor = exec.NewFixedPool[*entity.DeploymentRollout](32, 32*10)
	}
	if b.cache == nil {
		b.cache = &cache.ResourcesCache{} // set empty cache
	}
	version, err := b.client.KubernetesInterface.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	kubernetesVersion := version.GitVersion
	logger.Info("Kubernetes version=%s", kubernetesVersion)
	useNetworkingV1Ingress := semver.Compare(kubernetesVersion, "v1.22") >= 0

	watchEventHandlers := NewSharedWatchEventHandlers(b.watchExecutor, b.watchClientTimeout,
		b.client.CoreV1().RESTClient(),
		b.client.CertmanagerV1().RESTClient(),
		b.client.NetworkingV1().RESTClient(),
		b.client.ExtensionsV1beta1().RESTClient(),
	)
	if hasKindGatewayApi("HTTPRoute", b.client.KubernetesInterface.Discovery()) {
		err := gatewayv1.Install(scheme.Scheme)
		if err != nil {
			return nil, err
		}
		watchEventHandlers.WithHTTPRouteV1(b.watchExecutor, b.watchClientTimeout, b.client.GatewayV1().RESTClient())
	}

	if hasKindGatewayApi("GRPCRoute", b.client.KubernetesInterface.Discovery()) {
		err := gatewayv1.Install(scheme.Scheme)
		if err != nil {
			return nil, err
		}
		watchEventHandlers.WithGRPCRouteV1(b.watchExecutor, b.watchClientTimeout, b.client.GatewayV1().RESTClient())
	}

	ctx := context.Background() //todo make all parent functions to pass context, this context will be able to cancel everything inside Kubernetes srv when it's no longer needed
	cacheAdapters, err := NewCacheAdapters(ctx, b.namespace, b.cache, watchEventHandlers)
	if err != nil {
		return nil, err
	}
	return &Kubernetes{
		initCacheOnce:          sync.Once{},
		client:                 b.client,
		WatchExecutor:          b.watchExecutor,
		WatchHandlers:          watchEventHandlers,
		namespace:              b.namespace,
		Cache:                  b.cache,
		CacheAdapters:          cacheAdapters,
		BadResources:           b.badResources,
		UseNetworkingV1Ingress: useNetworkingV1Ingress,
		RolloutExecutor:        b.rolloutExecutor,
		BG2Enabled:             b.bg2Enabled,
	}, nil
}

func NewTestKubernetesClient(namespace string, client *backend.KubernetesApi) (*Kubernetes, error) {
	return NewKubernetesClientBuilder().WithClient(client).WithNamespace(namespace).Build()
}

func (kube *Kubernetes) getKubernetesClientVersion(namespace string) (string, error) {
	_, errApps := kube.getAppsV1Client().Deployments(namespace).List(context.Background(), v1.ListOptions{})
	if errApps == nil {
		return appsV1Client, nil
	}
	_, errExtension := kube.getExtensionsV1Client().Deployments(namespace).List(context.Background(), v1.ListOptions{})
	if errExtension == nil {
		return extensionV1Client, nil
	}
	return "", errors.New("can't get correct kubernetes client version")
}

func (kube *Kubernetes) GetCoreV1Client() corev1.CoreV1Interface {
	return kube.client.CoreV1()
}

func (kube *Kubernetes) getExtensionsV1Client() extensionsv1beta1.ExtensionsV1beta1Interface {
	return kube.client.ExtensionsV1beta1()
}

func (kube *Kubernetes) getNetworkingV1Client() networkingv1.NetworkingV1Interface {
	return kube.client.NetworkingV1()
}

func (kube *Kubernetes) getAppsV1Client() appsv1_client.AppsV1Interface {
	return kube.client.AppsV1()
}

func (kube *Kubernetes) getCertmanagerV1Client() certmanagerv1.CertmanagerV1Interface {
	return kube.client.CertmanagerV1()
}

func (kube *Kubernetes) GetCurrentNamespace() string {
	return kube.namespace
}

func (kube *Kubernetes) GetKubernetesVersion() (string, error) {
	version, err := kube.client.KubernetesInterface.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return version.GitVersion, err
}

func hasKindGatewayApi(kind string, discovery discovery.DiscoveryInterface) bool {
	resources, err := discovery.ServerResourcesForGroupVersion("gateway.networking.k8s.io/v1")
	if err != nil {
		logger.Errorf("can't get server group version: %v", err)
		return false
	}
	for _, res := range resources.APIResources {
		if res.Kind == kind {
			return true
		}
	}

	logger.Infof("gateway.networking.k8s.io for '%s' kind is not supported; skip it", kind)

	return false
}
