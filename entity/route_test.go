package entity

import (
	"testing"

	v1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	testDeploymentName  = "test-deployment"
	testIngress         = "test-ingress"
	testIngress2        = "test-ingress-2"
	testService         = "test-service"
	testPod             = "test-pod"
	testReplicaSet      = "test-rs"
	testName            = "testName"
	testNamespace       = "testNamespace"
	testAnnotationKey   = "testAnnotationKey"
	testAnnotationValue = "testAnnotationValue"
	testLabelKey        = "testLabelKey"
	testLabelValue      = "testLabelValue"
	testHost            = "testHost"
	testPath            = "testPath"
	testServiceName     = "testServiceName"
	testPathType        = "ImplementationSpecific"
	testPort            = 5555
)

var (
	testIngressClassName = "test-ingress-class-name"
)

func getRoute() Route {
	return Route{Metadata: Metadata{Name: testIngress, Namespace: testNamespace},
		Spec: RouteSpec{Host: "local", Path: "path", Port: RoutePort{TargetPort: int32(32)},
			Service: Target{Name: "target"}}}
}

func getOsRoute() *v1.Route {
	return &v1.Route{ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace},
		Spec: v1.RouteSpec{To: v1.RouteTargetReference{Name: "targetReference"},
			Host: "local", Port: &v1.RoutePort{TargetPort: intstr.IntOrString{IntVal: 32}},
			Path: "path"},
	}
}

func Test_RouteFromIngress_success(t *testing.T) {
	ingressPath := v1beta1.HTTPIngressPath{Path: "test-path", Backend: v1beta1.IngressBackend{ServiceName: "name",
		ServicePort: intstr.IntOrString{IntVal: int32(2)}}}

	httpIngressRuleValue := v1beta1.HTTPIngressRuleValue{Paths: []v1beta1.HTTPIngressPath{ingressPath}}

	ingress := v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace, ResourceVersion: "1"},
		Spec: v1beta1.IngressSpec{Rules: []v1beta1.IngressRule{
			{Host: "8080",
				IngressRuleValue: v1beta1.IngressRuleValue{HTTP: &httpIngressRuleValue},
			},
		},
		},
	}

	metadata := NewMetadata("Route", ingress.Name, ingress.Namespace,
		string(ingress.UID), ingress.Generation, ingress.ResourceVersion,
		ingress.Annotations, ingress.Labels)
	target := Target{Name: ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName}
	port := RoutePort{TargetPort: ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort.IntVal}
	routeSpec := RouteSpec{
		Service: target,
		Port:    port,
		Path:    ingress.Spec.Rules[0].HTTP.Paths[0].Path,
		Host:    ingress.Spec.Rules[0].Host,
	}

	route := Route{Spec: routeSpec, Metadata: metadata}
	testedRoute := RouteFromIngress(&ingress)
	assert.Equal(t, &route, testedRoute)
}

func Test_RouteFromIngressNetworkingV1_success(t *testing.T) {
	pathExample := networkingV1.PathType("TYPE")
	ingressServiceBackend := networkingV1.IngressServiceBackend{Name: "nameService",
		Port: networkingV1.ServiceBackendPort{Name: "aaa", Number: int32(80)}}

	ingressPath := networkingV1.HTTPIngressPath{PathType: &pathExample, Path: "test-path",
		Backend: networkingV1.IngressBackend{
			Service: &ingressServiceBackend}}

	httpIngressRuleValue := networkingV1.HTTPIngressRuleValue{Paths: []networkingV1.HTTPIngressPath{ingressPath}}

	ingress := networkingV1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace, ResourceVersion: "1"},
		Spec: networkingV1.IngressSpec{Rules: []networkingV1.IngressRule{
			{Host: "8080",
				IngressRuleValue: networkingV1.IngressRuleValue{HTTP: &httpIngressRuleValue},
			},
		},
		},
	}

	metadata := NewMetadata("Route", ingress.Name, ingress.Namespace,
		string(ingress.UID), ingress.Generation, ingress.ResourceVersion,
		ingress.Annotations, ingress.Labels)
	target := Target{Name: ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name}
	portNumber := ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number
	port := RoutePort{TargetPort: portNumber}

	routeSpec := RouteSpec{
		Service:  target,
		Port:     port,
		PathType: string(*ingress.Spec.Rules[0].HTTP.Paths[0].PathType),
		Path:     ingress.Spec.Rules[0].HTTP.Paths[0].Path,
		Host:     ingress.Spec.Rules[0].Host,
	}
	route := &Route{Metadata: metadata, Spec: routeSpec}
	assert.Equal(t, route, RouteFromIngressNetworkingV1(&ingress))
}

