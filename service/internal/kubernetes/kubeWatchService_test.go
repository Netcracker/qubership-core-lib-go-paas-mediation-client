package kubernetes

import (
	"sync"
	"testing"
	"time"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/golang/mock/gomock"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	. "github.com/smarty/assertions"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
	v1core "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const watchTimeout = 300 * time.Second

type fakeWatchExecutor struct {
	requestedResources string
	requestedOptions   *v1.ListOptions
	fakeWatcher        *watch.FakeWatcher
}

func newFakeWatchExecutor() *fakeWatchExecutor {
	return &fakeWatchExecutor{
		fakeWatcher: watch.NewFake(),
	}
}

func (e *fakeWatchExecutor) CreateWatchRequest(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
	e.requestedResources = resource.String()
	e.requestedOptions = options
	return &rest.Request{}
}

func (e *fakeWatchExecutor) Watch(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
	return e.fakeWatcher, nil
}

func TestWatchRoutesAddedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: "test-ns",
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}
	watchHandler, err := kubeClient.WatchRoutes(context.Background(), "test-namespace", filter.Meta{})
	r.Nil(err)
	fakeWatchExecutor.fakeWatcher.Add(createSimpleIngress())
	verifyWatchHandler(r, func() {
		watchEvent := <-watchHandler.Channel
		logger.Info("Get event %s", watchEvent.Type)
		r.Equal("ADDED", watchEvent.Type)
		ingress := entity.RouteFromIngress(createSimpleIngress())
		r.True(So(watchEvent.Object, ShouldResemble, ingress))
	})
}

func TestWatchRoutesAddedEventUseNetworkingV1(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: "test-ns",
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache(), UseNetworkingV1Ingress: true}
	watchHandler, err := kubeClient.WatchRoutes(context.Background(), "test-namespace", filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		go fakeWatchExecutor.fakeWatcher.Add(createSimpleIngressNetworkingV1())
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("ADDED", watchEvent.Type)
			ingress := entity.RouteFromIngressNetworkingV1(createSimpleIngressNetworkingV1())
			r.True(So(watchEvent.Object, ShouldResemble, ingress))
			break
		}
	})
}

//func TestWatchRoutesBadAddedEvent(t *testing.T) {
//	r := require.New(t)
//	fakeWatchExecutor := newFakeWatchExecutor()
//	badResources := BadResources{
//		Routes: NewBadRoutes(),
//	}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: fakeWatchExecutor, namespace: "test-namespace",
//		Cache: cache.NewTestResourcesCache(), BadResources: &badResources}
//	_, err := kubeClient.WatchRoutes(context.Background(), "test-namespace", filter.Meta{})
//	r.Nil(err)
//	verifyWatchHandler(r, func() {
//		go fakeWatchExecutor.fakeWatcher.Add(createBadSimpleIngress())
//		for {
//			if len(badResources.Routes.ToSliceMap()) == 1 &&
//				len(badResources.Routes.ToSliceMap()["test-namespace"]) == 1 &&
//				badResources.Routes.Exists("test-namespace", "kube-ingress") {
//				break
//			}
//		}
//	})
//}

//func TestWatchRoutesBadAddedEventUseNetworkingV1(t *testing.T) {
//	r := require.New(t)
//	fakeWatchExecutor := newFakeWatchExecutor()
//	badResources := BadResources{
//		Routes: NewBadRoutes(),
//	}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}},
//		WatchExecutor: fakeWatchExecutor, namespace: "test-namespace",
//		Cache: cache.NewTestResourcesCache(), BadResources: &badResources, UseNetworkingV1Ingress: true}
//	_, err := kubeClient.WatchRoutes(context.Background(), "test-namespace", filter.Meta{})
//	r.Nil(err)
//	verifyWatchHandler(r, func() {
//		go fakeWatchExecutor.fakeWatcher.Add(createBadSimpleIngressNetworkingV1())
//		for {
//			if len(badResources.Routes.ToSliceMap()) == 1 &&
//				len(badResources.Routes.ToSliceMap()["test-namespace"]) == 1 &&
//				badResources.Routes.Exists("test-namespace", "kube-ingress") {
//				break
//			}
//		}
//	})
//}

func TestWatchServicesAddedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}
	watchHandler, err := kubeClient.WatchServices(context.Background(), testNamespace1, filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		service := createSimpleService()
		go fakeWatchExecutor.fakeWatcher.Add(service)
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("ADDED", watchEvent.Type)
			srv := entity.NewService(service)
			r.True(So(watchEvent.Object, ShouldResemble, srv))
			break
		}
	})
}

func TestWatchConfigMapsAddedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: "test-namespace",
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}

	watchHandler, err := kubeClient.WatchConfigMaps(context.Background(), "test-namespace", filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		go fakeWatchExecutor.fakeWatcher.Add(createSimpleConfigMap())
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("ADDED", watchEvent.Type)
			configMap := entity.NewConfigMap(createSimpleConfigMap())
			r.True(So(watchEvent.Object, ShouldResemble, configMap))
			break
		}
	})
}

func TestWatchGatewayHTTPRoutesAddedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}
	kubeClient.WatchHandlers.WithHTTPRouteV1(fakeWatchExecutor, time.Second, nil)
	watchHandler, err := kubeClient.WatchGatewayHTTPRoutes(context.Background(), testNamespace1, filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		route := createHttpRoute("test-http-route", 1, "1")
		go fakeWatchExecutor.fakeWatcher.Add(route)
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("ADDED", watchEvent.Type)
			expected := entity.RouteFromHTTPRoute(route)
			r.True(So(watchEvent.Object, ShouldResemble, expected))
			break
		}
	})
}

func TestWatchGatewayGRPCRoutesAddedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}
	kubeClient.WatchHandlers.WithGRPCRouteV1(fakeWatchExecutor, time.Second, nil)
	watchHandler, err := kubeClient.WatchGatewayGRPCRoutes(context.Background(), testNamespace1, filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		route := createGrpcRoute("test-grpc-route", 1, "1")
		go fakeWatchExecutor.fakeWatcher.Add(route)
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("ADDED", watchEvent.Type)
			expected := entity.RouteFromGRPCRoute(route)
			r.True(So(watchEvent.Object, ShouldResemble, expected))
			break
		}
	})
}

func TestWatchGatewayHTTPRoutesDeletedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}
	kubeClient.WatchHandlers.WithHTTPRouteV1(fakeWatchExecutor, time.Second, nil)
	watchHandler, err := kubeClient.WatchGatewayHTTPRoutes(context.Background(), testNamespace1, filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		route := createHttpRoute("test-http-route", 2, "2")
		go fakeWatchExecutor.fakeWatcher.Delete(route)
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("DELETED", watchEvent.Type)
			expected := entity.RouteFromHTTPRoute(route)
			r.True(So(watchEvent.Object, ShouldResemble, expected))
			break
		}
	})
}

func TestWatchGatewayGRPCRoutesDeletedEvent(t *testing.T) {
	r := require.New(t)
	fakeWatchExecutor := newFakeWatchExecutor()
	clientset := &kubernetes.Clientset{}
	cert_client := &certClient.Clientset{}
	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client}, WatchExecutor: fakeWatchExecutor, namespace: testNamespace1,
		WatchHandlers: NewSharedWatchEventHandlers(fakeWatchExecutor, time.Second,
			clientset.CoreV1().RESTClient(),
			cert_client.CertmanagerV1().RESTClient(),
			clientset.NetworkingV1().RESTClient(),
			clientset.ExtensionsV1beta1().RESTClient()),
		Cache: cache.NewTestResourcesCache()}
	kubeClient.WatchHandlers.WithGRPCRouteV1(fakeWatchExecutor, time.Second, nil)
	watchHandler, err := kubeClient.WatchGatewayGRPCRoutes(context.Background(), testNamespace1, filter.Meta{})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		route := createGrpcRoute("test-grpc-route", 2, "2")
		go fakeWatchExecutor.fakeWatcher.Delete(route)
		for watchEvent := range watchHandler.Channel {
			logger.Info("Get event %s", watchEvent.Type)
			r.Equal("DELETED", watchEvent.Type)
			expected := entity.RouteFromGRPCRoute(route)
			r.True(So(watchEvent.Object, ShouldResemble, expected))
			break
		}
	})
}

