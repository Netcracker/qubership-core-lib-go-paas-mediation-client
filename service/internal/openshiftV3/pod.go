package openshiftV3

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/exec"
	kubernetes2 "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	deploymentConfig "github.com/openshift/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (os *OpenshiftV3Client) RolloutDeploymentsInParallel(ctx context.Context, namespace string, deploymentConfigNames []string) (*entity.DeploymentResponse, error) {
	return os.rolloutDeployments(ctx, namespace, deploymentConfigNames, true)
}

func (os *OpenshiftV3Client) RolloutDeployments(ctx context.Context, namespace string, deploymentConfigNames []string) (*entity.DeploymentResponse, error) {
	return os.rolloutDeployments(ctx, namespace, deploymentConfigNames, false)
}

func (os *OpenshiftV3Client) rolloutDeployments(ctx context.Context, namespace string, deploymentConfigNames []string, parallel bool) (*entity.DeploymentResponse, error) {
	logger.InfoC(ctx, "Start RolloutDeployments function with deploymentConfigNames=%v in openshift, in parallel=%t",
		deploymentConfigNames, parallel)
	var tasks []exec.Task[*entity.DeploymentRollout]
	for _, deploymentName := range deploymentConfigNames {
		tasks = append(tasks, &kubernetes2.DeployableTask{
			Name: deploymentName,
			Task: func(name string) (*entity.DeploymentRollout, error) {
				logger.InfoC(ctx, "Start RolloutDeployment func deploymentConfig=%s in openshift", name)
				return os.RolloutDeployment(ctx, name, namespace)
			},
		})
	}
	var executor exec.RolloutExecutor
	if parallel {
		executor = os.RolloutExecutor
	} else {
		executor = exec.NewFixedRolloutExecutor(1, len(deploymentConfigNames))
		defer executor.Stop()
	}
	taskResults, err := executor.Submit(tasks)
	if err != nil {
		logger.ErrorC(ctx, "%s", err.Error())
		return nil, err
	} else if taskResults.HasErrors() {
		tErr := taskResults.GetAsError()
		logger.ErrorC(ctx, "%s", tErr.Error())
		return nil, tErr
	} else {
		return entity.NewDeploymentResponse(taskResults.GetResults()), nil
	}
}

func (os *OpenshiftV3Client) RolloutDeployment(ctx context.Context, deploymentConfigName string, namespace string) (*entity.DeploymentRollout, error) {
	logger.InfoC(ctx, "Start RolloutDeployment function with param=%s in openshift", deploymentConfigName)
	logger.DebugC(ctx, "Start define param=%s deployment or deploymentConfig in openshift", deploymentConfigName)
	if deploymentConfigIsExist, errorDeploymentConfig := os.deploymentConfigIsExist(ctx, namespace, deploymentConfigName); !deploymentConfigIsExist {
		if errorDeploymentConfig != nil {
			logger.ErrorC(ctx, "Error while check DeploymentConfig is exist: %+v", errorDeploymentConfig)
			return nil, errorDeploymentConfig
		}
		logger.DebugC(ctx, "Param=%s is not deployment config", deploymentConfigName)
		deploymentEntity, errorDeployment := os.Kubernetes.RolloutDeployment(ctx, deploymentConfigName, namespace)
		if errorDeployment != nil {
			return nil, errors.New("Can't find neither deployment nor deployment config " + deploymentConfigName)
		}
		return deploymentEntity, nil
	}
	logger.DebugC(ctx, "Start get active replication controller deployment config=%s in openshift", deploymentConfigName)
	activeReplicationController, errorDeploymentConfig := os.getLatestReplicationController(ctx, namespace, deploymentConfigName)
	if errorDeploymentConfig != nil {
		logger.ErrorC(ctx, "Error while getting lastVersion ReplicationController: %+v", errorDeploymentConfig)
		return nil, errorDeploymentConfig
	}
	logger.DebugC(ctx, "Active replication=%s in openshift", activeReplicationController)

	logger.InfoC(ctx, "Start rollout deployment config=%s in openshift", deploymentConfigName)
	_, errorOP3 := os.AppsClient.DeploymentConfigs(namespace).Instantiate(ctx, deploymentConfigName, &deploymentConfig.DeploymentRequest{Name: deploymentConfigName, Latest: true, Force: true}, v1.CreateOptions{})
	if errorOP3 != nil {
		var result deploymentConfig.DeploymentConfig
		errorOP1 := os.RouteV1Client.RESTClient().Post().
			Namespace(namespace).
			Resource("deploymentconfigs").
			Name(deploymentConfigName).
			SubResource("instantiate").
			Body(&deploymentConfig.DeploymentRequest{Name: deploymentConfigName, Latest: true, Force: true}).
			Do(ctx).Into(&result)
		if errorOP1 != nil {
			logger.ErrorC(ctx, "Error while restarting DeploymentConfig: %+v", errorDeploymentConfig)
			return nil, errorDeploymentConfig
		}
	}
	logger.InfoC(ctx, "Success rollout deployment config=%s in openshift", deploymentConfigName)
	rollingReplicationController, errorDeploymentConfig := os.getLatestReplicationController(ctx, namespace, deploymentConfigName)
	if errorDeploymentConfig != nil {
		logger.ErrorC(ctx, "Error while getting DeploymentConfig: %+v", errorDeploymentConfig)
		return nil, errorDeploymentConfig
	}

	rollingReplicationController, errorDeploymentConfig = os.correctLastReplicationController(ctx, namespace, deploymentConfigName, rollingReplicationController, activeReplicationController)
	if errorDeploymentConfig != nil {
		logger.ErrorC(ctx, "Error while gettingReplicationController")
		return nil, errorDeploymentConfig
	}
	logger.DebugC(ctx, "Rolling replication=%s in openshift", rollingReplicationController)
	return entity.NewDeploymentConfigRolloutResponseObj(deploymentConfigName, rollingReplicationController.Name, activeReplicationController.Name), nil
}

