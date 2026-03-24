package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fasthttp/websocket"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/exec"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	pmTypes "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/types"
	pmWatch "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/watch"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type DeployableTask struct {
	Name string
	Task func(name string) (*entity.DeploymentRollout, error)
}

func (t *DeployableTask) Run() (*entity.DeploymentRollout, error) {
	return t.Task(t.Name)
}

func (kube *Kubernetes) GetPod(ctx context.Context, resourceName string, namespace string) (*entity.Pod, error) {
	return GetWrapper(ctx, resourceName, namespace, kube.GetCoreV1Client().Pods(namespace).Get,
		nil, entity.PodFromOsPod)
}

func (kube *Kubernetes) GetPodList(ctx context.Context, namespace string, filter filter.Meta) ([]entity.Pod, error) {
	return ListWrapper(ctx, filter, kube.GetCoreV1Client().Pods(namespace).List, nil,
		func(listObj *corev1.PodList) (result []entity.Pod) {
			for _, item := range listObj.Items {
				result = append(result, *entity.PodFromOsPod(&item))
			}
			return
		})
}

func (kube *Kubernetes) RolloutDeploymentsInParallel(ctx context.Context, namespace string, deploymentNames []string) (*entity.DeploymentResponse, error) {
	return kube.rolloutDeployments(ctx, namespace, deploymentNames, true)
}

func (kube *Kubernetes) RolloutDeployments(ctx context.Context, namespace string, deploymentNames []string) (*entity.DeploymentResponse, error) {
	return kube.rolloutDeployments(ctx, namespace, deploymentNames, false)
}

func (kube *Kubernetes) rolloutDeployments(ctx context.Context, namespace string, deploymentNames []string, parallel bool) (*entity.DeploymentResponse, error) {
	logger.InfoC(ctx, "Start RolloutDeployments function with deploymentNames=%v in kubernetes, in parallel=%t",
		deploymentNames, parallel)
	var tasks []exec.Task[*entity.DeploymentRollout]
	for _, deploymentName := range deploymentNames {
		tasks = append(tasks, &DeployableTask{
			Name: deploymentName,
			Task: func(name string) (*entity.DeploymentRollout, error) {
				return kube.RolloutDeployment(ctx, name, namespace)
			},
		})
	}
	var executor exec.RolloutExecutor
	if parallel {
		executor = kube.RolloutExecutor
	} else {
		executor = exec.NewFixedRolloutExecutor(1, len(deploymentNames))
		defer executor.Stop()
	}
	taskResults, err := executor.Submit(tasks)
	if err != nil {
		logger.ErrorC(ctx, "%v", err.Error())
		return nil, err
	} else if taskResults.HasErrors() {
		tErr := taskResults.GetAsError()
		logger.ErrorC(ctx, "%s", tErr.Error())
		return nil, tErr
	} else {
		return entity.NewDeploymentResponse(taskResults.GetResults()), nil
	}
}

func (kube *Kubernetes) RolloutDeployment(ctx context.Context, deploymentName string, namespace string) (*entity.DeploymentRollout, error) {
	var clientVersion string
	logger.InfoC(ctx, "Start RolloutDeployment function with param=%s in kubernetes", deploymentName)
	_, errExtension := kube.getExtensionsV1Client().Deployments(namespace).Get(ctx, deploymentName, v1.GetOptions{})
	_, errApps := kube.getAppsV1Client().Deployments(namespace).Get(ctx, deploymentName, v1.GetOptions{})
	if errExtension != nil && errApps != nil {
		logger.ErrorC(ctx, "Can't find requested deployment: %+v", errExtension)
		return nil, errExtension
	}
	if errApps == nil {
		clientVersion = appsV1Client
	} else {
		clientVersion = extensionV1Client
	}
	logger.DebugC(ctx, "Start get active replica set deployment config=%s in kubernetes", deploymentName)
	activeReplicaSet, err := kube.getLatestReplicaSet(ctx, namespace, deploymentName, clientVersion)
	if err != nil {
		logger.ErrorC(ctx, "Error while getting last Version ReplicaSet %+v", err)
		return nil, err
	}
	logger.InfoC(ctx, "Start rollout deployment=%s in kubernetes", deploymentName)
	jsonConfig := kube.DeploymentRolloutPatchData()
	if clientVersion == extensionV1Client {
		_, err = kube.getExtensionsV1Client().Deployments(namespace).Patch(ctx, deploymentName, types.StrategicMergePatchType, []byte(jsonConfig), v1.PatchOptions{})
	} else {
		_, err = kube.getAppsV1Client().Deployments(namespace).Patch(ctx, deploymentName, types.StrategicMergePatchType, []byte(jsonConfig), v1.PatchOptions{})
	}
	if err != nil {
		logger.ErrorC(ctx, "Error while restarting a Deployment: %+v", err)
		return nil, err
	}
	logger.DebugC(ctx, "Start get rolling replica set deployment config=%s in kubernetes", deploymentName)
	rollingReplicaSet, err := kube.getLatestReplicaSet(ctx, namespace, deploymentName, clientVersion)
	if err != nil {
		logger.ErrorC(ctx, "Error while getting last Version ReplicaSet: %+v", err)
		return nil, err
	}
	rollingReplicaSet, err = kube.correctLastReplicaSet(ctx, namespace, deploymentName, rollingReplicaSet, activeReplicaSet, clientVersion)
	if err != nil {
		return nil, err
	}
	logger.InfoC(ctx, "Success rollout deployment config=%s in kubernetes", deploymentName)
	return entity.NewDeploymentRolloutResponseObj(deploymentName, rollingReplicaSet.Name, activeReplicaSet.Name), nil
}

