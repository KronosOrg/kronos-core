package kronosapp

import (
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

func init() {
	metrics.Registry.MustRegister(scheduleInfo)
}
