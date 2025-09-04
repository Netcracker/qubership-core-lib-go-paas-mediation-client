package types

type PaasResourceType string

const (
	Certificates    PaasResourceType = "certificates"
	ConfigMaps      PaasResourceType = "configmaps"
	Deployments     PaasResourceType = "deployments"
	Ingresses       PaasResourceType = "ingresses"
	Namespaces      PaasResourceType = "namespaces"
	Pods            PaasResourceType = "pods"
	Secrets         PaasResourceType = "secrets"
	ServiceAccounts PaasResourceType = "serviceaccounts"
	Services        PaasResourceType = "services"
	HTTPRoutes      PaasResourceType = "httproutes"
	GRPCRoutes      PaasResourceType = "grpcroutes"

	Projects PaasResourceType = "projects" // todo deprecated, will be removed
	Routes   PaasResourceType = "routes"   // todo deprecated, will be removed
)

func (t PaasResourceType) String() string {
	return string(t)
}