func (kube *Kubernetes) correctLastReplicaSet(ctx context.Context, namespace string, deploymentName string,
	rollingReplicaSet *entity.ReplicaSet, activeReplicaSet *entity.ReplicaSet, clientVersion string) (*entity.ReplicaSet, error) {
	var err error
	logger.InfoC(ctx, "Enter in loop for find rollingReplicaSet in kubernetes")
	for activeReplicaSet.Name == rollingReplicaSet.Name {
		rollingReplicaSet, err = kube.getLatestReplicaSet(ctx, namespace, deploymentName, clientVersion)
		if err != nil {
			logger.ErrorC(ctx, "Error while getting ReplicaSetList: %+v", err)
			return nil, err
		}
	}
	return rollingReplicaSet, nil
}

func (kube *Kubernetes) DeploymentRolloutPatchData() string {
	return `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"` + time.Now().Format(time.RFC3339) + `"}}}}}`
}

func (kube *Kubernetes) getLatestReplicaSet(ctx context.Context, namespace string, deploymentName string, clientVersion string) (*entity.ReplicaSet, error) {
	logger.InfoC(ctx, "Start get last replica set in deployment config=%s in kubernetes", deploymentName)
	replicaSetsList, err := kube.getReplicaSetList(ctx, namespace, clientVersion)
	if err != nil {
		logger.ErrorC(ctx, "Error getting replicaSetsList: %+v", err)
		return nil, err
	}
	var latestRevisionNumber = -1
	var latestReplicaSet entity.ReplicaSet
	for _, currentReplicaSet := range *replicaSetsList {
		if strings.HasPrefix(currentReplicaSet.Name, deploymentName) {
			replicaSetCurrentVersionNumber, err := strconv.Atoi(currentReplicaSet.CurrentVersion)
			if err != nil {
				logger.ErrorC(ctx, "Error while parsing string: %+v", err)
				return nil, err
			}
			if latestRevisionNumber < replicaSetCurrentVersionNumber {
				latestRevisionNumber = replicaSetCurrentVersionNumber
				latestReplicaSet = currentReplicaSet
			}
		}
	}
	return &latestReplicaSet, nil
}

func (kube *Kubernetes) getReplicaSet(ctx context.Context, namespace string, replicaName string, clientVersion string) (*entity.ReplicaSet, error) {
	if clientVersion == extensionV1Client {
		replicaSet, err := kube.getExtensionsV1Client().ReplicaSets(namespace).Get(ctx, replicaName, v1.GetOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error while getting last replicaSet %+v", err)
			return nil, err
		}
		return entity.TransformationReplicaSetExtension(replicaSet), nil
	}
	if clientVersion == appsV1Client {
		replicaSet, err := kube.getAppsV1Client().ReplicaSets(namespace).Get(ctx, replicaName, v1.GetOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error while getting last replicaSet %+v", err)
			return nil, err
		}
		return entity.TransformationReplicaSetApp(replicaSet), nil
	}
	return nil, errors.New("unsupported client version: " + clientVersion)
}

