package kubernetes

import (
	"context"
	"fmt"
	"sync"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var (
	loggerSharedWatchHandler logging.Logger
	loggerWatchNotifier      logging.Logger
)

func init() {
	loggerSharedWatchHandler = logging.GetLogger("shared_watch_handler")
	loggerWatchNotifier = logging.GetLogger("watch_notifier")
}

type SharedWatchHandlers struct {
	Certificates          *SharedWatchHandler[*cmv1.Certificate, entity.Certificate]
	ConfigMaps            *SharedWatchHandler[*v1.ConfigMap, entity.ConfigMap]
	Services              *SharedWatchHandler[*v1.Service, entity.Service]
	ServiceAccounts       *SharedWatchHandler[*v1.ServiceAccount, entity.ServiceAccount]
	Secrets               *SharedWatchHandler[*v1.Secret, entity.Secret]
	IngressesNetworkingV1 *SharedWatchHandler[*networkingV1.Ingress, entity.Route]
	IngressesV1Beta1      *SharedWatchHandler[*v1beta1.Ingress, entity.Route]
	HTTPRouteV1           *SharedWatchHandler[*gatewayv1.HTTPRoute, entity.HttpRoute]
	GRPCRouteV1           *SharedWatchHandler[*gatewayv1.GRPCRoute, entity.GrpcRoute]
}

type SharedWatchHandler[F runtime.Object, T entity.HasMetadata] struct {
	kind                       types.PaasResourceType
	HandlerProvider            func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[F, T]
	SharedNamespaceHandlersMap map[string]*sharedNamespaceWatchHandler[F, T]
	ClientTimeout              time.Duration
	Mutex                      *sync.Mutex
}

type sharedNamespaceWatchHandler[F runtime.Object, T entity.HasMetadata] struct {
	namespace           string
	kind                types.PaasResourceType
	handlerProvider     func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[F, T]
	currentEventHandler *pmWatch.Handler
	currentCancelFunc   context.CancelFunc
	currentNotifier     *resourceVersionAwareClientNotifier[F, T]
	clientTimeout       time.Duration
	clientCount         int
	mutex               *sync.Mutex
	logger              logging.Logger
}

type sharedHandlerHolder struct {
	id      string
	ctx     context.Context
	handler chan pmWatch.ApiEvent
	Filter  filter.Meta
}

type resourceVersionAwareClientNotifier[F runtime.Object, T entity.HasMetadata] struct {
	id                string
	namespace         string
	kind              types.PaasResourceType
	previousNotifier  *resourceVersionAwareClientNotifier[F, T]
	clientsMap        map[string]*sharedHandlerHolder
	processedVersions map[namespaceAndName]versionAndType
	clientTimeout     time.Duration
	mutex             *sync.Mutex
	logger            logging.Logger
}

type namespaceAndName struct {
	namespace string
	name      string
}

type versionAndType struct {
	resourceVersion string
	eventType       string
}

func NewSharedWatchEventHandlers(executor pmWatch.Executor,
	clientTimeout time.Duration,
	coreV1 rest.Interface,
	cert rest.Interface,
	networkingV1Client rest.Interface,
	extensionsV1beta1 rest.Interface) *SharedWatchHandlers {
	return &SharedWatchHandlers{
		Certificates: NewSharedWatchEventHandler(types.Certificates, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*cmv1.Certificate, entity.Certificate] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*cmv1.Certificate, entity.Certificate] {
				return NewRestWatchHandler(namespace, kind, cert, executor, entity.NewCertificate)
			})
		}),
		ConfigMaps: NewSharedWatchEventHandler(types.ConfigMaps, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*v1.ConfigMap, entity.ConfigMap] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*v1.ConfigMap, entity.ConfigMap] {
				return NewRestWatchHandler(namespace, kind, coreV1, executor, entity.NewConfigMap)
			})
		}),
		Services: NewSharedWatchEventHandler(types.Services, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*v1.Service, entity.Service] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*v1.Service, entity.Service] {
				return NewRestWatchHandler(namespace, kind, coreV1, executor, entity.NewService)
			})
		}),
		ServiceAccounts: NewSharedWatchEventHandler(types.ServiceAccounts, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*v1.ServiceAccount, entity.ServiceAccount] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*v1.ServiceAccount, entity.ServiceAccount] {
				return NewRestWatchHandler(namespace, kind, coreV1, executor, entity.NewServiceAccount)
			})
		}),
		Secrets: NewSharedWatchEventHandler(types.Secrets, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*v1.Secret, entity.Secret] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*v1.Secret, entity.Secret] {
				return NewRestWatchHandler(namespace, kind, coreV1, executor, entity.NewSecret)
			})
		}),
		IngressesNetworkingV1: NewSharedWatchEventHandler(types.Ingresses, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*networkingV1.Ingress, entity.Route] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*networkingV1.Ingress, entity.Route] {
				return NewRestWatchHandler(namespace, kind, networkingV1Client, executor, entity.RouteFromIngressNetworkingV1)
			})
		}),
		IngressesV1Beta1: NewSharedWatchEventHandler(types.Ingresses, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*v1beta1.Ingress, entity.Route] {
			return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*v1beta1.Ingress, entity.Route] {
				return NewRestWatchHandler(namespace, kind, extensionsV1beta1, executor, entity.RouteFromIngress)
			})
		}),
	}
}

