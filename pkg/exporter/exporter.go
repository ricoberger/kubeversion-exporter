package exporter

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/ricoberger/kubeversion-exporter/pkg/docker"
	"github.com/ricoberger/kubeversion-exporter/pkg/kube"

	"github.com/mcuadros/go-version"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	log "github.com/sirupsen/logrus"
)

const namespace = "kubeversion"

var (
	imageVersionInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "image_info",
		Help:      "Information for the image",
	}, []string{"image", "running_version", "current_version"})

	imageVersionTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "images_total",
		Help:      "Total number of images",
	})

	imageVersionSuccessTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "images_success_total",
		Help:      "Total number of successfull processed images",
	})

	imageVersionErrorTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "images_error_total",
		Help:      "Total number of images with an error during the processing",
	})

	clusterVersionInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "cluster_info",
		Help:      "Information for the cluster",
	}, []string{"running_version", "current_version"})
)

// GitHubRelease is the structure of the response from the GitHub releases
// request. The struct only contains the necessary fields for the KuberVersion
// Exporter.
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// Metric is used to collect all metrics befor the vectors are setted for
// Prometheus
type Metric struct {
	Labels prometheus.Labels
	Status float64
}

// compareVersions normalize the running and the current version and returns the
// normalized string and a float64 value if the current version is newer then
// the running one. If the current version is newer the value will be 1 if not
// the value will be 0.
func compareVersions(runningVersion, currentVersion string) (string, string, float64) {
	runningVersion = version.Normalize(runningVersion)
	currentVersion = version.Normalize(currentVersion)

	if version.Compare(currentVersion, runningVersion, ">") {
		return runningVersion, currentVersion, 1
	}

	return runningVersion, currentVersion, 0
}

// clusterMetrics gets the running version and the current version of Kubernetes
// which is available on GitHub. The function fills the cluster_info metric with
// the version labels and if an update is available (2) or not (1).
func clusterMetrics(client *kube.Client) *Metric {
	runningVersion, err := client.GetClusterVersion()
	if err != nil {
		log.WithError(err).Errorf("Could not get running Kubernetes version")
		return nil
	}

	httpClient := http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/kubernetes/kubernetes/releases/latest", nil)
	if err != nil {
		log.WithError(err).Errorf("Could not create http request")
	}

	res, err := httpClient.Do(req)
	if err != nil {
		log.WithError(err).Errorf("Could not get current Kubernetes version")
		return nil
	}

	defer res.Body.Close()

	var release GitHubRelease
	err = json.NewDecoder(res.Body).Decode(&release)
	if err != nil {
		log.WithError(err).Errorf("Could not decode response from GitHub release page")
		return nil
	}

	var status float64
	currentVersion := version.Normalize(release.TagName)
	runningVersion, currentVersion, status = compareVersions(runningVersion, currentVersion)

	log.WithFields(log.Fields{
		"current_version": currentVersion,
		"running_version": runningVersion,
	}).Infof("Received versions")

	return &Metric{
		Labels: prometheus.Labels{
			"running_version": runningVersion,
			"current_version": currentVersion,
		},
		Status: status,
	}
}

// imageMetrics gets all images which are used in the Kubernetes cluster and
// checks the corresponding Docker registries if there are new versions
// available. The determination of new versions is handled by looking at the
// available tags and the comparison of these with the tag of the running
// image in the cluster. The function fills the image_info metric with
// the version labels and the image name and if an update is available (2) or
// not (1).
func imageMetrics(client *kube.Client) []Metric {
	images, err := client.GetImages()
	if err != nil {
		log.WithError(err).Errorf("Could not get running images in the Kubernetes cluster")
		return nil
	}

	log.WithFields(log.Fields{"images": images}).Debugf("Received images")

	imageVersionTotal.Set(float64(len(images)))
	imageVersionSuccessTotal.Set(0)
	imageVersionErrorTotal.Set(0)

	var metrics []Metric

	for _, image := range images {
		if imageParts := strings.Split(image, ":"); len(imageParts) == 2 {
			log.WithFields(log.Fields{"image": imageParts[0], "tag": imageParts[1]}).Debugf("Splitted image %s", image)

			if tags, err := docker.GetTags(imageParts[0]); err == nil && len(tags) > 0 {
				log.WithFields(log.Fields{"tags": tags}).Debugf("Received tags for image %s", image)

				version.Sort(tags)
				log.WithFields(log.Fields{"tags": tags}).Debugf("Sorted tags for image %s", image)

				var status float64
				runningVersion := imageParts[1]
				currentVersion := tags[len(tags)-1]
				runningVersion, currentVersion, status = compareVersions(runningVersion, currentVersion)

				log.WithFields(log.Fields{
					"current_version": currentVersion,
					"running_version": runningVersion,
				}).Infof("Received version for image %s", image)

				imageVersionSuccessTotal.Inc()

				metrics = append(metrics, Metric{
					Labels: prometheus.Labels{
						"image":           image,
						"running_version": runningVersion,
						"current_version": currentVersion,
					},
					Status: status,
				})
			} else {
				log.WithError(err).Errorf("Could not get tags for the image %s", image)
				imageVersionErrorTotal.Inc()
			}
		} else {
			log.Errorf("Could not get parts of the image %s", image)
			imageVersionErrorTotal.Inc()
		}
	}

	return metrics
}

// RecordMetrics records the information for the images.
// At first we try to get all used images in the Kubernetes Cluster and then
// we lookup the tags for all images.
func RecordMetrics(client *kube.Client, interval int64) {
	for {
		// Check version of cluster. First we try to get the metric and if the
		// metric is not nil we reset the old vector and create a new one with
		// the new values.
		log.Infof("Start checking Kubernetes for newer version")

		clusterMetric := clusterMetrics(client)
		if clusterMetric != nil {
			clusterVersionInfo.Reset()
			clusterVersionInfo.With(clusterMetric.Labels).Set(clusterMetric.Status)
		}

		log.Infof("Finished checking Kubernetes for new version")

		// Check version of images. First we try to get all image metrics and if
		// the length of the metrics slice is not nil we reset the old vector
		// and create a new one with the new values.
		log.Infof("Start checking images for new versions")

		imageMetrics := imageMetrics(client)
		if len(imageMetrics) != 0 {
			imageVersionInfo.Reset()
			for _, metric := range imageMetrics {
				imageVersionInfo.With(metric.Labels).Set(metric.Status)
			}
		}

		log.Infof("Finished checking images for new versions")

		// Wait the given interval befor the next run is started.
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