func TestWatchResourcesWithFilter(t *testing.T) {
	r := require.New(t)
	testWatchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))

	var labelsMap = make(map[string]string)
	labelsMap["test-label-1"] = "value-1"

	var wg sync.WaitGroup
	wg.Add(2)
	var actualListOptions *v1.ListOptions
	testWatchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
			actualListOptions = options
			defer wg.Done()
			return &rest.Request{}
		})
	testWatchExecutor.EXPECT().Watch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
			defer wg.Done()
			return watch.NewFake(), nil
		})
	watchEventHandler := NewRestWatchHandler("test-namespace", types.ConfigMaps, nil, testWatchExecutor, entity.NewConfigMap)
	_, err := watchEventHandler.Watch(context.Background(), filter.Meta{Labels: labelsMap})
	r.Nil(err)
	verifyWatchHandler(r, func() {
		wg.Wait()
		var expectedLabelSelector = labels.Set(labelsMap).String()
		r.Equal(expectedLabelSelector, actualListOptions.LabelSelector, "unexpected options' label selector")
	})
}

//func TestReconnectWithErrorOnServerSideClosure(t *testing.T) {
//	assertions := require.New(t)
//	watchRequestCount := 0
//	watchRequest1 := &rest.Request{}
//	watchRequest2 := &rest.Request{}
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
//		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
//			watchRequestCount++
//			if watchRequestCount == 1 {
//				return watchRequest1
//			} else {
//				return watchRequest2
//			}
//		}).MaxTimes(2)
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest1).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		fakeWatcher.Stop()
//		return fakeWatcher, nil
//	})
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest2).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		return nil, errors.New("test error")
//	})
//	badResources := BadResources{Routes: NewBadRoutes()}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: watchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//
//	watchHandler, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil,
//		func(out watch.Event, apiEvent chan pmWatch.ApiEvent) bool {
//			apiEvent <- pmWatch.ApiEvent{
//				Type:   string(out.Type),
//				Object: out.Object,
//			}
//			return false
//		})
//	assertions.Nil(err)
//	verifyWatchHandler(assertions, func() {
//		event := <-watchHandler.Channel
//		assertions.Equal(pmWatch.Error, event.Type)
//		assertions.Equal("test error", event.Object)
//		_, opened := <-watchHandler.Channel
//		assertions.False(opened)
//	})
//}
//
//func TestReconnectWithErrorOnServerSideClosureAfterBookmark(t *testing.T) {
//	assertions := require.New(t)
//	watchRequestCount := 0
//	watchRequest1 := &rest.Request{}
//	watchRequest2 := &rest.Request{}
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
//		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
//			watchRequestCount++
//			if watchRequestCount == 1 {
//				return watchRequest1
//			} else {
//				assertions.Equal("", options.ResourceVersion)
//				return watchRequest2
//			}
//		}).MaxTimes(2)
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest1).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		go func() {
//
//			fakeWatcher.Action(watch.Bookmark, &v1core.ConfigMap{ObjectMeta: v1.ObjectMeta{ResourceVersion: "1"}})
//			fakeWatcher.Action(watch.Error, &v1.Status{ListMeta: v1.ListMeta{},
//				Status: "Failure", Message: "too old resource version: 1 (1306951380)"})
//		}()
//		return fakeWatcher, nil
//	})
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest2).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		go func() {
//			fakeWatcher.Action(watch.Added, &v1core.ConfigMap{ObjectMeta: v1.ObjectMeta{ResourceVersion: "1306951380"}})
//		}()
//		return fakeWatcher, nil
//	})
//	// set reconnect wait interval to short
//	reconnectWaitInterval = time.Millisecond
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: watchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &BadResources{Routes: NewBadRoutes()}}
//
//	watchHandler, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil,
//		func(out watch.Event, apiEvent chan pmWatch.ApiEvent) bool {
//			apiEvent <- pmWatch.ApiEvent{
//				Type:   string(out.Type),
//				Object: out.Object,
//			}
//			return false
//		})
//	assertions.Nil(err)
//	verifyWatchHandler(assertions, func() {
//		event := <-watchHandler.Channel
//		assertions.Equal(string(watch.Added), event.Type)
//		assertions.Equal("1306951380", event.Object.(*v1core.ConfigMap).ResourceVersion)
//	})
//}
//
//func TestReconnectWithSuccessOnServerSideClosure(t *testing.T) {
//	assertions := require.New(t)
//	watchRequestCount := 0
//	watchRequest1 := &rest.Request{}
//	watchRequest2 := &rest.Request{}
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
//		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
//			watchRequestCount++
//			if watchRequestCount == 1 {
//				return watchRequest1
//			} else {
//				return watchRequest2
//			}
//		}).MaxTimes(2)
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest1).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		fakeWatcher.Stop()
//		return fakeWatcher, nil
//	})
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest2).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		go fakeWatcher.Add(nil)
//		return fakeWatcher, nil
//	})
//	badResources := BadResources{Routes: NewBadRoutes()}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: watchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//
//	watchHandler, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil,
//		func(out watch.Event, apiEvent chan pmWatch.ApiEvent) bool {
//			apiEvent <- pmWatch.ApiEvent{
//				Type:   string(out.Type),
//				Object: out.Object,
//			}
//			return false
//		})
//	assertions.Nil(err)
//	verifyWatchHandler(assertions, func() {
//		event := <-watchHandler.Channel
//		assertions.Equal(string(watch.Added), event.Type)
//	})
//}
//
//func TestReconnectWithSuccessOnServerSideErrorEvent(t *testing.T) {
//	assertions := require.New(t)
//	watchRequestCount := 0
//	watchRequest1 := &rest.Request{}
//	watchRequest2 := &rest.Request{}
//	var watcher1 *watch.FakeWatcher
//	var watcher2 *watch.FakeWatcher
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
//		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
//			watchRequestCount++
//			if watchRequestCount == 1 {
//				return watchRequest1
//			} else {
//				return watchRequest2
//			}
//		}).MaxTimes(2)
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest1).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		watcher1 = watch.NewFakeWithChanSize(1, false)
//		watcher1.Error(nil)
//		return watcher1, nil
//	})
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest2).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		watcher2 = watch.NewFakeWithChanSize(1, false)
//		watcher2.Add(nil)
//		return watcher2, nil
//	})
//	badResources := BadResources{Routes: NewBadRoutes()}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: watchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//
//	watchHandler, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil,
//		func(out watch.Event, apiEvent chan pmWatch.ApiEvent) bool {
//			apiEvent <- pmWatch.ApiEvent{
//				Type:   string(out.Type),
//				Object: out.Object,
//			}
//			return false
//		})
//	assertions.Nil(err)
//	verifyWatchHandler(assertions, func() {
//		event := <-watchHandler.Channel
//		assertions.Equal(string(watch.Added), event.Type)
//		assertions.NotNil(watcher1)
//		assertions.True(watcher1.IsStopped())
//		assertions.NotNil(watcher2)
//		assertions.False(watcher2.IsStopped())
//	})
//}
//
//func TestReconnectAfterBookmarkEvent(t *testing.T) {
//	assertions := require.New(t)
//	watchRequestCount := 0
//	watchRequest1 := &rest.Request{}
//	watchRequest2 := &rest.Request{}
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	var actualResourceVersion string
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).
//		DoAndReturn(func(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
//			watchRequestCount++
//			if watchRequestCount == 1 {
//				return watchRequest1
//			} else {
//				actualResourceVersion = options.ResourceVersion
//				return watchRequest2
//			}
//		}).MaxTimes(2)
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest1).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		go func() {
//			fakeWatcher.Action(watch.Bookmark, &v1core.ConfigMap{
//				ObjectMeta: v1.ObjectMeta{ResourceVersion: "testResourceVersion"},
//			})
//			fakeWatcher.Stop()
//		}()
//		return fakeWatcher, nil
//	})
//	watchExecutor.EXPECT().Watch(gomock.Any(), watchRequest2).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		go fakeWatcher.Add(nil)
//		return fakeWatcher, nil
//	})
//
//	badResources := BadResources{Routes: NewBadRoutes()}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: watchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//
//	watchHandler, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil,
//		func(out watch.Event, apiEvent chan pmWatch.ApiEvent) bool {
//			apiEvent <- pmWatch.ApiEvent{
//				Type:   string(out.Type),
//				Object: out.Object,
//			}
//			return false
//		})
//	assertions.Nil(err)
//	verifyWatchHandler(assertions, func() {
//		event := <-watchHandler.Channel
//		assertions.Equal(string(watch.Added), event.Type)
//		assertions.Equal("testResourceVersion", actualResourceVersion)
//	})
//}
//
//func TestClosureInWatchDelegate(t *testing.T) {
//	assertions := require.New(t)
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).Return(&rest.Request{})
//	watchExecutor.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		fakeWatcher := watch.NewFake()
//		go fakeWatcher.Action(watch.Added, nil)
//		return fakeWatcher, nil
//	})
//	badResources := BadResources{Routes: NewBadRoutes()}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: watchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//
//	watchHandler, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil,
//		func(out watch.Event, apiEvent chan pmWatch.ApiEvent) bool {
//			apiEvent <- pmWatch.ApiEvent{
//				Type:   string(out.Type),
//				Object: out.Object,
//			}
//			return true
//		})
//	assertions.Nil(err)
//	verifyWatchHandler(assertions, func() {
//		event := <-watchHandler.Channel
//		assertions.Equal(string(watch.Added), event.Type)
//		_, opened := <-watchHandler.Channel
//		assertions.False(opened)
//	})
//}
//
//func TestWatchResourcesAbortOnClientSideCancel(t *testing.T) {
//	assertions := require.New(t)
//	fakeWatchExecutor := newFakeWatchExecutor()
//	badResources := BadResources{
//		Routes: NewBadRoutes(),
//	}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: fakeWatchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//	ctx, cancel := context.WithCancel(context.Background())
//	cancel()
//	watchHandler, err := kubeClient.WatchResources(ctx, "test-namespace", types.ConfigMaps, filter.Meta{}, nil, nil)
//	assertions.Nil(err)
//	_, opened := <-watchHandler.Channel
//	assertions.False(opened)
//}
//
//func TestWatchResourcesValidationError(t *testing.T) {
//	assertions := require.New(t)
//	fakeWatchExecutor := newFakeWatchExecutor()
//	badResources := BadResources{
//		Routes: NewBadRoutes(),
//	}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}}, WatchExecutor: fakeWatchExecutor, namespace: "test-ns",
//		Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//	_, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps,
//		filter.Meta{Annotations: map[string]string{"test": "test"}}, nil, nil)
//	assertions.NotNil(err)
//	assertions.Equal("watch API does not support filtering by annotations, use labels instead", err.Error())
//}
//
//func TestWatchExecutorReturnsNilWatcher(t *testing.T) {
//	assertions := require.New(t)
//	watchExecutor := pmWatch.NewMockExecutor(gomock.NewController(t))
//	watchExecutor.EXPECT().CreateWatchRequest(gomock.Any(), gomock.Any(), gomock.Any()).Return(&rest.Request{})
//	watchExecutor.EXPECT().Watch(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error) {
//		return nil, nil
//	})
//	badResources := BadResources{
//		Routes: NewBadRoutes(),
//	}
//	kubeClient := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}},
//		WatchExecutor: watchExecutor, namespace: "test-ns", Cache:  cache.NewTestResourcesCache(), BadResources: &badResources}
//	_, err := kubeClient.WatchResources(context.Background(), "test-namespace", types.ConfigMaps, filter.Meta{}, nil, nil)
//	assertions.NotNil(err)
//	assertions.Equal("watcher is nil", err.Error())
//}
//
//func TestDefaultWatchExecutor_CreateWatchRequest(t *testing.T) {
//	// stupid test for SonarQube
//	r := require.New(t)
//	executor := &RestWatchExecutor{}
//	restClient := &fake.RESTClient{}
//	restClient.Req = &http.Request{}
//	restClient.Err = nil
//	req := executor.CreateWatchRequest(restClient, "", &v1.ListOptions{})
//	r.NotNil(req)
//}