func (h *SharedWatchHandlers) WithHTTPRouteV1(executor pmWatch.Executor, clientTimeout time.Duration, gatewayV1 rest.Interface) {
	h.HTTPRouteV1 = NewSharedWatchEventHandler(types.HTTPRoutes, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*gatewayv1.HTTPRoute, entity.HttpRoute] {
		return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*gatewayv1.HTTPRoute, entity.HttpRoute] {
			return NewRestWatchHandler(namespace, kind, gatewayV1, executor, entity.RouteFromHTTPRoute)
		})
	})
}

func (h *SharedWatchHandlers) WithGRPCRouteV1(executor pmWatch.Executor, clientTimeout time.Duration, gatewayV1 rest.Interface) {
	h.GRPCRouteV1 = NewSharedWatchEventHandler(types.GRPCRoutes, clientTimeout, func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[*gatewayv1.GRPCRoute, entity.GrpcRoute] {
		return newSharedNamespaceWatchHandler(namespace, kind, clientTimeout, func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[*gatewayv1.GRPCRoute, entity.GrpcRoute] {
			return NewRestWatchHandler(namespace, kind, gatewayV1, executor, entity.RouteFromGRPCRoute)
		})
	})
}

func NewSharedWatchEventHandler[F runtime.Object, T entity.HasMetadata](kind types.PaasResourceType, clientTimeout time.Duration, handlerProvider func(namespace string, kind types.PaasResourceType) *sharedNamespaceWatchHandler[F, T]) *SharedWatchHandler[F, T] {
	return &SharedWatchHandler[F, T]{
		kind:                       kind,
		HandlerProvider:            handlerProvider,
		SharedNamespaceHandlersMap: make(map[string]*sharedNamespaceWatchHandler[F, T]),
		ClientTimeout:              clientTimeout,
		Mutex:                      &sync.Mutex{}}
}

func newSharedNamespaceWatchHandler[F runtime.Object, T entity.HasMetadata](namespace string, kind types.PaasResourceType, clientTimeout time.Duration, handlerProvider func(namespace string, kind types.PaasResourceType) *RestWatchEventHandler[F, T]) *sharedNamespaceWatchHandler[F, T] {
	return &sharedNamespaceWatchHandler[F, T]{
		namespace: namespace, kind: kind, clientTimeout: clientTimeout, handlerProvider: handlerProvider,
		clientCount: 0, mutex: &sync.Mutex{}, logger: loggerSharedWatchHandler}
}

func (this *SharedWatchHandler[F, T]) Watch(ctx context.Context, namespace string, metaFilter filter.Meta) (*pmWatch.Handler, error) {
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	var handler *sharedNamespaceWatchHandler[F, T]
	if h, ok := this.SharedNamespaceHandlersMap[namespace]; ok {
		handler = h
	} else {
		handler = this.HandlerProvider(namespace, this.kind)
		this.SharedNamespaceHandlersMap[namespace] = handler
	}
	return handler.Watch(ctx, metaFilter)
}

