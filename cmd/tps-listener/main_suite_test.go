package main_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry-incubator/consuladapter/consulrunner"
	"github.com/cloudfoundry-incubator/tps/cmd/tpsrunner"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
	v1core "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"

	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"

	"testing"
)

var (
	consulRunner *consulrunner.ClusterRunner

	listenerPort int
	listenerAddr string
	listener     ifrit.Process
	runner       *ginkgomon.Runner

	listenerPath string

	fakeCC                *ghttp.Server
	k8sClient             v1core.CoreInterface
	fakeTrafficController *ghttp.Server

	logger *lagertest.TestLogger
)

func TestTPS(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TPS-Listener Suite")
}

var _ = SynchronizedBeforeSuite(func() []byte {
	tps, err := gexec.Build("github.com/cloudfoundry-incubator/tps/cmd/tps-listener", "-race")
	Expect(err).NotTo(HaveOccurred())

	payload, err := json.Marshal(map[string]string{
		"listener": tps,
	})
	Expect(err).NotTo(HaveOccurred())

	return payload
}, func(payload []byte) {
	binaries := map[string]string{}

	err := json.Unmarshal(payload, &binaries)
	Expect(err).NotTo(HaveOccurred())

	listenerPort = 1518 + GinkgoParallelNode()

	listenerPath = string(binaries["listener"])

	consulRunner = consulrunner.NewClusterRunner(
		9001+config.GinkgoConfig.ParallelNode*consulrunner.PortOffsetLength,
		1,
		"http",
	)

	logger = lagertest.NewTestLogger("test")

	consulRunner.Start()
	consulRunner.WaitUntilReady()
})

var _ = BeforeEach(func() {
	consulRunner.Reset()
	fakeCC = ghttp.NewServer()
	fakeTrafficController = ghttp.NewTLSServer()

	listenerAddr = fmt.Sprintf("127.0.0.1:%d", uint16(listenerPort))
	home := os.Getenv("HOME")
	config, _ := clientcmd.LoadFromFile(filepath.Join(home, ".kube", "config"))

	context := config.Contexts[config.CurrentContext]

	runner = tpsrunner.NewListener(
		string(listenerPath),
		listenerAddr,
		config.Clusters[context.Cluster].Server,
		config.AuthInfos[context.AuthInfo].ClientCertificate,
		config.AuthInfos[context.AuthInfo].ClientKey,
		config.Clusters[context.Cluster].CertificateAuthority,
		fakeTrafficController.URL(),
		consulRunner.URL(),
	)
})

var _ = AfterEach(func() {
	fakeCC.Close()
	fakeTrafficController.Close()
})

var _ = SynchronizedAfterSuite(func() {
	consulRunner.Stop()
}, func() {
	gexec.CleanupBuildArtifacts()
})
