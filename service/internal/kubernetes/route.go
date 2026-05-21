package kubernetes

import (
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

func (kube *Kubernetes) CreateRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if !kube.GatewaySystem.IsRouteCreationAllowed() {
		return nil, kube.GatewaySystem.RouteCreationNotAllowedError()
	}

	useGatewayAPI := kube.GatewaySystem.ShouldUseGatewayAPI()
	useLegacyIngress := kube.GatewaySystem.ShouldCreateLegacyIngress()

	var result *entity.Route
	var err error

	if useGatewayAPI {
		result, err = kube.createHTTPRoute(ctx, route, namespace)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTPRoute: %w", err)
		}
		logger.InfoC(ctx, "HTTPRoute created: %s", route.Name)
	}

	if !useLegacyIngress {
		return result, nil
	}

	if kube.UseNetworkingV1Ingress {
		ingress := route.ToIngressNetworkingV1()
		kube.configureIngress(ingress)

		createdIngress, e := kube.getNetworkingV1Client().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
		if e != nil {
			logger.ErrorC(ctx, "Error to create ingress: %+v", e)
			return nil, e
		}
		logger.InfoC(ctx, "Ingress created: %s", route.Name)
		ingressNetworkingV1 := entity.RouteFromIngressNetworkingV1(createdIngress)
		if kube.Cache.Ingresses != nil && ingressNetworkingV1 != nil {
			_, e := kube.Cache.Ingresses.Set(ctx, *ingressNetworkingV1)
			if e != nil {
				return nil, fmt.Errorf("faield to place ingress into cache: %w", e)
			}
		}
		return ingressNetworkingV1, nil
	} else {
		ingress := route.ToIngress()
		kube.configureIngress(ingress)

		createdIngress, e := kube.getExtensionsV1Client().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
		if e != nil {
			logger.ErrorC(ctx, "Error to create ingress: %+v", e)
			return nil, e
		}
		logger.InfoC(ctx, "Ingress created: %s", route.Name)
		routeFromIngress := entity.RouteFromIngress(createdIngress)
		if kube.Cache.Ingresses != nil && routeFromIngress != nil {
			_, e := kube.Cache.Ingresses.Set(ctx, *routeFromIngress)
			if e != nil {
				return nil, fmt.Errorf("faield to place ingress into cache: %w", e)
			}
		}
		return routeFromIngress, nil
	}
}

