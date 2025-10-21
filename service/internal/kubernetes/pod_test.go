package kubernetes

import (
	"context"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func getTestReplicaAppsV1() v1.ReplicaSet {
	return v1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet,
			Namespace: testNamespace1},
		Status: v1.ReplicaSetStatus{Replicas: 1}}
}

func getTestReplicaExtensionV1beta1() v1beta1.ReplicaSet {
	return v1beta1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet,
			Namespace: testNamespace1},
		Status: v1beta1.ReplicaSetStatus{Replicas: 1}}
}

func getTestRepController() corev1.ReplicationController {
	var a int32 = 1
	return corev1.ReplicationController{ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet,
		Namespace: testNamespace1}, Spec: corev1.ReplicationControllerSpec{Replicas: &a}}
}

func getTestPodsList() []corev1.Pod {
	testPods := []corev1.Pod{}
	testPods = append(testPods, corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet + "pod1",
			Namespace: testNamespace1}, Spec: corev1.PodSpec{ServiceAccountName: "build-robot"},
		Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: "Ready",
			Status: "True"}}}},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet + "pod2",
				Namespace: testNamespace1}, Spec: corev1.PodSpec{ServiceAccountName: "build-robot"},
			Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: "PodScheduled",
				Status: "True"}}}},
		corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet + "pod3",
				Namespace: testNamespace1}, Spec: corev1.PodSpec{ServiceAccountName: "deployer"},
			Status: corev1.PodStatus{Conditions: []corev1.PodCondition{{Type: "Ready",
				Status: "True"}}}})
	return testPods
}

func Test_GetPod_success(t *testing.T) {
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	podForClientSet := corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testPod, Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &podForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	pod, err := kube.GetPod(ctx, testPod, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, pod)
	assert.Equal(t, testPod, pod.Name)
}

func Test_GetPodsList_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	testPods := getTestPodsList()

	clientset := fake.NewClientset(&testPods[0], &testPods[1], &testPods[2])
	cert_client := &certClient.Clientset{}

	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	pods, err := kube.GetPodList(ctx, testNamespace1, filter.Meta{})
	r.Nil(err, "Unexpected error returned")
	r.NotNil(pods, "Pods must be not nil")
	r.Equal(3, len(pods), "Expected 3 pods")
}

func Test_getReplicaSet_appsV1Client_success(t *testing.T) {
	ctx := context.Background()
	testReplica := getTestReplicaAppsV1()

	clientset := fake.NewClientset(&testReplica)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	replica, err := kube.getReplicaSet(ctx, testNamespace1, testReplicaSet, appsV1Client)

	assert.NotNil(t, replica, "ReplicaSet must be not nil")
	assert.Nil(t, err, "Unexpected error returned")
	assert.Equal(t, testReplicaSet, replica.Name, "Mistake at received ReplicaSet name")
}

func Test_getReplicaSet_extensionV1Client_success(t *testing.T) {
	ctx := context.Background()
	testReplica := getTestReplicaExtensionV1beta1()

	clientset := fake.NewClientset(&testReplica)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	replica, err := kube.getReplicaSet(ctx, testNamespace1, testReplicaSet, extensionV1Client)

	assert.NotNil(t, replica, "ReplicaSet must be not nil")
	assert.Nil(t, err, "Unexpected error returned")
	assert.Equal(t, testReplicaSet, replica.Name, "Mistake at received ReplicaSet name")
}

func Test_getReplicaSetList_appsV1Client_success(t *testing.T) {
	testReplicaSetList := []v1.ReplicaSet{}
	testReplicaSetList = append(testReplicaSetList, v1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "set1",
			Namespace: testNamespace1},
		Spec: v1.ReplicaSetSpec{}}, v1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "set2",
			Namespace: testNamespace1},
		Spec: v1.ReplicaSetSpec{}})

	ctx := context.Background()
	clientset := fake.NewClientset(&testReplicaSetList[0], &testReplicaSetList[1])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	replica, err := kube.getReplicaSetList(ctx, testNamespace1, appsV1Client)
	assert.NotNil(t, replica, "ReplicaSet List must be not nil")
	assert.Nil(t, err, "Unexpected error returned")
	assert.Equal(t, 2, len(*replica), "Mistake at length of the list")
}

