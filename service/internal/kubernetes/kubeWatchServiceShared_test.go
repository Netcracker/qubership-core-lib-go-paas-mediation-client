package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/baseproviders"
	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/baseproviders/xrequestid"
	"github.com/netcracker/qubership-core-lib-go/v3/context-propagation/ctxmanager"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"github.com/stretchr/testify/require"
	v1core "k8s.io/api/core/v1"
	networkingV1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	fakeWatch "k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func initTestConfigLoader() {
	ctxmanager.Register(baseproviders.Get())
	//configloader.Init(&configloader.PropertySource{Provider: configloader.AsPropertyProvider(confmap.Provider(
	//	map[string]any{"log.level": "info"}, "."))})
	loggerSharedWatchHandler.SetLevel(logging.LvlDebug)
}

func TestMultipleClients(t *testing.T) {
	assertions := require.New(t)
	initTestConfigLoader()

	service := createSimpleService()

	watchersMap := map[int]watchReturnFunc{}

	amount := 100
	for i := 0; i < amount; i++ {
		watchExecutor := &testWatcher{channel: make(chan fakeWatch.Event, 1)}
		watchExecutor.channel <- watch.Event{Type: "ADDED", Object: service}
		watchersMap[i] = func() (fakeWatch.Interface, error) {
			return watchExecutor, nil
		}
	}
	watchExecutor := &testWatchExecutor{mutex: &sync.Mutex{}, watchers: watchersMap}

	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}

	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: watchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(watchExecutor, watchTimeout,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache(cache.ServiceCache)}

	handlers := make(chan *pmWatch.Handler, amount)
	for i := 0; i < amount; i++ {
		go func() {
			watchHandler, err := kubeClient.WatchServices(context.Background(), testNamespace1, filter.Meta{})
			assertions.Nil(err)
			handlers <- watchHandler
		}()
	}
	verifyWatchChannel := func(id int, handlers chan *pmWatch.Handler, wg *sync.WaitGroup) {
		timer := time.NewTimer(watchTimeout)
		select {
		case handler := <-handlers:
			timer.Stop()
			timer2 := time.NewTimer(watchTimeout)
			select {
			case watchEvent, ok := <-handler.Channel:
				timer2.Stop()
				assertions.True(ok, "channel %d is closed, but expected to be opened", id)
				logger.Info("received event: Type: %s, object: %+v", watchEvent.Type, *watchEvent.Object.(*entity.Service))
				assertions.Equal("ADDED", watchEvent.Type)
				assertions.Equal(*watchEvent.Object.(*entity.Service), *entity.NewService(service))
				wg.Done()
			case <-timer2.C:
				assertions.Fail("timed out to wait for event from channel", "channel id: %d", id)
			}
		case <-timer.C:
			assertions.Fail("timed out ot wait for handler for channel", "channel id: %d", id)
		}
	}
	wg := &sync.WaitGroup{}
	wg.Add(amount)
	for i := 0; i < amount; i++ {
		id := i + 1
		go verifyWatchChannel(id, handlers, wg)
	}
	assertions.True(waitWG(watchTimeout, wg))
	assertions.Equal(amount, kubeClient.WatchHandlers.Services.SharedNamespaceHandlersMap[testNamespace1].clientCount)
}

