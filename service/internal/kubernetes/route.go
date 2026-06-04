package kubernetes

import (
	"cmp"
	"context"
	"fmt"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	"k8s.io/api/extensions/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var BG2IngressClassName = "bg.mesh.netcracker.com"

const IgnoreApiConverterAnnotation = "gateway-api-converter.netcracker.com/ignore"
const errPlaceIngressIntoCache = "failed to place ingress into cache: %w"
const IngressCreated = "Ingress created: %s"
const RouteDeleteFailed = "Route delete failed: %s"

const (
	AnnotationBackendProtocol   = "nginx.ingress.kubernetes.io/backend-protocol"
	AnnotationSecureBackends    = "nginx.ingress.kubernetes.io/secure-backends"
	AnnotationAuthType          = "nginx.ingress.kubernetes.io/auth-type"
	AnnotationSSLPassthrough    = "nginx.ingress.kubernetes.io/ssl-passthrough"
	AnnotationConfigSnippet     = "nginx.ingress.kubernetes.io/configuration-snippet"
	AnnotationUpstreamVhost     = "nginx.ingress.kubernetes.io/upstream-vhost"
	AnnotationProxyRedirectFrom = "nginx.ingress.kubernetes.io/proxy-redirect-from"
	AnnotationProxyRedirectTo   = "nginx.ingress.kubernetes.io/proxy-redirect-to"
	EnvoyExtensionWarning       = "requires EnvoyExtensionPolicy with Lua"
	BackendTLSWarning           = "requires BackendTLSPolicy"
	BackendTlsOrTrafficWarning  = "requires BackendTLSPolicy (for HTTPS) or BackendTrafficPolicy (for GRPC)."
	SecurityPolicyWarning       = "requires SecurityPolicy"
	TlsRouteWarning             = "requires TLSRoute instead of HTTPRoute"
	ConfigSnippetWarning        = "requires manual conversion using RequestHeaderModifier/ResponseHeaderModifier filters"
)

const (
	dualModeRouteUpdateHint = " - try using Update endpoint"
	dualModeStatusFormat    = "%s, %s"
	routeStatusCreated      = "created"
	routeStatusUpdated      = "updated"
)

type routeResourceResult struct {
	route  *entity.Route
	status string
	err    error
}

func (kube *Kubernetes) CreateRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if !kube.GatewaySystem.IsRouteCreationAllowed() {
		return nil, kube.GatewaySystem.RouteCreationNotAllowedError()
	}

	useGatewayAPI := kube.GatewaySystem.IsGatewayAPIEnabled()
	useLegacyIngress := kube.GatewaySystem.IsIngressEnabled()

	var httpRouteRes, ingressRes routeResourceResult

	if useGatewayAPI {
		routeResult, err := kube.createHTTPRoute(ctx, route, namespace)
		httpRouteRes = routeResourceResult{route: routeResult, status: routeStatusCreated, err: err}
		if err == nil {
			logger.InfoC(ctx, "HTTPRoute created: %s", route.Name)
		}
	}

	if useLegacyIngress {
		routeResult, err := kube.createRouteLegacyIngress(ctx, route, namespace)
		ingressRes = routeResourceResult{route: routeResult, status: routeStatusCreated, err: err}
		if err == nil {
			logger.InfoC(ctx, IngressCreated, route.Name)
		}
	}

	return resolveRouteResult(useGatewayAPI, useLegacyIngress, httpRouteRes, ingressRes)
}

func resolveRouteResult(
	useGatewayAPI, useLegacyIngress bool,
	httpRouteRes, ingressRes routeResourceResult,
) (*entity.Route, error) {
	if useGatewayAPI && useLegacyIngress {
		if httpRouteRes.err == nil && ingressRes.err == nil {
			return finishRouteOperation(pickDualModeRouteResult(ingressRes, httpRouteRes), formatDualModeRouteStatus(httpRouteRes, ingressRes))
		}

		return nil, dualModeRouteError(httpRouteRes, ingressRes)
	}

	if useGatewayAPI {
		if httpRouteRes.err != nil {
			return nil, fmt.Errorf("httproute: error: %w", httpRouteRes.err)
		}
		return finishRouteOperation(httpRouteRes.route, formatRouteResourceStatus("httproute", httpRouteRes))
	}

	if ingressRes.err != nil {
		return nil, fmt.Errorf("ingress: error: %w", ingressRes.err)
	}
	return finishRouteOperation(ingressRes.route, formatRouteResourceStatus("ingress", ingressRes))
}

