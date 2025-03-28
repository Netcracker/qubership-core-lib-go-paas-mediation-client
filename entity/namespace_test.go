package entity

import (
	osV1 "github.com/openshift/api/project/v1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"testing"
)

func Test_NewNamespace(t *testing.T) {
	namespace := getNamespaceV1()
	expectedNamespace := getEntityNamespace(namespace.ObjectMeta)
	testedNamespace := NewNamespace(namespace)
	assert.Equal(t, expectedNamespace, testedNamespace)
}

func Test_NewNamespaceFromOsProject(t *testing.T) {
	project := getProjectV1()
	expectedNamespace := getEntityNamespace(project.ObjectMeta)
	namespace := NewNamespaceFromOsProject(project)
	assert.Equal(t, expectedNamespace, namespace)
}

func getNamespaceV1() *v1.Namespace {
	return &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test", UID: uuid.NewUUID(),
		Annotations: map[string]string{"test": "test1"},
		Labels:      map[string]string{"test": "test2"}}}
}

func getProjectV1() *osV1.Project {
	return &osV1.Project{ObjectMeta: metav1.ObjectMeta{Name: "test", UID: uuid.NewUUID(),
		Annotations: map[string]string{"test": "test1"},
		Labels:      map[string]string{"test": "test2"}}}
}

func getEntityNamespace(meta metav1.ObjectMeta) *Namespace {
	metadata := NewMetadata("Namespace", meta.Name, meta.Name, string(meta.UID), meta.Generation,
		meta.ResourceVersion, meta.Annotations, meta.Labels)
	return &Namespace{Metadata: metadata}
}
