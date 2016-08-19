package bulklrpstatus_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/tps/handler/bulklrpstatus"
	handlerfakes "github.com/cloudfoundry-incubator/tps/handler/handler_fakes"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Bulk Status", func() {
	const authorization = "something good"
	const guid1 = "my-guid1"
	const guid2 = "my-guid2"
	const logGuid1 = "log-guid1"
	const logGuid2 = "log-guid2"

	var (
		handler           http.Handler
		response          *httptest.ResponseRecorder
		request           *http.Request
		fakeKubeClient    *handlerfakes.FakeKubeClient
		logger            *lagertest.TestLogger
		fakeClock         *fakeclock.FakeClock
		fakePod           *handlerfakes.FakePod
		pod1              *v1.Pod
		pod2              *v1.Pod
		processGuid1      helpers.ProcessGuid
		processGuid2      helpers.ProcessGuid
		expectedSinceTime time.Time
		actualSinceTime   time.Time
	)

	BeforeEach(func() {
		var err error

		fakeKubeClient = &handlerfakes.FakeKubeClient{}
		logger = lagertest.NewTestLogger("test")
		fakeClock = fakeclock.NewFakeClock(time.Date(2008, 8, 8, 8, 8, 8, 8, time.UTC))
		handler = bulklrpstatus.NewHandler(fakeKubeClient, fakeClock, 15, logger)
		response = httptest.NewRecorder()
		url := "/v1/bulk_actual_lrp_status"
		request, err = http.NewRequest("GET", url, nil)
		Expect(err).NotTo(HaveOccurred())

		processGuid1, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())

		processGuid2, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())

		fakePod = &handlerfakes.FakePod{}
		expectedSinceTime = fakeClock.Now()
		actualSinceTime = fakeClock.Now()
		fakeClock.Increment(5 * time.Second)
		actualTime := unversioned.NewTime(actualSinceTime)

		pod1 = &v1.Pod{
			ObjectMeta: v1.ObjectMeta{
				Name:              "pod-name1",
				Namespace:         "namespace",
				UID:               "1234-5677",
				CreationTimestamp: unversioned.NewTime(actualSinceTime),
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
				StartTime: &actualTime,
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
				Name:              "pod-name2",
				Namespace:         "namespace",
				UID:               "1234-5678",
				CreationTimestamp: unversioned.NewTime(actualSinceTime),
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
						Name:  "application",
						Image: "cloudfoundry/cflinuxfs2:latest",
					},
				},
			},
			Status: v1.PodStatus{
				StartTime: &actualTime,
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
	})

	JustBeforeEach(func() {
		handler.ServeHTTP(response, request)
	})

	Describe("Validation", func() {
		BeforeEach(func() {
			request.Header.Set("Authorization", authorization)
		})

		Context("with no process guids", func() {
			It("fails with missing process guids", func() {
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})

		Context("with malformed process guids", func() {
			BeforeEach(func() {
				query := request.URL.Query()
				query.Set("guids", fmt.Sprintf("%s,,%s", guid1, guid2))
				request.URL.RawQuery = query.Encode()
			})

			It("fails", func() {
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("retrieves instance state for lrps specified", func() {
		var (
		//netInfo1, netInfo2                 models.ActualLRPNetInfo
		)

		BeforeEach(func() {

			// netInfo1 = models.NewActualLRPNetInfo(
			// 	"host",
			// 	models.NewPortMapping(5432, 7890),
			// 	models.NewPortMapping(1234, uint32(recipebuilder.DefaultPort)),
			// )
			// netInfo2 = models.NewActualLRPNetInfo(
			// 	"host2",
			// 	models.NewPortMapping(5432, 7890),
			// 	models.NewPortMapping(1234, uint32(recipebuilder.DefaultPort)),
			// )

			request.Header.Set("Authorization", authorization)

			query := request.URL.Query()
			query.Set("guids", fmt.Sprintf("%s,%s", processGuid1.String(), processGuid2.String()))
			request.URL.RawQuery = query.Encode()

			fakeKubeClient.PodsReturns(fakePod)
			fakePod.ListStub = func(opts api.ListOptions) (*v1.PodList, error) {
				if opts.LabelSelector.String() == "cloudfoundry.org/process-guid="+processGuid1.ShortenedGuid() {
					// return pod1
					return &v1.PodList{
						Items: []v1.Pod{*pod1},
					}, nil
				} else {
					return &v1.PodList{
						Items: []v1.Pod{*pod2},
					}, nil
				}
			}

		})

		Context("when the LRPs have been running for a while", func() {
			It("returns a map of status per index", func() {
				expectedLRPInstance1 := cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: "1234-5677",
					//NetInfo:      netInfo1,
					Index:  0,
					State:  cc_messages.LRPInstanceStateRunning,
					Since:  expectedSinceTime.Unix(),
					Uptime: 5,
				}
				expectedLRPInstance2 := cc_messages.LRPInstance{
					ProcessGuid:  processGuid2.String(),
					InstanceGuid: "1234-5678",
					//NetInfo:      netInfo2,
					Index:  0,
					State:  cc_messages.LRPInstanceStateRunning,
					Since:  expectedSinceTime.Unix(),
					Uptime: 5,
				}

				status := make(map[string][]cc_messages.LRPInstance)

				Expect(response.Code).To(Equal(http.StatusOK))
				Expect(response.Header().Get("Content-Type")).To(Equal("application/json"))

				err := json.Unmarshal(response.Body.Bytes(), &status)
				Expect(err).NotTo(HaveOccurred())
				fmt.Printf("status=%v", status)
				Expect(status[processGuid1.String()][0]).To(Equal(expectedLRPInstance1))
				Expect(status[processGuid2.String()][0]).To(Equal(expectedLRPInstance2))
			})
		})

		Context("when fetching one of the actualLRPs fails", func() {
			BeforeEach(func() {
				fakeKubeClient.PodsReturns(fakePod)
				fakePod.ListStub = func(opts api.ListOptions) (*v1.PodList, error) {
					if opts.LabelSelector.String() == "cloudfoundry.org/process-guid="+processGuid1.ShortenedGuid() {
						// return pod1
						return &v1.PodList{
							Items: []v1.Pod{*pod1},
						}, nil
					} else if opts.LabelSelector.String() == "cloudfoundry.org/process-guid="+processGuid2.ShortenedGuid() {
						return nil, errors.New("boom")
					} else {
						return nil, errors.New("UNEXPECTED GUID YO")
					}
				}
			})

			It("it is excluded from the result and logs the failure", func() {
				status := make(map[string][]cc_messages.LRPInstance)

				Expect(response.Code).To(Equal(http.StatusOK))
				Expect(response.Header().Get("Content-Type")).To(Equal("application/json"))

				err := json.Unmarshal(response.Body.Bytes(), &status)
				Expect(err).NotTo(HaveOccurred())

				Expect(len(status)).To(Equal(1))
				Expect(status[guid2]).To(BeNil())
				Expect(logger).To(Say("fetching-actual-lrps-info-failed"))
			})
		})
	})
})

func generateProcessGuid() (helpers.ProcessGuid, error) {
	appGuid, _ := uuid.NewV4()

	appVersion, _ := uuid.NewV4()

	return helpers.NewProcessGuid(appGuid.String() + "-" + appVersion.String())
}
