<div align="center">
  <img src="./assets/logo.png" width="20%" />
  <br><br>

  Find out which versions are running in your Kubernetes cluster and if there are updates available.
</div>

The **KubeVersion Exporter** checks which versions of images are running in your Kubernetes cluster. Then the Exporter checks if there are newer versions available in the corresponding Docker registry. When there is a new version of the image available the KubeVersion Exporter creates a metric with the currently running version of the image, the new version and the image name. Additionally the Exporter checks the used Kubernetes version and checks if there is a newer version of Kubernetes available.

## Features

The following features are planned for future versions of the KubeVersion Exporter:

- [ ] Check images from private Docker registries
- [ ] Improve the version comparison
- [ ] Check Helm charts for new versions
- [ ] Grafana dashboard
- [ ] Prometheus rules

## Installation

The **KubeVersion Exporter** can be installed via Helm chart:

```sh
helm repo add ricoberger https://ricoberger.github.io/helm-charts
helm repo update

helm upgrade --install kubeversion-exporter ricoberger/kubeversion-exporter
```

This will create a ClusterRole, ClusterRoleBinding and a ServiceAccount. The only permission which the Exporter needs is the right to list all pods in all Namespaces to get the image the pod is running. If you do not want to create the ClusterRole, ClusterRoleBinding or ServiceAccount you can deactivate the option in the `values.yaml`, but ensure that the KubeVersion Exporter has the right to list all pods in all Namespaces. Besides that the Helm chart creates a deployment and a service for the KubeVersion Exporter.

## Metrics

The KuberVersion Exporter exports the default Go and promhttp metrics, which are comming with the Go client for Prometheus. Additionally the following metrics are exported:

| Name | Description |
| ---- | ----------- |
| `kubeversion_cluster_info` | The running version and current version of Kubernetes. If the mtric is `1`, there is an update available. If not, the metric is `0`. |
| `kubeversion_image_info` | The running version of an image and the current version from the corresponding Docker registry. If the mtric is `1`, there is an update available. If not, the metric is `0`. |
| `kubeversion_images_total` | The total number of images running in the Cluster. |
| `kubeversion_images_success_total` | The total number of images, which where processed without an error. |
| `kubeversion_images_error_total` | The total number of images, which where processed with an error. |