func TestOldClientsDoNotReceiveAlreadyProcessedEvents(t *testing.T) {
	assertions := require.New(t)
	initTestConfigLoader()

	watchExecutor1 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
	watchExecutor2 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
	watchExecutor3 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}

	watchExecutor := &testWatchExecutor{mutex: &sync.Mutex{}, watchers: map[int]watchReturnFunc{
		0: func() (fakeWatch.Interface, error) { return watchExecutor1, nil },
		1: func() (fakeWatch.Interface, error) { return watchExecutor2, nil },
		2: func() (fakeWatch.Interface, error) { return watchExecutor3, nil },
	}}

	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: watchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(watchExecutor, watchTimeout,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache(cache.ConfigMapCache)}

	// 1. watch#1 established
	watchHandler1, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)
	// 2. configMap1 created
	configMap1Added := createConfigMap("test-1", 1, "1", map[string]string{"1": "added"})
	watchExecutor1.channel <- watch.Event{Type: "ADDED", Object: configMap1Added}
	// 3. watch#1 receives ADDED:configMap1
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})
	// 4. configMap1 updated
	configMap1Modified := createConfigMap("test-1", 2, "2", map[string]string{"1": "modified"})
	watchExecutor1.channel <- watch.Event{Type: "MODIFIED", Object: configMap1Modified}
	// 6. watch#1 receives MODIFIED:configMap1
	// 7. watch#1 receives ADDED:configMap2
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "MODIFIED", Object: entity.NewConfigMap(configMap1Modified)})

	// 8. watch#2 established
	watchHandler2, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)
	assertions.True(verifyChannelClosed(watchExecutor1.channel))

	// configMap1 updated
	configMap1Modified = createConfigMap("test-1", 3, "3", map[string]string{"1": "modified-#2"})
	// configMap2 created
	configMap2Added := createConfigMap("test-2", 1, "4", map[string]string{"2": "added"})
	// watch#1 receives ADDED:configMap2
	watchExecutor2.channel <- watch.Event{Type: "ADDED", Object: configMap1Modified}
	watchExecutor2.channel <- watch.Event{Type: "ADDED", Object: configMap2Added}
	// 9 watch#1 receives no events
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "MODIFIED", Object: entity.NewConfigMap(configMap1Modified)})
	verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Modified)})
	// 10. watch#2 receives ADDED:configMap1
	// 11. watch#2 receives ADDED:configMap2
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2Added)})
	verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2Added)})
	// 12. configMap2 updated
	configMap2Modified := createConfigMap("test-2", 2, "5", map[string]string{"2": "modified"})
	watchExecutor2.channel <- watch.Event{Type: "MODIFIED", Object: configMap2Modified}
	// 13. watch#1 receives MODIFIED:configMap2
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "MODIFIED", Object: entity.NewConfigMap(configMap2Modified)})
	// 14. watch#2 receives MODIFIED:configMap2
	verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "MODIFIED", Object: entity.NewConfigMap(configMap2Modified)})
	// 15. configMap1 deleted
	configMap1Deleted := configMap1Modified
	configMap1Deleted.ResourceVersion = "6"
	watchExecutor2.channel <- watch.Event{Type: "DELETED", Object: configMap1Deleted}
	// 15. watch#1 receives DELETED:configMap1
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "DELETED", Object: entity.NewConfigMap(configMap1Deleted)})
	// 16. watch#2 receives DELETED:configMap1
	verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "DELETED", Object: entity.NewConfigMap(configMap1Deleted)})

	// 17. watch#3 established
	watchHandler3, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{WatchBookmark: true})
	assertions.Nil(err)
	assertions.True(verifyChannelClosed(watchExecutor2.channel))
	// 18. configMap3 created
	configMap3Added := createConfigMap("test-3", 1, "7", map[string]string{"3": "added"})
	watchExecutor3.channel <- watch.Event{Type: "ADDED", Object: configMap2Modified}
	watchExecutor3.channel <- watch.Event{Type: "ADDED", Object: configMap3Added}
	// 19. watch#3 receives ADDED:configMap2
	verifyWatch(assertions, watchHandler3, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2Modified)})
	// 20. watch#1 receives ADDED:configMap3
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap3Added)})
	// 21. watch#2 receives ADDED:configMap3
	verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap3Added)})
	// 22. watch#3 receives ADDED:configMap3
	verifyWatch(assertions, watchHandler3, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap3Added)})

	currentHandler := kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1].currentNotifier
	assertions.Equal(1, len(currentHandler.clientsMap))
	// 23. BOOKMARK event received
	configMapBookmark := createConfigMap("", 0, "7", map[string]string{})
	watchExecutor3.channel <- watch.Event{Type: "BOOKMARK", Object: configMapBookmark}
	timer := time.NewTimer(watchTimeout)
	// 24. current resourceVersionAwareClientNotifier drains clients from previous ones
	for {
		select {
		case <-timer.C:
			assertions.Fail("failed to wait resourceVersionAwareClientNotifier to drain clients on BOOKMARK event")
		default:
			if len(currentHandler.clientsMap) == 3 && currentHandler.previousNotifier == nil {
				timer.Stop()
				verifyWatch(assertions, watchHandler1)
				verifyWatch(assertions, watchHandler2)
				verifyWatch(assertions, watchHandler3, pmWatch.ApiEvent{Type: "BOOKMARK", Object: entity.NewConfigMap(configMapBookmark)})
				return
			}
		}
	}
}

