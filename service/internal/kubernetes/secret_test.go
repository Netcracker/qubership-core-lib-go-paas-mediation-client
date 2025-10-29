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
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func Test_UpdateOrCreateSecret_CreateNew_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	ctx := context.Background()
	secret := entity.Secret{Metadata: entity.Metadata{Name: testSecret, Namespace: testNamespace1},
		Data: map[string][]byte{"body": {102, 97, 108, 99, 111, 110}}}
	clientset := fake.NewClientset(&namespace)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	updatedSecret, err := kube.UpdateOrCreateSecret(ctx, &secret, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, updatedSecret)
}

func Test_UpdateOrCreateSecret_Update_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	secretForClientSet := v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace1},
		Data: map[string][]byte{"body": {15, 11, 10}}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &secretForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	secret := entity.Secret{Metadata: entity.Metadata{Name: testSecret, Namespace: testNamespace1},
		Data: map[string][]byte{"body": {102, 97, 108, 99, 111, 110}}}
	updatedSecret, err := kube.UpdateOrCreateSecret(ctx, &secret, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, updatedSecret)
	assert.Equal(t, secret.Data, updatedSecret.Data)
}

func Test_DeleteSecret_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	secretForClientSet := v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace1},
		Data: map[string][]byte{"body": {15, 11, 10}}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &secretForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	err := kube.DeleteSecret(ctx, testSecret, testNamespace1)
	assert.Nil(t, err)
}

func Test_GetSecret_success(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	secretForClientSet := v1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace1},
		Data: map[string][]byte{"body": {15, 11, 10}}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &secretForClientSet)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	secret, err := kube.GetSecret(ctx, testSecret, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
}

func Test_GetSecret_usingCache_success(t *testing.T) {
	ctx := context.Background()

	secretTest := corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testNamespace1},
		Data: map[string][]byte{"body": {15, 11, 10}}}

	clientset := fake.NewClientset()
	clientset.Discovery().(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{GitVersion: "v1.23.0"}
	clientset.PrependReactor("get", "secrets", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
	})
	cert_client := &certClient.Clientset{}

	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	kube.Cache = cache.NewTestResourcesCache()
	ok, err := kube.Cache.Secrets.Set(ctx, *entity.NewSecret(&secretTest))
	assert.NoError(t, err)
	assert.True(t, ok)

	secret, err := kube.GetSecret(ctx, testSecret, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
}

func Test_GetSecretWithServiceAccountToken_kuber1_23(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	secretForClientSet := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount + "-token", Namespace: testNamespace1,
			Annotations: map[string]string{ServiceAccountNameAnnotation: testServiceAccount}},
		Data: map[string][]byte{"token": []byte("data")},
		Type: SecretTypeServiceAccountTokenAnnotation}
	serviceAccount := v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount, Namespace: testNamespace1},
		Secrets:    []v1.ObjectReference{{Name: testServiceAccount + "-token"}},
	}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &secretForClientSet, &serviceAccount)
	cert_client := &certClient.Clientset{}
	fakeDiscoveryClient := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscoveryClient.FakedServerVersion = &version.Info{GitVersion: "v1.23.0"}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	secret, err := kube.GetOrCreateSecretWithServiceAccountToken(ctx, testServiceAccount, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, secretForClientSet.Name, secret.Name)
}

func Test_GetSecretWithServiceAccountToken_kuber1_24(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	secretForClientSet := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount + "-token", Namespace: testNamespace1,
			Annotations: map[string]string{ServiceAccountNameAnnotation: testServiceAccount}},
		Data: map[string][]byte{"token": []byte("data")},
		Type: SecretTypeServiceAccountTokenAnnotation}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace, &secretForClientSet)
	cert_client := &certClient.Clientset{}
	fakeDiscoveryClient := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscoveryClient.FakedServerVersion = &version.Info{GitVersion: "v1.24.0"}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	secret, err := kube.GetOrCreateSecretWithServiceAccountToken(ctx, testServiceAccount, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, secretForClientSet.Name, secret.Name)
}

func Test_GetSecretWithServiceAccountToken_kuber1_24_SecretNotPresent(t *testing.T) {
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace1}}
	ctx := context.Background()
	clientset := fake.NewClientset(&namespace)
	cert_client := &certClient.Clientset{}
	fakeDiscoveryClient := clientset.Discovery().(*fakediscovery.FakeDiscovery)
	fakeDiscoveryClient.FakedServerVersion = &version.Info{GitVersion: "v1.24.0"}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientset, CertmanagerInterface: cert_client})
	secretGetCounter := 0
	clientset.Fake.PrependReactor("get", "secrets",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			defer func() { secretGetCounter++ }()
			if secretGetCounter == 0 {
				return true, ret, errors.NewNotFound(schema.GroupResource{}, testServiceAccount+"-token")
			} else if secretGetCounter == 1 {
				secret := v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount + "-token", Namespace: testNamespace1,
						Annotations: map[string]string{ServiceAccountNameAnnotation: testServiceAccount}},
					Type: SecretTypeServiceAccountTokenAnnotation}
				return true, &secret, nil
			} else if secretGetCounter >= 2 {
				secret := v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: testServiceAccount + "-token", Namespace: testNamespace1,
						Annotations: map[string]string{ServiceAccountNameAnnotation: testServiceAccount}},
					Data: map[string][]byte{"token": []byte("data")},
					Type: SecretTypeServiceAccountTokenAnnotation}
				return true, &secret, nil
			}
			return true, ret, nil
		})

	secret, err := kube.GetOrCreateSecretWithServiceAccountToken(ctx, testServiceAccount, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, testServiceAccount+"-token", secret.Name)
	assert.Equal(t, SecretTypeServiceAccountTokenAnnotation, secret.Type)
	assert.NotNil(t, secret.Annotations)
	assert.Equal(t, testServiceAccount, secret.Annotations[ServiceAccountNameAnnotation])
	secretFromClientSet, err := clientset.CoreV1().Secrets(testNamespace1).Get(ctx, testServiceAccount+"-token", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, secretFromClientSet)
	assert.Equal(t, testServiceAccount+"-token", secretFromClientSet.Name)
	assert.Equal(t, SecretTypeServiceAccountTokenAnnotation, string(secretFromClientSet.Type))
	assert.NotNil(t, secret.Annotations)
	assert.Equal(t, testServiceAccount, secretFromClientSet.Annotations[ServiceAccountNameAnnotation])
}
