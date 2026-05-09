package middleware

import (
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
	MaxAgeSeconds  int
}

func CORS(cfg *CORSConfig) func(http.Handler) http.Handler {
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	maxAgeSeconds := cfg.MaxAgeSeconds
	if maxAgeSeconds == 0 {
		maxAgeSeconds = 300
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if isAllowedOrigin(cfg.AllowedOrigins, origin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Add("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
				w.Header().Set("Access-Control-Allow-Methods", allowedMethods)
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAgeSeconds))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isAllowedOrigin(allowedOrigins []string, origin string) bool {
	if origin == "" {
		return false
	}
	if slices.Contains(allowedOrigins, origin) {
		return true
	}

	requested, err := url.Parse(origin)
	if err != nil || requested.Scheme == "" || requested.Host == "" {
		return false
	}
	if !isLoopbackHost(requested.Hostname()) {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		allowed, err := url.Parse(allowedOrigin)
		if err != nil || allowed.Scheme != requested.Scheme {
			continue
		}
		if isLoopbackHost(allowed.Hostname()) {
			return true
		}
	}

	return false
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func DevCORS(next http.Handler) http.Handler {
	return CORS(&CORSConfig{
		AllowedOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{"Authorization", "Content-Type"},
	})(next)
}
