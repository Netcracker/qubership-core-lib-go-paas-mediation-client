package entity

import (
	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

var (
	duration, _    = time.ParseDuration("2160h")
	renewBefore, _ = time.ParseDuration("1440h")
	testTime       = time.Now()
)

func Test_NewCertificate_Success(t *testing.T) {
	certificate := getV1Certificate()
	certificateExpected := getCertificate()

	result := NewCertificate(certificate)

	assert.Equalf(t, certificateExpected, *result, "Not expected Certificate")
}

func Test_NewCertificateList_Success(t *testing.T) {
	certificates := []*v1.Certificate{
		getV1Certificate(),
	}
	certificatesExpected := []Certificate{
		getCertificate(),
	}

	result := NewCertificateList(certificates)

	assert.Equalf(t, certificatesExpected, result, "Not expected Certificate")
}

func Test_ToCertificate_Success(t *testing.T) {
	certificate := getCertificate()
	certificateExpected := getV1Certificate()

	result := certificate.ToCertificate()

	assert.Equalf(t, certificateExpected, result, "Not expected Certificate")
}

func getCertificate() Certificate {
	return Certificate{
		Metadata: Metadata{
			Kind:       "Certificate",
			Name:       "test-certificate",
			Generation: 1,
		},
		Spec: CertificateSpec{
			Duration:    &duration,
			RenewBefore: &renewBefore,
			IssuerRef:   IssuerRef{},
			Usages:      []string{"server auth"},
			PrivateKey:  &CertificatePrivateKey{},
			Keystores: &CertificateKeystores{
				JKS:    &Keystore{},
				PKCS12: &Keystore{},
			},
			SecretTemplate: &CertificateSecretTemplate{},
		},
		Status: CertificateStatus{
			Conditions: []CertificateCondition{
				{
					Type:               "Ready",
					Status:             "True",
					LastTransitionTime: &testTime,
					Reason:             "test-reason",
					Message:            "test-message",
					ObservedGeneration: 1,
				},
			},
			LastFailureTime: &testTime,
			NotBefore:       &testTime,
			NotAfter:        &testTime,
			RenewalTime:     &testTime,
		},
	}
}

func getV1Certificate() *v1.Certificate {
	return &v1.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-certificate",
			Generation: 1,
		},
		Spec: v1.CertificateSpec{
			Duration:    &metav1.Duration{Duration: duration},
			RenewBefore: &metav1.Duration{Duration: renewBefore},
			IssuerRef:   cmmeta.IssuerReference{},
			Usages:      []v1.KeyUsage{"server auth"},
			PrivateKey:  &v1.CertificatePrivateKey{},
			Keystores: &v1.CertificateKeystores{
				JKS:    &v1.JKSKeystore{},
				PKCS12: &v1.PKCS12Keystore{},
			},
			SecretTemplate: &v1.CertificateSecretTemplate{},
		},
		Status: v1.CertificateStatus{
			Conditions: []v1.CertificateCondition{
				{
					Type:               "Ready",
					Status:             "True",
					LastTransitionTime: &metav1.Time{Time: testTime},
					Reason:             "test-reason",
					Message:            "test-message",
					ObservedGeneration: 1,
				},
			},
			LastFailureTime: &metav1.Time{Time: testTime},
			NotBefore:       &metav1.Time{Time: testTime},
			NotAfter:        &metav1.Time{Time: testTime},
			RenewalTime:     &metav1.Time{Time: testTime},
		},
	}
}
