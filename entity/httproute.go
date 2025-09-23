package entity

import (
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type HttpRoute struct {
	*v1.HTTPRoute
}

func (r HttpRoute) GetMetadata() Metadata {
	return *FromObjectMeta(r.Kind, &r.ObjectMeta)
}

func RouteFromHTTPRoute(httpRoute *v1.HTTPRoute) *HttpRoute {
	return &HttpRoute{httpRoute}
}
