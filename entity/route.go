package entity

import (
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var logger = logging.GetLogger("entity_route")

type (
	// todo change to Ingress in next major release AND REWRITE entity to comply with Ingress structure!
	Route struct {
		Metadata `json:"metadata"`
		Spec     RouteSpec `json:"spec"`
	}

	RouteSpec struct {
		Host             string    `json:"host"`
		PathType         string    `json:"pathType"`
		Path             string    `json:"path"`
		Service          Target    `json:"to"`
		Port             RoutePort `json:"port"`
		IngressClassName *string   `json:"ingressClassName"`
	}

	RoutePort struct {
		TargetPort int32 `json:"targetPort"`
	}

	Target struct {
		Name string `json:"name"`
	}
)

func (route Route) ToIngress() *v1beta1.Ingress {
	ingress := v1beta1.Ingress{ObjectMeta: *route.Metadata.ToObjectMeta()}
	ingress.Spec.Rules = []v1beta1.IngressRule{{Host: route.Spec.Host}}
	port := route.Spec.Port.TargetPort
	if port == 0 {
		port = 8080
	}
	servicePort := intstr.IntOrString{Type: intstr.Int, IntVal: port}

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
	ingress.Spec.IngressClassName = route.Spec.IngressClassName
	return &ingress
}

func (route Route) ToIngressNetworkingV1() *networkingV1.Ingress {
	ingress := networkingV1.Ingress{ObjectMeta: *route.Metadata.ToObjectMeta()}
	ingress.Spec.Rules = []networkingV1.IngressRule{{Host: route.Spec.Host}}

	port := route.Spec.Port.TargetPort
	if port == 0 {
		port = 8080
	}
	var pathType networkingV1.PathType
	if route.Spec.PathType == "" {
		pathType = "Prefix"
	} else {
		pathType = networkingV1.PathType(route.Spec.PathType)
	}
	path := route.Spec.Path
	if path == "" {
		path = "/"
	}
	ingress.Spec.Rules[0].HTTP = &networkingV1.HTTPIngressRuleValue{
		Paths: []networkingV1.HTTPIngressPath{
			{
				PathType: &pathType,
				Path:     path,
				Backend: networkingV1.IngressBackend{
					Service: &networkingV1.IngressServiceBackend{
						Name: route.Spec.Service.Name,
						Port: networkingV1.ServiceBackendPort{
							Number: port,
						},
					},
				},
			},
		},
	}
	ingress.Spec.IngressClassName = route.Spec.IngressClassName
	return &ingress
}

func (route Route) ToOsRoute() *v1.Route {
	osRoute := v1.Route{ObjectMeta: *route.Metadata.ToObjectMeta()}
	osRoute.Spec.Host = route.Spec.Host
	osRoute.Spec.Path = route.Spec.Path
	osRoute.Spec.To.Name = route.Spec.Service.Name

	if route.Spec.Port.TargetPort > 0 {
		targetPort := intstr.IntOrString{Type: intstr.Int, IntVal: route.Spec.Port.TargetPort}
		routePort := &v1.RoutePort{TargetPort: targetPort}
		osRoute.Spec.Port = routePort
	}
	return &osRoute
}

func RouteFromIngress(ingress *v1beta1.Ingress) *Route {
	logger.Debugf("Processing RouteFromIngress, ingress: %s", ingress.Name)
	metadata := *FromObjectMeta("Route", &ingress.ObjectMeta)
	// todo re-implement this!!!
	var routeSpec RouteSpec
	if len(ingress.Spec.Rules) > 0 && ingress.Spec.Rules[0].HTTP != nil && len(ingress.Spec.Rules[0].HTTP.Paths) > 0 {
		target := Target{Name: ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName}
		port := RoutePort{TargetPort: ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort.IntVal}
		routeSpec = RouteSpec{
			Service:          target,
			Port:             port,
			Path:             ingress.Spec.Rules[0].HTTP.Paths[0].Path,
			Host:             ingress.Spec.Rules[0].Host,
			IngressClassName: ingress.Spec.IngressClassName,
		}
	}
	return &Route{Spec: routeSpec, Metadata: metadata}
}

func RouteFromIngressNetworkingV1(ingress *networkingV1.Ingress) *Route {
	logger.Debugf("Processing RouteFromIngress, ingress: %s", ingress.Name)
	metadata := *FromObjectMeta("Route", &ingress.ObjectMeta)
	// todo re-implement this!!!
	var routeSpec RouteSpec
	if len(ingress.Spec.Rules) > 0 && ingress.Spec.Rules[0].HTTP != nil && len(ingress.Spec.Rules[0].HTTP.Paths) > 0 {
		target := Target{Name: ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Name}
		portNumber := ingress.Spec.Rules[0].HTTP.Paths[0].Backend.Service.Port.Number
		port := RoutePort{TargetPort: portNumber}
		routeSpec = RouteSpec{
			Service:          target,
			Port:             port,
			PathType:         string(*ingress.Spec.Rules[0].HTTP.Paths[0].PathType),
			Path:             ingress.Spec.Rules[0].HTTP.Paths[0].Path,
			Host:             ingress.Spec.Rules[0].Host,
			IngressClassName: ingress.Spec.IngressClassName,
		}
	}
	return &Route{Spec: routeSpec, Metadata: metadata}
}

func RouteListFromIngressList(ingressList []*v1beta1.Ingress) ([]Route, map[string]struct{}) {
	badRouteList := make(map[string]struct{})
	result := make([]Route, 0)
	for _, srcIngress := range ingressList {
		route := RouteFromIngress(srcIngress)
		if route != nil {
			result = append(result, *route)
		} else {
			badRouteList[srcIngress.Name] = struct{}{}
		}
	}
	return result, badRouteList
}

func RouteListFromIngressListNetworkingV1(ingressList []*networkingV1.Ingress) ([]Route, map[string]struct{}) {
	badRouteList := make(map[string]struct{})
	result := make([]Route, 0)
	for _, srcIngress := range ingressList {
		route := RouteFromIngressNetworkingV1(srcIngress)
		if route != nil {
			result = append(result, *route)
		} else {
			badRouteList[srcIngress.Name] = struct{}{}
		}
	}
	return result, badRouteList
}

func RouteFromOsRoute(route *v1.Route) *Route {
	defer func() {
		if err := recover(); err != nil {
			out, _ := WriteContext()
			logger.Error("panic occurred: %s with route:%s error:%s", out, route.Name, err)
		}
	}()
	logger.Debugf("Processing RouteFromOsRoute, Route: %s", route.Name)
	metadata := NewMetadata("Route", route.Name, route.Namespace,
		string(route.UID), route.Generation, route.ResourceVersion,
		route.Annotations, route.Labels)
	target := Target{Name: route.Spec.To.Name}
	port := RoutePort{}
	if route.Spec.Port != nil {
		port.TargetPort = route.Spec.Port.TargetPort.IntVal
	}
	routeSpec := RouteSpec{
		Port:    port,
		Service: target,
		Path:    route.Spec.Path,
		Host:    route.Spec.Host,
	}
	return &Route{Spec: routeSpec, Metadata: metadata}
}

func RouteListFromOsRouteList(osRouteList []*v1.Route) ([]Route, map[string]bool) {
	badRouteList := make(map[string]bool)
	result := make([]Route, 0)
	for _, osRoute := range osRouteList {
		route := RouteFromOsRoute(osRoute)
		if route != nil {
			result = append(result, *route)
		} else {
			badRouteList[osRoute.Name] = true
		}
	}
	return result, badRouteList
}

func NewRouteFromInterface(object any) *Route {
	metadataObj := object.(map[string]any)["metadata"]
	metadata := NewMetadataFromInterface("Route", metadataObj)
	specObj := object.(map[string]any)["spec"].(map[string]any)
	targetObject := specObj["to"].(map[string]any)
	serviceName := targetObject["name"].(string)
	target := Target{Name: serviceName}
	routeSpec := RouteSpec{
		Service: target,
		Host:    specObj["host"].(string),
	}
	if path := specObj["path"]; path != nil {
		routeSpec.Path = specObj["path"].(string)
	}
	return &Route{Spec: routeSpec, Metadata: metadata}
}

func (route Route) GetMetadata() Metadata {
	return route.Metadata
}