func dualModeRouteError(httpRouteRes, ingressRes routeResourceResult) error {
	if httpRouteRes.err == nil && ingressRes.err == nil {
		return nil
	}

	if httpRouteRes.err != nil && ingressRes.err != nil {
		return fmt.Errorf("httproute: error: %w, ingress: error: %w", httpRouteRes.err, ingressRes.err)
	}

	failedErr := cmp.Or(httpRouteRes.err, ingressRes.err)
	status := formatDualModeRouteStatusSummary(httpRouteRes, ingressRes)
	return fmt.Errorf("%s%s: %w", status, dualModeRouteUpdateHint, failedErr)
}

func finishRouteOperation(route *entity.Route, status string) (*entity.Route, error) {
	logger.Info("Route operation completed: %s", status)
	return route, nil
}

func pickDualModeRouteResult(ingressRes, httpRouteRes routeResourceResult) *entity.Route {
	if ingressRes.route != nil {
		return ingressRes.route
	}
	return httpRouteRes.route
}

func formatDualModeRouteStatus(httpRouteRes, ingressRes routeResourceResult) string {
	return fmt.Sprintf(dualModeStatusFormat,
		formatRouteResourceStatus("httproute", httpRouteRes),
		formatRouteResourceStatus("ingress", ingressRes),
	)
}

func formatDualModeRouteStatusSummary(httpRouteRes, ingressRes routeResourceResult) string {
	return fmt.Sprintf(dualModeStatusFormat,
		formatRouteResourceStatusSummary("httproute", httpRouteRes),
		formatRouteResourceStatusSummary("ingress", ingressRes),
	)
}

func formatRouteResourceStatus(name string, res routeResourceResult) string {
	if res.err != nil {
		return fmt.Sprintf("%s: error: %v", name, res.err)
	}
	return fmt.Sprintf("%s: %s", name, res.status)
}

func formatRouteResourceStatusSummary(name string, res routeResourceResult) string {
	if res.err != nil {
		return fmt.Sprintf("%s: error", name)
	}
	return fmt.Sprintf("%s: %s", name, res.status)
}

func isDeleteFailure(err error) bool {
	return err != nil && !paasErrors.IsNotFound(err)
}

func formatDualModeDeleteRouteStatus(httpRouteErr, ingressErr error) string {
	return fmt.Sprintf(dualModeStatusFormat,
		routeResourceDeleteStatus("httproute", httpRouteErr),
		routeResourceDeleteStatus("ingress", ingressErr),
	)
}

func routeResourceDeleteStatus(name string, err error) string {
	if err == nil {
		return name + ": deleted"
	}

	return fmt.Sprintf("%s: error: %v", name, err)
}

func (kube *Kubernetes) createRouteLegacyIngress(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if kube.UseNetworkingV1Ingress {
		return kube.createNetworkingV1IngressRoute(ctx, route, namespace)
	}
	return kube.createExtensionsV1IngressRoute(ctx, route, namespace)
}

func (kube *Kubernetes) cacheIngressRoute(ctx context.Context, route *entity.Route) error {
	if kube.Cache.Ingresses == nil || route == nil {
		return nil
	}
	if _, err := kube.Cache.Ingresses.Set(ctx, *route); err != nil {
		return fmt.Errorf(errPlaceIngressIntoCache, err)
	}
	return nil
}

