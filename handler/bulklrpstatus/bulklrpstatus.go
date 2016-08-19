package bulklrpstatus

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/tps/handler/lrpstatus"
	"github.com/cloudfoundry/gunk/workpool"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"k8s.io/kubernetes/pkg/api"
	v1core "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"
	"k8s.io/kubernetes/pkg/labels"
)

var processGuidPattern = regexp.MustCompile(`^([a-zA-Z0-9_-]+,)*[a-zA-Z0-9_-]+$`)

type handler struct {
	k8sClient                 v1core.CoreInterface
	clock                     clock.Clock
	logger                    lager.Logger
	bulkLRPStatusWorkPoolSize int
}

func NewHandler(k8sClient v1core.CoreInterface, clk clock.Clock, bulkLRPStatusWorkPoolSize int, logger lager.Logger) http.Handler {
	return &handler{
		k8sClient: k8sClient,
		clock:     clk,
		bulkLRPStatusWorkPoolSize: bulkLRPStatusWorkPoolSize,
		logger: logger,
	}
}

func (handler *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := handler.logger.Session("bulk-lrp-status")

	guidParameter := r.FormValue("guids")
	if !processGuidPattern.Match([]byte(guidParameter)) {
		logger.Error("failed-parsing-guids", nil, lager.Data{"guid-parameter": guidParameter})
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	guids := strings.Split(guidParameter, ",")
	works := []func(){}

	statusBundle := make(map[string][]cc_messages.LRPInstance)
	statusLock := sync.Mutex{}

	for _, processGuid := range guids {
		works = append(works, handler.getStatusForLRPWorkFunction(logger, processGuid, &statusLock, statusBundle))
	}

	throttler, err := workpool.NewThrottler(handler.bulkLRPStatusWorkPoolSize, works)
	if err != nil {
		logger.Error("failed-constructing-throttler", err, lager.Data{"max-workers": handler.bulkLRPStatusWorkPoolSize, "num-works": len(works)})
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	throttler.Work()

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(statusBundle)
	if err != nil {
		logger.Error("stream-response-failed", err, nil)
	}
}

func (handler *handler) getStatusForLRPWorkFunction(logger lager.Logger, processGuid string, statusLock *sync.Mutex, statusBundle map[string][]cc_messages.LRPInstance) func() {
	return func() {
		logger = logger.Session("fetching-actual-lrps-info", lager.Data{"process-guid": processGuid})
		logger.Info("start")
		defer logger.Info("complete")

		pg, err := helpers.NewProcessGuid(processGuid)
		if err != nil {
			logger.Error("invalid-process-guid", err)
			return
		}
		actualLRPGroups, err := handler.k8sClient.Pods(api.NamespaceAll).List(api.ListOptions{
			LabelSelector: labels.Set{"cloudfoundry.org/process-guid": pg.ShortenedGuid()}.AsSelector(),
		})
		if err != nil {
			logger.Error("fetching-actual-lrps-info-failed", err)
			return
		}

		instances := lrpstatus.LRPInstances(actualLRPGroups.Items,
			handler.clock,
		)

		statusLock.Lock()
		if instances != nil {
			statusBundle[processGuid] = instances
		}
		statusLock.Unlock()
	}
}
