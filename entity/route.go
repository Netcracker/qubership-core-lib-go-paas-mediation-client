package entity

import (
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	v1 "github.com/openshift/api/route/v1"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var logger = logging.GetLogger("entity_route")

const (
	AnnotationAffinity            = "nginx.ingress.kubernetes.io/affinity"
	AnnotationSessionCookieName   = "nginx.ingress.kubernetes.io/session-cookie-name"
	AnnotationSessionCookieMaxAge = "nginx.ingress.kubernetes.io/session-cookie-max-age"
	AnnotationProxyReadTimeout    = "nginx.ingress.kubernetes.io/proxy-read-timeout"
	AnnotationProxySendTimeout    = "nginx.ingress.kubernetes.io/proxy-send-timeout"
	AnnotationProxyConnectTimeout = "nginx.ingress.kubernetes.io/proxy-connect-timeout"
)

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

func (route Route) ToHTTPRoute(gatewayNamespace, gatewayName string) *gatewayv1.HTTPRoute {
	annotations := normalizeRouteAnnotations(route.Metadata.Annotations)

	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: *route.Metadata.ToObjectMeta(),
	}

	ns := gatewayv1.Namespace(gatewayNamespace)
	httpRoute.Spec.ParentRefs = []gatewayv1.ParentReference{
		{
			Name:      gatewayv1.ObjectName(gatewayName),
			Namespace: &ns,
		},
	}

	httpRoute.Spec.Hostnames = []gatewayv1.Hostname{gatewayv1.Hostname(route.Spec.Host)}
	httpRoute.Spec.Rules = []gatewayv1.HTTPRouteRule{buildHTTPRouteRule(route, annotations)}

	return httpRoute
}

func normalizeRouteAnnotations(annotations map[string]string) map[string]string {
	if annotations == nil {
		return make(map[string]string)
	}
	return annotations
}

func pathMatchTypeFromRoute(pathType string) gatewayv1.PathMatchType {
	if pathType == "" {
		return gatewayv1.PathMatchPathPrefix
	}
	switch pathType {
	case "Exact":
		return gatewayv1.PathMatchExact
	case "Prefix", "ImplementationSpecific":
		return gatewayv1.PathMatchPathPrefix
	default:
		return gatewayv1.PathMatchPathPrefix
	}
}

func routePathValue(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func routeBackendPort(targetPort int32) gatewayv1.PortNumber {
	if targetPort == 0 {
		return 8080
	}
	return gatewayv1.PortNumber(targetPort)
}

func buildHTTPRouteRule(route Route, annotations map[string]string) gatewayv1.HTTPRouteRule {
	pathType := pathMatchTypeFromRoute(route.Spec.PathType)
	path := routePathValue(route.Spec.Path)
	port := routeBackendPort(route.Spec.Port.TargetPort)

	rule := gatewayv1.HTTPRouteRule{
		Matches: []gatewayv1.HTTPRouteMatch{
			{
				Path: &gatewayv1.HTTPPathMatch{
					Type:  &pathType,
					Value: &path,
				},
			},
		},
		BackendRefs: []gatewayv1.HTTPBackendRef{
			{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(route.Spec.Service.Name),
						Port: &port,
					},
				},
			},
		},
	}

	if affinity, exists := annotations[AnnotationAffinity]; exists && affinity == "cookie" {
		if sessionPersistence := buildSessionPersistence(annotations); sessionPersistence != nil {
			rule.SessionPersistence = sessionPersistence
		}
	}

	if timeouts := buildTimeouts(annotations); timeouts != nil {
		rule.Timeouts = timeouts
	}

	return rule
}

func buildSessionPersistence(annotations map[string]string) *gatewayv1.SessionPersistence {
	persistenceType := gatewayv1.CookieBasedSessionPersistence
	sessionPersistence := &gatewayv1.SessionPersistence{
		Type: &persistenceType,
	}

	if cookieName := annotations["nginx.ingress.kubernetes.io/session-cookie-name"]; cookieName != "" {
		sessionPersistence.SessionName = &cookieName
	}

	if maxAge := annotations["nginx.ingress.kubernetes.io/session-cookie-max-age"]; maxAge != "" {
		timeout := gatewayv1.Duration(maxAge + "s")
		sessionPersistence.AbsoluteTimeout = &timeout
	}

	return sessionPersistence
}

func buildTimeouts(annotations map[string]string) *gatewayv1.HTTPRouteTimeouts {
	if connectTimeout, exists := annotations[AnnotationProxyConnectTimeout]; exists && connectTimeout != "" {
		logger.Warn("annotation %s=%s requires BackendTrafficPolicy and is not applied to HTTPRoute", AnnotationProxyConnectTimeout, connectTimeout)
	}

	requestTimeoutValue := ""
	if readTimeout, exists := annotations[AnnotationProxyReadTimeout]; exists && readTimeout != "" {
		requestTimeoutValue = readTimeout
	} else if sendTimeout, exists := annotations[AnnotationProxySendTimeout]; exists && sendTimeout != "" {
		requestTimeoutValue = sendTimeout
	}

	if requestTimeoutValue == "" {
		return nil
	}

	timeout := gatewayv1.Duration(requestTimeoutValue + "s")
	return &gatewayv1.HTTPRouteTimeouts{Request: &timeout}
}

func RouteFromHTTPRouteGatewayV1(httpRoute *gatewayv1.HTTPRoute) *Route {
	logger.Debugf("Processing RouteFromHTTPRouteGatewayV1, httpRoute: %s", httpRoute.Name)

	var routeSpec RouteSpec
	if len(httpRoute.Spec.Rules) > 0 &&
		len(httpRoute.Spec.Rules[0].BackendRefs) > 0 &&
		len(httpRoute.Spec.Rules[0].Matches) > 0 {

		backendRef := httpRoute.Spec.Rules[0].BackendRefs[0]
		match := httpRoute.Spec.Rules[0].Matches[0]

		target := Target{Name: string(backendRef.Name)}

		var port int32
		if backendRef.Port != nil {
			port = int32(*backendRef.Port)
		}

		var pathType string
		var path string
		if match.Path != nil {
			if match.Path.Type != nil {
				pathType = string(*match.Path.Type)
			}
			if match.Path.Value != nil {
				path = *match.Path.Value
			}
		}

		var host string
		if len(httpRoute.Spec.Hostnames) > 0 {
			host = string(httpRoute.Spec.Hostnames[0])
		}

		routeSpec = RouteSpec{
			Service:  target,
			Port:     RoutePort{TargetPort: port},
			PathType: pathType,
			Path:     path,
			Host:     host,
		}
	}

	metadata := *FromObjectMeta("Route", &httpRoute.ObjectMeta)
	return &Route{Spec: routeSpec, Metadata: metadata}
}
