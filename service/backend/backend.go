package backend

import (
	cm "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	appsv1client "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	projectv1client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	k8s "k8s.io/client-go/kubernetes"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

type (
	KubernetesInterface  = k8s.Interface
	CertmanagerInterface = cm.Interface
	GatewayInterface     = gatewayclient.Interface
)

type KubernetesApi struct {
	KubernetesInterface
	CertmanagerInterface
	GatewayInterface
}

type OpenshiftApi struct {
	RouteV1Interface   routev1client.RouteV1Interface
	ProjectV1Interface projectv1client.ProjectV1Interface
	AppsV1Interface    appsv1client.AppsV1Interface
}
