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
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

var BG2IngressClassName = "bg.mesh.netcracker.com"

func (kube *Kubernetes) CreateRoute(ctx context.Context, route *entity.Route, namespace string) (*entity.Route, error) {
	if kube.UseNetworkingV1Ingress {
		ingress := route.ToIngressNetworkingV1()
		kube.modifyIngressClassForBG2(ingress)
		createdIngress, e := kube.getNetworkingV1Client().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
		if e != nil {
			logger.ErrorC(ctx, "Error to create ingress: %+v", e)
			return nil, e
		}
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
		kube.modifyIngressClassForBG2(ingress)
		createdIngress, e := kube.getExtensionsV1Client().Ingresses(namespace).Create(ctx, ingress, v1.CreateOptions{})
		if e != nil {
			logger.ErrorC(ctx, "Error to create ingress: %+v", e)
			return nil, e
		}
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
	if kube.UseNetworkingV1Ingress {
		originalIngress, e := kube.getNetworkingV1Client().Ingresses(namespace).Get(ctx, route.Name, v1.GetOptions{})
		if e != nil {
			if paasErrors.IsNotFound(e) {
				logger.WarnC(ctx, "Ingress %s not found. Creating new", route.Name)
				return kube.CreateRoute(ctx, route, namespace)
			}
			logger.ErrorC(ctx, "Error to get ingress before update: %+v", e)
			return nil, e
		}
		ingressToUpdate := route.ToIngressNetworkingV1()
		ingressToUpdate.ResourceVersion = originalIngress.ResourceVersion
		if className := ingressToUpdate.Spec.IngressClassName; className == nil || *className == "" {
			ingressToUpdate.Spec.IngressClassName = originalIngress.Spec.IngressClassName
		}
		kube.modifyIngressClassForBG2(ingressToUpdate)
		updatedIngress, e := kube.getNetworkingV1Client().Ingresses(namespace).Update(ctx, ingressToUpdate, v1.UpdateOptions{})
		if e != nil {
			logger.ErrorC(ctx, "Error to update ingress: %+v", e)
			return nil, e
		}
		ingressNetworkingV1 := entity.RouteFromIngressNetworkingV1(updatedIngress)
		if kube.Cache.Ingresses != nil && ingressNetworkingV1 != nil {
			_, e := kube.Cache.Ingresses.Set(ctx, *ingressNetworkingV1)
			if e != nil {
				return nil, fmt.Errorf("faield to place ingress into cache: %w", e)
			}
		}
		return ingressNetworkingV1, nil
	} else {
		originalIngress, e := kube.getExtensionsV1Client().Ingresses(namespace).Get(ctx, route.Name, v1.GetOptions{})
		if e != nil {
			if paasErrors.IsNotFound(e) {
				logger.WarnC(ctx, "Ingress %s not found. Creating new", route.Name)
				return kube.CreateRoute(ctx, route, namespace)
			}
			logger.ErrorC(ctx, "Error to get ingress before update: %+v", e)
			return nil, e
		}
		ingressToUpdate := route.ToIngress()
		ingressToUpdate.ResourceVersion = originalIngress.ResourceVersion
		kube.modifyIngressClassForBG2(ingressToUpdate)
		updatedIngress, e := kube.getExtensionsV1Client().Ingresses(namespace).Update(ctx, ingressToUpdate, v1.UpdateOptions{})
		if e != nil {
			logger.ErrorC(ctx, "Error to update ingress: %+v", e)
			return nil, e
		}
		routeFromIngress := entity.RouteFromIngress(updatedIngress)
		if kube.Cache.Ingresses != nil && routeFromIngress != nil {
			_, e := kube.Cache.Ingresses.Set(ctx, *routeFromIngress)
			if e != nil {
				return nil, fmt.Errorf("faield to place ingress into cache: %w", e)
			}
		}
		return routeFromIngress, nil
	}
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
	if kube.UseNetworkingV1Ingress {
		err := kube.getNetworkingV1Client().
			Ingresses(namespace).
			Delete(ctx, routeName, v1.DeleteOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error while deleting a ingress=%s from kubernetes: %+v", routeName, err)
			return err
		}
		if kube.Cache.Ingresses != nil {
			kube.Cache.Ingresses.Delete(ctx, namespace, routeName)
		}
		return nil
	} else {
		err := kube.getExtensionsV1Client().
			Ingresses(namespace).
			Delete(ctx, routeName, v1.DeleteOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error while deleting a ingress=%s from kubernetes: %+v", routeName, err)
			return err
		}
		if kube.Cache.Ingresses != nil {
			kube.Cache.Ingresses.Delete(ctx, namespace, routeName)
		}
		return nil
	}
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
				route := entity.RouteFromHTTPRoute(&item)
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
