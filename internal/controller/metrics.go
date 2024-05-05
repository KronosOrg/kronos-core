package kronosapp

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

type Metrics struct {
	ScheduleInfo *prometheus.GaugeVec
}

func RegisterMetrics(prefix string) Metrics {
	sleepInfoMetrics := Metrics{
		ScheduleInfo: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: prefix,
			Name:      "schedule_info",
			Help:      "Current schedule information",
		}, []string{"name", "namespace"}),
	}
	return sleepInfoMetrics
}

func (additionalMetrics Metrics) MustRegister(registry metrics.RegistererGatherer) Metrics {
	registry.MustRegister(
		additionalMetrics.ScheduleInfo,
	)
	return additionalMetrics
}
