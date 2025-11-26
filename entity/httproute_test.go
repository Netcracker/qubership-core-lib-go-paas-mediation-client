package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestHttpRoute_GetMetadata(t *testing.T) {
	httpRoute := &HttpRoute{
		HTTPRoute: &v1.HTTPRoute{
			TypeMeta: metav1.TypeMeta{
				Kind: "HTTPRoute",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-http-route",
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

	metadata := httpRoute.GetMetadata()

	assert.Equal(t, "HTTPRoute", metadata.Kind)
	assert.Equal(t, "test-http-route", metadata.Name)
	assert.Equal(t, "test-namespace", metadata.Namespace)
	assert.Equal(t, "test-app", metadata.Labels["app"])
	assert.Equal(t, "test route", metadata.Annotations["description"])
}

func TestRouteFromHTTPRoute(t *testing.T) {
	httpRoute := &v1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-http-route",
			Namespace: "test-namespace",
		},
	}

	result := RouteFromHTTPRoute(httpRoute)

	assert.NotNil(t, result)
	assert.Equal(t, httpRoute, result.HTTPRoute)
	assert.Equal(t, "test-http-route", result.Name)
	assert.Equal(t, "test-namespace", result.Namespace)
}

func TestRouteFromHTTPRoute_NilInput(t *testing.T) {
	result := RouteFromHTTPRoute(nil)

	assert.NotNil(t, result)
	assert.Nil(t, result.HTTPRoute)
}
