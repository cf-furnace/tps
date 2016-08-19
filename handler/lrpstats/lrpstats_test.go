package lrpstats_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	kubeerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	handlerfakes "github.com/cloudfoundry-incubator/tps/handler/handler_fakes"
	"github.com/cloudfoundry-incubator/tps/handler/lrpstats"
	"github.com/cloudfoundry-incubator/tps/handler/lrpstats/fakes"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/nu7hatch/gouuid"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
)

var _ = Describe("Stats", func() {
	const authorization = "something good"
	const logGuid = "log-guid"

	var (
		handler        http.Handler
		response       *httptest.ResponseRecorder
		request        *http.Request
		noaaClient     *fakes.FakeNoaaClient
		fakeKubeClient *handlerfakes.FakeKubeClient
		logger         *lagertest.TestLogger
		fakeClock      *fakeclock.FakeClock
		fakePod        *handlerfakes.FakePod
		pod1           *v1.Pod
		processGuid1   helpers.ProcessGuid
		err            error
	)

	BeforeEach(func() {
		var err error

		fakeKubeClient = &handlerfakes.FakeKubeClient{}
		noaaClient = &fakes.FakeNoaaClient{}
		logger = lagertest.NewTestLogger("test")
		fakeClock = fakeclock.NewFakeClock(time.Date(2008, 8, 8, 8, 8, 8, 8, time.UTC))
		handler = lrpstats.NewHandler(fakeKubeClient, noaaClient, fakeClock, logger)
		response = httptest.NewRecorder()
		request, err = http.NewRequest("GET", "/v1/actual_lrps/:guid/stats", nil)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		handler.ServeHTTP(response, request)
	})

	Describe("Validation", func() {
		It("fails with a missing authorization header", func() {
			Expect(response.Code).To(Equal(http.StatusUnauthorized))
		})

		Context("with an authorization header", func() {
			BeforeEach(func() {
				request.Header.Set("Authorization", authorization)
			})

			It("fails with no guid", func() {
				Expect(response.Code).To(Equal(http.StatusBadRequest))
			})
		})
	})

	Describe("retrieve container metrics", func() {
		// var netInfo models.ActualLRPNetInfo

		BeforeEach(func() {

			processGuid1, err = generateProcessGuid()
			Expect(err).NotTo(HaveOccurred())

			request.Header.Set("Authorization", authorization)
			request.Form = url.Values{}
			request.Form.Add(":guid", processGuid1.String())

			noaaClient.ContainerMetricsReturns([]*events.ContainerMetric{
				{
					ApplicationId: proto.String("appId"),
					InstanceIndex: proto.Int32(5),
					CpuPercentage: proto.Float64(4),
					MemoryBytes:   proto.Uint64(1024),
					DiskBytes:     proto.Uint64(2048),
				},
			}, nil)

			// bbsClient.DesiredLRPByProcessGuidReturns(&models.DesiredLRP{
			// 	LogGuid:     logGuid,
			// 	ProcessGuid: guid,
			// }, nil)

			// netInfo = models.NewActualLRPNetInfo(
			// 	"host",
			// 	models.NewPortMapping(5432, 7890),
			// 	models.NewPortMapping(1234, uint32(recipebuilder.DefaultPort)),
			// )

			fakePod = &handlerfakes.FakePod{}
			actualTime := unversioned.NewTime(fakeClock.Now())
			pod1 = &v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:              "pod-name",
					Namespace:         "namespace",
					UID:               "1234-5677",
					CreationTimestamp: actualTime,
					Labels: map[string]string{
						"cloudfoundry.org/app-guid":     processGuid1.AppGuid.String(),
						"cloudfoundry.org/space-guid":   "my-space-id",
						"cloudfoundry.org/process-guid": processGuid1.ShortenedGuid(),
						"cloudfoundry.org/domain":       cc_messages.AppLRPDomain,
					},
					Annotations: map[string]string{
						"cloudfoundry.org/log-guid": logGuid,
					},
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

			fakeKubeClient.PodsReturns(fakePod)
			fakePod.ListReturns(&v1.PodList{
				Items: []v1.Pod{*pod1},
			}, nil)
		})

		Context("when the LRP has terminated", func() {
			var expectedSinceTime int64

			BeforeEach(func() {
				expectedSinceTime = fakeClock.Now().Unix()
				fakeClock.Increment(5 * time.Second)
				//actualLRP.State = models.ActualLRPStateCrashed
				pod1.Status.ContainerStatuses[0].State = v1.ContainerState{
					Terminated: &v1.ContainerStateTerminated{},
				}
			})

			It("returns a map of stats & status per index in the correct units", func() {
				expectedLRPInstance := cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: "1234-5677",
					Index:        0,
					State:        cc_messages.LRPInstanceStateDown,
					//Host:         "host",
					//Port:         1234,
					//NetInfo:      netInfo,
					Since:  expectedSinceTime,
					Uptime: 5,
					// Stats: &cc_messages.LRPInstanceStats{
					// 	Time:          time.Unix(0, 0),
					// 	CpuPercentage: 0,
					// 	MemoryBytes:   0,
					// 	DiskBytes:     0,
					// },
				}
				var stats []cc_messages.LRPInstance

				Expect(response.Code).To(Equal(http.StatusOK))
				Expect(response.Header().Get("Content-Type")).To(Equal("application/json"))
				err := json.Unmarshal(response.Body.Bytes(), &stats)
				Expect(err).NotTo(HaveOccurred())
				//Expect(stats[0].Stats.Time).NotTo(BeZero())
				//expectedLRPInstance.Stats.Time = stats[0].Stats.Time
				Expect(stats).To(ConsistOf(expectedLRPInstance))
			})
		})

		Context("when the LRP has been running for a while", func() {
			var expectedSinceTime int64

			BeforeEach(func() {
				expectedSinceTime = fakeClock.Now().Unix()
				fakeClock.Increment(5 * time.Second)
			})

			It("returns a map of stats & status per index in the correct units", func() {
				expectedLRPInstance := cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: "1234-5677",
					Index:        0,
					State:        cc_messages.LRPInstanceStateRunning,
					//Host:         "host",
					//Port:         1234,
					//NetInfo:      netInfo,
					Since:  expectedSinceTime,
					Uptime: 5,
					// Stats: &cc_messages.LRPInstanceStats{
					// 	Time:          time.Unix(0, 0),
					// 	CpuPercentage: 0.04,
					// 	MemoryBytes:   1024,
					// 	DiskBytes:     2048,
					// },
				}
				var stats []cc_messages.LRPInstance

				Expect(response.Code).To(Equal(http.StatusOK))
				Expect(response.Header().Get("Content-Type")).To(Equal("application/json"))
				err := json.Unmarshal(response.Body.Bytes(), &stats)
				Expect(err).NotTo(HaveOccurred())
				//Expect(stats[0].Stats.Time).NotTo(BeZero())
				//expectedLRPInstance.Stats.Time = stats[0].Stats.Time
				Expect(stats).To(ConsistOf(expectedLRPInstance))
			})
		})

		It("calls ContainerMetrics", func() {
			Expect(noaaClient.ContainerMetricsCallCount()).To(Equal(1))
			guid, token := noaaClient.ContainerMetricsArgsForCall(0)
			Expect(guid).To(Equal(logGuid))
			Expect(token).To(Equal(authorization))
		})

		Context("when ContainerMetrics fails", func() {
			var expectedLRPInstance cc_messages.LRPInstance
			BeforeEach(func() {
				noaaClient.ContainerMetricsReturns(nil, errors.New("bad stuff happened"))
				expectedLRPInstance = cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: "1234-5677",
					Index:        0,
					State:        cc_messages.LRPInstanceStateRunning,
					// Host:         "host",
					// Port:         1234,
					//NetInfo:      netInfo,
					Since:  fakeClock.Now().Unix(),
					Uptime: 0,
					Stats:  nil,
				}
			})

			It("responds with empty stats", func() {
				var stats []cc_messages.LRPInstance
				Expect(response.Code).To(Equal(http.StatusOK))
				Expect(response.Header().Get("Content-Type")).To(Equal("application/json"))
				err := json.Unmarshal(response.Body.Bytes(), &stats)
				Expect(err).NotTo(HaveOccurred())
				Expect(stats).To(ConsistOf(expectedLRPInstance))
			})

			It("logs the failure", func() {
				Expect(logger).To(Say("container-metrics-failed"))
			})
		})

		Context("when fetching the desiredLRP fails", func() {
			Context("when the desiredLRP is not found", func() {
				BeforeEach(func() {
					fakePod.ListReturns(&v1.PodList{
						Items: []v1.Pod{},
					}, &kubeerrors.StatusError{
						ErrStatus: unversioned.Status{
							Message: "replication controller not found",
							Status:  unversioned.StatusFailure,
							Reason:  unversioned.StatusReasonNotFound,
							Code:    http.StatusNotFound,
						},
					})
				})

				It("responds with a 404", func() {
					Expect(response.Code).To(Equal(http.StatusNotFound))
				})
			})

			Context("when another type of error occurs", func() {
				BeforeEach(func() {
					fakePod.ListReturns(&v1.PodList{
						Items: []v1.Pod{},
					}, errors.New("garbage"))
				})

				It("responds with a 500", func() {
					Expect(response.Code).To(Equal(http.StatusInternalServerError))
				})
			})
		})

		Context("when fetching actualLRPs fails", func() {
			BeforeEach(func() {
				fakePod.ListReturns(nil, errors.New("bad stuff happened"))
			})

			It("responds with a 500", func() {
				Expect(response.Code).To(Equal(http.StatusInternalServerError))
			})

			It("logs the failure", func() {
				Expect(logger).To(Say("fetching-actual-lrp-info-failed"))
			})
		})
	})
})

func generateProcessGuid() (helpers.ProcessGuid, error) {
	appGuid, _ := uuid.NewV4()

	appVersion, _ := uuid.NewV4()

	return helpers.NewProcessGuid(appGuid.String() + "-" + appVersion.String())
}