func (kube *Kubernetes) getReplicaSetList(ctx context.Context, namespace string, clientVersion string) (*[]entity.ReplicaSet, error) {
	if clientVersion == extensionV1Client {
		replicaSetList, err := kube.getExtensionsV1Client().ReplicaSets(namespace).List(ctx, v1.ListOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error getting replicaSetsList: %+v", err)
			return nil, err
		}
		return entity.TransformationReplicaSetListExtension(replicaSetList.Items), nil
	}
	if clientVersion == appsV1Client {
		replicaSetList, err := kube.getAppsV1Client().ReplicaSets(namespace).List(ctx, v1.ListOptions{})
		if err != nil {
			logger.ErrorC(ctx, "Error getting replicaSetsList: %+v", err)
			return nil, err
		}
		return entity.TransformationReplicaSetListApp(replicaSetList.Items), nil
	}
	return nil, errors.New("unsupported client version: " + clientVersion)
}

func (kube *Kubernetes) WatchPodsRestarting(ctx context.Context, namespace string, filter filter.Meta, replicasMap map[string][]string) (*pmWatch.Handler, error) {
	clientVersionForKubernetes, err := kube.getKubernetesClientVersion(namespace)
	if err != nil {
		return nil, err
	}
	handler, err := NewRestWatchHandler(namespace, pmTypes.Pods, kube.GetCoreV1Client().RESTClient(), kube.WatchExecutor, entity.PodFromOsPod).
		Watch(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to create watchEventHandler: %w", err)
	}
	proxyChannel := make(chan pmWatch.ApiEvent, 1)
	rolloutHandler := &pmWatch.Handler{
		Channel:      proxyChannel,
		StopWatching: handler.StopWatching,
	}
	go func(channel <-chan pmWatch.ApiEvent, replicasMap map[string][]string) {
		for {
			select {
			case podEvent, ok := <-channel:
				if !ok {
					return
				} else {
					pod := podEvent.Object.(*entity.Pod)
					if correctPod(pod, replicasMap) {
						proxyChannel <- pmWatch.ApiEvent{
							Type:   podEvent.Type,
							Object: entity.NewRolledOutPod(*pod),
						}
						webSocketResponse, prErr := kube.allPodsAreReady(ctx, namespace, replicasMap, clientVersionForKubernetes)
						if prErr != nil {
							proxyChannel <- pmWatch.ApiEvent{
								Type:   pmWatch.Error,
								Object: prErr,
							}
							logger.ErrorC(ctx, "Error while check that all pods already restarted: %v", prErr)
							handler.StopWatching()
							return
						}
						if webSocketResponse != nil {
							proxyChannel <- pmWatch.ApiEventConstructor(CloseControlMessage, webSocketResponse,
								pmWatch.ControlMessageDetails{CloseCode: websocket.CloseNormalClosure, MessageType: websocket.CloseMessage, CloseMessage: "Pods have been rolled out"})
							handler.StopWatching()
							return
						}
					}
				}
			}
		}
	}(handler.Channel, replicasMap)
	return rolloutHandler, nil
}

func (kube *Kubernetes) allPodsAreReady(ctx context.Context, namespace string, replicasMap map[string][]string,
	clientVersionForKubernetes string) (*map[string][]entity.RolloutedPod, error) {
	var replicationControllerList []*corev1.ReplicationController
	var tmpReplicationController *corev1.ReplicationController
	var replicaSetList []*entity.ReplicaSet
	var tmpReplicaSet *entity.ReplicaSet
	var err error

	podsList, err := kube.GetCoreV1Client().Pods(namespace).List(ctx, v1.ListOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error while get pods list: %+v", err)
		return nil, err
	}
	// todo next major change everywhere 'replication-controller' to 'replicationController'
	for _, replicationControllerName := range replicasMap["replication-controller"] {
		tmpReplicationController, err = kube.podReplicationControllerIsReady(ctx, namespace, replicationControllerName, podsList.Items)
		if err != nil {
			logger.ErrorC(ctx, "Error while check pod: %+v", err)
			return nil, err
		}
		// todo what return nil, nil means? refactor this
		if tmpReplicationController == nil {
			return nil, nil
		}
		replicationControllerList = append(replicationControllerList, tmpReplicationController)
	}

	// todo next major change everywhere 'replica-set' to 'replicaSet'
	for _, replicaSetName := range replicasMap["replica-set"] {
		tmpReplicaSet, err = kube.podReplicaSetIsReady(ctx, namespace, replicaSetName, podsList.Items, clientVersionForKubernetes)
		if err != nil {
			logger.ErrorC(ctx, "Error while check pod: %+v", err)
			return nil, err
		}
		// todo what return nil, nil means? refactor this
		if tmpReplicaSet == nil {
			return nil, nil
		}
		replicaSetList = append(replicaSetList, tmpReplicaSet)
	}

	mapForStruct := make(map[string][]corev1.Pod)
	reg, err := regexp.Compile(`-[^-]+?\z`)
	if err != nil {
		logger.ErrorC(ctx, "Error while compile regex: %+v", err)
		return nil, err
	}
	for _, replicaSet := range replicaSetList {
		var podListForStruct []corev1.Pod
		for _, pod := range podsList.Items {
			if strings.HasPrefix(pod.Name, replicaSet.Name) {
				podListForStruct = append(podListForStruct, pod)
			}
		}
		mapForStruct[reg.ReplaceAllString(replicaSet.Name, "")] = podListForStruct
	}
	for _, replicationController := range replicationControllerList {
		var podListForStruct []corev1.Pod
		for _, pod := range podsList.Items {
			if strings.HasPrefix(pod.Name, replicationController.Name) && pod.Spec.ServiceAccountName != "deployer" {
				podListForStruct = append(podListForStruct, pod)
			}
		}
		mapForStruct[reg.ReplaceAllString(replicationController.Name, "")] = podListForStruct
	}
	return entity.NewDeploymentsStatus(mapForStruct), nil
}