func Test_RouteFromOsRoute_success(t *testing.T) {
	osRoute := getOsRoute()

	metadata := NewMetadata("Route", osRoute.Name, osRoute.Namespace,
		string(osRoute.UID), osRoute.Generation, osRoute.ResourceVersion,
		osRoute.Annotations, osRoute.Labels)
	target := Target{Name: osRoute.Spec.To.Name}
	port := RoutePort{}
	if osRoute.Spec.Port != nil {
		port.TargetPort = osRoute.Spec.Port.TargetPort.IntVal
	}
	routeSpec := RouteSpec{
		Port:    port,
		Service: target,
		Path:    osRoute.Spec.Path,
		Host:    osRoute.Spec.Host,
	}
	route := &Route{Metadata: metadata, Spec: routeSpec}
	assert.Equal(t, route, RouteFromOsRoute(osRoute))
}

func Test_RouteListFromOsRouteList_success(t *testing.T) {
	osRoute1 := getOsRoute()
	osRoute2 := *osRoute1
	osRoute2.Spec.Port = &v1.RoutePort{TargetPort: intstr.IntOrString{IntVal: 64}}
	osRouteList := []*v1.Route{osRoute1, &osRoute2}
	result, badRouteList := RouteListFromOsRouteList(osRouteList)
	assert.Equal(t, 2, len(result))
	assert.Empty(t, badRouteList)
}

func Test_RouteFromIngressList_success(t *testing.T) {
	ingressPath := v1beta1.HTTPIngressPath{Path: "test-path", Backend: v1beta1.IngressBackend{ServiceName: "name",
		ServicePort: intstr.IntOrString{IntVal: int32(2)}}}

	httpIngressRuleValue := v1beta1.HTTPIngressRuleValue{Paths: []v1beta1.HTTPIngressPath{ingressPath}}

	ingress1 := &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace, ResourceVersion: "1"},
		Spec: v1beta1.IngressSpec{Rules: []v1beta1.IngressRule{
			{Host: "8080",
				IngressRuleValue: v1beta1.IngressRuleValue{HTTP: &httpIngressRuleValue},
			},
		},
		},
	}

	ingress2 := *ingress1
	ingress2.Name = testIngress2
	ingressList := []*v1beta1.Ingress{ingress1, &ingress2}
	result, badRouteList := RouteListFromIngressList(ingressList)
	assert.Equal(t, 2, len(result))
	assert.Empty(t, badRouteList)
}

func Test_RouteFromIngressListNetworkingV1_success(t *testing.T) {
	pathExample := networkingV1.PathType("TYPE")
	ingressServiceBackend := networkingV1.IngressServiceBackend{Name: "nameService",
		Port: networkingV1.ServiceBackendPort{Name: "aaa", Number: int32(80)}}

	ingressPath := networkingV1.HTTPIngressPath{PathType: &pathExample, Path: "test-path",
		Backend: networkingV1.IngressBackend{
			Service: &ingressServiceBackend}}

	httpIngressRuleValue := networkingV1.HTTPIngressRuleValue{Paths: []networkingV1.HTTPIngressPath{ingressPath}}

	ingress1 := &networkingV1.Ingress{
		ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace, ResourceVersion: "1"},
		Spec: networkingV1.IngressSpec{Rules: []networkingV1.IngressRule{
			{Host: "8080",
				IngressRuleValue: networkingV1.IngressRuleValue{HTTP: &httpIngressRuleValue},
			},
		},
		},
	}

	ingress2 := *ingress1
	ingress2.Name = testIngress2
	ingressList := []*networkingV1.Ingress{ingress1, &ingress2}
	result, badRouteList := RouteListFromIngressListNetworkingV1(ingressList)
	assert.Equal(t, 2, len(result))
	assert.Empty(t, badRouteList)
}

