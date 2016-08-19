package lrpstatus_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/lager/lagertest"

	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	handlerfakes "github.com/cloudfoundry-incubator/tps/handler/handler_fakes"
	"github.com/cloudfoundry-incubator/tps/handler/lrpstatus"

	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/clock/fakeclock"
)

var _ = Describe("LRPStatus", func() {
	var (
		fakeKubeClient *handlerfakes.FakeKubeClient
		handler        http.Handler
		response       *httptest.ResponseRecorder
		request        *http.Request
		//server      *httptest.Server
		fakePod      *handlerfakes.FakePod
		pod1         *v1.Pod
		pod2         *v1.Pod
		processGuid1 helpers.ProcessGuid
		processGuid2 helpers.ProcessGuid
		err          error
		fakeClock    *fakeclock.FakeClock
		logger       *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		fakeKubeClient = &handlerfakes.FakeKubeClient{}
		fakeClock = fakeclock.NewFakeClock(time.Now())
		processGuid1, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())
		processGuid2, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())

		fakePod = &handlerfakes.FakePod{}
		now := unversioned.Now()
		pod1 = &v1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:              "pod-name",
				Namespace:         "namespace",
				UID:               "1234-5677",
				CreationTimestamp: now,
				Labels: map[string]string{
					"cloudfoundry.org/app-guid":     processGuid1.AppGuid.String(),
					"cloudfoundry.org/space-guid":   "my-space-id",
					"cloudfoundry.org/process-guid": processGuid1.ShortenedGuid(),
					"cloudfoundry.org/domain":       cc_messages.AppLRPDomain,
				},
				Annotations: map[string]string{},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Name:  "application",
						Image: "cloudfoundry/cflinuxfs2:latest",
					},
				},
			},
			Status: v1.PodStatus{
				StartTime: &now,
				ContainerStatuses: []v1.ContainerStatus{
					v1.ContainerStatus{
						Name: "application",
						State: v1.ContainerState{
							Running: &v1.ContainerStateRunning{},
						},
						Ready:        true,
						RestartCount: int32(0),
					},
				},
			},
		}

		pod2 = &v1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:              "pod-name",
				Namespace:         "namespace",
				UID:               "1234-5678",
				CreationTimestamp: now,
				Labels: map[string]string{
					"cloudfoundry.org/app-guid":     processGuid2.AppGuid.String(),
					"cloudfoundry.org/space-guid":   "my-space-id",
					"cloudfoundry.org/process-guid": processGuid2.ShortenedGuid(),
					"cloudfoundry.org/domain":       cc_messages.AppLRPDomain,
				},
				Annotations: map[string]string{},
			},
			Spec: v1.PodSpec{
				Containers: []v1.Container{
					v1.Container{
						Name:  "application2",
						Image: "cloudfoundry/cflinuxfs2:latest",
					},
				},
			},
			Status: v1.PodStatus{
				StartTime: &now,
				ContainerStatuses: []v1.ContainerStatus{
					v1.ContainerStatus{
						Name: "application2",
						State: v1.ContainerState{
							Running: &v1.ContainerStateRunning{},
						},
						Ready:        true,
						RestartCount: int32(0),
					},
				},
			},
		}

		handler = lrpstatus.NewHandler(fakeKubeClient, fakeClock, logger)

		request, err = http.NewRequest("POST", "", nil)
		Expect(err).NotTo(HaveOccurred())
		request.Form = url.Values{
			":guid": []string{processGuid1.String()},
		}

		response = httptest.NewRecorder()
	})

	JustBeforeEach(func() {
		handler.ServeHTTP(response, request)
	})

	Describe("Instance state", func() {
		BeforeEach(func() {
			fakeKubeClient.PodsReturns(fakePod)
			fakePod.ListReturns(&v1.PodList{
				Items: []v1.Pod{*pod1},
			}, nil)
		})

		It("returns correct instance & state", func() {
			res := []cc_messages.LRPInstance{}
			err = json.NewDecoder(response.Body).Decode(&res)

			Expect(err).NotTo(HaveOccurred())

			Expect(res).To(HaveLen(1))
			Expect(res[0].State).To(Equal(cc_messages.LRPInstanceStateRunning))
		})
	})

	Describe("Instance state", func() {
		BeforeEach(func() {
			fakeKubeClient.PodsReturns(fakePod)
			fakePod.ListReturns(&v1.PodList{
				Items: []v1.Pod{*pod1, *pod2},
			}, nil)
		})

		It("ignores containers whose name is not called application", func() {
			res := []cc_messages.LRPInstance{}
			err = json.NewDecoder(response.Body).Decode(&res)

			Expect(err).NotTo(HaveOccurred())

			Expect(res).To(HaveLen(1))
			Expect(res[0].State).To(Equal(cc_messages.LRPInstanceStateRunning))
		})
	})
})

func generateProcessGuid() (helpers.ProcessGuid, error) {
	appGuid, _ := uuid.NewV4()

	appVersion, _ := uuid.NewV4()

	return helpers.NewProcessGuid(appGuid.String() + "-" + appVersion.String())
}
