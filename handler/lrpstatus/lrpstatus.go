package lrpstatus

import (
	"encoding/json"
	"net/http"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	tpshelpers "github.com/cloudfoundry-incubator/tps/helpers"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"

	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/v1"
	v1core "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"
	"k8s.io/kubernetes/pkg/labels"
)

type handler struct {
	k8sClient v1core.CoreInterface
	clock     clock.Clock
	logger    lager.Logger
}

func NewHandler(k8sClient v1core.CoreInterface, clk clock.Clock, logger lager.Logger) http.Handler {
	return &handler{
		k8sClient: k8sClient,
		clock:     clk,
		logger:    logger,
	}
}

func (handler *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	guid := r.FormValue(":guid")
	logger := handler.logger.Session("lrp-status", lager.Data{"process-guid": guid})

	logger.Info("fetching-actual-lrp-info")

	pg, err := helpers.NewProcessGuid(guid)
	if err != nil {
		logger.Error("invalid-process-guid", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Info("shortened-process-guid", lager.Data{"shortened-process-guid": pg.ShortenedGuid()})

	actualPodsList, err := handler.k8sClient.Pods(api.NamespaceAll).List(api.ListOptions{
		LabelSelector: labels.Set{"cloudfoundry.org/process-guid": pg.ShortenedGuid()}.AsSelector(),
	})

	if err != nil {
		logger.Error("failed-fetching-actual-lrp-info", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	instances := LRPInstances(actualPodsList.Items,
		handler.clock,
	)

	logger.Debug("failed-fetching-actual-lrp-info-instances", lager.Data{"instances": instances})
	err = json.NewEncoder(w).Encode(instances)
	if err != nil {
		logger.Error("stream-response-failed", err)
	}
}

func LRPInstances(
	actualPods []v1.Pod,
	clk clock.Clock,
) []cc_messages.LRPInstance {
	instances := make([]cc_messages.LRPInstance, len(actualPods))

	j := 0

	actualPods = tpshelpers.SortPods(actualPods)

	for i, pod := range actualPods {
		//actual, _ := actualLRPGroup.Resolve()
		shortenedGuid := pod.ObjectMeta.Labels["cloudfoundry.org/process-guid"]
		processGuid, err := helpers.DecodeProcessGuid(shortenedGuid)

		if err != nil {
			// ignore this LRPInstance
			//logger.Error("error get process guid", err)
		}
		instanceState := getApplicationContainerState(pod)
		if instanceState != "" {
			instance := cc_messages.LRPInstance{
				ProcessGuid:  processGuid.String(), // TODO: convert it to full pg
				InstanceGuid: string(pod.ObjectMeta.UID),
				Index:        uint(i),
				Since:        pod.Status.StartTime.UnixNano() / 1e9,
				Uptime:       (clk.Now().UnixNano() - pod.Status.StartTime.UnixNano()) / 1e9,
				State:        instanceState,
			}
			instances[j] = instance
			j = j + 1
		}
	}

	// only return instances up to j
	if j >= 1 {
		return instances[0:j]
	} else {
		return nil
	}
}

// return nil if we cannot find container name == "application"
func getApplicationContainerState(pod v1.Pod) cc_messages.LRPInstanceState {
	containerStatuses := pod.Status.ContainerStatuses
	for _, containerStatus := range containerStatuses {
		if containerStatus.Name == "application" {
			if containerStatus.State.Waiting != nil {
				return cc_messages.LRPInstanceStateStarting
			} else if containerStatus.State.Running != nil {
				return cc_messages.LRPInstanceStateRunning
			} else if containerStatus.State.Terminated != nil {
				return cc_messages.LRPInstanceStateDown
			} else {
				return cc_messages.LRPInstanceStateUnknown
			}
		}
	}

	return ""
}
