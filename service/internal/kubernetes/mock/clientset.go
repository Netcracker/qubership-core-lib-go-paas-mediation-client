package mock

import (
	cmclientset "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned"
	acmev1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/client/clientset/versioned/typed/certmanager/v1"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	clientset "k8s.io/client-go/kubernetes"
	admissionregistrationv1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1"
	admissionregistrationv1alfa1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
	internalv1alpha1 "k8s.io/client-go/kubernetes/typed/apiserverinternal/v1alpha1"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	appsv1beta1 "k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	appsv1beta2 "k8s.io/client-go/kubernetes/typed/apps/v1beta2"
	authenticationv1 "k8s.io/client-go/kubernetes/typed/authentication/v1"
	authenticationv1alfa1 "k8s.io/client-go/kubernetes/typed/authentication/v1alpha1"
	authenticationv1beta1 "k8s.io/client-go/kubernetes/typed/authentication/v1beta1"
	authorizationv1 "k8s.io/client-go/kubernetes/typed/authorization/v1"
	authorizationv1beta1 "k8s.io/client-go/kubernetes/typed/authorization/v1beta1"
	autoscalingv1 "k8s.io/client-go/kubernetes/typed/autoscaling/v1"
	autoscalingv2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2"
	autoscalingv2beta1 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta1"
	autoscalingv2beta2 "k8s.io/client-go/kubernetes/typed/autoscaling/v2beta2"
	batchv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	batchv1beta1 "k8s.io/client-go/kubernetes/typed/batch/v1beta1"
	certificatesv1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	certificatesv1alpha1 "k8s.io/client-go/kubernetes/typed/certificates/v1alpha1"
	certificatesv1beta1 "k8s.io/client-go/kubernetes/typed/certificates/v1beta1"
	coordinationv1 "k8s.io/client-go/kubernetes/typed/coordination/v1"
	coordinationv1alpha2 "k8s.io/client-go/kubernetes/typed/coordination/v1alpha2"
	coordinationv1beta1 "k8s.io/client-go/kubernetes/typed/coordination/v1beta1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	discoveryv1 "k8s.io/client-go/kubernetes/typed/discovery/v1"
	discoveryv1beta1 "k8s.io/client-go/kubernetes/typed/discovery/v1beta1"
	eventsv1 "k8s.io/client-go/kubernetes/typed/events/v1"
	eventsv1beta1 "k8s.io/client-go/kubernetes/typed/events/v1beta1"
	extensionsv1beta1 "k8s.io/client-go/kubernetes/typed/extensions/v1beta1"
	v1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1"
	flowcontrolv1beta1 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta1"
	flowcontrolv1beta2 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta2"
	flowcontrolv1beta3 "k8s.io/client-go/kubernetes/typed/flowcontrol/v1beta3"
	networkingv1 "k8s.io/client-go/kubernetes/typed/networking/v1"
	networkingv1alpha1 "k8s.io/client-go/kubernetes/typed/networking/v1alpha1"
	networkingv1beta1 "k8s.io/client-go/kubernetes/typed/networking/v1beta1"
	nodev1 "k8s.io/client-go/kubernetes/typed/node/v1"
	nodev1alpha1 "k8s.io/client-go/kubernetes/typed/node/v1alpha1"
	nodev1beta1 "k8s.io/client-go/kubernetes/typed/node/v1beta1"
	policyv1 "k8s.io/client-go/kubernetes/typed/policy/v1"
	policyv1beta1 "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	rbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
	rbacv1alpha1 "k8s.io/client-go/kubernetes/typed/rbac/v1alpha1"
	rbacv1beta1 "k8s.io/client-go/kubernetes/typed/rbac/v1beta1"
	resourcev1alpha3 "k8s.io/client-go/kubernetes/typed/resource/v1alpha3"
	"k8s.io/client-go/kubernetes/typed/resource/v1beta1"
	"k8s.io/client-go/kubernetes/typed/resource/v1beta2"
	schedulingv1 "k8s.io/client-go/kubernetes/typed/scheduling/v1"
	schedulingv1alpha1 "k8s.io/client-go/kubernetes/typed/scheduling/v1alpha1"
	schedulingv1beta1 "k8s.io/client-go/kubernetes/typed/scheduling/v1beta1"
	storagev1 "k8s.io/client-go/kubernetes/typed/storage/v1"
	storagev1alpha1 "k8s.io/client-go/kubernetes/typed/storage/v1alpha1"
	storagev1beta1 "k8s.io/client-go/kubernetes/typed/storage/v1beta1"
	storagemigrationv1alpha1 "k8s.io/client-go/kubernetes/typed/storagemigration/v1alpha1"
	"k8s.io/client-go/testing"
)

type KubeClientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
	tracker   testing.ObjectTracker
}

func (c *KubeClientset) ResourceV1beta2() v1beta2.ResourceV1beta2Interface {
	panic("implement me")
}

func (c *KubeClientset) NetworkingV1alpha1() v1alpha1.NetworkingV1alpha1Interface {
	panic("implement me")
}

func (c *KubeClientset) FlowcontrolV1() v1.FlowcontrolV1Interface {
	panic("not implemented")
}

func (c *KubeClientset) StoragemigrationV1alpha1() storagemigrationv1alpha1.StoragemigrationV1alpha1Interface {
	panic("not implemented")
}

func (c *KubeClientset) AdmissionregistrationV1alpha1() admissionregistrationv1alfa1.AdmissionregistrationV1alpha1Interface {
	panic("not implemented")
}

func (c *KubeClientset) AuthenticationV1alpha1() authenticationv1alfa1.AuthenticationV1alpha1Interface {
	panic("not implemented")
}

func (c *KubeClientset) FlowcontrolV1beta3() flowcontrolv1beta3.FlowcontrolV1beta3Interface {
	panic("implement me")
}

func (c *KubeClientset) ResourceV1alpha3() resourcev1alpha3.ResourceV1alpha3Interface {
	panic("implement me")
}

func (c *KubeClientset) CertificatesV1alpha1() certificatesv1alpha1.CertificatesV1alpha1Interface {
	panic("not implemented")
}

var _ clientset.Interface = &KubeClientset{}

func (c *KubeClientset) NetworkingV1alpha1() networkingv1alpha1.NetworkingV1alpha1Interface {
	panic("not implemented")
}

func (c *KubeClientset) Tracker() testing.ObjectTracker {
	return c.tracker
}

