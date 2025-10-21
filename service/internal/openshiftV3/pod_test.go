package openshiftV3

import (
	"context"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	kube "github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/kubernetes"
	openshiftappsfake "github.com/openshift/client-go/apps/clientset/versioned/fake"
	openshiftprojectfake "github.com/openshift/client-go/project/clientset/versioned/fake"
	openshiftroutefake "github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	v12 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func Test_getLatestReplicationController_success(t *testing.T) {
	desiredResultVersion := "5"
	testRepControllerList := []v12.ReplicationController{}
	testRepControllerList = append(testRepControllerList,
		v12.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "set1",
				Namespace:   testNamespace,
				Annotations: map[string]string{"openshift.io/deployment-config.latest-version": "3"}},
			Spec: v12.ReplicationControllerSpec{}},

		v12.ReplicationController{
			ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "set2",
				Namespace:   testNamespace,
				Annotations: map[string]string{"openshift.io/deployment-config.latest-version": desiredResultVersion}},
			Spec: v12.ReplicationControllerSpec{}})

	ctx := context.Background()
	clientset := fake.NewClientset(&testRepControllerList[0], &testRepControllerList[1])
	cert_client := &certClient.Clientset{}
	kubeClient, err := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)

	repController, err := os.getLatestReplicationController(ctx, testNamespace, testDeploymentName)
	assert.Equal(t, desiredResultVersion, repController.Annotations["openshift.io/deployment-config.latest-version"],
		"Wrong version of replica received")
	assert.NotNil(t, repController, "Latest replica must be not nil")
	assert.Nil(t, err, "Unexpected error returned")
}

func Test_deploymentConfigIsExist_success(t *testing.T) {
	deploymentConfigName := "test-name"

	repController := v12.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{Name: deploymentConfigName + "rep1", Namespace: testNamespace},
		Spec:       v12.ReplicationControllerSpec{}}

	ctx := context.Background()
	clientset := fake.NewClientset(&repController)
	cert_client := &certClient.Clientset{}
	kubeClient, err := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()

	appsV1Client := openshiftappsfake.NewClientset().AppsV1()

	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)
	checkExist, err := os.deploymentConfigIsExist(ctx, testNamespace, deploymentConfigName)
	assert.Nil(t, err)
	assert.True(t, checkExist)
}

func Test_RolloutDeployment_ConfigNotExist(t *testing.T) {
	ctx := context.Background()
	replica1 := v1beta1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set1",
			Namespace: testNamespace, Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			Labels: map[string]string{"app": "demo"}},
		Spec: v1beta1.ReplicaSetSpec{}}

	testDeployment := v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName,
			Namespace: testNamespace},
		Spec: v1beta1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo"},
			},
		},
	}

	clientset := fake.NewClientset(&testDeployment, &replica1)
	clientset.Fake.PrependReactor("patch", "*",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			replica2 := &v1beta1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set2",
					Namespace: testNamespace, Annotations: map[string]string{"deployment.kubernetes.io/revision": "2"},
					Labels: map[string]string{"app": "demo"}},
				Spec: v1beta1.ReplicaSetSpec{}}
			clientset.Tracker().Add(replica2)
			return true, &testDeployment, nil
		})

	cert_client := &certClient.Clientset{}
	kubeClient, err := kube.NewTestKubernetesClient(testNamespace, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	routeV1Client := openshiftroutefake.NewClientset().RouteV1()
	projectV1Client := openshiftprojectfake.NewClientset().ProjectV1()
	appsV1Client := openshiftappsfake.NewClientset().AppsV1()
	os := NewOpenshiftV3Client(routeV1Client, projectV1Client, appsV1Client, kubeClient)
	check, err := os.Kubernetes.RolloutDeployment(ctx, testDeploymentName, testNamespace)
	assert.Nil(t, err)
	assert.NotNil(t, check)
	assert.Equal(t, testDeploymentName+"-set1", check.Active)
	assert.Equal(t, testDeploymentName+"-set2", check.Rolling)
}