func (kube *Kubernetes) createNetworkingV1IngressRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	ingress := route.ToIngressNetworkingV1()
	kube.configureIngress(ingress)

	createdIngress, err := kube.getNetworkingV1Client().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error to create ingress: %+v", err)
		return nil, err
	}
	ingressRoute := entity.RouteFromIngressNetworkingV1(createdIngress)
	if err := kube.cacheIngressRoute(ctx, ingressRoute); err != nil {
		return nil, err
	}
	return ingressRoute, nil
}

func (kube *Kubernetes) createExtensionsV1IngressRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	ingress := route.ToIngress()
	kube.configureIngress(ingress)

	createdIngress, err := kube.getExtensionsV1Client().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error to create ingress: %+v", err)
		return nil, err
	}
	ingressRoute := entity.RouteFromIngress(createdIngress)
	if err := kube.cacheIngressRoute(ctx, ingressRoute); err != nil {
		return nil, err
	}
	return ingressRoute, nil
}

func (kube *Kubernetes) UpdateOrCreateRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if !kube.GatewaySystem.IsRouteCreationAllowed() {
		return nil, kube.GatewaySystem.RouteUpdateNotAllowedError()
	}

	useGatewayAPI := kube.GatewaySystem.IsGatewayAPIEnabled()
	useLegacyIngress := kube.GatewaySystem.IsIngressEnabled()

	var httpRouteRes, ingressRes routeResourceResult

	if useGatewayAPI {
		httpRouteRes = kube.upsertHTTPRoute(ctx, route, namespace)
	}

	if useLegacyIngress {
		ingressRes = kube.upsertLegacyIngress(ctx, route, namespace)
	}

	return resolveRouteResult(useGatewayAPI, useLegacyIngress, httpRouteRes, ingressRes)
}

func (kube *Kubernetes) upsertHTTPRoute(ctx context.Context, route *entity.Route, namespace string) routeResourceResult {
	originalHTTPRoute, err := kube.getGatewayV1Client().HTTPRoutes(namespace).Get(ctx, route.Name, v1.GetOptions{})
	if err != nil {
		if paasErrors.IsNotFound(err) {
			logger.WarnC(ctx, "HTTPRoute %s not found. Creating new", route.Name)
			createdHTTPRoute, createErr := kube.createHTTPRoute(ctx, route, namespace)
			if createErr == nil {
				logger.InfoC(ctx, "HTTPRoute created: %s", route.Name)
			}
			return routeResourceResult{route: createdHTTPRoute, status: routeStatusCreated, err: createErr}
		}
		logger.ErrorC(ctx, "Error to get HTTPRoute before update: %+v", err)
		return routeResourceResult{err: err}
	}

	httpRouteToUpdate := route.ToHTTPRoute(kube.GatewaySystem.Namespace, kube.GatewaySystem.Name)
	httpRouteToUpdate.ResourceVersion = originalHTTPRoute.ResourceVersion

	updatedHTTPRoute, err := kube.getGatewayV1Client().HTTPRoutes(namespace).Update(ctx, httpRouteToUpdate, v1.UpdateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error to update HTTPRoute: %+v", err)
		return routeResourceResult{err: err}
	}
	logger.InfoC(ctx, "HTTPRoute updated: %s", route.Name)

	routeFromHTTPRoute := entity.RouteFromHTTPRoute(updatedHTTPRoute)
	if kube.Cache.HTTPRoute != nil && routeFromHTTPRoute != nil {
		httpRouteEntity := entity.WrapHTTPRoute(updatedHTTPRoute)
		if _, err := kube.Cache.HTTPRoute.Set(ctx, *httpRouteEntity); err != nil {
			return routeResourceResult{err: fmt.Errorf("failed to place HTTPRoute into cache: %w", err)}
		}
	}
	return routeResourceResult{route: routeFromHTTPRoute, status: routeStatusUpdated}
}

func (kube *Kubernetes) upsertLegacyIngress(ctx context.Context, route *entity.Route, namespace string) routeResourceResult {
	if kube.UseNetworkingV1Ingress {
		return kube.upsertNetworkingV1Ingress(ctx, route, namespace)
	}
	return kube.upsertExtensionsV1Ingress(ctx, route, namespace)
}

