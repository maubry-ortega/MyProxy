package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"MyProxy/internal/health"
	"MyProxy/internal/middleware"
	"MyProxy/internal/telemetry"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type Route struct {
	Targets []*health.Target
	Index   int
	Limiter *rate.Limiter
	mu      sync.Mutex
}

func (r *Route) NextTarget() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.Targets) == 0 {
		return ""
	}

	for i := 0; i < len(r.Targets); i++ {
		target := r.Targets[r.Index%len(r.Targets)]
		r.Index++
		if target.Healthy {
			return target.URL
		}
	}

	return r.Targets[0].URL
}

type Router struct {
	Routes     map[string]*Route
	IpLimiter  *middleware.IPLimiter
	mu         sync.RWMutex
}

func NewRouter() *Router {
	return &Router{
		Routes:    make(map[string]*Route),
		IpLimiter: middleware.NewIPLimiter(),
	}
}

func (r *Router) GetRoutes() map[string]*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Return a shallow copy of the map
	copy := make(map[string]*Route)
	for k, v := range r.Routes {
		copy[k] = v
	}
	return copy
}

func (r *Router) AddRoute(domain, targetURL string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	newTarget := &health.Target{URL: targetURL, Healthy: true}

	if route, exists := r.Routes[domain]; exists {
		route.mu.Lock()
		found := false
		for _, t := range route.Targets {
			if t.URL == targetURL {
				found = true
				break
			}
		}
		if !found {
			route.Targets = append(route.Targets, newTarget)
		}
		route.mu.Unlock()
	} else {
		r.Routes[domain] = &Route{
			Targets: []*health.Target{newTarget},
		}
	}
	telemetry.Logger.Info("Route added", zap.String("domain", domain), zap.String("target", targetURL))
}

func (r *Router) SetRateLimit(domain string, rps float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if route, ok := r.Routes[domain]; ok {
		route.mu.Lock()
		if rps > 0 {
			route.Limiter = rate.NewLimiter(rate.Limit(rps), int(rps*2))
		} else {
			route.Limiter = nil
		}
		route.mu.Unlock()
		telemetry.Logger.Info("Rate limit updated", zap.String("domain", domain), zap.Float64("rps", rps))
	}
}

func (r *Router) RemoveTarget(domain, target string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	route, ok := r.Routes[domain]
	if !ok {
		return
	}

	route.mu.Lock()
	defer route.mu.Unlock()

	newTargets := []*health.Target{}
	for _, t := range route.Targets {
		if t.URL != target {
			newTargets = append(newTargets, t)
		}
	}

	if len(newTargets) == 0 {
		delete(r.Routes, domain)
		telemetry.Logger.Info("Route removed", zap.String("domain", domain))
		return
	}
	route.Targets = newTargets
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	domain := req.Host
	ip := strings.Split(req.RemoteAddr, ":")[0]

	middleware.ApplySecurityHeaders(w)

	if !r.IpLimiter.Allow(ip) {
		telemetry.RateLimitHits.WithLabelValues(domain, "ip").Inc()
		http.Error(w, "IP Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	r.mu.RLock()
	route, ok := r.Routes[domain]
	r.mu.RUnlock()

	if !ok {
		http.Error(w, "No route found", http.StatusNotFound)
		return
	}

	if route.Limiter != nil && !route.Limiter.Allow() {
		telemetry.RateLimitHits.WithLabelValues(domain, "domain").Inc()
		http.Error(w, "Domain Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	target := route.NextTarget()
	if target == "" {
		telemetry.Logger.Warn("No healthy targets for domain", zap.String("domain", domain))
		http.Error(w, "No healthy targets", http.StatusServiceUnavailable)
		return
	}

	targetURL, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		telemetry.Logger.Error("Proxy error", zap.Error(err), zap.String("domain", domain))
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}

	rw := &ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
	proxy.ServeHTTP(rw, req)

	duration := time.Since(start).Seconds()
	telemetry.RecordRequest(domain, rw.Status, duration)
}

type ResponseWriter struct {
	http.ResponseWriter
	Status int
}

func (rw *ResponseWriter) WriteHeader(code int) {
	rw.Status = code
	rw.ResponseWriter.WriteHeader(code)
}
