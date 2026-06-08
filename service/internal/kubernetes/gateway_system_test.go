package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewaySystem_TypeCheckedByContains_NotOrder(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                   string
		typ                    string
		wantGatewayAPI         bool
		wantIngress            bool
		wantBothGatewaySystems bool
	}{
		{name: "legacy only", typ: LegacyIngress, wantGatewayAPI: false, wantIngress: true, wantBothGatewaySystems: false},
		{name: "gateway api only", typ: GatewayApiDefault, wantGatewayAPI: true, wantIngress: false, wantBothGatewaySystems: false},
		{
			name:                   "both legacy first",
			typ:                    LegacyIngress + "," + GatewayApiDefault,
			wantGatewayAPI:         true,
			wantIngress:            true,
			wantBothGatewaySystems: true,
		},
		{
			name:                   "both gateway api first",
			typ:                    GatewayApiDefault + "," + LegacyIngress,
			wantGatewayAPI:         true,
			wantIngress:            true,
			wantBothGatewaySystems: true,
		},
		{
			name:                   "both with spaces",
			typ:                    GatewayApiDefault + ", " + LegacyIngress,
			wantGatewayAPI:         true,
			wantIngress:            true,
			wantBothGatewaySystems: true,
		},
		{name: "empty defaults to legacy ingress", typ: "", wantGatewayAPI: false, wantIngress: true, wantBothGatewaySystems: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gs := GatewaySystem{Type: tc.typ}
			require.Equal(t, tc.wantGatewayAPI, gs.IsGatewayAPIEnabled())
			require.Equal(t, tc.wantIngress, gs.IsIngressEnabled())
			require.Equal(t, tc.wantBothGatewaySystems, gs.IsBothGatewaySystemsEnabled())
		})
	}
}
