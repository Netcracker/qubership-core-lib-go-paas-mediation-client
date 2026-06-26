package kubernetes

import (
	"context"
	"errors"
	"time"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

const (
	CloseControlMessage = "CLOSE_CONTROL_MESSAGE"
)

var reconnectWaitInterval = 5 * time.Second

type WatchLoopState string

const (
	Connect           WatchLoopState = "Connect"
	Running           WatchLoopState = "Running"
	ServerSideClosure WatchLoopState = "ServerSideClosure"
	ServerSideError   WatchLoopState = "ServerSideError"
	ClientSideCancel  WatchLoopState = "ClientSideCancel"
)

type DefaultWatchExecutor struct {
}

func (de *DefaultWatchExecutor) Watch(ctx context.Context, request *rest.Request) (watch.Interface, error) {
	return request.Watch(ctx)
}

func (de *DefaultWatchExecutor) CreateWatchRequest(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request {
	return restClient.Get().Resource(resource.String()).VersionedParams(options, scheme.ParameterCodec)
}

func (s WatchLoopState) ok() bool {
	return s == Connect || s == Running
}

type RestWatchEventHandler[F runtime.Object, T entity.HasMetadata] struct {
	Namespace  string
	Kind       types.PaasResourceType
	RestClient rest.Interface
	Executor   pmWatch.Executor //todo get rid of this interface
	Converter  func(F) *T
}

func NewRestWatchHandler[F runtime.Object, T entity.HasMetadata](namespace string, kind types.PaasResourceType,
	restClient rest.Interface, executor pmWatch.Executor, converter func(F) *T) *RestWatchEventHandler[F, T] {
	return &RestWatchEventHandler[F, T]{Namespace: namespace, Kind: kind, RestClient: restClient, Executor: executor, Converter: converter}
}

func (handler *RestWatchEventHandler[F, T]) Watch(ctx context.Context, metaFilter filter.Meta) (*pmWatch.Handler, error) {
	kind := handler.Kind
	namespace := handler.Namespace
	logger.DebugC(ctx, "Performing watch operation for %s in %s", kind, namespace)
	var bookmark string
	watcher, err := handler.doWatchRequest(ctx, namespace, kind, metaFilter, bookmark)
	if err != nil {
		return nil, err
	} else if watcher == nil {
		return nil, errors.New("watcher is nil")
	}
	ctx, cancel := context.WithCancel(ctx)
	// don't make channel's size high, otherwise the handler will consume more memory if its consumers are slow to process events
	clientResultChannel := make(chan pmWatch.ApiEvent, 1)
	watchHandler := pmWatch.Handler{Channel: clientResultChannel, StopWatching: cancel}
	go func(ctx context.Context, watcher watch.Interface, clientResultChannel chan pmWatch.ApiEvent) {
		apiServerResultChannel := watcher.ResultChan()
		watchLoopState := Running
		defer func() {
			close(clientResultChannel)
			if watcher != nil {
				watcher.Stop()
			}
		}()
		logger.InfoC(ctx, "Starting watch loop for '%s' in namespace '%s'", kind, namespace)
		defer func() {
			logger.InfoC(ctx, "Stopped watch loop for '%s' in namespace '%s', watchLoopState = '%s'", kind, namespace, watchLoopState)
		}()

		for watchLoopState.ok() {
			if watchLoopState == Connect {
				watcher, err = handler.doWatchRequest(ctx, namespace, kind, metaFilter, bookmark)
				if err != nil {
					logger.ErrorC(ctx, "Failed to create watch handler. Sending err to the client and terminating. Err: %s", err.Error())
					clientResultChannel <- pmWatch.ApiEvent{Type: pmWatch.Error, Object: err.Error()}
					watchLoopState = ServerSideError
					return
				} else {
					newChan := watcher.ResultChan()
					if newChan == apiServerResultChannel {
						logger.ErrorC(ctx, "Watcher returned the same result channel after re-connect. Terminating")
						watchLoopState = ServerSideError
						return
					} else {
						apiServerResultChannel = newChan
						watchLoopState = Running
						continue
					}
				}
			}
			select {
			case <-ctx.Done():
				logger.DebugC(ctx, "Exiting watch loop for '%s' in namespace '%s' due to: %s", kind, namespace, ctx.Err().Error())
				watchLoopState = ClientSideCancel
				continue
			case event, opened := <-apiServerResultChannel:
				if !opened {
					// try to reconnect to Api server with last observed resourceVersion
					logger.InfoC(ctx, "Api Server watcher's result channel was closed. Trying to re-connect with bookmark=%s. Namespace=%s, resource=%s",
						bookmark, namespace, kind)
					watchLoopState = Connect
					continue
				}
				if event.Type == watch.Error {
					logger.WarnC(ctx, "Received Error event in namespace=%s, resource=%s, try to re-connect without bookmark after %s, event was : %+v",
						namespace, kind, reconnectWaitInterval.String(), event)
					watcher.Stop()
					bookmark = ""
					watchLoopState = Connect
					time.Sleep(reconnectWaitInterval)
					continue
				} else if event.Type == watch.Bookmark {
					if obj, ok := event.Object.(interface{ GetResourceVersion() string }); ok {
						bookmark = obj.GetResourceVersion()
						logger.DebugC(ctx, "Received bookmark event with resourceVersion=%s", bookmark)
						if metaFilter.WatchBookmark {
							clientResultChannel <- pmWatch.ApiEvent{
								Type:   string(event.Type),
								Object: handler.Converter(obj.(F)),
							}
						}
					} else {
						bookmark = "" //reset bookmark if it was set
						logger.WarnC(ctx, "Received unsupported bookmark event. Event: %+v", event)
					}
				} else {
					var obj F
					switch eventObject := event.Object.(type) {
					case F:
						obj = eventObject
					default:
						logger.ErrorC(ctx, "event has invalid Object field type: %v. Exiting watch loop.", eventObject)
						watchLoopState = ServerSideClosure
						continue
					}
					clientResultChannel <- pmWatch.ApiEvent{
						Type:   string(event.Type),
						Object: handler.Converter(obj),
					}
				}
			}
		}
	}(ctx, watcher, clientResultChannel)
	return &watchHandler, nil
}

func (handler *RestWatchEventHandler[F, T]) doWatchRequest(ctx context.Context, namespace string, resource types.PaasResourceType,
	filter filter.Meta, resourceVersion string) (watch.Interface, error) {
	logger.DebugC(ctx, "Try to establish web socket connection. Resource=%s, namespace=%s", resource, namespace)
	options := &v1.ListOptions{}
	if filter.Labels != nil {
		options.LabelSelector = labels.Set(filter.Labels).String()
		logger.DebugC(ctx, "Applying label selector: %s", options.LabelSelector)
	}
	if filter.Annotations != nil && len(filter.Annotations) > 0 {
		return nil, errors.New("watch API does not support filtering by annotations, use labels instead")
	}
	options.Watch = true
	options.AllowWatchBookmarks = true
	if resourceVersion != "" {
		options.ResourceVersion = resourceVersion
		logger.DebugC(ctx, "Setting ResourceVersion: %s", options.ResourceVersion)
	}
	watchRequest := handler.Executor.CreateWatchRequest(handler.RestClient, resource, options)
	if namespace != "" {
		watchRequest.Namespace(namespace)
	}
	if handler.RestClient != nil {
		logger.DebugC(ctx, "Watch url: %s", watchRequest.URL().String())
	}
	initialTime := time.Second
	maxTime := time.Minute
	jitterFactor := 0.1
	logger.DebugC(ctx, "Setting watchRequest's backoff: with initial=%v, max=%v, jitterFactor=%v", initialTime, maxTime, jitterFactor)
	watchRequest.BackOff(&rest.URLBackoff{Backoff: flowcontrol.NewBackOffWithJitter(initialTime, maxTime, jitterFactor)})
	return handler.Executor.Watch(ctx, watchRequest)
}
