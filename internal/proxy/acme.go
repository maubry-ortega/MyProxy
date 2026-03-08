package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"MyProxy/internal/telemetry"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"
)

type ACMEManager struct {
	manager    *autocert.Manager
	router     *Router
	localCert  *tls.Certificate
	certDir    string
}

func NewACMEManager(r *Router, certDir string) *ACMEManager {
	if err := os.MkdirAll(certDir, 0700); err != nil {
		telemetry.Logger.Fatal("Failed to create certs directory", zap.Error(err))
	}

	m := &autocert.Manager{
		Prompt: autocert.AcceptTOS,
		Cache:  autocert.DirCache(certDir),
		HostPolicy: func(ctx context.Context, host string) error {
			// Only allow public domains (not .my.os internals)
			if strings.HasSuffix(host, ".my.os") {
				return fmt.Errorf("acme: internal domain %q does not need ACME cert", host)
			}
			if r.DomainExists(host) || host == r.GetFallbackDomain() {
				return nil
			}
			return fmt.Errorf("acme: host %q not allowed", host)
		},
	}

	a := &ACMEManager{
		manager: m,
		router:  r,
		certDir: certDir,
	}

	// Load self-signed cert for .my.os domains
	certFile := certDir + "/myos.crt"
	keyFile := certDir + "/myos.key"
	if cert, err := tls.LoadX509KeyPair(certFile, keyFile); err == nil {
		a.localCert = &cert
		telemetry.Logger.Info("Loaded local wildcard cert for *.my.os")
	} else {
		telemetry.Logger.Warn("No local wildcard cert found, .my.os HTTPS will fail gracefully", zap.String("expected", certFile))
	}

	return a
}

func (a *ACMEManager) buildTLSConfig() *tls.Config {
	localCert := a.localCert

	return &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			// For .my.os or when no SNI (direct IP), use local self-signed cert
			if localCert != nil && (hello.ServerName == "" || strings.HasSuffix(hello.ServerName, ".my.os")) {
				return localCert, nil
			}
			// For public domains, delegate to ACME/autocert
			return a.manager.GetCertificate(hello)
		},
		NextProtos: []string{"h2", "http/1.1", "acme-tls/1"},
	}
}

func (a *ACMEManager) HTTPHandler(fallback http.Handler) http.Handler {
	return a.manager.HTTPHandler(fallback)
}

func (a *ACMEManager) StartHTTPS() {
	server := &http.Server{
		Addr:      ":443",
		Handler:   a.router,
		TLSConfig: a.buildTLSConfig(),
	}

	go func() {
		telemetry.Logger.Info("MyProxy HTTPS running", zap.String("addr", server.Addr))
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			telemetry.Logger.Fatal("HTTPS server error", zap.Error(err))
		}
	}()
}

func (a *ACMEManager) StartHTTPRedirect() {
	handler := a.HTTPHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		host, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			host = req.Host
		}

		// Don't redirect IP addresses, localhost, or internal .my.os domains — serve directly
		if net.ParseIP(host) != nil || host == "localhost" || strings.HasSuffix(host, ".my.os") {
			a.router.ServeHTTP(w, req)
			return
		}

		target := "https://" + req.Host + req.URL.Path
		if len(req.URL.RawQuery) > 0 {
			target += "?" + req.URL.RawQuery
		}
		http.Redirect(w, req, target, http.StatusMovedPermanently)
	}))

	server := &http.Server{
		Addr:    ":80",
		Handler: handler,
	}

	go func() {
		telemetry.Logger.Info("MyProxy HTTP running (Redirect/ACME)", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			telemetry.Logger.Fatal("HTTP server error", zap.Error(err))
		}
	}()
}
