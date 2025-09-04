package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func Test_RouteFromHTTPRoute(t *testing.T) {
	name := "test-http-route"
	ns := testNamespace
	var port gatewayv1.PortNumber = 9090
	path := "/api"
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns, ResourceVersion: "1"},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"example.com"},
			Rules: []gatewayv1.HTTPRouteRule{{
				Matches: []gatewayv1.HTTPRouteMatch{{Path: &gatewayv1.HTTPPathMatch{Value: &path}}},
				BackendRefs: []gatewayv1.HTTPBackendRef{{BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{Name: gatewayv1.ObjectName(testServiceName), Port: &port},
				}}},
			},
			},
		}}

	metadata := NewMetadata("HTTPRoute", httpRoute.Name, httpRoute.Namespace, string(httpRoute.UID), httpRoute.Generation, httpRoute.ResourceVersion, httpRoute.Annotations, httpRoute.Labels)
	expected := &HttpRoute{Metadata: metadata, Spec: HttpRouteSpec{
		Host:    string(httpRoute.Spec.Hostnames[0]),
		Path:    path,
		Service: HttpRouteTarget{Name: string(httpRoute.Spec.Rules[0].BackendRefs[0].Name)},
		Port:    HttpRoutePort{TargetPort: int32(*httpRoute.Spec.Rules[0].BackendRefs[0].Port)},
	}}
	assert.Equal(t, expected, RouteFromHTTPRoute(httpRoute))
}

func Test_RouteFromHTTPRoute_emptyRulesOrHostnames(t *testing.T) {
	http := &gatewayv1.HTTPRoute{ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: testNamespace}}
	metadata := NewMetadata("HTTPRoute", http.Name, http.Namespace, string(http.UID), http.Generation, http.ResourceVersion, http.Annotations, http.Labels)
	expected := &HttpRoute{Metadata: metadata, Spec: HttpRouteSpec{}}
	assert.Equal(t, expected, RouteFromHTTPRoute(http))
}