func (kube *Kubernetes) podReplicaSetIsReady(ctx context.Context, namespace string, replicaSetName string,
	podsList []corev1.Pod, clientVersionForKubernetes string) (*entity.ReplicaSet, error) {
	var desiredPods int
	replicaSet, err := kube.getReplicaSet(ctx, namespace, replicaSetName, clientVersionForKubernetes)
	if err != nil {
		logger.ErrorC(ctx, "Error while getting ReplicationSet")
		return nil, err
	}
	desiredReplicasString := replicaSet.DesiredReplicas
	if desiredReplicasString == "" {
		desiredPods = replicaSet.Replicas
	} else {
		desiredPods, err = strconv.Atoi(desiredReplicasString)
		if err != nil {
			return nil, err
		}
	}
	if kube.readyReplicasPods(ctx, replicaSetName, podsList) == desiredPods {
		return replicaSet, nil
	} else {
		logger.InfoC(ctx, "replica set '%s' id not ready", replicaSetName)
		logger.DebugC(ctx, "replica set is not ready %+v", replicaSet)
		return nil, nil
	}
}

func (kube *Kubernetes) podReplicationControllerIsReady(ctx context.Context, namespace string, replicationControllerName string, podsList []corev1.Pod) (*corev1.ReplicationController, error) {
	var desiredPods int
	replicationController, err := kube.GetCoreV1Client().ReplicationControllers(namespace).Get(ctx, replicationControllerName, v1.GetOptions{})
	if err != nil {
		logger.ErrorC(ctx, "Error while geting replicationController: %+v", err)
		return nil, err
	}
	desiredReplicasString := replicationController.Annotations["kubectl.kubernetes.io/desired-replicas"]
	if desiredReplicasString == "" {
		desiredPods = int(*replicationController.Spec.Replicas)
	} else {
		desiredPods, err = strconv.Atoi(desiredReplicasString)
		if err != nil {
			return nil, err
		}
	}
	if kube.readyReplicasPods(ctx, replicationControllerName, podsList) == desiredPods {
		return replicationController, nil
	} else {
		logger.InfoC(ctx, "replication controller=%s not ready", replicationControllerName)
		logger.DebugC(ctx, "replication controller not ready: %+v", replicationController)
		return nil, nil
	}
}

func (kube *Kubernetes) readyReplicasPods(ctx context.Context, replicaSetName string, podsList []corev1.Pod) int {
	counter := 0
	for _, pod := range podsList {
		if strings.HasPrefix(pod.Name, replicaSetName) && pod.Spec.ServiceAccountName != "deployer" {
			logger.DebugC(ctx, "pod status: %+v", pod)
			for _, condition := range pod.Status.Conditions {
				if condition.Type == "Ready" {
					if condition.Status == "True" {
						counter++
					}
				}
			}
		}
	}
	return counter
}

func correctPod(pod *entity.Pod, replicasMap map[string][]string) bool {
	for _, replicaNamesList := range replicasMap {
		for _, replicaName := range replicaNamesList {
			if strings.HasPrefix(pod.Name, replicaName) {
				return true
			}
		}
	}
	return false
}
