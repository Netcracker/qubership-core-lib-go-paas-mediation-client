package service

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	cm "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	bgMonitor "github.com/netcracker/qubership-core-lib-go-bg-state-monitor/v2"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/exec"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/openshiftV3"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	"github.com/netcracker/qubership-core-lib-go-rest-utils/v2/consul-propertysource"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	appsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8s "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

var (
	logger              logging.Logger
	isLocal             bool
	consulRetries       = 20
	consulRetryDuration = 5 * time.Second
)

func init() {
	logger = logging.GetLogger("pm-service")
	if flag.Lookup("local") == nil {
		flag.BoolVar(&isLocal, "local", false, "flag for local run")
	}
	//klog.InitFlags(nil)
}

func initFlag() bool {
	return flag.Lookup("local").Value.(flag.Getter).Get().(bool)
}

func createPlatformService(builder PlatformClientBuilder) (PlatformService, error) {
	flag.Parse()
	if vFlag := flag.Lookup("v"); vFlag != nil {
		flag.Set("v", vFlag.Value.String())
	}
	isLocal = initFlag()
	var platformType types.PlatformType
	if builder.platformType == nil {
		platformType = types.PlatformType(strings.ToUpper(configloader.GetKoanf().MustString("paas.platform")))
	} else {
		platformType = *builder.platformType
	}
	logger.Info("Paas platform=%s, local flag is %t", platformType, isLocal)
	var k8sClientErr error
	var kubernetesClient *backend.KubernetesApi
	if builder.kubernetesApi == nil {
		if isLocal {
			kubernetesClient, k8sClientErr = initLocalClient()
		} else {
			kubernetesClient, k8sClientErr = initInClusterClient()
		}
	} else {
		kubernetesClient = builder.kubernetesApi
	}
	if k8sClientErr != nil {
		return nil, k8sClientErr
	}
	var namespace string
	if builder.namespace == nil {
		namespace = configloader.GetOrDefaultString("microservice.namespace", "")
	} else {
		namespace = *builder.namespace
	}
	var rolloutExecutor exec.RolloutExecutor
	var rolloutParallelism int
	if builder.rolloutParallelism == nil {
		rolloutParallelism = 32
	} else {
		rolloutParallelism = *builder.rolloutParallelism
	}
	if builder.rolloutExecutor == nil {
		rolloutExecutor = exec.NewFixedRolloutExecutor(rolloutParallelism, rolloutParallelism*10)
	}
	var consulEnabled bool
	var consulUrl string
	var consulToken string
	if builder.consulEnabled == nil {
		consulEnabled = configloader.GetKoanf().Bool("consul.enabled")
		if consulEnabled {
			consulUrl = configloader.GetKoanf().String("consul.url")
			consulToken = configloader.GetKoanf().String("consul.token")
		}
	} else {
		consulEnabled = *builder.consulEnabled
		consulUrl = *builder.consulUrl
		consulToken = *builder.consulToken
	}
	bg2Enabled := atomic.Bool{}
	// determine if BlueGreen 2 is enabled in namespace
	if consulEnabled {
		ctx := context.Background()
		var consulTokenSupplier func(ctx context.Context) (string, error)
		if consulToken != "" {
			logger.DebugC(ctx, "Using provided consul token")
			consulTokenSupplier = func(ctx context.Context) (string, error) { return consulToken, nil }
		} else {
			logger.DebugC(ctx, "Using consul M2M client to acquire consul token")
			consulClient := consul.NewClient(consul.ClientConfig{
				Address:   consulUrl,
				Namespace: namespace,
				Ctx:       ctx,
			})
			var err error
			retries := consulRetries
			for {
				err = consulClient.Login()
				retries--
				if err != nil && retries >= 0 {
					duration := consulRetryDuration
					logger.WarnC(ctx, "Failed to login to consul: '%s' retrying after: %s, retries left: %d", err.Error(), duration.String(), retries)
					// retry because probably key-manager has not configured consul yet
					time.Sleep(duration)
				} else {
					break
				}
			}
			if err != nil {
				return nil, fmt.Errorf("failed to start consulClient: %w", err)
			}
			consulTokenSupplier = func(ctx context.Context) (string, error) { return consulClient.SecretId(), nil }
		}
		// initiate bg state publisher to listen for events about BG from Consul
		statePublisher, err := bgMonitor.NewPublisher(ctx, consulUrl, namespace, consulTokenSupplier)
		if err != nil {
			return nil, fmt.Errorf("failed to start BlueGreenStatePublisher: %w", err)
		}
		statePublisher.Subscribe(ctx, func(state bgMonitor.BlueGreenState) {
			enabled := state.Sibling != nil
			old := bg2Enabled.Swap(enabled)
			if old != enabled {
				logger.InfoC(ctx, "BlueGreen mode enabled: %t", enabled)
			}
		})
	}
	bg2EnabledFunc := func() bool {
		return bg2Enabled.Load()
	}
	var watchClientTimeout time.Duration
	if builder.watchClientTimeout != nil {
		watchClientTimeout = *builder.watchClientTimeout
	} else {
		watchClientTimeout = time.Minute
	}
	resourcesCache, err := initCaches(builder.caches, builder.cacheNumItems, builder.cacheMaxSizeInBytes, builder.cacheMaxItemSizeInBytes, builder.cacheItemTTL)
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewKubernetesClientBuilder().
		WithNamespace(namespace).
		WithClient(kubernetesClient).
		WithRolloutExecutor(rolloutExecutor).
		WithWatchExecutor(builder.watchExecutor).
		WithWatchClientTimeout(watchClientTimeout).
		WithBadResources(builder.badResources).
		WithCache(resourcesCache).
		WithBG2Enabled(bg2EnabledFunc).
		Build()

	if err != nil {
		return nil, err
	}
	if platformType == types.Kubernetes {
		return kubeClient, nil
	}
	return initOpenshiftClient(kubeClient, builder.openshiftApi, isLocal)
}

