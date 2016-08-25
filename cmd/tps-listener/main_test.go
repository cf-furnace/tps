package main_test

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/api/v1"
	clientset "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3"
	v1core "k8s.io/kubernetes/pkg/client/clientset_generated/release_1_3/typed/core/v1"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/labels"

	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gogo/protobuf/proto"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit/ginkgomon"
	"github.com/tedsuo/rata"

	"github.com/cloudfoundry-incubator/nsync/helpers"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/tps"
	tpshelpers "github.com/cloudfoundry-incubator/tps/helpers"
)

var _ = Describe("TPS-Listener", func() {
	var (
		httpClient       *http.Client
		requestGenerator *rata.RequestGenerator

		err error

		processGuid1     helpers.ProcessGuid
		processGuid2     helpers.ProcessGuid
		processGuid3     helpers.ProcessGuid
		k8sClient        v1core.CoreInterface
		defaultNamespace string
	)

	BeforeEach(func() {
		requestGenerator = rata.NewRequestGenerator(fmt.Sprintf("http://%s", listenerAddr), tps.Routes)
		httpClient = &http.Client{
			Transport: &http.Transport{},
		}
		processGuid1, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())
		processGuid2, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())
		processGuid3, err = generateProcessGuid()
		Expect(err).NotTo(HaveOccurred())

		config := loadClientConfig()
		k8sConfig, err := clientset.NewForConfig(config)
		defaultNamespace = "default"

		if err != nil {
			logger.Fatal("Can't create kubernetes client", err)
		}

		k8sClient = k8sConfig.Core()

	})

	JustBeforeEach(func() {
		listener = ginkgomon.Invoke(runner)
	})

	AfterEach(func() {
		if listener != nil {
			listener.Signal(os.Kill)
			Eventually(listener.Wait()).Should(Receive())
		}
		fmt.Printf("calling after each")

		deleteReplicationController(k8sClient, defaultNamespace, processGuid1)
		deleteReplicationController(k8sClient, defaultNamespace, processGuid2)
		deleteReplicationController(k8sClient, defaultNamespace, processGuid3)
	})

	Describe("Initialization", func() {
		It("registers itself with consul", func() {
			services, err := consulRunner.NewClient().Agent().Services()
			Expect(err).NotTo(HaveOccurred())
			Expect(services).Should(HaveKeyWithValue("tps",
				&consulapi.AgentService{
					Service: "tps",
					ID:      "tps",
					Port:    listenerPort,
				}))
		})

		It("registers a TTL healthcheck", func() {
			checks, err := consulRunner.NewClient().Agent().Checks()
			Expect(err).NotTo(HaveOccurred())
			Expect(checks).Should(HaveKeyWithValue("service:tps",
				&consulapi.AgentCheck{
					Node:        "0",
					CheckID:     "service:tps",
					Name:        "Service 'tps' check",
					Status:      "passing",
					ServiceID:   "tps",
					ServiceName: "tps",
				}))
		})
	})

	Describe("GET actual LRP for a given guid", func() {
		Context("when the kubernetes is running", func() {
			JustBeforeEach(func() {
				newRC1 := generateReplicationController(processGuid1)
				_, err := k8sClient.ReplicationControllers(newRC1.ObjectMeta.Namespace).Create(newRC1)
				Expect(err).NotTo(HaveOccurred())

				Eventually(replicationControllers(k8sClient, processGuid1.AppGuid.String()), 1*time.Minute).Should(HaveLen(1))
				Eventually(pods(k8sClient, processGuid1.AppGuid.String()), 1*time.Minute).Should(HaveLen(3))

				// wait till containers all reach running
				time.Sleep(30 * time.Second)
			})

			It("reports the state of the given process guid's instances", func() {
				getLRPs, err := requestGenerator.CreateRequest(
					tps.LRPStatus,
					rata.Params{"guid": processGuid1.String()},
					nil,
				)
				Expect(err).NotTo(HaveOccurred())
				response, err := httpClient.Do(getLRPs)
				Expect(err).NotTo(HaveOccurred())
				//time.Sleep(time.Second * 120)
				var lrpInstances []cc_messages.LRPInstance
				err = json.NewDecoder(response.Body).Decode(&lrpInstances)
				Expect(err).NotTo(HaveOccurred())

				podsList, err := k8sClient.Pods(defaultNamespace).List(api.ListOptions{
					LabelSelector: labels.Set{"cloudfoundry.org/process-guid": processGuid1.ShortenedGuid()}.AsSelector(),
				})

				items := podsList.Items
				items = tpshelpers.SortPods(items)
				Expect(len(items)).To(Equal(3))

				for _, pod := range items {
					containerStatuses := pod.Status.ContainerStatuses
					Eventually(containerStatuses).ShouldNot(BeNil())
					Eventually(containerStatuses[0].State.Running).ShouldNot(BeNil())
				}

				Expect(lrpInstances).To(HaveLen(3))

				for i, _ := range lrpInstances {
					Expect(lrpInstances[i]).NotTo(BeZero())
					lrpInstances[i].Since = 0

					Eventually(lrpInstances[i]).ShouldNot(BeZero())
					lrpInstances[i].Uptime = 0
				}

				Expect(lrpInstances).To(ContainElement(cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: string(items[0].ObjectMeta.UID),
					Index:        0,
					State:        cc_messages.LRPInstanceStateRunning,
				}))

				Expect(lrpInstances).To(ContainElement(cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: string(items[1].ObjectMeta.UID),
					Index:        1,
					State:        cc_messages.LRPInstanceStateRunning,
				}))

				Expect(lrpInstances).To(ContainElement(cc_messages.LRPInstance{
					ProcessGuid:  processGuid1.String(),
					InstanceGuid: string(items[2].ObjectMeta.UID),
					Index:        2,
					State:        cc_messages.LRPInstanceStateRunning,
				}))
			})
		})
	})

	Describe("get actual lrp for all guids when kubernetes is running", func() {
		JustBeforeEach(func() {
			newRC1 := generateReplicationController(processGuid2)
			_, err = k8sClient.ReplicationControllers(newRC1.ObjectMeta.Namespace).Create(newRC1)
			Expect(err).NotTo(HaveOccurred())
			newRC2 := generateReplicationController(processGuid3)
			_, err = k8sClient.ReplicationControllers(newRC2.ObjectMeta.Namespace).Create(newRC2)
			Expect(err).NotTo(HaveOccurred())
			Eventually(replicationControllers(k8sClient, processGuid2.AppGuid.String()), 1*time.Minute).Should(HaveLen(1))
			Eventually(replicationControllers(k8sClient, processGuid3.AppGuid.String()), 1*time.Minute).Should(HaveLen(1))
			Eventually(pods(k8sClient, processGuid2.AppGuid.String()), 1*time.Minute).Should(HaveLen(3))
			Eventually(pods(k8sClient, processGuid3.AppGuid.String()), 1*time.Minute).Should(HaveLen(3))
			// wait till containers all reach running
			time.Sleep(60 * time.Second)
		})

		It("reports the status for all the process guids supplied", func() {
			getLRPStatus, err := requestGenerator.CreateRequest(
				tps.BulkLRPStatus,
				nil,
				nil,
			)
			Expect(err).NotTo(HaveOccurred())
			getLRPStatus.Header.Add("Authorization", "I can do this.")

			query := getLRPStatus.URL.Query()
			query.Set("guids", processGuid2.String()+","+processGuid3.String())
			getLRPStatus.URL.RawQuery = query.Encode()

			response, err := httpClient.Do(getLRPStatus)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			var lrpInstanceStatus map[string][]cc_messages.LRPInstance
			err = json.NewDecoder(response.Body).Decode(&lrpInstanceStatus)
			Expect(err).NotTo(HaveOccurred())

			Expect(lrpInstanceStatus).To(HaveLen(2))
			for guid, instances := range lrpInstanceStatus {
				pg, err := helpers.NewProcessGuid(guid)
				Expect(err).To(BeNil())
				podsList, err := k8sClient.Pods(defaultNamespace).List(api.ListOptions{
					LabelSelector: labels.Set{"cloudfoundry.org/process-guid": pg.ShortenedGuid()}.AsSelector(),
				})
				Expect(err).To(BeNil())

				items := podsList.Items
				items = tpshelpers.SortPods(items)

				Expect(len(items)).To(Equal(3))

				for i, _ := range instances {
					Expect(instances[i]).NotTo(BeZero())
					instances[i].Since = 0

					Eventually(instances[i]).ShouldNot(BeZero())
					instances[i].Uptime = 0
				}

				Expect(instances).To(ContainElement(cc_messages.LRPInstance{
					ProcessGuid:  guid,
					InstanceGuid: string(items[0].ObjectMeta.UID),
					Index:        0,
					State:        cc_messages.LRPInstanceStateRunning,
				}))

				Expect(instances).To(ContainElement(cc_messages.LRPInstance{
					ProcessGuid:  guid,
					InstanceGuid: string(items[1].ObjectMeta.UID),
					Index:        1,
					//	NetInfo:      netInfo,
					State: cc_messages.LRPInstanceStateRunning,
				}))

				Expect(instances).To(ContainElement(cc_messages.LRPInstance{
					ProcessGuid:  guid,
					InstanceGuid: string(items[2].ObjectMeta.UID),
					Index:        2,
					State:        cc_messages.LRPInstanceStateRunning,
				}))
			}
		})
	})
})

