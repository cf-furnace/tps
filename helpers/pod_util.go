package helpers

import (
	"sort"

	"k8s.io/kubernetes/pkg/api/v1"
)

// simple sort of a pod based on pod uid
func SortPods(actualPods []v1.Pod) []v1.Pod {
	// sort the pods by the pod UID
	sortedPods := []v1.Pod{}
	sortedUID := []string{}
	for _, pod := range actualPods {
		sortedUID = append(sortedUID, string(pod.ObjectMeta.UID))

	}
	sort.Strings(sortedUID)
	for _, uid := range sortedUID {
		// get the pod for the given uid
		for _, pod := range actualPods {
			if string(pod.ObjectMeta.UID) == uid {
				sortedPods = append(sortedPods, pod)
				continue
			}
		}
	}

	return sortedPods
}
