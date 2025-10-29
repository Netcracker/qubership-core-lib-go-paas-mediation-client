package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestGrpcRoute_GetMetadata(t *testing.T) {
	grpcRoute := &GrpcRoute{
		GRPCRoute: &v1.GRPCRoute{
			TypeMeta: metav1.TypeMeta{
				Kind: "GRPCRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-grpc-route",
				Namespace: "test-namespace",
				Labels: map[string]string{
					"app": "test-app",
				},
				Annotations: map[string]string{
					"description": "test route",
				},
			},
		},
	}

	metadata := grpcRoute.GetMetadata()

	assert.Equal(t, "GRPCRoute", metadata.Kind)
	assert.Equal(t, "test-grpc-route", metadata.Name)
	assert.Equal(t, "test-namespace", metadata.Namespace)
	assert.Equal(t, "test-app", metadata.Labels["app"])
	assert.Equal(t, "test route", metadata.Annotations["description"])
}

func TestRouteFromGRPCRoute(t *testing.T) {
	grpcRoute := &v1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-grpc-route",
			Namespace: "test-namespace",
		},
	}

	result := RouteFromGRPCRoute(grpcRoute)

	assert.NotNil(t, result)
	assert.Equal(t, grpcRoute, result.GRPCRoute)
	assert.Equal(t, "test-grpc-route", result.Name)
	assert.Equal(t, "test-namespace", result.Namespace)
}

func TestRouteFromGRPCRoute_NilInput(t *testing.T) {
	result := RouteFromGRPCRoute(nil)

	assert.NotNil(t, result)
	assert.Nil(t, result.GRPCRoute)
}