// todo delete this func, use waitWG instead
func verifyWatchHandler(r *require.Assertions, verifyFunc func()) {
	done := make(chan bool)
	go func() {
		verifyFunc()
		done <- true
	}()
	timeout := time.After(watchTimeout)
	select {
	case <-timeout:
		r.Fail("timed out waiting for event to be received")
	case result := <-done:
		logger.Info("verification done=%t", result)
	}
}

func waitWG(timeout time.Duration, groups ...*sync.WaitGroup) bool {
	finalWg := &sync.WaitGroup{}
	finalWg.Add(len(groups))
	c := make(chan struct{})
	for _, wg := range groups {
		go func(wg *sync.WaitGroup) {
			wg.Wait()
			finalWg.Done()
		}(wg)
	}
	go func() {
		finalWg.Wait()
		c <- struct{}{}
	}()

	timer := time.NewTimer(timeout)
	select {
	case <-c:
		timer.Stop()
		return true
	case <-timer.C:
		return false
	}
}

func createSimpleIngress() *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: v1.ObjectMeta{Name: "kube-ingress", Namespace: "namespace"},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{
				Host: "test-host",
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path:    "/",
							Backend: v1beta1.IngressBackend{ServiceName: "target-service-name"}},
						}}}}},
			IngressClassName: &testIngressClassName},
	}
}

