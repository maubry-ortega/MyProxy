package proxy

import (
	"encoding/json"
	"net/http"

	"MyProxy/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

type AdminServer struct {
	router *Router
}

func NewAdminServer(r *Router) *AdminServer {
	return &AdminServer{router: r}
}

func (s *AdminServer) Start(addr string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("/api/apps", s.handleListApps)

	go func() {
		telemetry.Logger.Info("Admin server running", zap.String("addr", addr))
		if err := http.ListenAndServe(addr, mux); err != nil {
			telemetry.Logger.Error("Admin server error", zap.Error(err))
		}
	}()
}

func (s *AdminServer) handleListApps(w http.ResponseWriter, r *http.Request) {
	routes := s.router.GetRoutes()
	apps := make([]string, 0, len(routes))
	fallback := s.router.GetFallbackDomain()
	
	for domain := range routes {
		if domain != fallback {
			apps = append(apps, domain)
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apps)
}
