package watch

import (
	"context"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

//go:generate mockgen -source=executor.go -destination=executor_mock.go -package=watch -mock_names=Executor=MockExecutor

const Error = "ERROR"

type Executor interface {
	CreateWatchRequest(restClient rest.Interface, resource types.PaasResourceType, options *v1.ListOptions) *rest.Request
	Watch(ctx context.Context, watchRequest *rest.Request) (watch.Interface, error)
}

type ApiEvent struct {
	Type                  string                `json:"type"`
	Object                any                   `json:"object"`
	controlMessageDetails ControlMessageDetails //todo delete this in the next major release
}

func (w *ApiEvent) GetControlMessageDetails() ControlMessageDetails {
	return w.controlMessageDetails
}

func ApiEventConstructor(watchApiType string, object any, controlMessage ControlMessageDetails) ApiEvent {
	return ApiEvent{watchApiType, object, controlMessage}
}

type ControlMessageDetails struct {
	CloseCode    int
	MessageType  int
	CloseMessage string
}

type Handler struct {
	Channel      <-chan ApiEvent
	StopWatching func() // todo delete in next major release, close channel via context.cancel()
}

// todo re-design it in next major release to:
//type ExecutorV2[T entity.HasMetadata] interface {
//	Watch(ctx context.Context, namespace string, filter filter.Meta) (*HandlerV2[T], error)
//}
//
//type ApiEventV2[T entity.HasMetadata] struct {
//	Type   string `json:"type"`
//	Object T      `json:"object"`
//}
//
//type HandlerV2[T entity.HasMetadata] struct {
//	Channel <-chan *ApiEventV2[T]
//}