func TestClientStopWatching(t *testing.T) {
	assertions := require.New(t)
	initTestConfigLoader()

	for _, scenario := range []string{"Cancel", "StopWatching"} {
		t.Run(fmt.Sprint(scenario), func(t *testing.T) {
			watchExecutor1 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
			watchExecutor2 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
			watchExecutor3 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}

			watchExecutor := &testWatchExecutor{mutex: &sync.Mutex{}, watchers: map[int]watchReturnFunc{
				0: func() (fakeWatch.Interface, error) { return watchExecutor1, nil },
				1: func() (fakeWatch.Interface, error) { return watchExecutor2, nil },
				2: func() (fakeWatch.Interface, error) { return watchExecutor3, nil },
			}}
			clientset := &kubernetes.Clientset{}
			cert_client := &certClient.Clientset{}
			kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: watchExecutor, namespace: testNamespace1,
				WatchHandlers: NewSharedWatchEventHandlers(watchExecutor, time.Second,
					clientset.CoreV1().RESTClient(),
					cert_client.CertmanagerV1().RESTClient(),
					clientset.NetworkingV1().RESTClient(),
					clientset.ExtensionsV1beta1().RESTClient()),
				Cache: cache.NewTestResourcesCache(cache.ConfigMapCache)}

			watchProxy := func(namespace string, metaFilter filter.Meta,
				WatchConfigMaps func(ctx context.Context, namespace string, metaFilter filter.Meta) (*pmWatch.Handler, error)) (handler *pmWatch.Handler, cancelFunc func(), err error) {
				parentCtx, _ := ctxmanager.SetContextObject(context.Background(), xrequestid.X_REQUEST_ID_COTEXT_NAME, xrequestid.NewXRequestIdContextObject(""))
				if scenario == "Cancel" {
					var ctx context.Context
					ctx, cancelFunc = context.WithCancel(parentCtx)
					handler, err = WatchConfigMaps(ctx, namespace, metaFilter)
				} else if scenario == "StopWatching" {
					handler, err = WatchConfigMaps(parentCtx, namespace, metaFilter)
					cancelFunc = handler.StopWatching
				} else {
					panic("unsupported scenario: " + scenario)
				}
				return
			}
			watchHandler1, cancelFunc1, err := watchProxy(testNamespace1, filter.Meta{}, kubeClient.WatchConfigMaps)
			assertions.Nil(err)
			configMap1Added := createConfigMap("test-1", 1, "1", map[string]string{"1": "added"})
			watchExecutor1.channel <- watch.Event{Type: "ADDED", Object: configMap1Added}
			verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})

			watchHandler2, cancelFunc2, err := watchProxy(testNamespace1, filter.Meta{}, kubeClient.WatchConfigMaps)
			assertions.Nil(err)
			cancelFunc1()
			watchExecutor2.channel <- watch.Event{Type: "ADDED", Object: configMap1Added}
			verifyWatch(assertions, watchHandler1)
			verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})
			cancelFunc2()
			validateCleanedUp := func() {
				timer := time.NewTimer(watchTimeout)
				sharedNamespaceHandler := kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1]
				for {
					select {
					case <-timer.C:
						assertions.Fail("failed to wait for sharedNamespaceHandler to be cleaned up")
					default:
						if sharedNamespaceHandler.currentNotifier.clientsCount() == 0 {
							return
						}
					}
				}
			}
			validateCleanedUp()

			watchHandler3, cancelFunc3, err := watchProxy(testNamespace1, filter.Meta{}, kubeClient.WatchConfigMaps)
			assertions.Nil(err)
			watchExecutor3.channel <- watch.Event{Type: "ADDED", Object: configMap1Added}
			verifyWatch(assertions, watchHandler1)
			verifyWatch(assertions, watchHandler2)
			verifyWatch(assertions, watchHandler3, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})
			cancelFunc3()
			validateCleanedUp()
		})
	}
}

