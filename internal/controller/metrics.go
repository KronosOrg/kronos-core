package kronosapp

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	scheduleInfo = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "schedule_info",
			Help: "Current schedule information",
		},
		[]string{
			"name",
			"namespace",
		},
	)
)

func MetricsInit() {
	fmt.Println("Registering Metric")
	metrics.Registry.MustRegister(scheduleInfo)
}