func Test_ToOsRoute(t *testing.T) {
	route := getRoute()
	targetPort := intstr.IntOrString{Type: intstr.Int, IntVal: route.Spec.Port.TargetPort}
	routePort := &v1.RoutePort{TargetPort: targetPort}
	routeV1 := v1.Route{ObjectMeta: metav1.ObjectMeta{Name: testIngress, Namespace: testNamespace},
		Spec: v1.RouteSpec{Host: "local", Path: "path", To: v1.RouteTargetReference{Name: "target"}, Port: routePort}}
	osRoute := route.ToOsRoute()
	assert.Equal(t, &routeV1, osRoute)
}

func Test_ToIngressNetworkingV1_success(t *testing.T) {
	route := getRoute()

	ingress := networkingV1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: route.Name, Namespace: route.Namespace}}
	ingress.Spec.Rules = []networkingV1.IngressRule{{Host: route.Spec.Host}}

	var pathType networkingV1.PathType = "Prefix"
	ingress.Spec.Rules[0].HTTP = &networkingV1.HTTPIngressRuleValue{
		Paths: []networkingV1.HTTPIngressPath{
			{
				PathType: &pathType,
				Path:     route.Spec.Path,
				Backend: networkingV1.IngressBackend{
					Service: &networkingV1.IngressServiceBackend{
						Name: route.Spec.Service.Name,
						Port: networkingV1.ServiceBackendPort{
							Number: route.Spec.Port.TargetPort,
						},
					},
				},
			},
		},
	}

	ingressFromFunction := route.ToIngressNetworkingV1()
	assert.Equal(t, &ingress, ingressFromFunction)
}

func Test_ToIngress_success(t *testing.T) {
	route := getRoute()

	ingress := v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Name: route.Name, Namespace: route.Namespace}}
	ingress.Spec.Rules = []v1beta1.IngressRule{{Host: route.Spec.Host}}
	servicePort := intstr.IntOrString{Type: intstr.Int, IntVal: route.Spec.Port.TargetPort}

	ingress.Spec.Rules[0].HTTP = &v1beta1.HTTPIngressRuleValue{
		Paths: []v1beta1.HTTPIngressPath{
			{
				Path: route.Spec.Path,
				Backend: v1beta1.IngressBackend{
					ServiceName: route.Spec.Service.Name,
					ServicePort: servicePort,
				},
			},
		},
	}
	ingressFromFunc := route.ToIngress()
	assert.Equal(t, &ingress, ingressFromFunc)
}

func createSimpleRoute(pathType string, port int32) *Route {
	return &Route{
		Metadata: Metadata{
			Kind:        "Route",
			Name:        testName,
			Namespace:   testNamespace,
			Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
			Labels:      map[string]string{testLabelKey: testLabelValue},
		},
		Spec: RouteSpec{
			Host:             testHost,
			Path:             testPath,
			PathType:         pathType,
			Service:          Target{Name: testServiceName},
			Port:             RoutePort{TargetPort: port},
			IngressClassName: &testIngressClassName,
		},
	}
}

func createRouteMap() map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"name":        testName,
			"namespace":   testNamespace,
			"annotations": map[string]any{testAnnotationKey: testAnnotationValue},
			"labels":      map[string]any{testLabelKey: testLabelValue},
		},
		"spec": map[string]any{
			"to": map[string]any{
				"name": testServiceName},
			"host": testHost,
			"path": testPath},
		"ingressClassName": &testIngressClassName,
	}
}

func createSimpleIngress() *v1beta1.Ingress {
	return &v1beta1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testName,
			Namespace:   testNamespace,
			Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
			Labels:      map[string]string{testLabelKey: testLabelValue},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{{Host: testHost,
				IngressRuleValue: v1beta1.IngressRuleValue{HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: []v1beta1.HTTPIngressPath{
						{
							Path: testPath,
							Backend: v1beta1.IngressBackend{
								ServiceName: testServiceName,
								ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(testPort)},
							},
						},
					},
				}},
			}},
			IngressClassName: &testIngressClassName,
		},
	}
}

