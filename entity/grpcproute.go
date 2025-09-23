package entity

import (
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GrpcRoute struct {
	*v1.GRPCRoute
}

func (r GrpcRoute) GetMetadata() Metadata {
	return *FromObjectMeta(r.Kind, &r.ObjectMeta)
}

func RouteFromGRPCRoute(grpcRoute *v1.GRPCRoute) *GrpcRoute {
	return &GrpcRoute{grpcRoute}
}
