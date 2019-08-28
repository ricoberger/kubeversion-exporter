package kube

import (
	"errors"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	// ErrConfigNotFound is thrown if there is not a confgiuration file for Kubernetes.
	ErrConfigNotFound = errors.New("config not found")
)

// Client implements an API client for a Kubernetes cluster.
type Client struct {
	config    *rest.Config
	clientset *kubernetes.Clientset
}

// homeDir returns the users home directory, where the '.kube' directory is located.
// The '.kube' directory contains the configuration file for a Kubernetes cluster.
func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}

	// Get the home directory on windows.
	return os.Getenv("USERPROFILE")
}

// NewClient returns a new API client for a Kubernetes cluster.
// If the cluster parameter is true the client is configured inside the
// cluster. If the cluster parameter is false the client is configures outside
// the cluster.
func NewClient(cluster bool, kubeconfig string) (*Client, error) {
	// Authenticating inside the cluster.
	if cluster == true {
		// Creates the in-cluster config.
		config, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}

		// Creates the clientset.
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return nil, err
		}

		return &Client{
			config:    config,
			clientset: clientset,
		}, nil
	}

	// Authenticating outside the cluster.
	if kubeconfig == "" {
		if os.Getenv("KUBECONFIG") == "" {
			if home := homeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			} else {
				return nil, ErrConfigNotFound
			}
		} else {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
	}

	// Use the current context in kubeconfig.
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		config:    config,
		clientset: clientset,
	}, nil
}

// GetImages returns all images of all containers in the Kubernetes cluster.
func (c *Client) GetImages() ([]string, error) {
	pods, err := c.clientset.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var images []string
	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			images = append(images, container.Image)
		}
	}

	return images, nil
}

// GetClusterVersion returns the Kubernetes API version.
func (c *Client) GetClusterVersion() (string, error) {
	version, err := c.clientset.ServerVersion()
	if err != nil {
		return "", err
	}

	return version.String(), nil
}