func createSimpleIngressNetworkingV1() *networkingV1.Ingress {
	pathType := networkingV1.PathTypeImplementationSpecific
	return &networkingV1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        testName,
			Namespace:   testNamespace,
			Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
			Labels:      map[string]string{testLabelKey: testLabelValue},
		},
		Spec: networkingV1.IngressSpec{
			Rules: []networkingV1.IngressRule{{
				Host: testHost,
				IngressRuleValue: networkingV1.IngressRuleValue{HTTP: &networkingV1.HTTPIngressRuleValue{
					Paths: []networkingV1.HTTPIngressPath{
						{
							PathType: &pathType,
							Path:     testPath,
							Backend: networkingV1.IngressBackend{
								Service: &networkingV1.IngressServiceBackend{
									Name: testServiceName,
									Port: networkingV1.ServiceBackendPort{Number: int32(testPort)},
								},
							},
						},
					},
				}},
			}},
			IngressClassName: &testIngressClassName,
		},
	}
}

func createSimpleOsRoute() *v1.Route {
	return &v1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        testName,
			Namespace:   testNamespace,
			Annotations: map[string]string{testAnnotationKey: testAnnotationValue},
			Labels:      map[string]string{testLabelKey: testLabelValue},
		},
		Spec: v1.RouteSpec{
			Host: testHost,
			Path: testPath,
			To: v1.RouteTargetReference{
				Name: testServiceName,
			},
			AlternateBackends: nil,
			Port:              &v1.RoutePort{TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(testPort)}},
		},
	}
}

func TestToIngress(t *testing.T) {
	assert.Equal(t, createSimpleIngress(), createSimpleRoute("", int32(testPort)).ToIngress())
}

func TestToIngressNetworkingV1(t *testing.T) {
	assert.Equal(t, createSimpleIngressNetworkingV1(),
		createSimpleRoute(testPathType, int32(testPort)).ToIngressNetworkingV1())
}

func TestToOsRoute(t *testing.T) {
	assert.Equal(t, createSimpleOsRoute(), createSimpleRoute("", int32(testPort)).ToOsRoute())
}

func TestRouteFromIngress(t *testing.T) {
	assert.Equal(t, createSimpleRoute("", int32(testPort)), RouteFromIngress(createSimpleIngress()))
}

func TestRouteFromIngressNetworkingV1(t *testing.T) {
	assert.Equal(t, createSimpleRoute(testPathType, int32(testPort)),
		RouteFromIngressNetworkingV1(createSimpleIngressNetworkingV1()))
}

func TestRouteListFromIngressList(t *testing.T) {
	ingress := createSimpleIngress()
	routeList, _ := RouteListFromIngressList([]*v1beta1.Ingress{ingress})
	assert.Equal(t, 1, len(routeList))
	assert.Equal(t, []Route{*createSimpleRoute("", int32(testPort))}, routeList)
}

func TestRouteListFromIngressListNetworkingV1(t *testing.T) {
	ingress := createSimpleIngressNetworkingV1()
	routeList, _ := RouteListFromIngressListNetworkingV1([]*networkingV1.Ingress{ingress})
	assert.Equal(t, 1, len(routeList))
	assert.Equal(t, []Route{*createSimpleRoute(testPathType, int32(testPort))}, routeList)
}

func TestRouteFromOsRoute(t *testing.T) {
	route := createSimpleRoute("", int32(testPort))
	route.Spec.IngressClassName = nil // OpenShift does not support IngressClassName
	assert.Equal(t, route, RouteFromOsRoute(createSimpleOsRoute()))
}

func TestRouteListFromOsRouteList(t *testing.T) {
	osRoute := createSimpleOsRoute()
	routeList, _ := RouteListFromOsRouteList([]*v1.Route{osRoute})
	assert.Equal(t, 1, len(routeList))
	simpleRoute := createSimpleRoute("", int32(testPort))
	simpleRoute.Spec.IngressClassName = nil // OpenShift does not support IngressClassName
	assert.Equal(t, []Route{*simpleRoute}, routeList)
}