func (kube *Kubernetes) upsertNetworkingV1Ingress(ctx context.Context, route *entity.Route, namespace string) routeResourceResult {
	originalIngress, err := kube.getNetworkingV1Client().Ingresses(namespace).Get(ctx, route.Name, v1.GetOptions{})
	if err != nil {
		if paasErrors.IsNotFound(err) {
			logger.WarnC(ctx, "Ingress %s not found. Creating new", route.Name)
			ingressResult, createErr := kube.createRouteLegacyIngress(ctx, route, namespace)
			if createErr == nil {
				logger.InfoC(ctx, IngressCreated, route.Name)
			}
			return routeResourceResult{route: ingressResult, status: routeStatusCreated, err: createErr}
		}
		logger.ErrorC(ctx, "Error to get ingress before update: %+v", err)
		return routeResourceResult{err: err}
	}

	ingressToUpdate := route.ToIngressNetworkingV1()
	ingressToUpdate.ResourceVersion = originalIngress.ResourceVersion
	if className := ingressToUpdate.Spec.IngressClassName; className == nil || *className == "" {
		ingressToUpdate.Spec.IngressClassName = originalIngress.Spec.IngressClassName
	}
	kube.configureIngress(ingressToUpdate)

	updatedIngress, err := kube.getNetworkingV1Client().Ingresses(namespace).Update(ctx, ingressToUpdate, v1.UpdateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error to update ingress: %+v", err)
		return routeResourceResult{err: err}
	}
	logger.InfoC(ctx, "Ingress updated: %s", route.Name)

	ingressNetworkingV1 := entity.RouteFromIngressNetworkingV1(updatedIngress)
	if err := kube.cacheIngressRoute(ctx, ingressNetworkingV1); err != nil {
		return routeResourceResult{err: err}
	}
	return routeResourceResult{route: ingressNetworkingV1, status: routeStatusUpdated}
}

func (kube *Kubernetes) upsertExtensionsV1Ingress(ctx context.Context, route *entity.Route, namespace string) routeResourceResult {
	originalIngress, err := kube.getExtensionsV1Client().Ingresses(namespace).Get(ctx, route.Name, v1.GetOptions{})
	if err != nil {
		if paasErrors.IsNotFound(err) {
			logger.WarnC(ctx, "Ingress %s not found. Creating new", route.Name)
			ingressResult, createErr := kube.createRouteLegacyIngress(ctx, route, namespace)
			if createErr == nil {
				logger.InfoC(ctx, IngressCreated, route.Name)
			}
			return routeResourceResult{route: ingressResult, status: routeStatusCreated, err: createErr}
		}
		logger.ErrorC(ctx, "Error to get ingress before update: %+v", err)
		return routeResourceResult{err: err}
	}

	ingressToUpdate := route.ToIngress()
	ingressToUpdate.ResourceVersion = originalIngress.ResourceVersion
	kube.configureIngress(ingressToUpdate)

	updatedIngress, err := kube.getExtensionsV1Client().Ingresses(namespace).Update(ctx, ingressToUpdate, v1.UpdateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error to update ingress: %+v", err)
		return routeResourceResult{err: err}
	}
	logger.InfoC(ctx, "Ingress updated: %s", route.Name)

	routeFromIngress := entity.RouteFromIngress(updatedIngress)
	if err := kube.cacheIngressRoute(ctx, routeFromIngress); err != nil {
		return routeResourceResult{err: err}
	}
	return routeResourceResult{route: routeFromIngress, status: routeStatusUpdated}
}

func (kube *Kubernetes) GetRoute(ctx context.Context, resourceName string, namespace string) (*entity.Route, error) {
	if kube.UseNetworkingV1Ingress {
		return GetWrapper(ctx, resourceName, namespace, kube.getNetworkingV1Client().Ingresses(namespace).Get,
			kube.Cache.Ingresses, entity.RouteFromIngressNetworkingV1)
	} else {
		return GetWrapper(ctx, resourceName, namespace, kube.getExtensionsV1Client().Ingresses(namespace).Get,
			kube.Cache.Ingresses, entity.RouteFromIngress)
	}
}