func TestClientNotResponding(t *testing.T) {
	assertions := require.New(t)
	initTestConfigLoader()

	watchExecutor1 := &testWatcher{channel: make(chan fakeWatch.Event, 1)}
	watchExecutor2 := &testWatcher{channel: make(chan fakeWatch.Event, 1)}

	watchExecutor := &testWatchExecutor{mutex: &sync.Mutex{}, watchers: map[int]watchReturnFunc{
		0: func() (fakeWatch.Interface, error) { return watchExecutor1, nil },
		1: func() (fakeWatch.Interface, error) { return watchExecutor2, nil }},
	}

	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	clientTimeout := 500 * time.Millisecond
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: watchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(watchExecutor, clientTimeout,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache(cache.ConfigMapCache)}

	watchHandler1, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)
	watchHandler2, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)

	assertions.Equal(2, kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1].currentNotifier.clientsCount())

	configMap1 := createConfigMap("test-1", 1, "1", map[string]string{"1": "added"})
	configMap2 := createConfigMap("test-2", 1, "1", map[string]string{"2": "added"})
	watchExecutor2.channel <- watch.Event{Type: "ADDED", Object: configMap1}
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1)})
	watchExecutor2.channel <- watch.Event{Type: "ADDED", Object: configMap2}
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2)})

	time.Sleep(clientTimeout * 11 / 10)

	assertions.True(verifyChannelOpened(watchHandler1.Channel))
	assertions.True(verifyChannelClosed(watchHandler2.Channel))
	assertions.Equal(1, kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1].currentNotifier.clientsCount())
}

func TestServerChannelClosed(t *testing.T) {
	assertions := require.New(t)
	initTestConfigLoader()

	watchExecutor1 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
	watchExecutor2 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
	watchExecutor3 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
	watchExecutor4 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}

	watchExecutor := &testWatchExecutor{mutex: &sync.Mutex{}, watchers: map[int]watchReturnFunc{
		0: func() (fakeWatch.Interface, error) { return watchExecutor1, nil },
		1: func() (fakeWatch.Interface, error) { return watchExecutor2, nil },
		2: func() (fakeWatch.Interface, error) { return watchExecutor3, nil },
		3: func() (fakeWatch.Interface, error) { return nil, fmt.Errorf("test error") },
		4: func() (fakeWatch.Interface, error) { return watchExecutor4, nil },
	}}

	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: watchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(watchExecutor, watchTimeout,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache(cache.ConfigMapCache)}

	watchHandler1, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)
	watchHandler2, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)
	watchHandler3, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)

	configMap1Added := createConfigMap("test-1", 1, "1", map[string]string{"1": "added"})
	watchExecutor3.channel <- watch.Event{Type: "ADDED", Object: configMap1Added}
	verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})
	verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})
	verifyWatch(assertions, watchHandler3, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})

	assertions.True(verifyChannelClosed(watchExecutor1.channel))
	assertions.True(verifyChannelClosed(watchExecutor2.channel))
	assertions.True(verifyChannelOpened(watchExecutor3.channel))

	assertions.Equal(3, kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1].currentNotifier.clientsCount())

	// imitate k8s closing watch connection
	close(watchExecutor3.channel)

	assertions.True(verifyChannelClosed(watchHandler1.Channel))
	assertions.True(verifyChannelClosed(watchHandler2.Channel))
	assertions.True(verifyChannelClosed(watchHandler3.Channel))

	assertions.Equal(0, kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1].currentNotifier.clientsCount())

	watchHandler4, err := kubeClient.WatchConfigMaps(context.Background(), testNamespace1, filter.Meta{})
	assertions.Nil(err)

	watchExecutor4.channel <- watch.Event{Type: "ADDED", Object: configMap1Added}
	verifyWatch(assertions, watchHandler4, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1Added)})
	assertions.True(verifyChannelOpened(watchExecutor4.channel))
	assertions.Equal(1, kubeClient.WatchHandlers.ConfigMaps.SharedNamespaceHandlersMap[testNamespace1].currentNotifier.clientsCount())
}

