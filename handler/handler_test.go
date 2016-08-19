package handler_test

import (
	"net/http"
	"net/http/httptest"
	"sync"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/unversioned"
	"k8s.io/kubernetes/pkg/api/v1"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/tps/handler"
	handlerfakes "github.com/cloudfoundry-incubator/tps/handler/handler_fakes"
	"github.com/cloudfoundry-incubator/tps/handler/lrpstats/fakes"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Handler", func() {

	Describe("rate limiting", func() {

		var (
			noaaClient     *fakes.FakeNoaaClient
			fakeKubeClient *handlerfakes.FakeKubeClient

			logger *lagertest.TestLogger

			server                 *httptest.Server
			fakeActualLRPResponses chan *v1.PodList
			statsRequest           *http.Request
			statusRequest          *http.Request
			httpClient             *http.Client
			fakePod                *handlerfakes.FakePod
			pod                    *v1.Pod
		)

		BeforeEach(func() {
			var err error
			var httpHandler http.Handler

			httpClient = &http.Client{}
			logger = lagertest.NewTestLogger("test")
			fakeKubeClient = &handlerfakes.FakeKubeClient{}
			noaaClient = &fakes.FakeNoaaClient{}

			httpHandler, err = handler.New(fakeKubeClient, noaaClient, 2, 15, logger)
			Expect(err).NotTo(HaveOccurred())

			server = httptest.NewServer(httpHandler)
			processGuid, err := helpers.NewProcessGuid("8d58c09b-b305-4f16-bcfe-b78edcb77100-3f258eb0-9dac-460c-a424-b43fe92bee27")
			Expect(err).NotTo(HaveOccurred())

			fakeActualLRPResponses = make(chan *v1.PodList, 2)
			statsRequest, err = http.NewRequest("GET", server.URL+"/v1/actual_lrps/"+processGuid.String()+"/stats", nil)
			Expect(err).NotTo(HaveOccurred())
			statsRequest.Header.Set("Authorization", "something")

			statusRequest, err = http.NewRequest("GET", server.URL+"/v1/actual_lrps/"+processGuid.String(), nil)
			Expect(err).NotTo(HaveOccurred())

			fakePod = &handlerfakes.FakePod{}

			now := unversioned.Now()
			pod = &v1.Pod{
				ObjectMeta: v1.ObjectMeta{
					Name:              "pod-name",
					Namespace:         "namespace",
					UID:               "1234-5677",
					CreationTimestamp: now,
					Labels: map[string]string{
						"cloudfoundry.org/app-guid":     processGuid.AppGuid.String(),
						"cloudfoundry.org/space-guid":   "my-space-id",
						"cloudfoundry.org/process-guid": processGuid.ShortenedGuid(),
						"cloudfoundry.org/domain":       cc_messages.AppLRPDomain,
					},
					Annotations: map[string]string{
						"cloudfoundry.org/log-guid": "my-log-guid",
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

			fakeKubeClient.PodsReturns(fakePod)
			fakePod.ListReturns(&v1.PodList{
				Items: []v1.Pod{*pod},
			}, nil)

			fakePod.ListStub = func(opts api.ListOptions) (*v1.PodList, error) {
				return <-fakeActualLRPResponses, nil
			}
			// bbsClient.ActualLRPGroupsByProcessGuidStub = func(lager.Logger, string) ([]*models.ActualLRPGroup, error) {
			// 	return <-fakeActualLRPResponses, nil
			// }

			noaaClient.ContainerMetricsReturns([]*events.ContainerMetric{
				{
					ApplicationId: proto.String("appId"),
					InstanceIndex: proto.Int32(0),
					CpuPercentage: proto.Float64(4),
					MemoryBytes:   proto.Uint64(1024),
					DiskBytes:     proto.Uint64(2048),
				},
			}, nil)
		})

		AfterEach(func() {
			server.Close()
		})

		It("returns 503 if the limit is exceeded", func() {
			// hit both status and stats endpoints once, make fake bbs hang
			var wg sync.WaitGroup

			defer close(fakeActualLRPResponses)

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()

				res, err := httpClient.Do(statusRequest)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.StatusCode).To(Equal(http.StatusOK))
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()

				res, err := httpClient.Do(statsRequest)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.StatusCode).To(Equal(http.StatusOK))
			}()

			//Eventually(bbsClient.ActualLRPGroupsByProcessGuidCallCount).Should(Equal(2))
			Eventually(fakePod.ListCallCount).Should(Equal(2))
			// hit it again, assert we get a 503
			res, err := httpClient.Do(statusRequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusServiceUnavailable))

			res, err = httpClient.Do(statsRequest)
			Expect(err).NotTo(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusServiceUnavailable))

			// un-hang http calls
			fakeActualLRPResponses <- &v1.PodList{Items: []v1.Pod{*pod}}
			fakeActualLRPResponses <- &v1.PodList{Items: []v1.Pod{*pod}}
			wg.Wait()

			// confirm we can request again
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()

				res, err := httpClient.Do(statusRequest)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.StatusCode).To(Equal(http.StatusOK))
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer GinkgoRecover()

				res, err := httpClient.Do(statsRequest)
				Expect(err).NotTo(HaveOccurred())
				Expect(res.StatusCode).To(Equal(http.StatusOK))
			}()

			fakeActualLRPResponses <- &v1.PodList{Items: []v1.Pod{*pod}}
			fakeActualLRPResponses <- &v1.PodList{Items: []v1.Pod{*pod}}
			wg.Wait()

		})
	})
})
