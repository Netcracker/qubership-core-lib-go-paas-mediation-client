package kubernetes

import "strings"

const (
	LegacyIngress     = "legacy-ingress"
	GatewayApiDefault = "gateway-api-default"

	DefaultGatewaySystemNamespace = "gateway-system"
	DefaultGatewaySystemName      = "default-external-gateway"
)

type GatewaySystem struct {
	Type      string
	Namespace string
	Name      string
}

func (g GatewaySystem) ShouldUseGatewayAPI() bool {
	if g.Type == "" {
		return false
	}
	return strings.Contains(g.Type, GatewayApiDefault)
}

func (g GatewaySystem) ShouldCreateLegacyIngress() bool {
	if g.Type == "" {
		return true
	}
	return strings.Contains(g.Type, LegacyIngress)
}

func (g GatewaySystem) ShouldIgnoreIngressForConverter() bool {
	if g.Type == "" {
		return false
	}
	return strings.Contains(g.Type, LegacyIngress) && strings.Contains(g.Type, GatewayApiDefault)
}
