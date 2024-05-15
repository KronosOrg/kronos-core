package kronosapp

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Metrics struct {
	ScheduleInfo *prometheus.GaugeVec
	InDepthScheduleInfo *prometheus.GaugeVec
}

func RegisterMetrics() Metrics {
	sleepInfoMetrics := Metrics{
		ScheduleInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "schedule_info",
			Help: "Current schedule information",
		}, []string{"name", "namespace"}),
		InDepthScheduleInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "indepth_schedule_info",
			Help: "Current schedule information",
		}, []string{"name", "namespace", "status", "reason", "handled_resources", "next_operation"}),
	}
	return sleepInfoMetrics
}

func (additionalMetrics Metrics) MustRegister(registry metrics.RegistererGatherer) Metrics {
	registry.MustRegister(
		additionalMetrics.ScheduleInfo,
		additionalMetrics.InDepthScheduleInfo,
	)
	return additionalMetrics
}