func (this *sharedNamespaceWatchHandler[F, T]) Watch(ctx context.Context, metaFilter filter.Meta) (*pmWatch.Handler, error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.clientCount++
	id := fmt.Sprintf("%06d", this.clientCount)

	channel := make(chan pmWatch.ApiEvent)
	h := &pmWatch.Handler{
		Channel: channel,
		StopWatching: func() {
			this.delete(ctx, id)
		},
	}
	clientHolder := &sharedHandlerHolder{id: id, ctx: ctx, handler: channel, Filter: metaFilter}
	if this.currentCancelFunc != nil {
		// cancel previous handler's watcher
		this.currentCancelFunc()
	}
	// use new context, because it will be used for the handler shared by multiple clients
	handlerCtx, cancelWatching := context.WithCancel(context.Background())
	this.currentCancelFunc = cancelWatching
	provider := this.handlerProvider(this.namespace, this.kind)
	// need to start up Handler with WatchBookmark=true filter
	var err error
	this.currentEventHandler, err = provider.Watch(handlerCtx, filter.Meta{WatchBookmark: true})
	if err != nil {
		return nil, fmt.Errorf("failed to initiate handler watch handler: %w", err)
	}

	this.currentNotifier = &resourceVersionAwareClientNotifier[F, T]{
		id:                id,
		namespace:         this.namespace,
		kind:              this.kind,
		previousNotifier:  this.currentNotifier,
		clientsMap:        map[string]*sharedHandlerHolder{id: clientHolder},
		processedVersions: map[namespaceAndName]versionAndType{},
		clientTimeout:     this.clientTimeout,
		mutex:             &sync.Mutex{},
		logger:            loggerWatchNotifier,
	}
	go this.processServerChannel(handlerCtx, this.currentEventHandler.Channel, this.currentNotifier)
	// listen for client's context cancellation
	go func(handler *pmWatch.Handler) {
		select {
		case <-ctx.Done():
			handler.StopWatching()
		}
	}(h)
	this.logger.InfoC(ctx, "Added new watch client %s for %s@%s and filter: '%+v'", id, this.kind, this.namespace, metaFilter)
	return h, nil
}

func (this *sharedNamespaceWatchHandler[F, T]) delete(ctx context.Context, id string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.currentNotifier.deleteClient(ctx, id)
	if this.currentNotifier.clientsCount() == 0 {
		// need to stop Handler
		this.logger.InfoC(ctx, "Stopping '%s' because there is no active watch client", this.currentNotifier.String())
		this.currentEventHandler.StopWatching()
	}
}

func (this *sharedNamespaceWatchHandler[F, T]) cleanupOnHandlerClosed(ctx context.Context, handler *resourceVersionAwareClientNotifier[F, T]) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	logger.InfoC(ctx, "(%s) server channel was closed. Closing watch clients, if any", handler.String())
	handler.cleanup(ctx)
}

func (this *sharedNamespaceWatchHandler[F, T]) processServerChannel(ctx context.Context,
	serverChan <-chan pmWatch.ApiEvent, handler *resourceVersionAwareClientNotifier[F, T]) {
	for {
		select {
		case <-ctx.Done():
			// we were substituted by another Handler
			// do not clean up clients, just exit from this listen loop
			this.logger.DebugC(ctx, "Stopping processServerChannel loop for id: '%s' because it was cancelled", handler.id)
			return
		case event, ok := <-serverChan:
			if ok {
				switch obj := event.Object.(type) {
				case *T:
					meta := getMeta(obj)
					this.logger.DebugC(ctx, "Received event from the api server: event: '%s', namespace: '%s', kind: '%s', name: '%s', resourceVersion: '%s'. "+
						"Sending to %d client(s) for acceptance",
						event.Type, meta.Namespace, meta.Kind, meta.Name, meta.ResourceVersion, handler.clientsCount())
					handler.acceptEvent(ctx, &event)
				}
			} else {
				select {
				// one more time check on ctx.Done() after the serverChan was closed
				case <-ctx.Done():
					continue
				default:
					// our rest handler decided to close! need to close all clients, they all have to re-connect
					this.cleanupOnHandlerClosed(ctx, handler)
					return
				}
			}
		}
	}
}

