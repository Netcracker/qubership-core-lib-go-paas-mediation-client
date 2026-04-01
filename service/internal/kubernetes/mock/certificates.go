package mock

import (
	"context"
	"errors"

	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	clientCertV1 "github.com/cert-manager/cert-manager/pkg/client/applyconfigurations/certmanager/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeCertificates struct {
	Fake *FakeCertmanagerV1
	ns   string
}

func (f *FakeCertificates) Apply(_ context.Context, _ *clientCertV1.CertificateApplyConfiguration, _ metav1.ApplyOptions) (result *v1.Certificate, err error) {
	panic("not implemented")
}

func (f *FakeCertificates) ApplyStatus(_ context.Context, _ *clientCertV1.CertificateApplyConfiguration, _ metav1.ApplyOptions) (result *v1.Certificate, err error) {
	panic("not implemented")
}

func (f *FakeCertificates) Create(_ context.Context, _ *v1.Certificate, _ metav1.CreateOptions) (*v1.Certificate, error) {
	panic("not implemented")
}

func (f *FakeCertificates) Update(_ context.Context, _ *v1.Certificate, _ metav1.UpdateOptions) (*v1.Certificate, error) {
	panic("not implemented")
}

func (f *FakeCertificates) UpdateStatus(_ context.Context, _ *v1.Certificate, _ metav1.UpdateOptions) (*v1.Certificate, error) {
	panic("not implemented")
}

func (f *FakeCertificates) Delete(_ context.Context, _ string, _ metav1.DeleteOptions) error {
	panic("not implemented")
}

func (f *FakeCertificates) DeleteCollection(_ context.Context, _ metav1.DeleteOptions, _ metav1.ListOptions) error {
	panic("not implemented")
}

func (f *FakeCertificates) Get(_ context.Context, _ string, _ metav1.GetOptions) (*v1.Certificate, error) {
	return nil, errors.New("error test")
}

func (f *FakeCertificates) List(_ context.Context, _ metav1.ListOptions) (*v1.CertificateList, error) {
	return nil, errors.New("error test")
}

func (f *FakeCertificates) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	panic("not implemented")
}

func (f *FakeCertificates) Patch(_ context.Context, _ string, _ types.PatchType, _ []byte, _ metav1.PatchOptions, _ ...string) (result *v1.Certificate, err error) {
	panic("not implemented")
}
