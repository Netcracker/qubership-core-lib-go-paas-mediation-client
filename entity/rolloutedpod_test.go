package entity

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func Test_NewRolloutedPod_Ready(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testPod, Namespace: testNamespace},
		Status: v1.PodStatus{Conditions: []v1.PodCondition{{Type: "Ready", Status: "True"}}}}
	rollouted := &RolloutedPod{testPod, "Ready"}
	assert.Equal(t, rollouted, NewRolloutedPod(pod))
}

func Test_NewRolloutedPod_NotReady(t *testing.T) {
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testPod, Namespace: testNamespace},
		Status: v1.PodStatus{Conditions: []v1.PodCondition{{Type: "Ready", Status: "False"}}}}
	rollouted := &RolloutedPod{testPod, "Not ready"}
	assert.Equal(t, rollouted, NewRolloutedPod(pod))
}

func Test_NewDeploymentsStatus(t *testing.T) {
	pod1 := v1.Pod{ObjectMeta: metav1.ObjectMeta{Name: testPod, Namespace: testNamespace},
		Status: v1.PodStatus{Conditions: []v1.PodCondition{{Type: "Ready", Status: "True"}}}}
	pod2 := pod1
	pod2.Name = "pod3"
	pod2.Status.Conditions[0].Status = "False"
	podsMap := map[string][]v1.Pod{testDeploymentName: {pod1, pod2}}
	assert.Nil(t, NewDeploymentsStatus(podsMap))
	podsMap[testDeploymentName][0].Status.Conditions[0].Status = "True"
	result := *(NewDeploymentsStatus(podsMap))
	assert.Equal(t, 2, len(result[testDeploymentName]))
}

func Test_NewRolledOutPod_Ready(t *testing.T) {
	pod := Pod{
		Metadata: Metadata{
			Name: testPod,
		},
		Status: PodStatus{
			Conditions: []StatusCondition{
				{Type: "Ready", Status: "True"},
			},
		},
	}
	rollouted := &RolloutedPod{testPod, "Ready"}
	assert.Equal(t, rollouted, NewRolledOutPod(pod))
}

func Test_NewRolledOutPod_NotReady(t *testing.T) {
	pod := Pod{
		Metadata: Metadata{
			Name: testPod,
		},
		Status: PodStatus{
			Conditions: []StatusCondition{
				{Type: "Ready", Status: "False"},
			},
		},
	}
	rollouted := &RolloutedPod{testPod, "Not ready"}
	assert.Equal(t, rollouted, NewRolledOutPod(pod))
}

func Test_NewRolledOutPod_NoReadyCondition(t *testing.T) {
	pod := Pod{
		Metadata: Metadata{
			Name: testPod,
		},
		Status: PodStatus{
			Conditions: []StatusCondition{
				{Type: "Initialized", Status: "True"},
			},
		},
	}
	rollouted := &RolloutedPod{testPod, ""}
	assert.Equal(t, rollouted, NewRolledOutPod(pod))
}
