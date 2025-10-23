package kubernetes

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	fake_cert "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/fake"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fake_k8s "k8s.io/client-go/kubernetes/fake"
	kube_test "k8s.io/client-go/testing"
)

func Test_GetCertificate_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	certificate := v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCertificate,
			Namespace: testNamespace1,
		},
		Spec: v1.CertificateSpec{},
	}
	k8sclient := fake_k8s.NewClientset()
	certclient := fake_cert.NewSimpleClientset(&certificate)
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8sclient, CertmanagerInterface: certclient})
	// found
	result, err := kube.GetCertificate(ctx, testCertificate, testNamespace1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "certificate must be not nil")
	r.Equal(testCertificate, result.Metadata.Name)

	// not found
	result2, err2 := kube.GetCertificate(ctx, "test-name1", testNamespace1)
	r.Nil(result2, "certificate must be nil")
	r.Nil(err2, "unexpected error returned")
}

// todo certificates cache not supported yet
//func Test_GetCertificate_usingCache_success(t *testing.T) {
//	ctx := context.Background()
//
//	k8s_client := fake_k8s.NewSimpleClientset()
//	cert_client := fake_cert.NewSimpleClientset()
//	cert_client.PrependReactor("*", "*", func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
//		return true, nil, errors.NewInternalError(fmt.Errorf("test api server error"))
//	})
//	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})
//	kube.Cache = cache.NewTestResourcesCache()
//
//	certificateTest := entity.Certificate{Metadata: entity.Metadata{Name: testCertificate, Namespace: testNamespace1}}
//	ok, err := kube.Cache.Certificates.Set(ctx, certificateTest)
//	assert.NoError(t, err)
//	assert.True(t, ok)
//
//	certificate, err := kube.GetCertificate(ctx, testCertificate, testNamespace1)
//	assert.Nil(t, err)
//	assert.NotNil(t, certificate)
//}

func Test_GetCertificate_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset()
	expectedError := fmt.Errorf("test error during list Certificate")
	cert_client.Fake.PrependReactor("get", "certificates",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	_, err := kube.GetCertificate(ctx, testCertificate, testNamespace1)
	r.Equal(expectedError, err, "unexpected error returned")
}

// there are 3 certificates:
// certificate1 {name: "test-name1", labels: {"test-label": "test-label-name"}, annotations: {"test-annotation":"test-annotation-name"}}
// certificate2 {name: "test-name2", labels: {"test-label": "test-label-name"}, annotations: {"test-annotation":"test-annotation-name2"}}
// certificate2 {name: "test-name3", labels: {"test-label": "test-label-name3"}}
//
// filter1 {labels: {"test-label": "test-label-name"}, annotations: {"test-annotation":"test-annotation-name"}}
// expectation: certificate1
//
// filter2 {annotations: {"test-annotation":"test-annotation-name2"}}
// expectation: certificate2
func Test_GetCertificateList_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	labels1 := make(map[string]string)
	labels1["test-label"] = "test-label-name"
	annotations1 := make(map[string]string)
	annotations1["test-annotation"] = "test-annotation-name"
	certificate1 := v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-name1",
			Namespace:   testNamespace1,
			Labels:      labels1,
			Annotations: annotations1,
		},
		Spec: v1.CertificateSpec{}}

	labels2 := make(map[string]string)
	labels2["test-label"] = "test-label-name"
	annotations2 := make(map[string]string)
	annotations2["test-annotation"] = "test-annotation-name2"
	certificate2 := v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-name2",
			Namespace:   testNamespace1,
			Labels:      labels2,
			Annotations: annotations2,
		},
		Spec: v1.CertificateSpec{}}

	labels3 := make(map[string]string)
	labels3["test-label"] = "test-label-name3"
	certificate3 := v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-name3",
			Namespace: testNamespace1,
			Labels:    labels3,
		},
		Spec: v1.CertificateSpec{}}

	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset(&certificate1, &certificate2, &certificate3)
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	filter1 := filter.Meta{
		Labels:      labels1,
		Annotations: annotations1,
	}

	certificates, err := kube.GetCertificateList(ctx, testNamespace1, filter1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(certificates, "certificate must be not nil")
	r.Equal(1, len(certificates), "expected 1 certificate")
	r.Equal("test-name1", (certificates)[0].Metadata.Name, "expected first certificate")

	filter2 := filter.Meta{
		Annotations: annotations2,
	}

	certificates2, err := kube.GetCertificateList(ctx, testNamespace1, filter2)
	r.Nil(err, "unexpected error returned")
	r.NotNil(certificates2, "certificate must be not nil")
	r.Equal(1, len(certificates2), "expected 1 certificate")
	r.Equal("test-name2", (certificates2)[0].Metadata.Name, "expected second certificate")

	filter3 := filter.Meta{}

	certificates3, err := kube.GetCertificateList(ctx, testNamespace1, filter3)
	r.Nil(err, "unexpected error returned")
	r.NotNil(certificates3, "certificate must be not nil")
	r.Equal(3, len(certificates3), "expected 3 certificate")
}

