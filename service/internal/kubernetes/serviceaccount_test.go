package kubernetes

import (
	"context"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_GetServiceAccount_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	serviceAccountForClientSet := v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount,
		Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &serviceAccountForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	service, err := kube.GetServiceAccount(ctx, testServiceAccount, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, testServiceAccount, service.Name)
}

func Test_GetServiceAccountList_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	firstServiceAccountForClientSet := v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount + "-first", Namespace: testNamespace1}}
	secondServiceAccountForClientSet := v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount + "-second", Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &firstServiceAccountForClientSet, &secondServiceAccountForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	services, err := kube.GetServiceAccountList(ctx, testNamespace1, filter.Meta{})
	assert.Nil(t, err)
	assert.NotNil(t, services)
	assert.Len(t, services, 2)
	assert.Equal(t, testServiceAccount+"-first", services[0].Name)
	assert.Equal(t, testServiceAccount+"-second", services[1].Name)
}

func Test_DeleteServiceAccount_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	serviceForClientSet := v1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount,
		Namespace: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &serviceForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	err := kube.DeleteServiceAccount(ctx, testServiceAccount, testNamespace1)
	assert.Nil(t, err)
}

func Test_CreateServiceAccount_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	serviceAccount := entity.ServiceAccount{Metadata: entity.Metadata{Name: testServiceAccount,
		Namespace: testNamespace1}, Secrets: []entity.SecretInfo{{Name: "firstInfo"}, {Name: "secondInfo"}}}
	newServiceAccount, err := kube.CreateServiceAccount(ctx, &serviceAccount, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, newServiceAccount)
	assert.Equal(t, 2, len(newServiceAccount.Secrets))
}
