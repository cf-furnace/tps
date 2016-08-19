package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHandler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handler Suite")
}

func newTestRequest(body interface{}) *http.Request {
	var reader io.Reader
	switch body := body.(type) {
	case string:
		reader = strings.NewReader(body)
	case []byte:
		reader = bytes.NewReader(body)
	default:
		jsonBytes, err := json.Marshal(body)
		Expect(err).NotTo(HaveOccurred())
		reader = bytes.NewReader(jsonBytes)
	}

	request, err := http.NewRequest("", "", reader)
	Expect(err).NotTo(HaveOccurred())
	return request
}

//go:generate counterfeiter -o handler_fakes/fake_replication_controller.go --fake-name FakeReplicationController ../../../../k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1 ReplicationControllerInterface
//go:generate counterfeiter -o handler_fakes/fake_pod.go --fake-name FakePod ../../../../k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1 PodInterface
//go:generate counterfeiter -o handler_fakes/fake_k8s_client.go --fake-name FakeKubeClient ../../../../k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1 CoreInterface
//go:generate counterfeiter -o handler_fakes/fake_namespace.go --fake-name FakeNamespace ../../../../k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1 NamespaceInterface
