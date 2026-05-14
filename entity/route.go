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
	httpRoute := &gatewayv1.HTTPRoute{
		ObjectMeta: *route.Metadata.ToObjectMeta(),
	}

	annotations := route.Metadata.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	ns := gatewayv1.Namespace(gatewayNamespace)
	httpRoute.Spec.ParentRefs = []gatewayv1.ParentReference{
		{
			Name:      gatewayv1.ObjectName(gatewayName),
			Namespace: &ns,
		},
	}

	httpRoute.Spec.Hostnames = []gatewayv1.Hostname{gatewayv1.Hostname(route.Spec.Host)}

	pathType := gatewayv1.PathMatchPathPrefix
	if route.Spec.PathType != "" {
		switch route.Spec.PathType {
		case "Exact":
			pathType = gatewayv1.PathMatchExact
		case "Prefix", "ImplementationSpecific":
			pathType = gatewayv1.PathMatchPathPrefix
		default:
			pathType = gatewayv1.PathMatchPathPrefix
		}
	}

	path := route.Spec.Path
	if path == "" {
		path = "/"
	}

	port := gatewayv1.PortNumber(route.Spec.Port.TargetPort)
	if port == 0 {
		port = 8080
	}

	var filters []gatewayv1.HTTPRouteFilter

	if enableCors, exists := annotations["nginx.ingress.kubernetes.io/enable-cors"]; exists && enableCors == "true" {
		corsFilter := buildCORSFilter(annotations)
		if corsFilter != nil {
			filters = append(filters, *corsFilter)
		}
	}

	if snippet, exists := annotations["nginx.ingress.kubernetes.io/configuration-snippet"]; exists && snippet != "" {
		logger.Warn("Ingress annotation 'configuration-snippet' found but cannot be automatically converted to HTTPRoute. " +
			"Manual conversion required using RequestHeaderModifier/ResponseHeaderModifier filters. See GatewayAPIMigration.md section 6")
	}

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

	if len(filters) > 0 {
		rule.Filters = filters
	}

	if affinity, exists := annotations["nginx.ingress.kubernetes.io/affinity"]; exists && affinity == "cookie" {
		sessionPersistence := buildSessionPersistence(annotations)
		if sessionPersistence != nil {
			rule.SessionPersistence = sessionPersistence
		}
	}

	if timeouts := buildTimeouts(annotations); timeouts != nil {
		rule.Timeouts = timeouts
	}

	httpRoute.Spec.Rules = []gatewayv1.HTTPRouteRule{rule}

	logAdditionalResourceWarnings(annotations, route.Metadata.Name)

	return httpRoute
}

func buildCORSFilter(annotations map[string]string) *gatewayv1.HTTPRouteFilter {

	logger.Warn("CORS annotation detected. Note: CORS configuration in HTTPRoute may differ between Gateway API implementations. " +
		"See GatewayAPIMigration.md section 8 for manual configuration")

	return nil
}

func buildSessionPersistence(annotations map[string]string) *gatewayv1.SessionPersistence {
	persistenceType := gatewayv1.CookieBasedSessionPersistence
	sessionPersistence := &gatewayv1.SessionPersistence{
		Type: &persistenceType,
	}

	if cookieName, exists := annotations["nginx.ingress.kubernetes.io/session-cookie-name"]; exists && cookieName != "" {
		sessionPersistence.SessionName = &cookieName
	}

	if maxAge, exists := annotations["nginx.ingress.kubernetes.io/session-cookie-max-age"]; exists && maxAge != "" {
		timeout := gatewayv1.Duration(maxAge + "s")
		sessionPersistence.AbsoluteTimeout = &timeout
	}

	return sessionPersistence
}

func buildTimeouts(annotations map[string]string) *gatewayv1.HTTPRouteTimeouts {
	var hasTimeouts bool
	timeoutKeys := []string{
		"nginx.ingress.kubernetes.io/proxy-read-timeout",
		"nginx.ingress.kubernetes.io/proxy-send-timeout",
		"nginx.ingress.kubernetes.io/proxy-connect-timeout",
	}
	for _, key := range timeoutKeys {
		if _, exists := annotations[key]; exists {
			hasTimeouts = true
			break
		}
	}

	if !hasTimeouts {
		return nil
	}

	timeouts := &gatewayv1.HTTPRouteTimeouts{}

	if readTimeout, exists := annotations["nginx.ingress.kubernetes.io/proxy-read-timeout"]; exists && readTimeout != "" {
		timeout := gatewayv1.Duration(readTimeout + "s")
		timeouts.Request = &timeout
	}

	return timeouts
}

func logAdditionalResourceWarnings(annotations map[string]string, routeName string) {
	if protocol, exists := annotations["nginx.ingress.kubernetes.io/backend-protocol"]; exists && protocol == "HTTPS" {
		logger.Warn("Route '%s': Ingress annotation 'backend-protocol: HTTPS' requires BackendTLSPolicy resource. "+
			"See GatewayAPIMigration.md section 5", routeName)
	}
	if _, exists := annotations["nginx.ingress.kubernetes.io/secure-backends"]; exists {
		logger.Warn("Route '%s': Ingress annotation 'secure-backends' requires BackendTLSPolicy resource. "+
			"See GatewayAPIMigration.md section 5", routeName)
	}

	if protocol, exists := annotations["nginx.ingress.kubernetes.io/backend-protocol"]; exists && protocol == "GRPC" {
		logger.Warn("Route '%s': Ingress annotation 'backend-protocol: GRPC' requires BackendTrafficPolicy resource with useClientProtocol. "+
			"See GatewayAPIMigration.md section 8", routeName)
	}

	if _, exists := annotations["nginx.ingress.kubernetes.io/proxy-connect-timeout"]; exists {
		logger.Warn("Route '%s': Ingress annotation 'proxy-connect-timeout' requires BackendTrafficPolicy resource. "+
			"See GatewayAPIMigration.md section 9", routeName)
	}

	if _, exists := annotations["nginx.ingress.kubernetes.io/auth-type"]; exists {
		logger.Warn("Route '%s': Ingress annotation 'auth-type' requires SecurityPolicy resource. "+
			"See GatewayAPIMigration.md section 13", routeName)
	}

	if passthrough, exists := annotations["nginx.ingress.kubernetes.io/ssl-passthrough"]; exists && passthrough == "true" {
		logger.Warn("Route '%s': Ingress annotation 'ssl-passthrough' requires TLSRoute resource instead of HTTPRoute. "+
			"See GatewayAPIMigration.md section 12", routeName)
	}

	if _, exists := annotations["nginx.ingress.kubernetes.io/app-root"]; exists {
		logger.Warn("Route '%s': Ingress annotation 'app-root' requires additional HTTPRoute rule with RequestRedirect filter. "+
			"See GatewayAPIMigration.md section 7", routeName)
	}
}

func RouteFromHTTPRouteGatewayV1(httpRoute *gatewayv1.HTTPRoute) *Route {
	logger.Debugf("Processing RouteFromHTTPRouteGatewayV1, httpRoute: %s", httpRoute.Name)
	metadata := *FromObjectMeta("Route", &httpRoute.ObjectMeta)

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

	return &Route{Spec: routeSpec, Metadata: metadata}
}