func TestNewRouteFromInterface(t *testing.T) {
	route := createSimpleRoute("", 0)
	route.Spec.IngressClassName = nil // OpenShift does not support IngressClassName
	assert.Equal(t, route, NewRouteFromInterface(createRouteMap()))
}

func TestToHTTPRoute(t *testing.T) {
	route := createSimpleRoute("Prefix", int32(testPort))
	httpRoute := route.ToHTTPRoute("gateway-system", "default-external-gateway")

	assert.Equal(t, testName, httpRoute.Name)
	assert.Equal(t, testNamespace, httpRoute.Namespace)
	assert.Equal(t, 1, len(httpRoute.Spec.ParentRefs))
	assert.Equal(t, "default-external-gateway", string(httpRoute.Spec.ParentRefs[0].Name))
	assert.Equal(t, "gateway-system", string(*httpRoute.Spec.ParentRefs[0].Namespace))
	assert.Equal(t, 1, len(httpRoute.Spec.Hostnames))
	assert.Equal(t, testHost, string(httpRoute.Spec.Hostnames[0]))
	assert.Equal(t, 1, len(httpRoute.Spec.Rules))
	assert.Equal(t, testPath, *httpRoute.Spec.Rules[0].Matches[0].Path.Value)
	assert.Equal(t, testServiceName, string(httpRoute.Spec.Rules[0].BackendRefs[0].Name))
	assert.Equal(t, int32(testPort), int32(*httpRoute.Spec.Rules[0].BackendRefs[0].Port))
}

func TestToHTTPRoute_WithDefaults(t *testing.T) {
	route := &Route{
		Metadata: Metadata{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: RouteSpec{
			Host:    testHost,
			Service: Target{Name: testServiceName},
		},
	}

	httpRoute := route.ToHTTPRoute("gateway-system", "default-external-gateway")

	assert.Equal(t, "/", *httpRoute.Spec.Rules[0].Matches[0].Path.Value)
	assert.Equal(t, int32(8080), int32(*httpRoute.Spec.Rules[0].BackendRefs[0].Port))
	assert.Equal(t, "PathPrefix", string(*httpRoute.Spec.Rules[0].Matches[0].Path.Type))
}

func TestToHTTPRoute_WithSessionAffinity(t *testing.T) {
	route := createSimpleRoute("Prefix", int32(testPort))
	route.Metadata.Annotations = map[string]string{
		AnnotationAffinity:            "cookie",
		AnnotationSessionCookieName:   "my-cookie",
		AnnotationSessionCookieMaxAge: "3600",
	}

	httpRoute := route.ToHTTPRoute("gateway-system", "default-external-gateway")

	assert.NotNil(t, httpRoute.Spec.Rules[0].SessionPersistence)
	assert.Equal(t, "my-cookie", *httpRoute.Spec.Rules[0].SessionPersistence.SessionName)
	assert.Equal(t, "3600s", string(*httpRoute.Spec.Rules[0].SessionPersistence.AbsoluteTimeout))
}

func TestToHTTPRoute_WithTimeouts(t *testing.T) {
	route := createSimpleRoute("Prefix", int32(testPort))
	route.Metadata.Annotations = map[string]string{
		AnnotationProxyReadTimeout: "1800",
	}

	httpRoute := route.ToHTTPRoute("gateway-system", "default-external-gateway")

	assert.NotNil(t, httpRoute.Spec.Rules[0].Timeouts)
	assert.Equal(t, "1800s", string(*httpRoute.Spec.Rules[0].Timeouts.Request))
}

func TestPathMatchTypeFromRoute(t *testing.T) {
	tests := []struct {
		name     string
		pathType string
		expected string
	}{
		{"Empty defaults to PathPrefix", "", "PathPrefix"},
		{"Exact maps to Exact", "Exact", "Exact"},
		{"Prefix maps to PathPrefix", "Prefix", "PathPrefix"},
		{"ImplementationSpecific maps to PathPrefix", "ImplementationSpecific", "PathPrefix"},
		{"Unknown defaults to PathPrefix", "Unknown", "PathPrefix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pathMatchTypeFromRoute(tt.pathType)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestRoutePathValue(t *testing.T) {
	assert.Equal(t, "/", routePathValue(""))
	assert.Equal(t, "/api", routePathValue("/api"))
}

func TestRouteBackendPort(t *testing.T) {
	assert.Equal(t, int32(8080), int32(routeBackendPort(0)))
	assert.Equal(t, int32(9090), int32(routeBackendPort(9090)))
}

func TestRouteFromHTTPRouteGatewayV1(t *testing.T) {
	pathType := "PathPrefix"
	pathValue := "/test-path"
	port := int32(8080)
	hostname := "example.com"

	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{gatewayv1.Hostname(hostname)},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  (*gatewayv1.PathMatchType)(&pathType),
								Value: &pathValue,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: gatewayv1.ObjectName(testServiceName),
									Port: (*gatewayv1.PortNumber)(&port),
								},
							},
						},
					},
				},
			},
		},
	}

	route := RouteFromHTTPRouteGatewayV1(httpRoute)

	assert.Equal(t, testName, route.Metadata.Name)
	assert.Equal(t, testNamespace, route.Metadata.Namespace)
	assert.Equal(t, hostname, route.Spec.Host)
	assert.Equal(t, pathValue, route.Spec.Path)
	assert.Equal(t, pathType, route.Spec.PathType)
	assert.Equal(t, testServiceName, route.Spec.Service.Name)
	assert.Equal(t, port, route.Spec.Port.TargetPort)
}

