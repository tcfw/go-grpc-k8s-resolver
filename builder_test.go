package k8sresolver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetNamespace(t *testing.T) {
	ns := getNamespaceFromHost("test.notdefault")
	assert.Equal(t, "notdefault", ns)

	ns = getNamespaceFromHost("test")
	assert.Equal(t, "default", ns)

	ns = getNamespaceFromHost("test.maybedefault.cluster")
	assert.Equal(t, "maybedefault", ns)
}