func (kube *Kubernetes) UpdateOrCreateRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if !kube.GatewaySystem.IsRouteCreationAllowed() {
		return nil, kube.GatewaySystem.RouteUpdateNotAllowedError()
	}

	useGatewayAPI := kube.GatewaySystem.ShouldUseGatewayAPI()
	useLegacyIngress := kube.GatewaySystem.ShouldCreateLegacyIngress()

	var result *entity.Route
	var httpRouteExists bool
	var ingressExists bool

	if useGatewayAPI {
		originalHTTPRoute, err := kube.getGatewayV1Client().HTTPRoutes(namespace).Get(ctx, route.Name, v1.GetOptions{})
		if err != nil {
			if paasErrors.IsNotFound(err) {
				logger.WarnC(ctx, "HTTPRoute %s not found. Creating new", route.Name)
				return kube.CreateRoute(ctx, route, namespace)
			}
			logger.ErrorC(ctx, "Error to get HTTPRoute before update: %+v", err)
			return nil, err
		}
		httpRouteExists = true
		httpRouteToUpdate := route.ToHTTPRoute(kube.GatewaySystem.Namespace, kube.GatewaySystem.Name)
		httpRouteToUpdate.ResourceVersion = originalHTTPRoute.ResourceVersion

		updatedHTTPRoute, err := kube.getGatewayV1Client().HTTPRoutes(namespace).Update(ctx, httpRouteToUpdate, v1.UpdateOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error to update HTTPRoute: %+v", err)
			return nil, err
		}
		logger.InfoC(ctx, "HTTPRoute updated: %s", route.Name)

		routeFromHTTPRoute := entity.RouteFromHTTPRoute(updatedHTTPRoute)
		if kube.Cache.HTTPRoute != nil && routeFromHTTPRoute != nil {
			httpRouteEntity := entity.WrapHTTPRoute(updatedHTTPRoute)
			_, err := kube.Cache.HTTPRoute.Set(ctx, *httpRouteEntity)
			if err != nil {
				return nil, fmt.Errorf("failed to place HTTPRoute into cache: %w", err)
			}
		}
		result = routeFromHTTPRoute
	}

	if useLegacyIngress {
		if kube.UseNetworkingV1Ingress {
			originalIngress, err := kube.getNetworkingV1Client().Ingresses(namespace).Get(ctx, route.Name, v1.GetOptions{})
			if err != nil {
				if paasErrors.IsNotFound(err) {
					logger.WarnC(ctx, "Ingress %s not found. Creating new", route.Name)
					return kube.CreateRoute(ctx, route, namespace)
				}
				logger.ErrorC(ctx, "Error to get ingress before update: %+v", err)
				return nil, err
			}

			ingressExists = true
			ingressToUpdate := route.ToIngressNetworkingV1()
			ingressToUpdate.ResourceVersion = originalIngress.ResourceVersion
			if className := ingressToUpdate.Spec.IngressClassName; className == nil || *className == "" {
				ingressToUpdate.Spec.IngressClassName = originalIngress.Spec.IngressClassName
			}
			kube.configureIngress(ingressToUpdate)

			updatedIngress, err := kube.getNetworkingV1Client().Ingresses(namespace).Update(ctx, ingressToUpdate, v1.UpdateOptions{})
			if err != nil {
				logger.ErrorC(ctx, "Error to update ingress: %+v", err)
				return nil, err
			}
			logger.InfoC(ctx, "Ingress updated: %s", route.Name)

			ingressNetworkingV1 := entity.RouteFromIngressNetworkingV1(updatedIngress)
			if kube.Cache.Ingresses != nil && ingressNetworkingV1 != nil {
				_, err := kube.Cache.Ingresses.Set(ctx, *ingressNetworkingV1)
				if err != nil {
					return nil, fmt.Errorf("faield to place ingress into cache: %w", err)
				}
			}
			if result == nil {
				result = ingressNetworkingV1
			}
		} else {
			originalIngress, err := kube.getExtensionsV1Client().Ingresses(namespace).Get(ctx, route.Name, v1.GetOptions{})
			if err != nil {
				if paasErrors.IsNotFound(err) {
					logger.WarnC(ctx, "Ingress %s not found. Creating new", route.Name)
					return kube.CreateRoute(ctx, route, namespace)
				}
				logger.ErrorC(ctx, "Error to get ingress before update: %+v", err)
				return nil, err
			}
			ingressExists = true
			ingressToUpdate := route.ToIngress()
			ingressToUpdate.ResourceVersion = originalIngress.ResourceVersion
			kube.configureIngress(ingressToUpdate)

			updatedIngress, err := kube.getExtensionsV1Client().Ingresses(namespace).Update(ctx, ingressToUpdate, v1.UpdateOptions{})
			if err != nil {
				logger.ErrorC(ctx, "Error to update ingress: %+v", err)
				return nil, err
			}
			logger.InfoC(ctx, "Ingress updated: %s", route.Name)

			routeFromIngress := entity.RouteFromIngress(updatedIngress)
			if kube.Cache.Ingresses != nil && routeFromIngress != nil {
				_, err := kube.Cache.Ingresses.Set(ctx, *routeFromIngress)
				if err != nil {
					return nil, fmt.Errorf("faield to place ingress into cache: %w", err)
				}
			}
			if result == nil {
				result = routeFromIngress
			}
		}
	}

	if !httpRouteExists && !ingressExists {
		logger.WarnC(ctx, "Route %s not found. Creating new", route.Name)
		return kube.CreateRoute(ctx, route, namespace)
	}

	return result, nil
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

func (kube *Kubernetes) DeleteRoute(ctx context.Context, routeName string, namespace string) error {
	var firstErr error

	if kube.GatewaySystem.ShouldUseGatewayAPI() {
		if err := kube.deleteRouteHTTPRoute(ctx, routeName, namespace); err != nil {
			firstErr = err
		}
	}

	if kube.GatewaySystem.ShouldCreateLegacyIngress() {
		if err := kube.deleteRouteLegacyIngress(ctx, routeName, namespace); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (kube *Kubernetes) deleteRouteHTTPRoute(ctx context.Context, routeName, namespace string) error {
	err := kube.getGatewayV1Client().HTTPRoutes(namespace).Delete(ctx, routeName, v1.DeleteOptions{})
	if err == nil {
		logger.InfoC(ctx, "HTTPRoute deleted: %s", routeName)
		if kube.Cache.HTTPRoute != nil {
			kube.Cache.HTTPRoute.Delete(ctx, namespace, routeName)
		}
		return nil
	}
	if paasErrors.IsNotFound(err) {
		return nil
	}
	logger.ErrorC(ctx, "Error while deleting HTTPRoute=%s from kubernetes: %+v", routeName, err)
	return err
}

func (kube *Kubernetes) deleteRouteLegacyIngress(ctx context.Context, routeName, namespace string) error {
	var err error
	if kube.UseNetworkingV1Ingress {
		err = kube.getNetworkingV1Client().Ingresses(namespace).Delete(ctx, routeName, v1.DeleteOptions{})
	} else {
		err = kube.getExtensionsV1Client().Ingresses(namespace).Delete(ctx, routeName, v1.DeleteOptions{})
	}
	if err == nil {
		logger.InfoC(ctx, "Ingress deleted: %s", routeName)
		if kube.Cache.Ingresses != nil {
			kube.Cache.Ingresses.Delete(ctx, namespace, routeName)
		}
		return nil
	}
	if paasErrors.IsNotFound(err) {
		return nil
	}
	logger.ErrorC(ctx, "Error while deleting ingress=%s from kubernetes: %+v", routeName, err)
	return err
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
