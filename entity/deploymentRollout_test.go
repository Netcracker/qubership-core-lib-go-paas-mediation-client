package entity

import (
	"testing"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	"github.com/stretchr/testify/require"
)

func Test_NewDeploymentResponse(t *testing.T) {
	deployment1 := DeploymentRollout{Name: "1", Kind: "ReplicaSet", Rolling: "1-2", Active: "1-1"}
	deployment2 := DeploymentRollout{Name: "2", Kind: "ReplicationController", Rolling: "2-2", Active: "2-1"}
	deployments := []*DeploymentRollout{&deployment1, &deployment2}
	response := NewDeploymentResponse(deployments)
	require.Equal(t, []DeploymentRollout{deployment1, deployment2}, response.Deployments)
}

func Test_NewDeploymentResponseEmpty(t *testing.T) {
	deployments := []*DeploymentRollout{}
	response := NewDeploymentResponse(deployments)
	require.Equal(t, []DeploymentRollout{}, response.Deployments)
}

func Test_NewDeploymentRolloutResponseObj(t *testing.T) {
	expected := &DeploymentRollout{Name: testDeploymentName, Kind: types.ReplicaSet, Active: "active", Rolling: "rolling"}
	require.Equal(t, expected, NewDeploymentRolloutResponseObj(testDeploymentName, "rolling", "active"))
}

func Test_NewDeploymentConfigRolloutResponseObj(t *testing.T) {
	expected := &DeploymentRollout{Name: testDeploymentName, Kind: types.ReplicationController, Active: "active", Rolling: "rolling"}
	require.Equal(t, expected, NewDeploymentConfigRolloutResponseObj(testDeploymentName, "rolling", "active"))
}