func (kube *Kubernetes) DeleteRoute(ctx context.Context, routeName, namespace string) error {
	useGatewayAPI := kube.GatewaySystem.IsGatewayAPIEnabled()
	useLegacyIngress := kube.GatewaySystem.IsIngressEnabled()

	var httpRouteErr error
	var ingressErr error

	if useGatewayAPI {
		httpRouteErr = kube.deleteRouteHTTPRoute(ctx, routeName, namespace)
	}

	if useLegacyIngress {
		ingressErr = kube.deleteRouteLegacyIngress(ctx, routeName, namespace)
	}

	return resolveDeleteRouteResult(useGatewayAPI, useLegacyIngress, routeName, httpRouteErr, ingressErr)
}

func resolveDeleteRouteResult(
	useGatewayAPI, useLegacyIngress bool,
	routeName string,
	httpRouteErr, ingressErr error,
) error {
	if useGatewayAPI && useLegacyIngress {
		status := formatDualModeDeleteRouteStatus(httpRouteErr, ingressErr)

		if paasErrors.IsNotFound(httpRouteErr) && paasErrors.IsNotFound(ingressErr) {
			logger.Warn(RouteDeleteFailed, status)
			return newRouteDeleteNotFoundError(routeName, status)
		}

		if isDeleteFailure(httpRouteErr) || isDeleteFailure(ingressErr) {
			logger.Error(RouteDeleteFailed, status)
			return dualModeDeleteError(httpRouteErr, ingressErr)
		}

		logger.Info("Route delete completed: %s", status)
		return nil
	}

	if useGatewayAPI {
		return resolveSingleResourceDeleteResult("httproute", httpRouteErr)
	}

	if useLegacyIngress {
		return resolveSingleResourceDeleteResult("ingress", ingressErr)
	}

	return nil
}

func resolveSingleResourceDeleteResult(resourceName string, err error) error {
	status := routeResourceDeleteStatus(resourceName, err)
	if err == nil {
		logger.Info("Route delete completed: %s", status)
		return nil
	}
	if isDeleteFailure(err) {
		logger.Error(RouteDeleteFailed, status)
		return fmt.Errorf("%s: error: %w", resourceName, err)
	}
	logger.Warn(RouteDeleteFailed, status)
	return err
}

func dualModeDeleteError(httpRouteErr, ingressErr error) error {
	httpFailed := isDeleteFailure(httpRouteErr)
	ingressFailed := isDeleteFailure(ingressErr)

	if httpFailed && ingressFailed {
		return fmt.Errorf("httproute: error: %w, ingress: error: %w", httpRouteErr, ingressErr)
	}
	if httpFailed {
		return fmt.Errorf("httproute: error: %w", httpRouteErr)
	}
	return fmt.Errorf("ingress: error: %w", ingressErr)
}

func newRouteDeleteNotFoundError(routeName, status string) error {
	notFound := paasErrors.NewNotFound(schema.GroupResource{Resource: "routes"}, routeName)
	notFound.ErrStatus.Message = status
	return notFound
}

func (kube *Kubernetes) deleteRouteHTTPRoute(ctx context.Context, routeName, namespace string) error {
	err := kube.getGatewayV1Client().HTTPRoutes(namespace).Delete(ctx, routeName, v1.DeleteOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error while deleting HTTPRoute=%s from kubernetes: %+v", routeName, err)
		return err
	}
	logger.InfoC(ctx, "HTTPRoute deleted: %s", routeName)
	if kube.Cache.HTTPRoute != nil {
		kube.Cache.HTTPRoute.Delete(ctx, namespace, routeName)
	}
	return nil
}

