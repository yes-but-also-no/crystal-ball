package monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"net/http"
)

var (
	QueueGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "queue_size",
		Namespace: "crystal_ball",
		Help:      "Amount of scheduled jobs in the queue",
	})
	AccountBalanceGauge = promauto.NewGauge(prometheus.GaugeOpts{
		Name:      "balance",
		Namespace: "crystal_ball",
		Help:      "Validator account balance",
	})
	ExecutedJobsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "executed_jobs",
		Namespace: "crystal_ball",
		Help:      "Amount of jobs that were executed",
	})
	FailedJobsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Name:      "failed_jobs",
		Namespace: "crystal_ball",
		Help:      "Amount of jobs that could not be executed",
	})
)

func StartMonitoring(host string) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(host, nil)
	log.Error().Err(err).Caller().Msg("prometheus monitoring exited")
}