func (this *resourceVersionAwareClientNotifier[F, T]) acceptEvent(ctx context.Context, event *pmWatch.ApiEvent) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.logger.DebugC(ctx, "(%s) received new event of type: '%s', object: '%+v'",
		this.String(), event.Type, event.Object)
	switch obj := event.Object.(type) {
	case *T:
		meta := getMeta(obj)
		switch event.Type {
		case "BOOKMARK":
			this.logger.DebugC(ctx, "(%s). Received Bookmark event with resourceVersion: '%s'. "+
				"Draining clients from previous handlers and updating processedVersions",
				this.String(), meta.ResourceVersion)
			// 1) transfer clients from all PreviousHandler(s) if any to ourselves
			this.clientsMap = this.drainPrevious(ctx)
			// 2) cleanup any DELETED events from our processedVersions
			updatedMap := map[namespaceAndName]versionAndType{}
			for k, v := range this.processedVersions {
				if v.eventType != "DELETED" {
					updatedMap[k] = v
				}
			}
			this.processedVersions = updatedMap
			this.sendToClients(*event, meta.Kind, this.namespace, "", meta.ResourceVersion, func(holder *sharedHandlerHolder) bool {
				return holder.Filter.WatchBookmark
			})
		default:
			if this.previousNotifier != nil {
				if event.Type == "ADDED" && meta.Generation > 1 {
					// for clients processed by previous handler need to change eventType from ADDED to PREVIOUS
					// so the type will be correctly substituted depending on whether it was already handled or not
					this.previousNotifier.acceptEvent(ctx, &pmWatch.ApiEvent{Type: "PREVIOUS", Object: event.Object})
				} else {
					this.previousNotifier.acceptEvent(ctx, event)
				}
			}
			verAndType, ok := this.processedVersions[namespaceAndName{namespace: meta.Namespace, name: meta.Name}]
			if ok && verAndType.resourceVersion == meta.ResourceVersion {
				// skip, already processed
				this.logger.DebugC(ctx, "(%s). Skipping event (already received from previous handler) of type: '%s' "+
					"with object kind: '%s', namespace: '%s', name: '%s', resourceVersion: '%s'",
					this.String(), verAndType.eventType, meta.Kind, meta.Namespace, meta.Name, meta.ResourceVersion)
				return
			} else {
				if event.Type == "PREVIOUS" {
					if verAndType.resourceVersion == "" {
						event.Type = "ADDED"
					} else {
						event.Type = "MODIFIED"
					}
					this.logger.DebugC(ctx, "(%s). Resolved internal PREVIOUS event (we are owned by previous handler) to '%s'"+
						" object kind: '%s', namespace: '%s', name: '%s', resourceVersion: '%s'",
						this.String(), event.Type, meta.Kind, meta.Namespace, meta.Name, meta.ResourceVersion)
				}
				this.processedVersions[namespaceAndName{namespace: meta.Namespace, name: meta.Name}] = versionAndType{
					resourceVersion: meta.ResourceVersion,
					eventType:       event.Type,
				}
				// send event to the clients
				this.sendToClients(*event, meta.Kind, meta.Namespace, meta.Name, meta.ResourceVersion, func(holder *sharedHandlerHolder) bool {
					return acceptMeta(&meta, &holder.Filter)
				})
			}
		}
	}
}

