package tpsrunner

import (
	"os/exec"

	"github.com/tedsuo/ifrit/ginkgomon"
)

func NewListener(bin, listenAddr, kubeCluster, kubeClientCert, kubeClientKey, kubeCACert, trafficControllerURL, consulCluster string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name: "tps-listener",
		Command: exec.Command(
			bin,
			"-kubeCluster", kubeCluster,
			"-kubeClientCert", kubeClientCert,
			"-kubeClientKey", kubeClientKey,
			"-kubeCACert", kubeCACert,
			"-listenAddr", listenAddr,
			"-trafficControllerURL", trafficControllerURL,
			"-skipSSLVerification",
			"-consulCluster", consulCluster,
		),
		StartCheck: "tps-listener.started",
	})
}

func NewWatcher(bin, bbsAddress, ccBaseURL, consulCluster string) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name: "tps-watcher",
		Command: exec.Command(
			bin,
			"-bbsAddress", bbsAddress,
			"-ccBaseURL", ccBaseURL,
			"-lockRetryInterval", "1s",
			"-consulCluster", consulCluster,
		),
		StartCheck: "tps-watcher.started",
	})
}