func createContainerMetric(appId string, instanceIndex int32, cpuPercentage float64, memoryBytes uint64, diskByte uint64, timestamp int64) *events.Envelope {
	if timestamp == 0 {
		timestamp = time.Now().UnixNano()
	}

	cm := &events.ContainerMetric{
		ApplicationId: proto.String(appId),
		InstanceIndex: proto.Int32(instanceIndex),
		CpuPercentage: proto.Float64(cpuPercentage),
		MemoryBytes:   proto.Uint64(memoryBytes),
		DiskBytes:     proto.Uint64(diskByte),
	}

	return &events.Envelope{
		ContainerMetric: cm,
		EventType:       events.Envelope_ContainerMetric.Enum(),
		Origin:          proto.String("fake-origin-1"),
		Timestamp:       proto.Int64(timestamp),
	}
}

func marshalMessage(message *events.Envelope) []byte {
	data, err := proto.Marshal(message)
	if err != nil {
		log.Println(err.Error())
	}

	return data
}

func generateProcessGuid() (helpers.ProcessGuid, error) {
	appGuid, _ := uuid.NewV4()

	appVersion, _ := uuid.NewV4()

	return helpers.NewProcessGuid(appGuid.String() + "-" + appVersion.String())
}

func loadClientConfig() *restclient.Config {
	home := os.Getenv("HOME")
	config, err := clientcmd.LoadFromFile(filepath.Join(home, ".kube", "config"))
	Expect(err).NotTo(HaveOccurred())

	context := config.Contexts[config.CurrentContext]

	clientConfig := &restclient.Config{
		Host:     config.Clusters[context.Cluster].Server,
		Username: config.AuthInfos[context.AuthInfo].Username,
		Password: config.AuthInfos[context.AuthInfo].Password,
		Insecure: config.Clusters[context.Cluster].InsecureSkipTLSVerify,
		TLSClientConfig: restclient.TLSClientConfig{
			CertFile: config.AuthInfos[context.AuthInfo].ClientCertificate,
			KeyFile:  config.AuthInfos[context.AuthInfo].ClientKey,
			CAFile:   config.Clusters[context.Cluster].CertificateAuthority,
		},
	}
	return clientConfig
}

