//go:generate mockgen -source=service.go -destination=service_mock.go -package=service

package service

import (
	"context"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
)

type RouteInterface interface {
	CreateRoute(ctx context.Context, request *entity.Route, namespace string) (*entity.Route, error)
	DeleteRoute(ctx context.Context, routeName string, namespace string) error
	GetBadRouteLists(ctx context.Context) (map[string][]string, error)
	// GetRoute returns Route entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetRoute(ctx context.Context, resourceName string, namespace string) (*entity.Route, error)
	GetRouteList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Route, error)
	UpdateOrCreateRoute(ctx context.Context, request *entity.Route, namespace string) (*entity.Route, error)
	WatchRoutes(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type HttpRouteInterface interface {
	GetHttpRouteList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.HttpRoute, error)
	WatchGatewayHTTPRoutes(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type GrpcRouteInterface interface {
	GetGrpcRouteList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.GrpcRoute, error)
	WatchGatewayGRPCRoutes(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type ConfigMapInterface interface {
	CreateConfigMap(ctx context.Context, configMap *entity.ConfigMap, namespace string) (*entity.ConfigMap, error)
	DeleteConfigMap(ctx context.Context, resourceName string, namespace string) error
	// GetConfigMap returns ConfigMap entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetConfigMap(ctx context.Context, resourceName string, namespace string) (*entity.ConfigMap, error)
	GetConfigMapList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.ConfigMap, error)
	UpdateOrCreateConfigMap(ctx context.Context, configMap *entity.ConfigMap, namespace string) (*entity.ConfigMap, error)
	WatchConfigMaps(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type PodInterface interface {
	// GetPod returns Pod entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetPod(ctx context.Context, resourceName string, namespace string) (*entity.Pod, error)
	GetPodList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Pod, error)
	RolloutDeployment(ctx context.Context, deploymentName string, namespace string) (*entity.DeploymentRollout, error)
	RolloutDeployments(ctx context.Context, namespace string, deploymentNames []string) (*entity.DeploymentResponse, error)
	RolloutDeploymentsInParallel(ctx context.Context, namespace string, deploymentNames []string) (*entity.DeploymentResponse, error)
	// todo in next major release delete filter.Meta from params of this method - filtering is not applicable in this case
	WatchPodsRestarting(ctx context.Context, namespace string, filter filter.Meta, deploymentsMap map[string][]string) (*watch.Handler, error)
}

type ServiceInterface interface {
	CreateService(ctx context.Context, service *entity.Service, namespace string) (*entity.Service, error)
	DeleteService(ctx context.Context, resourceName string, namespace string) error
	// GetService returns Service entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetService(ctx context.Context, resourceName string, namespace string) (*entity.Service, error)
	GetServiceList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Service, error)
	UpdateOrCreateService(ctx context.Context, service *entity.Service, namespace string) (*entity.Service, error)
	WatchServices(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type SecretInterface interface {
	CreateSecret(ctx context.Context, secret *entity.Secret, namespace string) (*entity.Secret, error)
	DeleteSecret(ctx context.Context, resourceName string, namespace string) error
	GetOrCreateSecretWithServiceAccountToken(ctx context.Context, serviceAccountName string, namespace string) (*entity.Secret, error)
	// GetSecret returns Secret entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetSecret(ctx context.Context, resourceName string, namespace string) (*entity.Secret, error)
	GetSecretList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Secret, error)
	UpdateOrCreateSecret(ctx context.Context, secret *entity.Secret, namespace string) (*entity.Secret, error)
	WatchSecrets(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type NamespaceInterface interface {
	GetNamespace(ctx context.Context, name string) (*entity.Namespace, error)
	GetNamespaces(ctx context.Context, filter filter.Meta) ([]entity.Namespace, error)
	WatchNamespaces(ctx context.Context, namespace string) (*watch.Handler, error)
}

type ServiceAccountInterface interface {
	CreateServiceAccount(ctx context.Context, serviceAccount *entity.ServiceAccount, namespace string) (*entity.ServiceAccount, error)
	DeleteServiceAccount(ctx context.Context, resourceName string, namespace string) error
	// GetServiceAccount returns ServiceAccount entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetServiceAccount(ctx context.Context, resourceName string, namespace string) (*entity.ServiceAccount, error)
	GetServiceAccountList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.ServiceAccount, error)
	WatchServiceAccounts(ctx context.Context, namespace string, filter filter.Meta) (*watch.Handler, error)
}

type DeploymentFamilyVersionsInterface interface {
	GetDeploymentFamilyVersions(ctx context.Context, familyMame string, namespace string) ([]entity.DeploymentFamilyVersion, error)
}

type DeploymentInterface interface {
	// GetDeployment returns Deployment entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetDeployment(ctx context.Context, resourceName string, namespace string) (*entity.Deployment, error)
	GetDeploymentList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Deployment, error)
}

type CertificateInterface interface {
	CreateCertificate(ctx context.Context, certificate *entity.Certificate, namespace string) (*entity.Certificate, error)
	DeleteCertificate(ctx context.Context, resourceName string, namespace string) error
	// GetCertificate returns Certificate entity in case it exists, nil otherwise, error in case api server returned non 404 error response
	GetCertificate(ctx context.Context, resourceName string, namespace string) (*entity.Certificate, error)
	GetCertificateList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Certificate, error)
	UpdateOrCreateCertificate(ctx context.Context, certificate *entity.Certificate, namespace string) (*entity.Certificate, error)
}

type PlatformService interface {
	CertificateInterface
	ConfigMapInterface
	DeploymentFamilyVersionsInterface
	DeploymentInterface
	NamespaceInterface
	PodInterface
	RouteInterface
	SecretInterface
	ServiceAccountInterface
	ServiceInterface
	HttpRouteInterface
	GrpcRouteInterface
}
