package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/cloudfoundry-incubator/cf-debug-server"
	"github.com/cloudfoundry-incubator/cf-lager"
	"github.com/cloudfoundry-incubator/consuladapter"
	"github.com/cloudfoundry-incubator/locket"
	"github.com/cloudfoundry-incubator/tps/handler"
	"github.com/cloudfoundry/dropsonde"
	"github.com/cloudfoundry/noaa/consumer"
	"github.com/hashicorp/consul/api"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"

	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3"
	"k8s.io/kubernetes/pkg/client/restclient"
)

var listenAddr = flag.String(
	"listenAddr",
	"0.0.0.0:1518", // p and s's offset in the alphabet, do not change
	"listening address of api server",
)

var dropsondePort = flag.Int(
	"dropsondePort",
	3457,
	"port the local metron agent is listening on",
)

var trafficControllerURL = flag.String(
	"trafficControllerURL",
	"",
	"URL of TrafficController",
)

var skipSSLVerification = flag.Bool(
	"skipSSLVerification",
	true,
	"Skip SSL verification",
)

var maxInFlightRequests = flag.Int(
	"maxInFlightRequests",
	200,
	"number of requests to handle at a time; any more will receive 503",
)

var kubeCluster = flag.String(
	"kubeCluster",
	"",
	"kubernetes API server URL (scheme://ip:port)",
)

var kubeCACert = flag.String(
	"kubeCACert",
	"",
	"path to kubernetes API server CA certificate",
)

var kubeClientCert = flag.String(
	"kubeClientCert",
	"",
	"path to client certificate for authentication with the kubernetes API server",
)

var kubeClientKey = flag.String(
	"kubeClientKey",
	"",
	"path to client key for authentication with the kubernetes API server",
)

var bulkLRPStatusWorkers = flag.Int(
	"bulkLRPStatusWorkers",
	15,
	"Max concurrency for fetching bulk lrps",
)

var consulCluster = flag.String(
	"consulCluster",
	"",
	"Consul Agent URL",
)

const (
	dropsondeOrigin = "tps_listener"
)

func main() {
	cf_debug_server.AddFlags(flag.CommandLine)
	cf_lager.AddFlags(flag.CommandLine)
	flag.Parse()

	logger, reconfigurableSink := cf_lager.New("tps-listener")
	initializeDropsonde(logger)
	noaaClient := consumer.New(*trafficControllerURL, &tls.Config{InsecureSkipVerify: *skipSSLVerification}, nil)
	defer noaaClient.Close()
	clientSet := initializeK8sClient(logger)
	apiHandler := initializeHandler(logger, noaaClient, *maxInFlightRequests, clientSet)

	consulClient, err := consuladapter.NewClientFromUrl(*consulCluster)
	if err != nil {
		logger.Fatal("new-client-failed", err)
	}

	registrationRunner := initializeRegistrationRunner(logger, consulClient, *listenAddr, clock.NewClock())

	members := grouper.Members{
		{"api", http_server.New(*listenAddr, apiHandler)},
		{"registration-runner", registrationRunner},
	}

	if dbgAddr := cf_debug_server.DebugAddress(flag.CommandLine); dbgAddr != "" {
		members = append(grouper.Members{
			{"debug-server", cf_debug_server.Runner(dbgAddr, reconfigurableSink)},
		}, members...)
	}

	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))

	logger.Info("started")

	err = <-monitor.Wait()
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

func initializeHandler(logger lager.Logger, noaaClient *consumer.Consumer, maxInFlight int, k8sClient clientset.Interface) http.Handler {
	apiHandler, err := handler.New(k8sClient.Core(), noaaClient, maxInFlight, *bulkLRPStatusWorkers, logger)
	if err != nil {
		logger.Fatal("initialize-handler.failed", err)
	}

	return apiHandler
}

func initializeRegistrationRunner(logger lager.Logger, consulClient consuladapter.Client, listenAddress string, clock clock.Clock) ifrit.Runner {
	_, portString, err := net.SplitHostPort(listenAddress)
	if err != nil {
		logger.Fatal("failed-invalid-listen-address", err)
	}
	portNum, err := net.LookupPort("tcp", portString)
	if err != nil {
		logger.Fatal("failed-invalid-listen-port", err)
	}

	registration := &api.AgentServiceRegistration{
		Name: "tps",
		Port: portNum,
		Check: &api.AgentServiceCheck{
			TTL: "3s",
		},
	}

	return locket.NewRegistrationRunner(logger, registration, consulClient, locket.RetryInterval, clock)
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
