package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"

	"github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/locket"
	"github.com/cloudfoundry-incubator/tps"
	"github.com/cloudfoundry-incubator/tps/cc_client"
	"github.com/cloudfoundry-incubator/tps/watcher"
	"github.com/cloudfoundry/dropsonde"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"

	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3"
	"k8s.io/kubernetes/pkg/client/restclient"
)

var consulCluster = flag.String(
	"consulCluster",
	"",
	"comma-separated list of consul server URLs (scheme://ip:port)",
)

var lockTTL = flag.Duration(
	"lockTTL",
	locket.LockTTL,
	"TTL for service lock",
)

var lockRetryInterval = flag.Duration(
	"lockRetryInterval",
	locket.RetryInterval,
	"interval to wait before retrying a failed lock acquisition",
)

var dropsondePort = flag.Int(
	"dropsondePort",
	3457,
	"port the local metron agent is listening on",
)

var ccBaseURL = flag.String(
	"ccBaseURL",
	"",
	"URI to acccess the Cloud Controller",
)

var ccUsername = flag.String(
	"ccUsername",
	"",
	"Basic auth username for CC internal API",
)

var ccPassword = flag.String(
	"ccPassword",
	"",
	"Basic auth password for CC internal API",
)

var skipCertVerify = flag.Bool(
	"skipCertVerify",
	false,
	"skip SSL certificate verification",
)

var eventHandlingWorkers = flag.Int(
	"eventHandlingWorkers",
	500,
	"Max concurrency for handling lrp events",
)

const (
	dropsondeOrigin = "tps_watcher"
)

func main() {
	cf_debug_server.AddFlags(flag.CommandLine)
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, reconfigurableSink := cf_lager.New("tps-watcher")
	initializeDropsonde(logger)

	lockMaintainer := initializeLockMaintainer(logger)

	ccClient := cc_client.NewCcClient(*ccBaseURL, *ccUsername, *ccPassword, *skipCertVerify)

	watcher := ifrit.RunFunc(func(signals <-chan os.Signal, ready chan<- struct{}) error {

		w, err := watcher.NewWatcher(logger,
			*eventHandlingWorkers,
			watcher.DefaultRetryPauseInterval,
			initializeK8sClient(logger).Core(), ccClient)

		if err != nil {
			return err
		}

		return w.Run(signals, ready)
	})

	members := grouper.Members{
		{"lock-maintainer", lockMaintainer},
		{"watcher", watcher},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		members = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, reconfigurableSink)},
		}, members...)
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	logger.Info("started")

	err := <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}

	logger.Info("exited")
}

func initializeDropsonde(logger lager.Logger) {
	dropsondeDestination := fmt.Sprint("localhost:", *dropsondePort)
	err := dropsonde.Initialize(dropsondeDestination, dropsondeOrigin)
	if err != nil {
		logger.Error("failed to initialize dropsonde: %v", err)
	}
}

func initializeServiceClient(logger lager.Logger) tps.ServiceClient {
	consulClient, err := consuladapter.NewClientFromUrl(*consulCluster)
	if err != nil {
		logger.Fatal("new-client-failed", err)
	}

	return tps.NewServiceClient(consulClient, clock.NewClock())
}

func initializeLockMaintainer(logger lager.Logger) ifrit.Runner {
	serviceClient := initializeServiceClient(logger)

	uuid, err := uuid.NewV4()
	if err != nil {
		logger.Fatal("Couldn't generate uuid", err)
	}

	return serviceClient.NewTPSWatcherLockRunner(logger, uuid.String(), *lockRetryInterval, *lockTTL)
}

func initializeK8sClient(logger lager.Logger) clientset.Interface {
	k8sClient, err := clientset.NewForConfig(&restclient.Config{
		Host: *kubeCluster,
		TLSClientConfig: restclient.TLSClientConfig{
			CertFile: *kubeClientCert,
			KeyFile:  *kubeClientKey,
			CAFile:   *kubeCACert,
		},
	})

	if err != nil {
		logger.Fatal("Can't create Kubernetes Client", err, lager.Data{"address": *kubeCluster})
	}

	return k8sClient
}
