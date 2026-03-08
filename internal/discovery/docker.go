package discovery

import (
	"context"
	"strconv"
	"strings"

	"MyProxy/internal/proxy"
	"MyProxy/internal/telemetry"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"go.uber.org/zap"
)

const CorporateDomain = ".my.os"

func Start(r *proxy.Router) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		telemetry.Logger.Fatal("Docker client error", zap.Error(err))
	}

	ctx := context.Background()

	containers, err := cli.ContainerList(ctx, container.ListOptions{})
	if err == nil {
		for _, c := range containers {
			RegisterContainer(cli, r, c.ID)
		}
	}

	msgs, errs := cli.Events(ctx, events.ListOptions{})
	go func() {
		for {
			select {
			case event := <-msgs:
				if event.Type == events.ContainerEventType {
					switch event.Action {
					case "start":
						RegisterContainer(cli, r, event.Actor.ID)
					case "die", "stop":
						UnregisterContainer(cli, r, event.Actor.ID)
					}
				}
			case err := <-errs:
				if err != nil {
					telemetry.Logger.Error("Docker event error", zap.Error(err))
				}
			}
		}
	}()
}

func RegisterContainer(cli *client.Client, r *proxy.Router, containerID string) {
	ctx := context.Background()
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return
	}

	domainLabel := inspect.Config.Labels["myproxy.domain"]
	if domainLabel == "" {
		domainLabel = inspect.Config.Labels["my.os.domain"]
	}
	isFallback := inspect.Config.Labels["myproxy.fallback"] == "true"
	if domainLabel == "" && !isFallback {
		return
	}

	port := inspect.Config.Labels["myproxy.port"]
	if port == "" {
		port = "3000"
	}

	var ip string
	for _, net := range inspect.NetworkSettings.Networks {
		if net.IPAddress != "" {
			ip = net.IPAddress
			break
		}
	}
	if ip == "" {
		return
	}

	target := "http://" + ip + ":" + port
	domains := strings.Split(domainLabel, ",")

	rateLimitStr := inspect.Config.Labels["myproxy.rate_limit"]
	var rateLimit float64
	if rateLimitStr != "" {
		rateLimit, _ = strconv.ParseFloat(rateLimitStr, 64)
	}

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if domain == "" {
			continue
		}
		if strings.HasSuffix(domain, CorporateDomain) {
			r.AddRoute(domain, target)
			if isFallback {
				r.SetFallbackDomain(domain)
				telemetry.Logger.Info("Fallback domain registered", zap.String("domain", domain))
			}
			if rateLimit > 0 {
				r.SetRateLimit(domain, rateLimit)
			}
		}
	}
}

func UnregisterContainer(cli *client.Client, r *proxy.Router, containerID string) {
	ctx := context.Background()
	inspect, err := cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return
	}

	domainLabel := inspect.Config.Labels["myproxy.domain"]
	if domainLabel == "" {
		domainLabel = inspect.Config.Labels["my.os.domain"]
	}
	isFallback := inspect.Config.Labels["myproxy.fallback"] == "true"
	if domainLabel == "" && !isFallback {
		return
	}

	var ip string
	for _, net := range inspect.NetworkSettings.Networks {
		if net.IPAddress != "" {
			ip = net.IPAddress
			break
		}
	}
	if ip == "" {
		return
	}

	port := inspect.Config.Labels["myproxy.port"]
	if port == "" {
		port = "3000"
	}
	target := "http://" + ip + ":" + port
	domains := strings.Split(domainLabel, ",")

	for _, domain := range domains {
		domain = strings.TrimSpace(domain)
		if strings.HasSuffix(domain, CorporateDomain) {
			r.RemoveTarget(domain, target)
			if isFallback {
				if r.GetFallbackDomain() == domain {
					r.SetFallbackDomain("")
				}
			}
		}
	}
}