func (this *resourceVersionAwareClientNotifier[F, T]) sendToClients(event pmWatch.ApiEvent, kind, namespace, name, resourceVersion string,
	accept func(*sharedHandlerHolder) bool) {
	for id, client := range this.clientsMap {
		if accept(client) {
			// start timeout timer and try sending event to the client's channel
			timer := time.NewTimer(this.clientTimeout)
			this.logger.DebugC(client.ctx, "(%s). Sending event to the client with id: '%s' "+
				"of type: '%s' with object kind: '%s', namespace: '%s', name: '%s', resourceVersion: '%s'",
				this.String(), client.id, event.Type, kind, namespace, name, resourceVersion)
			select {
			case client.handler <- event:
				timer.Stop()
				this.logger.DebugC(client.ctx, "(%s). Sent event to the client with id: '%s' "+
					"of type: '%s' with object kind: '%s', namespace: '%s', name: '%s', resourceVersion: '%s'",
					this.String(), client.id, event.Type, kind, namespace, name, resourceVersion)
			case <-timer.C:
				// client's channel is full after timeout - close it, if client is alive but slow, it will re-connect
				this.logger.WarnC(client.ctx, "(%s). Failed to send event to the client with id: '%s' after timeout: '%s'."+
					"event type: '%s', object's kind: '%s', namespace: '%s', name: '%s', resourceVersion: '%s'. "+
					"Closing client's channel and removing the client from the notifier.",
					this.String(), client.id, this.clientTimeout.String(), event.Type, kind, namespace, name, resourceVersion)
				close(client.handler)
				delete(this.clientsMap, id)
				return
			}
		} else {
			this.logger.DebugC(client.ctx, "(%s). Skipping (due to filter returnd 'accept=false') event for the client with id: '%s' "+
				"of type: '%s' with object kind: '%s', namespace: '%s', name: '%s', resourceVersion: '%s'",
				this.String(), client.id, event.Type, kind, namespace, name, resourceVersion)
		}
	}
}

func (this *resourceVersionAwareClientNotifier[F, T]) drainPrevious(ctx context.Context) map[string]*sharedHandlerHolder {
	result := map[string]*sharedHandlerHolder{}
	if this.previousNotifier != nil {
		previous := this.previousNotifier.drainPrevious(ctx)
		this.logger.InfoC(ctx, "(%s). Drained %d client(s) from handler: %s", this.String(), len(previous), this.previousNotifier.id)
		for k, v := range previous {
			result[k] = v
		}
		this.previousNotifier = nil
		for k, v := range this.clientsMap {
			result[k] = v
		}
		return result
	} else {
		return this.clientsMap
	}
}

func (this *resourceVersionAwareClientNotifier[F, T]) deleteClient(ctx context.Context, id string) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.previousNotifier != nil {
		this.previousNotifier.deleteClient(ctx, id)
	}

	if _, ok := this.clientsMap[id]; ok {
		delete(this.clientsMap, id)
		this.logger.InfoC(ctx, "(%s). Deleted watch client with id: '%s'", this.String(), id)
	}
}

func (this *resourceVersionAwareClientNotifier[F, T]) cleanup(ctx context.Context) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if this.previousNotifier != nil {
		this.previousNotifier.cleanup(ctx)
	}

	// clean up only clients that belong to us (previousNotifier's clients must be left along)
	if len(this.clientsMap) > 0 {
		this.logger.InfoC(ctx, "(%s). Deleting %d watch client(s)", this.String(), len(this.clientsMap))
		for id, client := range this.clientsMap {
			delete(this.clientsMap, id)
			close(client.handler)
			this.logger.InfoC(ctx, "(%s). Deleted and closed watch client with id: '%s'", this.String(), id)
		}
	}
}

func (this *resourceVersionAwareClientNotifier[F, T]) clientsCount() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	thisCount := len(this.clientsMap)
	if this.previousNotifier != nil {
		thisCount += this.previousNotifier.clientsCount()
	}
	return thisCount
}

func getMeta[T entity.HasMetadata](obj *T) entity.Metadata {
	object := *obj
	return object.GetMetadata()
}

func (this *resourceVersionAwareClientNotifier[F, T]) String() string {
	return fmt.Sprintf("WatchHandler %s: %s@%s", this.id, this.kind, this.namespace)
}

func acceptMeta(meta *entity.Metadata, metaFilter *filter.Meta) bool {
	return matchMeta(metaFilter.Labels, meta.Labels) && matchMeta(metaFilter.Annotations, meta.Annotations)
}

func matchMeta(expected, actual map[string]string) bool {
	for expectedK, expectedV := range expected {
		if actualV, ok := actual[expectedK]; !ok || (actualV != expectedV && expectedV != anyValue) {
			return false
		}
	}
	return true
}
