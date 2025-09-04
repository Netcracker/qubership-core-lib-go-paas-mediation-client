package entity

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type (
	HttpRoute struct {
		Metadata `json:"metadata"`
		Spec     HttpRouteSpec `json:"spec"`
	}

	HttpRouteSpec struct {
		Host     string          `json:"host"`
		PathType string          `json:"pathType"`
		Path     string          `json:"path"`
		Service  HttpRouteTarget `json:"to"`
		Port     HttpRoutePort   `json:"port"`
	}

	HttpRoutePort struct {
		TargetPort int32 `json:"targetPort"`
	}

	HttpRouteTarget struct {
		Name string `json:"name"`
	}
)

func (httpRoute HttpRoute) GetMetadata() Metadata {
	return httpRoute.Metadata
}

func RouteFromHTTPRoute(httpRoute *gatewayv1.HTTPRoute) *HttpRoute {
	logger.Debugf("Processing RouteFrom HTTPRoute, httproute: %s", httpRoute.Name)
	metadata := *FromObjectMeta("HTTPRoute", &httpRoute.ObjectMeta)

	if httpRoute.Spec.Rules == nil || len(httpRoute.Spec.Rules) == 0 || httpRoute.Spec.Hostnames == nil || len(httpRoute.Spec.Hostnames) == 0 {
		return &HttpRoute{Spec: HttpRouteSpec{}, Metadata: metadata}
	}
	port := httpRoute.Spec.Rules[0].BackendRefs[0].Port
	var portNumber int32 = 8080
	if port != nil {
		portNumber = int32(*port)
	}
	path := httpRoute.Spec.Rules[0].Matches[0].Path.Value
	pathValue := "/"
	if path != nil {
		pathValue = *path
	}

	routeSpec := HttpRouteSpec{
		Service: HttpRouteTarget{Name: string(httpRoute.Spec.Rules[0].BackendRefs[0].Name)},
		Port:    HttpRoutePort{TargetPort: portNumber},
		Path:    pathValue,
		Host:    string(httpRoute.Spec.Hostnames[0]),
	}
	return &HttpRoute{Spec: routeSpec, Metadata: metadata}
}

func hasRules(spec *gatewayv1.HTTPRouteSpec) bool {
	return spec != nil && spec.Rules != nil && len(spec.Rules) > 0
}
