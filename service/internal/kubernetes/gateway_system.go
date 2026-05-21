package kubernetes

import (
	"fmt"
	"strings"
)

const (
	LegacyIngress     = "legacy-ingress"
	GatewayApiDefault = "gateway-api-default"

	DefaultGatewaySystemNamespace = "gateway-system"
	DefaultGatewaySystemName      = "default-external-gateway"

	GatewaySystemTypeProperty      = "gateway.system.type"
	GatewaySystemNamespaceProperty = "gateway.system.namespace"
	GatewaySystemNameProperty      = "gateway.system.name"
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

func (g GatewaySystem) IsBothGatewaySystemsEnabled() bool {
	if g.Type == "" {
		return false
	}
	return strings.Contains(g.Type, LegacyIngress) && strings.Contains(g.Type, GatewayApiDefault)
}

func (g GatewaySystem) IsRouteCreationAllowed() bool {
	return g.ShouldUseGatewayAPI() || g.ShouldCreateLegacyIngress()
}

func (g GatewaySystem) RouteCreationNotAllowedError() error {
	return fmt.Errorf("GATEWAY_SYSTEM_TYPE=%s does not allow any Route creation", g.Type)
}

func (g GatewaySystem) RouteUpdateNotAllowedError() error {
	return fmt.Errorf("GATEWAY_SYSTEM_TYPE=%s does not allow any Route update", g.Type)
}
