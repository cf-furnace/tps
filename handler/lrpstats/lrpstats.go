package lrpstats

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cloudfoundry-incubator/bbs/models"
	"github.com/cloudfoundry-incubator/nsync/helpers"
	"github.com/cloudfoundry-incubator/nsync/recipebuilder"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/tps/handler/lrpstatus"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	kubeerrors "k8s.io/kubernetes/pkg/api/errors"

	"k8s.io/kubernetes/pkg/api"
	v1core "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"
	"k8s.io/kubernetes/pkg/labels"
)

//go:generate counterfeiter -o fakes/fake_noaaclient.go . NoaaClient
type NoaaClient interface {
	ContainerMetrics(appGuid string, authToken string) ([]*events.ContainerMetric, error)
	Close() error
}

type handler struct {
	k8sClient  v1core.CoreInterface
	noaaClient NoaaClient
	clock      clock.Clock
	logger     lager.Logger
}

func NewHandler(k8sClient v1core.CoreInterface, noaaClient NoaaClient, clk clock.Clock, logger lager.Logger) http.Handler {
	return &handler{k8sClient: k8sClient, noaaClient: noaaClient, clock: clk, logger: logger}
}

func (handler *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")
	if authorization == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	guid := r.FormValue(":guid")

	logger := handler.logger.Session("lrp-stats", lager.Data{"process-guid": guid})

	if guid == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pg, err := helpers.NewProcessGuid(guid)
	if err != nil {
		logger.Error("invalid-process-guid", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Info("fetching-actual-lrp-info")
	actualLRPs, err := handler.k8sClient.Pods(api.NamespaceAll).List(api.ListOptions{
		LabelSelector: labels.Set{"cloudfoundry.org/process-guid": pg.ShortenedGuid()}.AsSelector(),
	})

	if actualLRPs != nil && len(actualLRPs.Items) == 0 && responseCodeFromError(err) == http.StatusNotFound {
		logger.Info("fetching-actual-lrp-not-found")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err != nil {
		logger.Error("fetching-actual-lrp-info-failed", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if actualLRPs == nil || len(actualLRPs.Items) == 0 {
		logger.Error("fetching-actual-lrp-info-failed", errors.New("invalid-actual-lrp"))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logGuid := actualLRPs.Items[0].ObjectMeta.Annotations["cloudfoundry.org/log-guid"]
	logger.Info("fetching-container-metrics", lager.Data{
		"log-guid": logGuid,
	})
	metrics, err := handler.noaaClient.ContainerMetrics(logGuid, authorization)
	if err != nil {
		handler.logger.Error("fetching-container-metrics-failed", err, lager.Data{
			"log-guid": logGuid,
		})
	}

	metricsByInstanceIndex := make(map[uint]*cc_messages.LRPInstanceStats)
	currentTime := handler.clock.Now()
	for _, metric := range metrics {
		cpuPercentageAsDecimal := metric.GetCpuPercentage() / 100
		metricsByInstanceIndex[uint(metric.GetInstanceIndex())] = &cc_messages.LRPInstanceStats{
			Time:          currentTime,
			CpuPercentage: cpuPercentageAsDecimal,
			MemoryBytes:   metric.GetMemoryBytes(),
			DiskBytes:     metric.GetDiskBytes(),
		}
	}

	instances := lrpstatus.LRPInstances(actualLRPs.Items,
		handler.clock,
	)

	for i, instance := range instances {
		if instance.State == cc_messages.LRPInstanceStateCrashed {
			instances[i].Uptime = 0
			if instances[i].Stats != nil {
				instances[i].Stats.CpuPercentage = 0
				instances[i].Stats.MemoryBytes = 0
				instances[i].Stats.DiskBytes = 0
			}
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(instances)
	if err != nil {
		handler.logger.Error("stream-response-failed", err, lager.Data{"guid": guid})
	}
}

func getDefaultPort(mappings []*models.PortMapping) uint16 {
	for _, mapping := range mappings {
		if mapping.ContainerPort == recipebuilder.DefaultPort {
			return uint16(mapping.HostPort)
		}
	}

	return 0
}

func responseCodeFromError(err error) int {
	switch err := err.(type) {
	case *kubeerrors.StatusError:
		switch err.ErrStatus.Code {
		case http.StatusNotFound:
			return http.StatusNotFound
		default:
			return http.StatusInternalServerError
		}
	default:
		return http.StatusInternalServerError
	}
}
