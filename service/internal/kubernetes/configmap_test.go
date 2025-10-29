package kubernetes

import (
	"context"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/internal/cache"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func Test_GetConfigMap_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	configmapForClientSet := v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: testConfigMap, Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &configmapForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	configMap, err := kube.GetConfigMap(ctx, testConfigMap, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, configMap)
	assert.Equal(t, testConfigMap, configMap.Name)
}

func Test_GetConfigMap_usingCache_success(t *testing.T) {
	ctx := context.Background()

	clientset := fake.NewClientset()
	cert_client := &certClient.Clientset{}
	//	version, err := b.client.KubernetesInterface.Discovery().ServerVersion()
	clientset.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "v1.23.0"}
	clientset.PrependReactor("get", "configmaps", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
	})
	kube, err := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	assert.Nil(t, err)
	kube.Cache = cache.NewTestResourcesCache()
	configmapTest := entity.ConfigMap{Metadata: entity.Metadata{Name: testConfigMap, Namespace: testNamespace1}}
	ok, err := kube.Cache.ConfigMaps.Set(ctx, configmapTest)
	assert.NoError(t, err)
	assert.True(t, ok)
	secret, err := kube.GetConfigMap(ctx, testConfigMap, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
}

func Test_UpdateOrCreateConfigMap_CreateNew_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	ctx := context.Background()
	configMap := entity.ConfigMap{Metadata: entity.Metadata{Name: testConfigMap, Namespace: testNamespace1},
		Data: map[string]string{"body": "test1"}}
	clientset := fake.NewClientset(&namespace)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	updatedConfigmap, err := kube.UpdateOrCreateConfigMap(ctx, &configMap, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, updatedConfigmap)
}

func Test_UpdateOrCreateConfigMap_Update_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	configmapForClientSet := v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: testConfigMap, Namespace: testNamespace1},
		Data: map[string]string{"body": "test1"}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &configmapForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	configMap := entity.ConfigMap{Metadata: entity.Metadata{Name: testConfigMap, Namespace: testNamespace1},
		Data: map[string]string{"body": "test2"}}
	updatedConfigmap, err := kube.UpdateOrCreateConfigMap(ctx, &configMap, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, updatedConfigmap)
	assert.Equal(t, configMap.Data, updatedConfigmap.Data)
}

func Test_DeleteConfigMap_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	configmapForClientSet := v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: testConfigMap, Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &configmapForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	err := kube.DeleteConfigMap(ctx, testConfigMap, testNamespace1)
	assert.Nil(t, err)
}