func TestFilterByLabelsOrAnnotations(t *testing.T) {
	assertions := require.New(t)
	initTestConfigLoader()

	for _, scenario := range []string{"labels", "annotations", "labels+annotations"} {
		t.Run(fmt.Sprint(scenario), func(t *testing.T) {
			watchExecutor1 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
			watchExecutor2 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
			watchExecutor3 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}
			watchExecutor4 := &testWatcher{channel: make(chan fakeWatch.Event, 5)}

			watchExecutor := &testWatchExecutor{mutex: &sync.Mutex{}, watchers: map[int]watchReturnFunc{
				0: func() (fakeWatch.Interface, error) { return watchExecutor1, nil },
				1: func() (fakeWatch.Interface, error) { return watchExecutor2, nil },
				2: func() (fakeWatch.Interface, error) { return watchExecutor3, nil },
				3: func() (fakeWatch.Interface, error) { return watchExecutor4, nil },
			}}

			clientset := &kubernetes.Clientset{}
			cert_client := &certClient.Clientset{}
			kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: watchExecutor, namespace: testNamespace1,
				WatchHandlers: NewSharedWatchEventHandlers(watchExecutor, watchTimeout,
					clientset.CoreV1().RESTClient(),
					cert_client.CertmanagerV1().RESTClient(),
					clientset.NetworkingV1().RESTClient(),
					clientset.ExtensionsV1beta1().RESTClient()),
				Cache: cache.NewTestResourcesCache(cache.ConfigMapCache)}
			setLabelsOrAnnotations := func(labelsOrAnnotations map[string]string) filter.Meta {
				metaFilter := filter.Meta{}
				if scenario == "labels" {
					metaFilter.Labels = labelsOrAnnotations
				} else if scenario == "annotations" {
					metaFilter.Annotations = labelsOrAnnotations
				} else if scenario == "labels+annotations" {
					metaFilter.Labels = labelsOrAnnotations
					metaFilter.Annotations = labelsOrAnnotations
				} else {
					panic("invalid scenario: " + scenario)
				}
				return metaFilter
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			watchHandler1, err := kubeClient.WatchConfigMaps(ctx, testNamespace1,
				setLabelsOrAnnotations(map[string]string{"key-1": "value-1"}))
			assertions.Nil(err)
			watchHandler2, err := kubeClient.WatchConfigMaps(ctx, testNamespace1,
				setLabelsOrAnnotations(map[string]string{"key-1": "*"}))
			assertions.Nil(err)
			watchHandler3, err := kubeClient.WatchConfigMaps(ctx, testNamespace1,
				setLabelsOrAnnotations(map[string]string{"key-2": "value-2"}))
			watchHandler4, err := kubeClient.WatchConfigMaps(ctx, testNamespace1,
				setLabelsOrAnnotations(nil))
			assertions.Nil(err)

			var createFunc func(name string, generation int64,
				resourceVersion string, data map[string]string, annotations map[string]string) *v1core.ConfigMap

			if scenario == "labels" {
				createFunc = createConfigMapWithLabels
			} else if scenario == "annotations" {
				createFunc = createConfigMapWithAnnotations
			} else if scenario == "labels+annotations" {
				createFunc = func(name string, generation int64, resourceVersion string, data map[string]string, labelsOrAnnotations map[string]string) *v1core.ConfigMap {
					configMap := createConfigMap(name, generation, resourceVersion, data)
					configMap.Labels = labelsOrAnnotations
					configMap.Annotations = labelsOrAnnotations
					return configMap
				}
			} else {
				panic("invalid scenario: " + scenario)
			}

			configMap1_1 := createFunc("test-1-1", 1, "1",
				map[string]string{}, map[string]string{"key-1": "value-1"})
			configMap1_2 := createFunc("test-1-2", 1, "2",
				map[string]string{}, map[string]string{"key-1": "value-2"})
			configMap2_1 := createFunc("test-2-1", 1, "3",
				map[string]string{}, map[string]string{"key-2": "value-1"})
			configMap2_2 := createFunc("test-2-2", 1, "4",
				map[string]string{}, map[string]string{"key-2": "value-2"})

			assertions.True(verifyChannelClosed(watchExecutor1.channel))
			assertions.True(verifyChannelClosed(watchExecutor2.channel))
			assertions.True(verifyChannelClosed(watchExecutor3.channel))
			assertions.True(verifyChannelOpened(watchExecutor4.channel))

			watchExecutor4.channel <- watch.Event{Type: "ADDED", Object: configMap1_1}
			watchExecutor4.channel <- watch.Event{Type: "ADDED", Object: configMap1_2}
			watchExecutor4.channel <- watch.Event{Type: "ADDED", Object: configMap2_1}
			watchExecutor4.channel <- watch.Event{Type: "ADDED", Object: configMap2_2}

			verifyWatch(assertions, watchHandler1, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1_1)})
			verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1_1)})
			verifyWatch(assertions, watchHandler4, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1_1)})

			verifyWatch(assertions, watchHandler2, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1_2)})
			verifyWatch(assertions, watchHandler4,
				pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap1_2)},
				pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2_1)})

			verifyWatch(assertions, watchHandler3, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2_2)})
			verifyWatch(assertions, watchHandler4, pmWatch.ApiEvent{Type: "ADDED", Object: entity.NewConfigMap(configMap2_2)})
		})
	}
}

