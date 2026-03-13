// Package server provides an HTTP API that mirrors the Django dashboard's
// endpoints:
//
//	POST /api/ping        { "target": "8.8.8.8", "count": 4 }
//	POST /api/traceroute  { "target": "8.8.8.8", "max_hops": 30 }
//	POST /api/audit       { "targets": ["8.8.8.8"], "traceroute": true }
//	GET  /health
//
// All responses are JSON. The server is single-binary — no templates needed.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/wyvern/netaudit/internal/ping"
	"github.com/wyvern/netaudit/internal/report"
	"github.com/wyvern/netaudit/internal/traceroute"
)

// Server wraps http.Server with our handlers.
type Server struct {
	http *http.Server
	log  *log.Logger
}

// New constructs a Server bound to addr (e.g. ":8080").
func New(addr string, logger *log.Logger) *Server {
	mux := http.NewServeMux()
	s := &Server{log: logger}

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /api/ping", s.handlePing)
	mux.HandleFunc("POST /api/traceroute", s.handleTraceroute)
	mux.HandleFunc("POST /api/audit", s.handleAudit)

	s.http = &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(logger, jsonMiddleware(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 120 * time.Second, // traceroutes can be slow
		IdleTimeout:  30 * time.Second,
	}
	return s
}

// ListenAndServe starts the server and blocks until it exits.
func (s *Server) ListenAndServe() error {
	s.log.Printf("server listening on %s", s.http.Addr)
	return s.http.ListenAndServe()
}

// Shutdown gracefully drains connections.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// ─── handlers ────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// pingRequest is the POST body for /api/ping.
type pingRequest struct {
	Target  string `json:"target"`
	Count   int    `json:"count"`    // default 4
	Timeout int    `json:"timeout"`  // seconds, default 5
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	var req pingRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	opts := ping.DefaultOptions()
	if req.Count > 0 {
		opts.Count = req.Count
	}
	if req.Timeout > 0 {
		opts.Timeout = time.Duration(req.Timeout) * time.Second
	}

	ctx, cancel := context.WithTimeout(r.Context(), opts.Timeout*time.Duration(opts.Count)+5*time.Second)
	defer cancel()

	result := ping.Run(ctx, req.Target, opts)
	writeJSON(w, http.StatusOK, result)
}

// traceRequest is the POST body for /api/traceroute.
type traceRequest struct {
	Target  string `json:"target"`
	MaxHops int    `json:"max_hops"` // default 30
}

func (s *Server) handleTraceroute(w http.ResponseWriter, r *http.Request) {
	var req traceRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	opts := traceroute.DefaultOptions()
	if req.MaxHops > 0 {
		opts.MaxHops = req.MaxHops
	}

	ctx, cancel := context.WithTimeout(r.Context(), opts.Timeout)
	defer cancel()

	result := traceroute.Run(ctx, req.Target, opts)
	writeJSON(w, http.StatusOK, result)
}

// auditRequest drives the bulk concurrent audit endpoint.
type auditRequest struct {
	Targets    []string `json:"targets"`
	RunPing    bool     `json:"ping"`       // default true
	RunTrace   bool     `json:"traceroute"` // default false (slow)
	PingCount  int      `json:"ping_count"`
	MaxHops    int      `json:"max_hops"`
	Concurrency int     `json:"concurrency"` // goroutine cap, default 10
}

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	var req auditRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Targets) == 0 {
		writeError(w, http.StatusBadRequest, "at least one target is required")
		return
	}

	// Defaults
	if !req.RunPing && !req.RunTrace {
		req.RunPing = true // ping at minimum
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 10
	}
	if req.PingCount <= 0 {
		req.PingCount = 4
	}
	if req.MaxHops <= 0 {
		req.MaxHops = 30
	}

	start := time.Now()
	audits := runConcurrentAudit(r.Context(), req)

	rpt := report.New(audits, time.Since(start))
	writeJSON(w, http.StatusOK, rpt)
}

// ─── concurrent audit engine ─────────────────────────────────────────────────

// runConcurrentAudit fans out audit tasks across a semaphore-limited goroutine
// pool. The semaphore pattern is the idiomatic Go approach: a buffered channel
// of capacity N acts as a token pool. Each goroutine acquires a token before
// work and releases it after — clean, zero-dependency rate limiting.
func runConcurrentAudit(ctx context.Context, req auditRequest) []report.TargetAudit {
	results := make([]report.TargetAudit, len(req.Targets))
	sem := make(chan struct{}, req.Concurrency)
	var wg sync.WaitGroup

	pOpts := ping.DefaultOptions()
	pOpts.Count = req.PingCount

	tOpts := traceroute.DefaultOptions()
	tOpts.MaxHops = req.MaxHops

	for i, target := range req.Targets {
		wg.Add(1)
		go func(idx int, tgt string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire token
			defer func() { <-sem }() // release token

			audit := report.TargetAudit{Target: tgt}

			if req.RunPing {
				pr := ping.Run(ctx, tgt, pOpts)
				audit.Ping = &pr
			}
			if req.RunTrace {
				tr := traceroute.Run(ctx, tgt, tOpts)
				audit.Traceroute = &tr
			}

			results[idx] = audit
		}(i, target)
	}

	wg.Wait()
	return results
}

// ─── helpers ─────────────────────────────────────────────────────────────────

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// At this point headers are sent; log and move on
		log.Printf("writeJSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func decodeJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// jsonMiddleware sets Content-Type: application/json for all responses.
func jsonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs method, path, and duration for every request.
func loggingMiddleware(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf("%s %s %s", r.Method, r.URL.Path,
			fmt.Sprintf("%.2fms", float64(time.Since(start).Microseconds())/1000))
	})
}
