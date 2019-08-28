package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ricoberger/kubeversion-exporter/pkg/exporter"
	"github.com/ricoberger/kubeversion-exporter/pkg/kube"
	"github.com/ricoberger/kubeversion-exporter/pkg/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	interval      int64
	inCluster     bool
	kubeconfig    string
	listenAddress string
	logLevel      string
	logOutput     string
	metricsPath   string
)

var rootCmd = &cobra.Command{
	Use:   "kubeversion-exporter",
	Short: "kubeversion-exporter - exports the running version of an image in a Kubernetes cluster and if available the new version of the image.",
	Long:  "kubeversion-exporter - exports the running version of an image in a Kubernetes cluster and if available the new version of the image.",
	Run: func(cmd *cobra.Command, args []string) {
		if logOutput == "json" {
			log.SetFormatter(&log.JSONFormatter{})
		} else {
			log.SetFormatter(&log.TextFormatter{})
		}

		log.SetReportCaller(true)
		log.SetLevel(getLogLevel(logLevel))

		log.Infof(version.Info())
		log.Infof(version.BuildContext())

		// Create API client for the Kubernetes cluster.
		client, err := kube.NewClient(inCluster, kubeconfig)
		if err != nil {
			log.WithError(err).Fatalf("Could not create API client for the Kubernetes cluster.\n")
		}

		// Scrape version informationn for all used images in the Kubernetes
		// cluster.
		go exporter.RecordMetrics(client, interval)

		// Create the HTTP server for the kubeversion-exporter.
		router := http.NewServeMux()
		router.Handle("/metrics", promhttp.Handler())
		router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`<html>
			<head><title>kubeversion-exporter</title></head>
			<body>
			<h1>kubeversion-exporter</h1>
			<p><a href='/metrics'>Metrics</a></p>
			<p>
			<ul>
			<li>version: ` + version.Version + `</li>
			<li>branch: ` + version.Branch + `</li>
			<li>revision: ` + version.Revision + `</li>
			<li>go version: ` + version.GoVersion + `</li>
			<li>build user: ` + version.BuildUser + `</li>
			<li>build date: ` + version.BuildDate + `</li>
			</ul>
			</p>
			</body>
			</html>`))
		})

		server := &http.Server{
			Addr:    listenAddress,
			Handler: router,
		}

		// Listen for SIGINT and SIGTERM signals and try to gracefully shutdown
		// the HTTP server. This ensures that enabled connections are not
		// interrupted.
		go func() {
			term := make(chan os.Signal, 1)
			signal.Notify(term, os.Interrupt, syscall.SIGTERM)
			select {
			case <-term:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := server.Shutdown(ctx)
				if err != nil {
					log.WithError(err).Fatalf("Failed to shutdown kubeversion-exporter gracefully\n")
					os.Exit(1)
				}

				log.Infof("Shutdown kubeversion-exporter...\n")
				os.Exit(0)
			}
		}()

		log.Infof("Server listen on: %s", listenAddress)

		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.WithError(err).Fatalf("HTTP server died unexpected\n")
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information for kubeversion-exporter.",
	Long:  "Print version information for kubeversion-exporter.",
	Run: func(cmd *cobra.Command, args []string) {
		// Print the version information for kubeversion-exporter.
		v, err := version.Print("kubeversion-exporter")
		if err != nil {
			log.Fatalf("Failed to print version information: %#v", err)
		}

		fmt.Fprintln(os.Stdout, v)
		os.Exit(0)
	},
}

func getLogLevel(level string) log.Level {
	switch level {
	case "trace":
		return log.TraceLevel
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	case "panic":
		return log.PanicLevel
	default:
		return log.InfoLevel
	}
}

func init() {
	rootCmd.AddCommand(versionCmd)

	rootCmd.PersistentFlags().BoolVar(&inCluster, "cluster", false, "Authenticating inside the Kubernetes cluster.")
	rootCmd.PersistentFlags().Int64Var(&interval, "interval", 3600, "Interval in which to check for new image versions in seconds.")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "Path to the kubeconfig file to use for CLI requests.")
	rootCmd.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the log level. Must be one of the follwing values: trace, debug, info, warn, error, fatal or panic.")
	rootCmd.PersistentFlags().StringVar(&logOutput, "logoutput", "plain", "Set the output format of the log line. Must be plain or json.")
	rootCmd.PersistentFlags().StringVar(&listenAddress, "web.listen-address", ":9637", "Address to listen on for web interface and telemetry.")
	rootCmd.PersistentFlags().StringVar(&metricsPath, "web.telemetry-path", "/metrics", "Path under which to expose metrics.")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to initialize kubetop: %#v", err)
	}
}
