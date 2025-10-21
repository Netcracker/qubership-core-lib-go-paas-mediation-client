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

func Test_GetService_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	serviceForClientSet := v1.Service{ObjectMeta: metav1.ObjectMeta{Name: testService, Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &serviceForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	srv, err := kube.GetService(ctx, testService, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, srv)
	assert.Equal(t, testService, srv.Name)
}

func Test_GetService_usingCache_success(t *testing.T) {
	ctx := context.Background()

	serviceTest := v1.Service{ObjectMeta: metav1.ObjectMeta{Name: testService, Namespace: testNamespace1}}

	clientset := fake.NewClientset()
	clientset.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "v1.23.0"}
	clientset.PrependReactor("get", "services", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
	})

	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	kube.Cache = cache.NewTestResourcesCache()
	ok, err := kube.Cache.Services.Set(ctx, *entity.NewService(&serviceTest))
	assert.NoError(t, err)
	assert.True(t, ok)

	srv, err := kube.GetService(ctx, testService, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, srv)
	assert.Equal(t, testService, srv.Name)
}

func Test_DeleteService_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	serviceForClientSet := v1.Service{ObjectMeta: metav1.ObjectMeta{Name: testService, Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &serviceForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	err := kube.DeleteService(ctx, testService, testNamespace1)
	assert.Nil(t, err)
}

func Test_UpdateOrCreateSecrvice_CreateNew_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	ctx := context.Background()
	srv := entity.Service{Metadata: entity.Metadata{Name: testService, Namespace: testNamespace1}}
	clientset := fake.NewClientset(&namespace)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	updatedService, err := kube.UpdateOrCreateService(ctx, &srv, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, updatedService)
}

func Test_UpdateOrCreateService_Update_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	srv := v1.Service{ObjectMeta: metav1.ObjectMeta{Name: testService, Namespace: testNamespace1},
		Spec: v1.ServiceSpec{Type: "1"}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &srv)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})

	serviceToUpdate := entity.Service{Metadata: entity.Metadata{Name: testService, Namespace: testNamespace1},
		Spec: entity.ServiceSpec{Type: "2"}}
	updatedService, err := kube.UpdateOrCreateService(ctx, &serviceToUpdate, testNamespace1)

	assert.Nil(t, err)
	assert.NotNil(t, updatedService)
	assert.Equal(t, serviceToUpdate.Spec.Type, updatedService.Spec.Type)
}