func generateReplicationController(pg helpers.ProcessGuid) *v1.ReplicationController {
	return &v1.ReplicationController{
		ObjectMeta: v1.ObjectMeta{
			Name:      pg.ShortenedGuid(),
			Namespace: "default",
			Labels: map[string]string{
				"cloudfoundry.org/process-guid": pg.ShortenedGuid(),
				"cloudfoundry.org/domain":       cc_messages.AppLRPDomain,
				"cloudfoundry.org/app-guid":     pg.AppGuid.String(),
			},
			Annotations: map[string]string{
				"cloudfoundry.org/log-guid":      "the-log-guid",
				"cloudfoundry.org/metrics-guid":  "the-metrics-guid",
				"cloudfoundry.org/storage-space": "storage-space",
				"cloudfoundry.org/etag":          "current-etag",
			},
		},
		Spec: v1.ReplicationControllerSpec{
			Replicas: helpers.Int32Ptr(3),
			Selector: map[string]string{
				"cloudfoundry.org/process-guid": pg.ShortenedGuid(),
			},
			Template: &v1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: map[string]string{
						"cloudfoundry.org/app-guid":     pg.AppGuid.String(),
						"cloudfoundry.org/process-guid": pg.ShortenedGuid(),
						"cloudfoundry.org/domain":       cc_messages.AppLRPDomain,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:    "application",
						Image:   "busybox",
						Command: []string{"sleep", "3600"},
						Resources: v1.ResourceRequirements{
							Limits: v1.ResourceList{
								v1.ResourceCPU:                   resource.MustParse("100m"),
								v1.ResourceMemory:                resource.MustParse(fmt.Sprintf("%dMi", 256)),
								"cloudfoundry.org/storage-space": resource.MustParse(fmt.Sprintf("%dMi", 1024)),
							},
						},
					}},
				},
			},
		},
	}
}

func replicationControllers(client v1core.CoreInterface, appGuid string) func() []string {
	return func() []string {
		rcList, err := client.ReplicationControllers(api.NamespaceAll).List(api.ListOptions{
			LabelSelector: labels.Set{"cloudfoundry.org/app-guid": appGuid}.AsSelector(),
		})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "List replication controller failed: %s\n", err.Error())
			return nil
		}

		result := []string{}
		for _, rc := range rcList.Items {
			result = append(result, rc.Name)
		}

		return result
	}
}

func pods(client v1core.CoreInterface, appGuid string) func() []string {
	return func() []string {
		podList, err := client.Pods(api.NamespaceAll).List(api.ListOptions{
			LabelSelector: labels.Set{"cloudfoundry.org/app-guid": appGuid}.AsSelector(),
		})
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "List pods failed: %s\n", err.Error())
			return nil
		}

		result := []string{}
		for _, pod := range podList.Items {
			result = append(result, pod.Name)
		}

		return result
	}
}

func deleteReplicationController(k8sClient v1core.CoreInterface, namespace string, pg helpers.ProcessGuid) {
	k8sClient.ReplicationControllers(namespace).Delete(pg.ShortenedGuid(), nil)

	podsList, _ := k8sClient.Pods(namespace).List(api.ListOptions{
		LabelSelector: labels.Set{"cloudfoundry.org/process-guid": pg.ShortenedGuid()}.AsSelector(),
	})

	items := podsList.Items
	for _, pod := range items {
		k8sClient.Pods(namespace).Delete(pod.ObjectMeta.Name, nil)
	}
}