func verifyChannelClosed[T any](channel <-chan T) bool {
	return verifyChannel(channel, false)
}

func verifyChannelOpened[T any](channel <-chan T) bool {
	return verifyChannel(channel, true)
}

func verifyChannel[T any](channel <-chan T, expectOpened bool) bool {
	timer := time.NewTimer(watchTimeout)
	for {
		select {
		case _, opened := <-channel:
			if opened == expectOpened {
				return true
			}
		case <-timer.C:
			return false
		default:
			if expectOpened {
				return true
			}
		}
	}
}

func verifyWatch(assertions *require.Assertions, handler *pmWatch.Handler, expectedEvents ...pmWatch.ApiEvent) {
	for _, expectedEvent := range expectedEvents {
		timer := time.NewTimer(watchTimeout)
		select {
		case watchEvent, ok := <-handler.Channel:
			assertions.True(ok, "channel expected to be opened, but is closed")
			assertions.Equal(expectedEvent.Type, watchEvent.Type)
			assertions.Equal(*expectedEvent.Object.(*entity.ConfigMap), *watchEvent.Object.(*entity.ConfigMap))
		case <-timer.C:
			assertions.Fail("timed out to wait for event")
		}
	}
	select {
	case watchEvent := <-handler.Channel:
		assertions.Fail("unexpected event received", fmt.Sprintf("Type: %s, Object: %+v", watchEvent.Type, watchEvent.Object))
	default:
		return
	}
}