func (c *KubeClientset) Discovery() discovery.DiscoveryInterface { return c.discovery }
func (c *KubeClientset) AdmissionregistrationV1() admissionregistrationv1.AdmissionregistrationV1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AdmissionregistrationV1beta1() admissionregistrationv1beta1.AdmissionregistrationV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) InternalV1alpha1() internalv1alpha1.InternalV1alpha1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AppsV1() appsv1.AppsV1Interface                { panic("not implemented") }
func (c *KubeClientset) AppsV1beta1() appsv1beta1.AppsV1beta1Interface { panic("not implemented") }
func (c *KubeClientset) AppsV1beta2() appsv1beta2.AppsV1beta2Interface { panic("not implemented") }
func (c *KubeClientset) AuthenticationV1() authenticationv1.AuthenticationV1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AuthenticationV1beta1() authenticationv1beta1.AuthenticationV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AuthorizationV1() authorizationv1.AuthorizationV1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AuthorizationV1beta1() authorizationv1beta1.AuthorizationV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AutoscalingV1() autoscalingv1.AutoscalingV1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AutoscalingV2() autoscalingv2.AutoscalingV2Interface {
	panic("not implemented")
}
func (c *KubeClientset) AutoscalingV2beta1() autoscalingv2beta1.AutoscalingV2beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) AutoscalingV2beta2() autoscalingv2beta2.AutoscalingV2beta2Interface {
	panic("not implemented")
}
func (c *KubeClientset) BatchV1() batchv1.BatchV1Interface                { panic("not implemented") }
func (c *KubeClientset) BatchV1beta1() batchv1beta1.BatchV1beta1Interface { panic("not implemented") }
func (c *KubeClientset) CertificatesV1() certificatesv1.CertificatesV1Interface {
	panic("not implemented")
}
func (c *KubeClientset) CertificatesV1beta1() certificatesv1beta1.CertificatesV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) CoordinationV1() coordinationv1.CoordinationV1Interface {
	panic("not implemented")
}
func (c *KubeClientset) CoordinationV1beta1() coordinationv1beta1.CoordinationV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) CoordinationV1alpha2() coordinationv1alpha2.CoordinationV1alpha2Interface {
	panic("not implemented")
}
func (c *KubeClientset) ResourceV1beta1() v1beta1.ResourceV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) CoreV1() corev1.CoreV1Interface                { return &FakeCoreV1{Fake: &c.Fake} }
func (c *KubeClientset) DiscoveryV1() discoveryv1.DiscoveryV1Interface { panic("not implemented") }
func (c *KubeClientset) DiscoveryV1beta1() discoveryv1beta1.DiscoveryV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) EventsV1() eventsv1.EventsV1Interface { panic("not implemented") }
func (c *KubeClientset) EventsV1beta1() eventsv1beta1.EventsV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) ExtensionsV1beta1() extensionsv1beta1.ExtensionsV1beta1Interface {
	return &FakeExtensionsV1beta1{Fake: &c.Fake}
}
func (c *KubeClientset) FlowcontrolV1beta1() flowcontrolv1beta1.FlowcontrolV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) FlowcontrolV1beta2() flowcontrolv1beta2.FlowcontrolV1beta2Interface {
	panic("not implemented")
}
func (c *KubeClientset) NetworkingV1() networkingv1.NetworkingV1Interface { panic("not implemented") }
func (c *KubeClientset) NetworkingV1beta1() networkingv1beta1.NetworkingV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) NodeV1() nodev1.NodeV1Interface                   { panic("not implemented") }
func (c *KubeClientset) NodeV1alpha1() nodev1alpha1.NodeV1alpha1Interface { panic("not implemented") }
func (c *KubeClientset) NodeV1beta1() nodev1beta1.NodeV1beta1Interface    { panic("not implemented") }
func (c *KubeClientset) PolicyV1() policyv1.PolicyV1Interface             { panic("not implemented") }
func (c *KubeClientset) PolicyV1beta1() policyv1beta1.PolicyV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) RbacV1() rbacv1.RbacV1Interface                   { panic("not implemented") }
func (c *KubeClientset) RbacV1beta1() rbacv1beta1.RbacV1beta1Interface    { panic("not implemented") }
func (c *KubeClientset) RbacV1alpha1() rbacv1alpha1.RbacV1alpha1Interface { panic("not implemented") }
func (c *KubeClientset) SchedulingV1alpha1() schedulingv1alpha1.SchedulingV1alpha1Interface {
	panic("not implemented")
}
func (c *KubeClientset) SchedulingV1beta1() schedulingv1beta1.SchedulingV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) SchedulingV1() schedulingv1.SchedulingV1Interface { panic("not implemented") }
func (c *KubeClientset) StorageV1beta1() storagev1beta1.StorageV1beta1Interface {
	panic("not implemented")
}
func (c *KubeClientset) StorageV1() storagev1.StorageV1Interface { panic("not implemented") }
func (c *KubeClientset) StorageV1alpha1() storagev1alpha1.StorageV1alpha1Interface {
	panic("not implemented")
}

type CmClientset struct {
	testing.Fake
}

var _ cmclientset.Interface = &CmClientset{}

func (c *CmClientset) Discovery() discovery.DiscoveryInterface {
	panic("not implemented")
}

func (c *CmClientset) AcmeV1() acmev1.AcmeV1Interface {
	panic("not implemented")
}

func (c *CmClientset) CertmanagerV1() certmanagerv1.CertmanagerV1Interface {
	return &FakeCertmanagerV1{Fake: &c.Fake}
}