func createSimpleIngressNetworkingV1() *networkingV1.Ingress {
	pathType := networkingV1.PathTypeExact
	return &networkingV1.Ingress{
		ObjectMeta: v1.ObjectMeta{Name: "kube-ingress", Namespace: "namespace"},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{{Host: "test-host",
				IngressRuleValue: networkingV1.IngressRuleValue{HTTP: &networkingV1.HTTPIngressRuleValue{Paths: []networkingV1.HTTPIngressPath{{
					Path:     "/",
					PathType: &pathType,
					Backend: networkingV1.IngressBackend{Service: &networkingV1.IngressServiceBackend{
						Name: "target-service-name",
						Port: networkingV1.ServiceBackendPort{
							Number: 8080,
						},
					}}}}}}}},
			IngressClassName: &testIngressClassName}}
}

func createBadSimpleIngress() *v1beta1.Ingress {
	ingress := v1beta1.Ingress{}
	ingress.Name = "kube-ingress"
	ingress.Namespace = "namespace"
	ingress.Spec.Rules = []v1beta1.IngressRule{{Host: "test-host"}}
	return &ingress
}

func createBadSimpleIngressNetworkingV1() *networkingV1.Ingress {
	ingress := networkingV1.Ingress{}
	ingress.Name = "kube-ingress"
	ingress.Namespace = "namespace"
	ingress.Spec.Rules = []networkingV1.IngressRule{{Host: "test-host"}}
	return &ingress
}

func createSimpleService() *v1core.Service {
	srv := &v1core.Service{}
	srv.Name = "kube-service"
	srv.Namespace = testNamespace1
	return srv
}

func createSimpleConfigMap() *v1core.ConfigMap {
	configMap := v1core.ConfigMap{}
	configMap.Name = "kube-config-map"
	configMap.Namespace = "test-namespace"
	return &configMap
}
