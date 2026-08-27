package entity

import (
	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

type (
	Certificate struct {
		Metadata `json:"metadata"`
		Spec     CertificateSpec   `json:"spec"`
		Status   CertificateStatus `json:"status"`
	}

	CertificateSpec struct {
		SecretName     string                     `json:"secretName"`
		Duration       *time.Duration             `json:"duration,omitempty"`
		RenewBefore    *time.Duration             `json:"renewBefore,omitempty"`
		CommonName     string                     `json:"commonName,omitempty"`
		DNSNames       []string                   `json:"dnsNames,omitempty"`
		IPAddresses    []string                   `json:"ipAddresses,omitempty"`
		IssuerRef      IssuerRef                  `json:"issuerRef"`
		IsCA           bool                       `json:"isCA,omitempty"`
		Usages         []string                   `json:"usages,omitempty"`
		PrivateKey     *CertificatePrivateKey     `json:"privateKey,omitempty"`
		Keystores      *CertificateKeystores      `json:"keystores,omitempty"`
		SecretTemplate *CertificateSecretTemplate `json:"secretTemplate,omitempty"`
	}

	IssuerRef struct {
		Name  string `json:"name"`
		Kind  string `json:"kind,omitempty"`
		Group string `json:"group,omitempty"`
	}

	CertificatePrivateKey struct {
		RotationPolicy string `json:"rotationPolicy,omitempty"`
		Encoding       string `json:"encoding,omitempty"`
		Algorithm      string `json:"algorithm,omitempty"`
		Size           int    `json:"size,omitempty"`
	}

	CertificateKeystores struct {
		JKS    *Keystore `json:"jks,omitempty"`
		PKCS12 *Keystore `json:"pkcs12,omitempty"`
	}

	Keystore struct {
		Create            bool              `json:"create"`
		PasswordSecretRef SecretKeySelector `json:"passwordSecretRef"`
	}

	SecretKeySelector struct {
		Name string `json:"name"`
		Key  string `json:"key,omitempty"`
	}
	CertificateSecretTemplate struct {
		Annotations map[string]string `json:"annotations,omitempty"`
		Labels      map[string]string `json:"labels,omitempty"`
	}

	CertificateStatus struct {
		Conditions               []CertificateCondition `json:"conditions,omitempty"`
		LastFailureTime          *time.Time             `json:"lastFailureTime,omitempty"`
		NotBefore                *time.Time             `json:"notBefore,omitempty"`
		NotAfter                 *time.Time             `json:"notAfter,omitempty"`
		RenewalTime              *time.Time             `json:"renewalTime,omitempty"`
		Revision                 *int                   `json:"revision,omitempty"`
		NextPrivateKeySecretName *string                `json:"nextPrivateKeySecretName,omitempty"`
		FailedIssuanceAttempts   *int                   `json:"failedIssuanceAttempts,omitempty"`
	}

	CertificateCondition struct {
		Type               string     `json:"type"`
		Status             string     `json:"status"`
		LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
		Reason             string     `json:"reason,omitempty"`
		Message            string     `json:"message,omitempty"`
		ObservedGeneration int64      `json:"observedGeneration,omitempty"`
	}
)

func NewCertificate(certificate *v1.Certificate) *Certificate {
	return &Certificate{
		Metadata: *FromObjectMeta("Certificate", &certificate.ObjectMeta),
		Spec:     NewCertificateSpec(&certificate.Spec),
		Status:   NewCertificateStatus(certificate.Status),
	}
}

func NewCertificateSpec(cs *v1.CertificateSpec) CertificateSpec {
	certificateSpec := CertificateSpec{}
	certificateSpec.SecretName = cs.SecretName
	certificateSpec.CommonName = cs.CommonName
	certificateSpec.DNSNames = cs.DNSNames
	certificateSpec.IPAddresses = cs.IPAddresses
	certificateSpec.IsCA = cs.IsCA
	if cs.Duration != nil {
		certificateSpec.Duration = &cs.Duration.Duration
	}
	if cs.RenewBefore != nil {
		certificateSpec.RenewBefore = &cs.RenewBefore.Duration
	}

	certificateSpec.IssuerRef = NewIssuerRef(cs.IssuerRef)
	if cs.PrivateKey != nil {
		certificateSpec.PrivateKey = &CertificatePrivateKey{
			RotationPolicy: string(cs.PrivateKey.RotationPolicy),
			Encoding:       string(cs.PrivateKey.Encoding),
			Algorithm:      string(cs.PrivateKey.Algorithm),
			Size:           cs.PrivateKey.Size,
		}
	}

	certificateSpec.Usages = NewKeyUsages(cs.Usages)
	certificateSpec.Keystores = NewCertificateKeystores(cs.Keystores)
	certificateSpec.SecretTemplate = NewCertificateSecretTemplate(cs.SecretTemplate)
	return certificateSpec
}

func NewIssuerRef(ir cmmeta.IssuerReference) IssuerRef {
	return IssuerRef{
		Name:  ir.Name,
		Group: ir.Group,
		Kind:  ir.Kind,
	}
}

func NewKeyUsages(ky []v1.KeyUsage) []string {
	var keyUsages []string
	for _, usage := range ky {
		keyUsages = append(keyUsages, string(usage))
	}
	return keyUsages
}

func NewCertificateKeystores(ck *v1.CertificateKeystores) *CertificateKeystores {
	if ck == nil {
		return nil
	}
	keystores := &CertificateKeystores{}
	if ck.JKS != nil {
		jks := &Keystore{}
		jks.Create = ck.JKS.Create
		jks.PasswordSecretRef.Key = ck.JKS.PasswordSecretRef.Key
		jks.PasswordSecretRef.Name = ck.JKS.PasswordSecretRef.Name
		keystores.JKS = jks
	}
	if ck.PKCS12 != nil {
		pkcs12 := &Keystore{}
		pkcs12.Create = ck.PKCS12.Create
		pkcs12.PasswordSecretRef.Key = ck.PKCS12.PasswordSecretRef.Key
		pkcs12.PasswordSecretRef.Name = ck.PKCS12.PasswordSecretRef.Name
		keystores.PKCS12 = pkcs12
	}
	return keystores
}

func NewCertificateSecretTemplate(cst *v1.CertificateSecretTemplate) *CertificateSecretTemplate {
	if cst == nil {
		return nil
	}
	return &CertificateSecretTemplate{
		Annotations: cst.Annotations,
		Labels:      cst.Labels,
	}
}

func NewCertificateStatus(cs v1.CertificateStatus) CertificateStatus {
	status := CertificateStatus{}
	status.Revision = cs.Revision
	status.NextPrivateKeySecretName = cs.NextPrivateKeySecretName
	status.FailedIssuanceAttempts = cs.FailedIssuanceAttempts
	if cs.LastFailureTime != nil {
		status.LastFailureTime = &cs.LastFailureTime.Time
	}
	if cs.NotBefore != nil {
		status.NotBefore = &cs.NotBefore.Time
	}
	if cs.RenewalTime != nil {
		status.RenewalTime = &cs.RenewalTime.Time
	}
	if cs.NotAfter != nil {
		status.NotAfter = &cs.NotAfter.Time
	}
	status.Conditions = NewCertificateConditions(cs.Conditions)
	return status
}

func NewCertificateConditions(cc []v1.CertificateCondition) []CertificateCondition {
	var conditions []CertificateCondition
	for _, certificateCondition := range cc {
		condition := CertificateCondition{}
		condition.Type = string(certificateCondition.Type)
		condition.Status = string(certificateCondition.Status)
		condition.Reason = certificateCondition.Reason
		condition.Message = certificateCondition.Message
		if certificateCondition.LastTransitionTime != nil {
			condition.LastTransitionTime = &certificateCondition.LastTransitionTime.Time
		}
		condition.ObservedGeneration = certificateCondition.ObservedGeneration
		conditions = append(conditions, condition)
	}
	return conditions
}

func NewCertificateList(kubeCertificateList []*v1.Certificate) []Certificate {
	result := make([]Certificate, 0, len(kubeCertificateList))
	for _, kubeCertificate := range kubeCertificateList {
		result = append(result, *NewCertificate(kubeCertificate))
	}
	return result
}

func (c Certificate) ToCertificate() *v1.Certificate {
	defer func() {
		if err := recover(); err != nil {
			out, _ := WriteContext()
			logger.Error("panic occurred: %s with Certificate:%s error:%s", out, c.Name, err)
		}
	}()
	return &v1.Certificate{
		ObjectMeta: *c.Metadata.ToObjectMeta(),
		Spec:       c.Spec.ToCertificateSpec(),
		Status:     c.Status.ToCertificateStatus(),
	}
}

func (cs *CertificateSpec) ToCertificateSpec() v1.CertificateSpec {
	certificateSpec := v1.CertificateSpec{}
	certificateSpec.SecretName = cs.SecretName
	certificateSpec.CommonName = cs.CommonName
	certificateSpec.DNSNames = cs.DNSNames
	certificateSpec.IPAddresses = cs.IPAddresses
	certificateSpec.IsCA = cs.IsCA
	if cs.Duration != nil {
		certificateSpec.Duration = &metav1.Duration{Duration: *cs.Duration}
	}
	if cs.RenewBefore != nil {
		certificateSpec.RenewBefore = &metav1.Duration{Duration: *cs.RenewBefore}
	}
	certificateSpec.IssuerRef = cs.IssuerRef.ToIssuerRef()

	var keyUsages []v1.KeyUsage
	for _, usage := range cs.Usages {
		keyUsages = append(keyUsages, v1.KeyUsage(usage))
	}
	certificateSpec.Usages = keyUsages

	if cs.PrivateKey != nil {
		certificateSpec.PrivateKey = cs.PrivateKey.ToCertificatePrivateKey()
	}
	if cs.Keystores != nil {
		certificateSpec.Keystores = cs.Keystores.ToCertificatePrivateKey()
	}

	if cs.SecretTemplate != nil {
		certificateSpec.SecretTemplate = &v1.CertificateSecretTemplate{
			Annotations: cs.SecretTemplate.Annotations,
			Labels:      cs.SecretTemplate.Labels,
		}
	}
	return certificateSpec
}

func (ir *IssuerRef) ToIssuerRef() cmmeta.IssuerReference {
	return cmmeta.IssuerReference{
		Name:  ir.Name,
		Kind:  ir.Kind,
		Group: ir.Group,
	}
}

func (cpk *CertificatePrivateKey) ToCertificatePrivateKey() *v1.CertificatePrivateKey {
	return &v1.CertificatePrivateKey{
		RotationPolicy: v1.PrivateKeyRotationPolicy(cpk.RotationPolicy),
		Encoding:       v1.PrivateKeyEncoding(cpk.Encoding),
		Algorithm:      v1.PrivateKeyAlgorithm(cpk.Algorithm),
		Size:           cpk.Size,
	}
}

func (ck *CertificateKeystores) ToCertificatePrivateKey() *v1.CertificateKeystores {
	keystores := &v1.CertificateKeystores{}
	if ck.JKS != nil {
		jks := &v1.JKSKeystore{}
		jks.Create = ck.JKS.Create
		jks.PasswordSecretRef.Key = ck.JKS.PasswordSecretRef.Key
		jks.PasswordSecretRef.Name = ck.JKS.PasswordSecretRef.Name
		keystores.JKS = jks
	}
	if ck.PKCS12 != nil {
		pkcs12 := &v1.PKCS12Keystore{}
		pkcs12.Create = ck.PKCS12.Create
		pkcs12.PasswordSecretRef.Key = ck.PKCS12.PasswordSecretRef.Key
		pkcs12.PasswordSecretRef.Name = ck.PKCS12.PasswordSecretRef.Name
		keystores.PKCS12 = pkcs12
	}
	return keystores
}

func (cs *CertificateStatus) ToCertificateStatus() v1.CertificateStatus {
	status := v1.CertificateStatus{}
	status.Revision = cs.Revision
	status.NextPrivateKeySecretName = cs.NextPrivateKeySecretName
	status.FailedIssuanceAttempts = cs.FailedIssuanceAttempts
	if cs.LastFailureTime != nil {
		status.LastFailureTime = &metav1.Time{Time: *cs.LastFailureTime}
	}
	if cs.NotBefore != nil {
		status.NotBefore = &metav1.Time{Time: *cs.NotBefore}
	}
	if cs.RenewalTime != nil {
		status.RenewalTime = &metav1.Time{Time: *cs.RenewalTime}
	}
	if cs.NotAfter != nil {
		status.NotAfter = &metav1.Time{Time: *cs.NotAfter}
	}
	var conditions []v1.CertificateCondition
	for _, certificateCondition := range cs.Conditions {
		condition := v1.CertificateCondition{}
		condition.Type = v1.CertificateConditionType(certificateCondition.Type)
		condition.Status = cmmeta.ConditionStatus(certificateCondition.Status)
		condition.Reason = certificateCondition.Reason
		condition.Message = certificateCondition.Message
		if certificateCondition.LastTransitionTime != nil {
			condition.LastTransitionTime = &metav1.Time{Time: *certificateCondition.LastTransitionTime}
		}
		condition.ObservedGeneration = certificateCondition.ObservedGeneration
		conditions = append(conditions, condition)
	}
	status.Conditions = conditions
	return status
}

func (c Certificate) GetMetadata() Metadata {
	return c.Metadata
}