func Test_getReplicaSetList_extensionV1Client_success(t *testing.T) {
	testReplicaSet := v1beta1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "set1",
			Namespace: testNamespace1},
		Spec: v1beta1.ReplicaSetSpec{}}

	ctx := context.Background()
	clientset := fake.NewClientset(&testReplicaSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	replica, err := kube.getReplicaSetList(ctx, testNamespace1, extensionV1Client)
	assert.NotNil(t, replica, "ReplicaSet List must be not nil")
	assert.Nil(t, err, "Unexpected error returned")
	assert.Equal(t, 1, len(*replica), "Mistake at length of the list")
}

func Test_getLatestReplicaSet_appsV1Client_success(t *testing.T) {
	desiredResultVersion := "5"
	testReplicaSetList := []v1.ReplicaSet{}
	testReplicaSetList = append(testReplicaSetList,
		v1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "set1",
				Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "3"}},
			Spec: v1.ReplicaSetSpec{}},

		v1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "set2",
				Namespace:   testNamespace1,
				Annotations: map[string]string{"deployment.kubernetes.io/revision": desiredResultVersion}},
			Spec: v1.ReplicaSetSpec{}})

	ctx := context.Background()
	clientset := fake.NewClientset(&testReplicaSetList[0], &testReplicaSetList[1])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	replica, err := kube.getLatestReplicaSet(ctx, testNamespace1, testDeploymentName, appsV1Client)
	assert.Equal(t, desiredResultVersion, replica.CurrentVersion, "Wrong version of replica received")
	assert.NotNil(t, replica, "Latest replica must be not nil")
	assert.Nil(t, err, "Unexpected error returned")
}

func Test_readyReplicasPods_success(t *testing.T) {
	testPods := getTestPodsList()
	ctx := context.Background()
	clientset := fake.NewClientset(&testPods[0], &testPods[1], &testPods[2])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	counter := kube.readyReplicasPods(ctx, testReplicaSet, testPods)
	assert.Equal(t, 1, counter, "1 pod should be returned!")

}

// podReplicationControllerIsReady(ctx context.Context, namespace string, replicationControllerName string, podsList []corev1.Pod) (*corev1.ReplicationController, error) {
func Test_podReplicationControllerIsReady_success(t *testing.T) {
	testPods := getTestPodsList()
	ctx := context.Background()
	testRepController := getTestRepController()

	clientset := fake.NewClientset(&testRepController, &testPods[0], &testPods[1], &testPods[2])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	repController, err := kube.podReplicationControllerIsReady(ctx, testNamespace1, testReplicaSet, testPods)
	assert.NotNil(t, repController, "ReplicationController must not be nil")
	assert.Nil(t, err, "Some error occurred while getting ReplicationController")
}

func Test_podReplicationControllerIsReady_failure(t *testing.T) {
	testPods := getTestPodsList()
	ctx := context.Background()
	testRepController := corev1.ReplicationController{ObjectMeta: metav1.ObjectMeta{Name: testReplicaSet,
		Namespace: testNamespace1, Annotations: map[string]string{"kubectl.kubernetes.io/desired-replicas": "3"}}}

	clientset := fake.NewClientset(&testRepController, &testPods[0], &testPods[1], &testPods[2])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	repController, err := kube.podReplicationControllerIsReady(ctx, testNamespace1, testReplicaSet, testPods)
	assert.Nil(t, repController, "ReplicationController must be nil: not enough desired replicas")
	assert.Nil(t, err, "Some error occurred while getting ReplicationController")
}

func Test_podReplicaSetIsReady_appsv1client_success(t *testing.T) {
	testPods := getTestPodsList()
	ctx := context.Background()
	testReplica := getTestReplicaAppsV1()
	clientset := fake.NewClientset(&testReplica, &testPods[0], &testPods[1], &testPods[2])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	replicaSet, err := kube.podReplicaSetIsReady(ctx, testNamespace1, testReplicaSet, testPods, appsV1Client)
	assert.Nil(t, err, "Some error occurred while getting ReplicaSet")
	assert.NotNil(t, replicaSet, "ReplicaSet must not be nil")
}

func replicasMapInit() map[string][]string {
	replicasMap := make(map[string][]string)
	replicasMap["replica-set"] = []string{testReplicaSet}
	replicasMap["replication-controller"] = []string{testReplicaSet}
	return replicasMap
}

func Test_allPodsAreReady_appsV1client_success(t *testing.T) {
	testPods := getTestPodsList()[0]
	ctx := context.Background()
	testRepController := getTestRepController()
	testReplica := getTestReplicaAppsV1()

	replicasMap := replicasMapInit()
	clientset := fake.NewClientset(&testRepController, &testReplica, &testPods)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	mapPod, err := kube.allPodsAreReady(ctx, testNamespace1, replicasMap, appsV1Client)
	assert.NotNil(t, mapPod)
	assert.Nil(t, err, "Some error occurred while getting ReplicaSet")
}

