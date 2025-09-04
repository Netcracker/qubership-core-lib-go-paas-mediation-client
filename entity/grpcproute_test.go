package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func Test_RouteFromGRPCRoute(t *testing.T) {
	name := "test-grpc-route"
	ns := testNamespace
	var port gatewayv1.PortNumber = 7070
	grpc := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, ResourceVersion: "1"},
		Spec: gatewayv1.GRPCRouteSpec{
			Hostnames: []gatewayv1.Hostname{"grpc.example.com"},
			Rules: []gatewayv1.GRPCRouteRule{{
				BackendRefs: []gatewayv1.GRPCBackendRef{{BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{Name: gatewayv1.ObjectName(testServiceName), Port: &port},
				}}},
			},
			},
		}}

	metadata := NewMetadata("GRPCRoute", grpc.Name, grpc.Namespace, string(grpc.UID), grpc.Generation, grpc.ResourceVersion, grpc.Annotations, grpc.Labels)
	expected := &GrpcRoute{Metadata: metadata, Spec: GrpcRouteSpec{
		Host:    string(grpc.Spec.Hostnames[0]),
		Path:    "/",
		Service: GrpcRouteTarget{Name: string(grpc.Spec.Rules[0].BackendRefs[0].Name)},
		Port:    GrpcRoutePort{TargetPort: int32(*grpc.Spec.Rules[0].BackendRefs[0].Port)},
	}}
	assert.Equal(t, expected, RouteFromGRPCRoute(grpc))
}

func Test_RouteFromGRPCRoute_emptyRulesOrHostnames(t *testing.T) {
	grpc := &gatewayv1.GRPCRoute{ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: testNamespace}}
	metadata := NewMetadata("GRPCRoute", grpc.Name, grpc.Namespace, string(grpc.UID), grpc.Generation, grpc.ResourceVersion, grpc.Annotations, grpc.Labels)
	expected := &GrpcRoute{Metadata: metadata, Spec: GrpcRouteSpec{}}
	assert.Equal(t, expected, RouteFromGRPCRoute(grpc))
}