func (os *OpenshiftV3Client) correctLastReplicationController(ctx context.Context, namespace string, deploymentConfigName string,
	rollingReplicationController *v12.ReplicationController,
	activeReplicationController *v12.ReplicationController) (*v12.ReplicationController, error) {
	var err error
	logger.InfoC(ctx, "Enter in loop for find rollingReplicationController in openshift")
	for activeReplicationController.Name == rollingReplicationController.Name {
		rollingReplicationController, err = os.getLatestReplicationController(ctx, namespace, deploymentConfigName)
		if err != nil {
			logger.ErrorC(ctx, "Error while try get last replication controller")
			return nil, err
		}
	}
	return rollingReplicationController, nil
}

func (os *OpenshiftV3Client) getLatestReplicationController(ctx context.Context, namespace string, deploymentConfigName string) (*v12.ReplicationController, error) {
	logger.InfoC(ctx, "Start get last replication controller in deployment config=%s in openshift", deploymentConfigName)
	replicationControllersList, err := os.GetCoreV1Client().ReplicationControllers(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error while getting ReplicationControllerList")
		return nil, err
	}
	var latestRevisionNumber = -1
	var latestReplicationController v12.ReplicationController
	for _, currentReplicationController := range replicationControllersList.Items {
		if strings.HasPrefix(currentReplicationController.Name, deploymentConfigName) {
			currentVersionNumber, err := strconv.Atoi(currentReplicationController.Annotations["openshift.io/deployment-config.latest-version"])
			if err != nil {
				logger.ErrorC(ctx, "Error while parsing string")
				return nil, err
			}
			if latestRevisionNumber < currentVersionNumber {
				latestRevisionNumber = currentVersionNumber
				latestReplicationController = currentReplicationController
			}
		}
	}
	return &latestReplicationController, nil
}

func (os *OpenshiftV3Client) deploymentConfigIsExist(ctx context.Context, namespace string, deploymentConfigName string) (bool, error) {
	logger.InfoC(ctx, "Start get last replication controller in deployment config=%s in openshift", deploymentConfigName)
	replicationControllersList, err := os.GetCoreV1Client().ReplicationControllers(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error while getting ReplicationControllerList")
		return false, err
	}
	for _, replicationController := range replicationControllersList.Items {
		if strings.HasPrefix(replicationController.Name, deploymentConfigName) {
			return true, nil
		}
	}
	return false, nil
}
