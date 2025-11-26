package kubernetes

import (
	"context"
	"fmt"
	"testing"

	certClient "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/entity"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/filter"
	"github.com/netcracker/qubership-core-lib-go-paas-mediation-client/v8/service/backend"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	paasErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	kubeTest "k8s.io/client-go/testing"
)

func TestKubernetes_GetNamespace(t *testing.T) {
	r := require.New(t)
	namespace1 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace1}}
	expectedNamespace1 := entity.NewNamespace(&namespace1)
	ctx := context.Background()
	clientSet := fake.NewClientset(&namespace1)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientSet, CertmanagerInterface: cert_client})
	res, err := kube.GetNamespace(ctx, testNamespace1)
	r.NoError(err)
	r.NotNil(res)
	r.Equal(*res, *expectedNamespace1)
}

func TestKubernetes_GetNamespaceErr(t *testing.T) {
	r := require.New(t)
	namespace1 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace1, Namespace: testNamespace1}}
	expectedNamespace1 := entity.NewNamespace(&namespace1)
	ctx := context.Background()
	clientSet := fake.NewClientset(&namespace1)
	expectedError := paasErrors.StatusError{ErrStatus: metaV1.Status{Reason: metaV1.StatusFailure}}
	clientSet.Fake.PrependReactor("get", "namespaces",
		func(action kubeTest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, &expectedError
		})
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientSet, CertmanagerInterface: cert_client})
	res, err := kube.GetNamespace(ctx, testNamespace1)
	r.NoError(err)
	r.NotNil(res)
	r.Equal(*res, *expectedNamespace1)
}

func TestKubernetes_GetNamespaces(t *testing.T) {
	r := require.New(t)
	namespace1 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace1}}
	namespace2 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace2}}
	expectedNamespace1 := entity.NewNamespace(&namespace1)
	expectedNamespace2 := entity.NewNamespace(&namespace2)
	ctx := context.Background()
	clientSet := fake.NewClientset(&namespace1, &namespace2)
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientSet, CertmanagerInterface: cert_client})
	res, err := kube.GetNamespaces(ctx, filter.Meta{})
	r.Nil(err)
	r.NotNil(res)
	r.Equal(2, len(res))
	r.Contains(res, *expectedNamespace1)
	r.Contains(res, *expectedNamespace2)
}

func TestKubernetes_GetNamespacesForbiddenError(t *testing.T) {
	r := require.New(t)
	namespace1 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace1}}
	namespace2 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace2}}
	expectedNamespace1 := entity.NewNamespace(&namespace1)
	expectedError := paasErrors.StatusError{ErrStatus: metaV1.Status{Reason: metaV1.StatusReasonForbidden}}
	ctx := context.Background()
	clientSet := fake.NewClientset(&namespace1, &namespace2)
	clientSet.Fake.PrependReactor("list", "namespaces",
		func(action kubeTest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, &expectedError
		})
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientSet, CertmanagerInterface: cert_client})
	res, err := kube.GetNamespaces(ctx, filter.Meta{})
	r.Nil(err)
	r.NotNil(res)
	r.Equal(1, len(res))
	r.Contains(res, *expectedNamespace1)
}

func TestKubernetes_GetNamespacesForbiddenError_And_GetNamespaceError(t *testing.T) {
	r := require.New(t)
	namespace1 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace1}}
	namespace2 := v1.Namespace{ObjectMeta: metaV1.ObjectMeta{Name: testNamespace2}}
	expectedNamespace1 := entity.Namespace{Metadata: entity.Metadata{Kind: "Namespace", Name: testNamespace1, Namespace: testNamespace1}}
	expectedError1 := paasErrors.StatusError{ErrStatus: metaV1.Status{Reason: metaV1.StatusReasonForbidden}}
	expectedError2 := fmt.Errorf("error on get namespace")
	ctx := context.Background()
	clientSet := fake.NewClientset(&namespace1, &namespace2)
	clientSet.Fake.PrependReactor("list", "namespaces",
		func(action kubeTest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, &expectedError1
		})
	clientSet.Fake.PrependReactor("get", "namespace",
		func(action kubeTest.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, expectedError2
		})
	cert_client := &certClient.Clientset{}
	kube, _ := NewTestKubernetesClient(testNamespace1, &backend.KubernetesApi{KubernetesInterface: clientSet, CertmanagerInterface: cert_client})
	res, err := kube.GetNamespaces(ctx, filter.Meta{})
	r.Nil(err)
	r.NotNil(res)
	r.Equal(1, len(res))
	r.Contains(res, expectedNamespace1)
}

func TestKubernetes_WatchNamespaces(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()
	cert_client := &certClient.Clientset{}
	kube := &Kubernetes{client: &backend.KubernetesApi{KubernetesInterface: &kubernetes.Clientset{}, CertmanagerInterface: cert_client},
		WatchExecutor: newFakeWatchExecutor(), namespace: testNamespace1}
	res, err := kube.WatchNamespaces(ctx, testNamespace1)
	r.Nil(err)
	r.NotNil(res)
}