func Test_allPodsAreReady_appsV1client_failure(t *testing.T) {
	testPods := getTestPodsList()
	ctx := context.Background()
	testRepController := getTestRepController()
	testReplica := getTestReplicaAppsV1()
	clientset := fake.NewClientset(&testRepController, &testReplica, &testPods[0], &testPods[1], &testPods[2])
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	replicasMap := replicasMapInit()
	mapPod, err := kube.allPodsAreReady(ctx, testNamespace1, replicasMap, appsV1Client)
	assert.Nil(t, mapPod)
	assert.Nil(t, err, "Some error occurred while getting ReplicaSet")
}

func Test_RolloutDeployment_appsV1_success(t *testing.T) {
	ctx := context.Background()
	replica1 := v1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set1",
			Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			Labels: map[string]string{"app": "demo"}},
		Spec: v1.ReplicaSetSpec{}}

	testDeployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName,
			Namespace: testNamespace1},
		Spec: v1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo"},
			},
		},
	}

	clientset := fake.NewClientset(&testDeployment, &replica1)
	cert_client := &certClient.Clientset{}
	clientset.Fake.PrependReactor("patch", "*",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			replica2 := &v1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set2",
					Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "2"},
					Labels: map[string]string{"app": "demo"}},
				Spec: v1.ReplicaSetSpec{}}
			clientset.Tracker().Add(replica2)
			return true, &testDeployment, nil
		})

	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	check, err := kube.RolloutDeployment(ctx, testDeploymentName, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, check)
	assert.Equal(t, testDeploymentName+"-set1", check.Active)
	assert.Equal(t, testDeploymentName+"-set2", check.Rolling)
}

func Test_RolloutDeployment_extensionV1_success(t *testing.T) {
	ctx := context.Background()
	replica1 := v1beta1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set1",
			Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			Labels: map[string]string{"app": "demo"}},
		Spec: v1beta1.ReplicaSetSpec{}}

	testDeployment := v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName,
			Namespace: testNamespace1},
		Spec: v1beta1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo"},
			},
		},
	}

	clientset := fake.NewClientset(&testDeployment, &replica1)
	cert_client := &certClient.Clientset{}
	clientset.Fake.PrependReactor("patch", "*",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			replica2 := &v1beta1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set2",
					Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "2"},
					Labels: map[string]string{"app": "demo"}},
				Spec: v1beta1.ReplicaSetSpec{}}
			clientset.Tracker().Add(replica2)
			return true, &testDeployment, nil
		})

	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	check, err := kube.RolloutDeployment(ctx, testDeploymentName, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, check)
	assert.Equal(t, testDeploymentName+"-set1", check.Active)
	assert.Equal(t, testDeploymentName+"-set2", check.Rolling)
}

func Test_RolloutDeployments(t *testing.T) {
	ctx := context.Background()
	replica1 := v1beta1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set1",
			Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "1"},
			Labels: map[string]string{"app": "demo"}},
		Spec: v1beta1.ReplicaSetSpec{}}

	testDeployment := v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName,
			Namespace: testNamespace1},
		Spec: v1beta1.DeploymentSpec{},
	}

	clientset := fake.NewClientset(&testDeployment, &replica1)
	cert_client := &certClient.Clientset{}
	clientset.Fake.PrependReactor("patch", "deployments",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			replica2 := &v1beta1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: testDeploymentName + "-set2",
					Namespace: testNamespace1, Annotations: map[string]string{"deployment.kubernetes.io/revision": "2"},
					Labels: map[string]string{"app": "demo"}},
				Spec: v1beta1.ReplicaSetSpec{}}
			clientset.Tracker().Add(replica2)
			return true, &testDeployment, nil
		})

	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	checkPtr, err := kube.RolloutDeployments(ctx, testNamespace1, []string{testDeploymentName})
	check := *checkPtr
	firstDeployment := check.Deployments[0]
	assert.Nil(t, err)
	assert.NotNil(t, check)
	assert.Equal(t, testDeploymentName+"-set1", firstDeployment.Active)
	assert.Equal(t, testDeploymentName+"-set2", firstDeployment.Rolling)
	assert.Equal(t, "ReplicaSet", firstDeployment.Kind)
}