func TestRouteFromHTTPRouteGatewayV1_EmptyRules(t *testing.T) {
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testName,
			Namespace: testNamespace,
		},
		Spec: gatewayv1.HTTPRouteSpec{},
	}

	route := RouteFromHTTPRouteGatewayV1(httpRoute)

	assert.Equal(t, testName, route.Metadata.Name)
	assert.Equal(t, "", route.Spec.Host)
	assert.Equal(t, "", route.Spec.Path)
	assert.Equal(t, "", route.Spec.Service.Name)
	assert.Equal(t, int32(0), route.Spec.Port.TargetPort)
}

func TestBuildSessionPersistence(t *testing.T) {
	annotations := map[string]string{
		AnnotationAffinity:            "cookie",
		AnnotationSessionCookieName:   "test-cookie",
		AnnotationSessionCookieMaxAge: "7200",
	}

	sessionPersistence := buildSessionPersistence(annotations)

	assert.NotNil(t, sessionPersistence)
	assert.Equal(t, "test-cookie", *sessionPersistence.SessionName)
	assert.Equal(t, "7200s", string(*sessionPersistence.AbsoluteTimeout))
}

func TestBuildTimeouts(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expectNil   bool
	}{
		{
			name: "With proxy-read-timeout",
			annotations: map[string]string{
				AnnotationProxyReadTimeout: "1800",
			},
			expectNil: false,
		},
		{
			name: "With proxy-send-timeout",
			annotations: map[string]string{
				AnnotationProxySendTimeout: "900",
			},
			expectNil: false,
		},
		{
			name: "With proxy-connect-timeout",
			annotations: map[string]string{
				AnnotationProxyConnectTimeout: "600",
			},
			expectNil: false,
		},
		{
			name:        "Without timeout annotations",
			annotations: map[string]string{},
			expectNil:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeouts := buildTimeouts(tt.annotations)
			if tt.expectNil {
				assert.Nil(t, timeouts)
			} else {
				assert.NotNil(t, timeouts)
			}
		})
	}
}

func TestNormalizeRouteAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		expectEmpty bool
	}{
		{
			name:        "Nil annotations",
			annotations: nil,
			expectEmpty: true,
		},
		{
			name:        "Non-nil annotations",
			annotations: map[string]string{"key": "value"},
			expectEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeRouteAnnotations(tt.annotations)
			assert.NotNil(t, result)
			if tt.expectEmpty {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.annotations, result)
			}
		})
	}
}
