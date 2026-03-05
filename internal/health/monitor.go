package health

import (
	"net/http"
	"time"

	"MyProxy/internal/telemetry"
	"go.uber.org/zap"
)

type Target struct {
	URL     string
	Healthy bool
}

func Monitor(targetsGetter func() map[string][]*Target, interval time.Duration) {
	ticker := time.NewTicker(interval)
	client := &http.Client{Timeout: 3 * time.Second}

	for range ticker.C {
		targets := targetsGetter()
		for domain, list := range targets {
			healthyCount := 0
			for _, t := range list {
				resp, err := client.Get(t.URL + "/health")
				oldStatus := t.Healthy
				t.Healthy = (err == nil && resp != nil && resp.StatusCode == http.StatusOK)
				if resp != nil {
					resp.Body.Close()
				}

				if t.Healthy {
					healthyCount++
				}

				if oldStatus != t.Healthy {
					telemetry.Logger.Warn("Health status changed",
						zap.String("domain", domain),
						zap.String("target", t.URL),
						zap.Bool("healthy", t.Healthy))
				}
			}
			telemetry.ActiveBackends.WithLabelValues(domain).Set(float64(healthyCount))
		}
	}
}
