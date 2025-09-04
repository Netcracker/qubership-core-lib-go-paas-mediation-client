package entity

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type (
	GrpcRoute struct {
		Metadata `json:"metadata"`
		Spec     GrpcRouteSpec `json:"spec"`
	}

	GrpcRouteSpec struct {
		Host     string          `json:"host"`
		PathType string          `json:"pathType"`
		Path     string          `json:"path"`
		Service  GrpcRouteTarget `json:"to"`
		Port     GrpcRoutePort   `json:"port"`
	}

	GrpcRoutePort struct {
		TargetPort int32 `json:"targetPort"`
	}

	GrpcRouteTarget struct {
		Name string `json:"name"`
	}
)

func (grpcRoute GrpcRoute) GetMetadata() Metadata {
	return grpcRoute.Metadata
}

func RouteFromGRPCRoute(grpcRoute *gatewayv1.GRPCRoute) *GrpcRoute {
	logger.Debugf("Processing RouteFrom GrpcRoute, grpcroute: %s", grpcRoute.Name)
	metadata := *FromObjectMeta("GRPCRoute", &grpcRoute.ObjectMeta)

	if grpcRoute.Spec.Rules == nil || len(grpcRoute.Spec.Rules) == 0 || grpcRoute.Spec.Hostnames == nil || len(grpcRoute.Spec.Hostnames) == 0 {
		return &GrpcRoute{Spec: GrpcRouteSpec{}, Metadata: metadata}
	}
	port := grpcRoute.Spec.Rules[0].BackendRefs[0].Port
	var portNumber int32 = 8080
	if port != nil {
		portNumber = int32(*port)
	}

	routeSpec := GrpcRouteSpec{
		Service: GrpcRouteTarget{Name: string(grpcRoute.Spec.Rules[0].BackendRefs[0].Name)},
		Port:    GrpcRoutePort{TargetPort: portNumber},
		Path:    "/",
		Host:    string(grpcRoute.Spec.Hostnames[0]),
	}
	return &GrpcRoute{Spec: routeSpec, Metadata: metadata}
}