func (kube *Kubernetes) deleteRouteLegacyIngress(ctx context.Context, routeName, namespace string) error {
	var err error
	if kube.UseNetworkingV1Ingress {
		err = kube.getNetworkingV1Client().Ingresses(namespace).Delete(ctx, routeName, v1.DeleteOptions{})
	} else {
		err = kube.getExtensionsV1Client().Ingresses(namespace).Delete(ctx, routeName, v1.DeleteOptions{})
	}
	if err != nil {
		logger.ErrorC(ctx, "Error while deleting ingress=%s from kubernetes: %+v", routeName, err)
		return err
	}
	logger.InfoC(ctx, "Ingress deleted: %s", routeName)
	if kube.Cache.Ingresses != nil {
		kube.Cache.Ingresses.Delete(ctx, namespace, routeName)
	}
	return nil
}

func (kube *Kubernetes) GetRouteList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Route, error) {
	if kube.UseNetworkingV1Ingress {
		return ListWrapper(ctx, filter, kube.getNetworkingV1Client().Ingresses(namespace).List, kube.Cache.Ingresses,
			func(listObj *networkingV1.IngressList) (result []entity.Route) {
				for _, item := range listObj.Items {
					route := entity.RouteFromIngressNetworkingV1(&item)
					if route != nil {
						result = append(result, *route)
					}
				}
				return
			})
	} else {
		return ListWrapper(ctx, filter, kube.getExtensionsV1Client().Ingresses(namespace).List, kube.Cache.Ingresses,
			func(listObj *v1beta1.IngressList) (result []entity.Route) {
				for _, item := range listObj.Items {
					route := entity.RouteFromIngress(&item)
					if route != nil {
						result = append(result, *route)
					}
				}
				return
			})
	}
}

func (kube *Kubernetes) GetHttpRouteList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.HttpRoute, error) {
	return ListWrapper(ctx, filter, kube.getGatewayV1Client().HTTPRoutes(namespace).List, kube.Cache.HTTPRoute,
		func(listObj *gatewayv1.HTTPRouteList) (result []entity.HttpRoute) {
			for _, item := range listObj.Items {
				route := entity.WrapHTTPRoute(&item)
				if route != nil {
					result = append(result, *route)
				}
			}
			return
		})
}

func (kube *Kubernetes) GetGrpcRouteList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.GrpcRoute, error) {
	return ListWrapper(ctx, filter, kube.getGatewayV1Client().GRPCRoutes(namespace).List, kube.Cache.GRPCRoute,
		func(listObj *gatewayv1.GRPCRouteList) (result []entity.GrpcRoute) {
			for _, item := range listObj.Items {
				route := entity.RouteFromGRPCRoute(&item)
				if route != nil {
					result = append(result, *route)
				}
			}
			return
		})
}

func (kube *Kubernetes) GetBadRouteLists(ctx context.Context) (map[string][]string, error) {
	return kube.BadResources.Routes.ToSliceMap(), nil
}

func (kube *Kubernetes) WatchRoutes(ctx context.Context, namespace string, metaFilter filter.Meta) (*pmWatch.Handler, error) {
	if kube.UseNetworkingV1Ingress {
		return kube.WatchHandlers.IngressesNetworkingV1.Watch(ctx, namespace, metaFilter)
	} else {
		return kube.WatchHandlers.IngressesV1Beta1.Watch(ctx, namespace, metaFilter)
	}
}

func (kube *Kubernetes) WatchGatewayHTTPRoutes(ctx context.Context, namespace string, metaFilter filter.Meta) (*pmWatch.Handler, error) {
	if kube.WatchHandlers.HTTPRouteV1 == nil {
		return nil, fmt.Errorf("k8s HTTPRoute is not supported")
	}
	return kube.WatchHandlers.HTTPRouteV1.Watch(ctx, namespace, metaFilter)
}

func (kube *Kubernetes) WatchGatewayGRPCRoutes(ctx context.Context, namespace string, metaFilter filter.Meta) (*pmWatch.Handler, error) {
	if kube.WatchHandlers.GRPCRouteV1 == nil {
		return nil, fmt.Errorf("k8s GRPCRoute is not supported")
	}
	return kube.WatchHandlers.GRPCRouteV1.Watch(ctx, namespace, metaFilter)
}