func Test_GetCertificateList_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset()
	expectedError := fmt.Errorf("test error during list Certificate")
	cert_client.Fake.PrependReactor("list", "certificates",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	labels1 := make(map[string]string)
	labels1["test-label"] = "test-label-name"

	filter2 := filter.Meta{
		Labels: labels1,
	}

	_, err := kube.GetCertificateList(ctx, testNamespace1, filter2)
	r.Equal(expectedError, err, "unexpected error returned")
}

func Test_CreateCertificate_Success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset()
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	certificate := entity.Certificate{
		Metadata: entity.Metadata{
			Name:      testCertificate,
			Namespace: testNamespace1,
		},
		Spec:   entity.CertificateSpec{},
		Status: entity.CertificateStatus{},
	}
	result, err := kube.CreateCertificate(ctx, &certificate, testNamespace1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "certificate must be not nil")
	r.Equal(testCertificate, result.Metadata.Name)
}

func Test_CreateCertificate_Error(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset()
	expectedError := fmt.Errorf("test error during list Certificate")
	cert_client.Fake.PrependReactor("create", "certificates",
		func(action kube_test.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError
		})
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	certificate := entity.Certificate{
		Metadata: entity.Metadata{},
		Spec:     entity.CertificateSpec{},
		Status:   entity.CertificateStatus{},
	}
	_, err := kube.CreateCertificate(ctx, &certificate, testNamespace1)
	r.Equal(expectedError, err, "unexpected error returned")
}

func Test_UpdateOrCreateCertificate_CreateNew_success(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset()
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	certificate := entity.Certificate{
		Metadata: entity.Metadata{
			Name:      testCertificate,
			Namespace: testNamespace1,
		},
		Spec:   entity.CertificateSpec{},
		Status: entity.CertificateStatus{},
	}
	result, err := kube.UpdateOrCreateCertificate(ctx, &certificate, testNamespace1)
	r.Nil(err, "unexpected error returned")
	r.NotNil(result, "certificate must be not nil")
	r.Equal(testCertificate, result.Metadata.Name)
}

func Test_UpdateOrCreateCertificate_Update_success(t *testing.T) {
	ctx := context.Background()

	certificateForClientSet := v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCertificate,
			Namespace: testNamespace1,
		},
		Spec: v1.CertificateSpec{
			SecretName: "secret-name",
		},
	}
	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset(&certificateForClientSet)
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})
	certificate := entity.Certificate{
		Metadata: entity.Metadata{
			Name:      testCertificate,
			Namespace: testNamespace1,
		},
		Spec: entity.CertificateSpec{
			SecretName: "new-secret-name",
		},
		Status: entity.CertificateStatus{},
	}
	updatedCertificate, err := kube.UpdateOrCreateCertificate(ctx, &certificate, testNamespace1)
	assert.Nil(t, err)
	assert.NotNil(t, updatedCertificate)
	assert.Equal(t, certificate.Spec.SecretName, updatedCertificate.Spec.SecretName)
}

func Test_DeleteCertificate_Success(t *testing.T) {
	ctx := context.Background()

	certificate := v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testCertificate,
			Namespace: testNamespace1,
		},
		Spec: v1.CertificateSpec{},
	}
	k8s_client := fake_k8s.NewClientset()
	cert_client := fake_cert.NewSimpleClientset(&certificate)
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: k8s_client, CertmanagerInterface: cert_client})

	err := kube.DeleteCertificate(ctx, testCertificate, testNamespace1)
	assert.Nil(t, err)
}