func initOpenshiftClient(kubernetesClient *kubernetes.Kubernetes,
	openshiftBackendClient *backend.OpenshiftApi,
	isLocal bool) (PlatformService, error) {
	logger.Info("Start init openshift client")
	if openshiftBackendClient == nil {
		config, err := openshiftV3.GetKubeConfig(isLocal)
		if err != nil {
			return nil, err
		}
		routeV1Client, err := routev1client.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		projectV1Client, err := projectv1client.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		appsV1Client, err := appsv1client.NewForConfig(config)
		if err != nil {
			return nil, err
		}
		openshiftBackendClient = &backend.OpenshiftApi{
			RouteV1Interface:   routeV1Client,
			ProjectV1Interface: projectV1Client,
			AppsV1Interface:    appsV1Client,
		}
	}
	openshiftClient := openshiftV3.NewOpenshiftV3Client(
		openshiftBackendClient.RouteV1Interface,
		openshiftBackendClient.ProjectV1Interface,
		openshiftBackendClient.AppsV1Interface,
		kubernetesClient)
	return openshiftClient, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func initLocalClient() (*backend.KubernetesApi, error) {
	defaultKubeConfigPath := filepath.Join(homeDir(), ".kube", "config")
	kubeConfigPath := configloader.GetOrDefaultString("kubeconfig", defaultKubeConfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, err
	}
	// create the kubernetes client
	kubernetesClient, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// create the cert-manager client
	certmanagerClient, err := cm.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// create the gateway api client
	gatewayApiClient, err := gatewayclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &backend.KubernetesApi{KubernetesInterface: kubernetesClient, CertmanagerInterface: certmanagerClient, GatewayInterface: gatewayApiClient}, nil
}

func initInClusterClient() (*backend.KubernetesApi, error) {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	// create the kubernetes client
	kubernetesClient, err := k8s.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// create the cert-manager client
	certmanagerClient, err := cm.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	// create the gateway api client
	gatewayApiClient, err := gatewayclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &backend.KubernetesApi{KubernetesInterface: kubernetesClient, CertmanagerInterface: certmanagerClient, GatewayInterface: gatewayApiClient}, nil
}

func initCaches(caches map[cache.CacheName]struct{}, numItems int64, maxSizeInBytes int64, maxItemSizeInBytes int64, ttl time.Duration) (*cache.ResourcesCache, error) {
	if len(caches) == 0 {
		logger.Info("Cache is disabled")
		return &cache.ResourcesCache{}, nil
	}
	logger.Info("Initiating caches with numItems: %v, maxSizeInBytes: %v, maxItemSizeInBytes: %v", numItems,
		resource.NewQuantity(maxSizeInBytes, resource.BinarySI).String(), resource.NewQuantity(maxItemSizeInBytes, resource.BinarySI).String())

	var cacheTypes []cache.CacheName
	for cacheType := range caches {
		cacheTypes = append(cacheTypes, cacheType)
	}
	return cache.NewResourcesCache(numItems, maxSizeInBytes, maxItemSizeInBytes, ttl, cacheTypes...)
}