func (kube *Kubernetes) configureIngress(ingress any) {
	kube.modifyIngressClassForBG2(ingress)
	if kube.GatewaySystem.IsBothGatewaySystemsEnabled() {
		kube.ignoreApiConverter(ingress)
	}
}

func applyIgnoreApiConverterAnnotation(meta *v1.ObjectMeta) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	meta.Annotations[IgnoreApiConverterAnnotation] = "true"
}

func (kube *Kubernetes) ignoreApiConverter(ingress any) {
	switch i := ingress.(type) {
	case *networkingV1.Ingress:
		applyIgnoreApiConverterAnnotation(&i.ObjectMeta)
	case *v1beta1.Ingress:
		applyIgnoreApiConverterAnnotation(&i.ObjectMeta)
	}
}

func (kube *Kubernetes) modifyIngressClassForBG2(ingress any) {
	if kube.BG2Enabled == nil || !kube.BG2Enabled() {
		return
	}
	switch i := ingress.(type) {
	case *networkingV1.Ingress:
		if i.Spec.IngressClassName == nil {
			logger.Info("Adding ingress class '%s' (BlueGreen mode) to the ingress '%s'", BG2IngressClassName, i.Name)
			i.Spec.IngressClassName = &BG2IngressClassName
		}
	case *v1beta1.Ingress:
		if i.Spec.IngressClassName == nil {
			logger.Info("Adding ingress class '%s' (BlueGreen mode) to the ingress '%s'", BG2IngressClassName, i.Name)
			i.Spec.IngressClassName = &BG2IngressClassName
		}
	}
}

func (kube *Kubernetes) validateAnnotationsForGatewayAPI(annotations map[string]string) error {
	criticalAnnotations := map[string]string{
		AnnotationBackendProtocol:   BackendTlsOrTrafficWarning,
		AnnotationSecureBackends:    BackendTLSWarning,
		AnnotationAuthType:          SecurityPolicyWarning,
		AnnotationSSLPassthrough:    TlsRouteWarning,
		AnnotationConfigSnippet:     ConfigSnippetWarning,
		AnnotationUpstreamVhost:     EnvoyExtensionWarning,
		AnnotationProxyRedirectFrom: EnvoyExtensionWarning,
		AnnotationProxyRedirectTo:   EnvoyExtensionWarning,
	}

	var fieldErrors field.ErrorList
	for key, message := range criticalAnnotations {
		if val := annotations[key]; val != "" {
			fieldErrors = append(fieldErrors, field.Invalid(
				field.NewPath("metadata", "annotations").Key(key),
				val,
				fmt.Sprintf("not supported for HTTPRoute creation: %s", message),
			))
		}
	}
	if len(fieldErrors) == 0 {
		return nil
	}
	return paasErrors.NewInvalid(
		schema.GroupKind{Group: gatewayv1.GroupName, Kind: "HTTPRoute"},
		"",
		fieldErrors,
	)
}

func (kube *Kubernetes) createHTTPRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if err := kube.validateAnnotationsForGatewayAPI(route.Metadata.Annotations); err != nil {
		return nil, err
	}

	httpRoute := route.ToHTTPRoute(
		kube.GatewaySystem.Namespace,
		kube.GatewaySystem.Name,
	)

	createdHTTPRoute, err := kube.getGatewayV1Client().HTTPRoutes(namespace).Create(ctx, httpRoute, v1.CreateOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error to create HTTPRoute: %+v", err)
		return nil, err
	}

	routeFromHTTPRoute := entity.RouteFromHTTPRoute(createdHTTPRoute)
	if kube.Cache.HTTPRoute != nil && routeFromHTTPRoute != nil {
		httpRouteEntity := entity.WrapHTTPRoute(createdHTTPRoute)
		_, err := kube.Cache.HTTPRoute.Set(ctx, *httpRouteEntity)
		if err != nil {
			return nil, fmt.Errorf("failed to place HTTPRoute into cache: %w", err)
		}
	}

	return routeFromHTTPRoute, nil
}
