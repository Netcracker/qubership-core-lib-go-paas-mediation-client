package entity

import (
	"testing"

	v1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