func createConfigMapWithLabels(name string, generation int64,
	resourceVersion string, data map[string]string, labels map[string]string) *v1core.ConfigMap {
	configMap := createConfigMap(name, generation, resourceVersion, data)
	configMap.Labels = labels
	return configMap
}

func createConfigMapWithAnnotations(name string, generation int64,
	resourceVersion string, data map[string]string, annotations map[string]string) *v1core.ConfigMap {
	configMap := createConfigMap(name, generation, resourceVersion, data)
	configMap.Annotations = annotations
	return configMap
}

func createConfigMap(name string, generation int64, resourceVersion string, data map[string]string) *v1core.ConfigMap {
	configMap := &v1core.ConfigMap{}
	configMap.Name = name
	configMap.Namespace = testNamespace1
	configMap.ResourceVersion = resourceVersion
	configMap.Generation = generation
	configMap.Data = data
	return configMap
}

func createSecret(name string, generation int64, resourceVersion string, data map[string][]byte) *v1core.Secret {
	secret := &v1core.Secret{}
	secret.Name = name
	secret.Namespace = testNamespace1
	secret.ResourceVersion = resourceVersion
	secret.Generation = generation
	secret.Data = data
	return secret
}

func createService(name string, generation int64, resourceVersion string) *v1core.Service {
	service := &v1core.Service{}
	service.Name = name
	service.Namespace = testNamespace1
	service.ResourceVersion = resourceVersion
	service.Generation = generation
	service.Spec = v1core.ServiceSpec{}
	return service
}

func createIngress(name string, generation int64, resourceVersion string) *networkingV1.Ingress {
	ingress := &networkingV1.Ingress{}
	ingress.Name = name
	ingress.Namespace = testNamespace1
	ingress.ResourceVersion = resourceVersion
	ingress.Generation = generation
	ingress.Spec = networkingV1.IngressSpec{}
	return ingress
}

func createHttpRoute(name string, generation int64, resourceVersion string) *gatewayv1.HTTPRoute {
	httpRoute := &gatewayv1.HTTPRoute{}
	httpRoute.Name = name
	httpRoute.Namespace = testNamespace1
	httpRoute.ResourceVersion = resourceVersion
	httpRoute.Generation = generation
	httpRoute.Spec = gatewayv1.HTTPRouteSpec{}
	return httpRoute
}

func createGrpcRoute(name string, generation int64, resourceVersion string) *gatewayv1.GRPCRoute {
	grpcRoute := &gatewayv1.GRPCRoute{}
	grpcRoute.Name = name
	grpcRoute.Namespace = testNamespace1
	grpcRoute.ResourceVersion = resourceVersion
	grpcRoute.Generation = generation
	grpcRoute.Spec = gatewayv1.GRPCRouteSpec{}
	return grpcRoute
}

func createCertificate(name string, generation int64, resourceVersion string) *cmv1.Certificate {
	certificate := &cmv1.Certificate{}
	certificate.Name = name
	certificate.Namespace = testNamespace1
	certificate.ResourceVersion = resourceVersion
	certificate.Generation = generation
	certificate.Spec = cmv1.CertificateSpec{}
	return certificate
}

type testWatchExecutor struct {
	newAttempts int
	watchers    map[int]watchReturnFunc
	mutex       *sync.Mutex
}

func (this *testWatchExecutor) CreateWatchRequest(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
	return &rest.Request{}
}

func (this *testWatchExecutor) Watch(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	watcherFunc := this.watchers[this.newAttempts]
	this.newAttempts++
	return watcherFunc()
}

type watchReturnFunc func() (watch.Interface, error)

type testWatcher struct {
	channel chan watch.Event
}

func (this *testWatcher) Stop() {
	close(this.channel)
}

func (this *testWatcher) ResultChan() <-chan watch.Event {
	return this.channel
}
