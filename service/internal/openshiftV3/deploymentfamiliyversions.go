package openshiftV3

import (
	"context"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
)

func (os *OpenshiftV3Client) GetDeploymentFamilyVersions(ctx context.Context, familyName string, namespace string) ([]entity.DeploymentFamilyVersion, error) {
	labels := map[string]string{entity.FamilyNameProp: familyName}

	deploymentConfigList, err := os.GetDeploymentList(ctx, namespace, filter.Meta{Labels: labels})
	if err != nil {
		logger.ErrorC(ctx, "Error to get a list of deploymentconfig: %+v", err)
		return nil, err
	}
	var result []entity.DeploymentFamilyVersion
	//goland:noinspection GoPreferNilSlice
	logger.InfoC(ctx, "Found %d deployment configs and deployments filtered by family_name=%s", len(deploymentConfigList), familyName)
	for _, deployment := range deploymentConfigList {
		item := entity.DeploymentToDeploymentFamilyVersion(deployment.Metadata.Labels)
		result = append(result, item)
	}
	return result, nil
}
