// Command netaudit is the Wyvern Network Audit tool.
//
// Usage:
//
//	netaudit ping     <target> [flags]
//	netaudit trace    <target> [flags]
//	netaudit audit    <target,...> [flags]
//	netaudit serve    [flags]
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wyvern/netaudit/internal/ping"
	"github.com/wyvern/netaudit/internal/report"
	"github.com/wyvern/netaudit/internal/server"
	"github.com/wyvern/netaudit/internal/traceroute"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	logger := log.New(os.Stderr, "[netaudit] ", log.LstdFlags)

	switch os.Args[1] {
	case "ping":
		runPing(os.Args[2:], logger)
	case "trace", "traceroute":
		runTrace(os.Args[2:], logger)
	case "audit":
		runAudit(os.Args[2:], logger)
	case "serve":
		runServe(os.Args[2:], logger)
	case "version", "-v", "--version":
		fmt.Printf("wyvern-netaudit v%s\n", version)
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

// ─── ping subcommand ─────────────────────────────────────────────────────────

func runPing(args []string, logger *log.Logger) {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	count := fs.Int("c", 4, "number of echo requests")
	timeout := fs.Duration("t", 5*time.Second, "per-probe timeout")
	jsonOut := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: netaudit ping <target> [flags]")
		fs.PrintDefaults()
		os.Exit(1)
	}

	target := fs.Arg(0)
	opts := ping.Options{Count: *count, Timeout: *timeout}
	ctx, cancel := context.WithTimeout(context.Background(),
		*timeout*time.Duration(*count)+5*time.Second)
	defer cancel()

	result := ping.Run(ctx, target, opts)

	if *jsonOut {
		printJSON(result)
		return
	}

	fmt.Println(result.Summary())
	if result.RawOutput != "" {
		fmt.Println()
		fmt.Println(result.RawOutput)
	}
}

// ─── traceroute subcommand ───────────────────────────────────────────────────

func runTrace(args []string, logger *log.Logger) {
	fs := flag.NewFlagSet("trace", flag.ExitOnError)
	maxHops := fs.Int("m", 30, "maximum hops")
	jsonOut := fs.Bool("json", false, "output JSON")
	_ = fs.Parse(args)

	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: netaudit trace <target> [flags]")
		fs.PrintDefaults()
		os.Exit(1)
	}

	target := fs.Arg(0)
	opts := traceroute.Options{MaxHops: *maxHops, Timeout: 60 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	result := traceroute.Run(ctx, target, opts)

	if *jsonOut {
		printJSON(result)
		return
	}

	fmt.Println(result.Summary())
	fmt.Println()
	fmt.Println(result.RawOutput)
}

// ─── audit subcommand ────────────────────────────────────────────────────────

func runAudit(args []string, logger *log.Logger) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	targets := fs.String("targets", "", "comma-separated list of targets")
	doTrace := fs.Bool("trace", false, "also run traceroute per target")
	pingCount := fs.Int("c", 4, "ping count per target")
	concurrency := fs.Int("p", 10, "max concurrent probes")
	jsonOut := fs.Bool("json", false, "output JSON")
	outFile := fs.String("o", "", "write report to file")
	_ = fs.Parse(args)

	// Targets can come from -targets flag or positional args
	var hosts []string
	if *targets != "" {
		hosts = splitTargets(*targets)
	}
	hosts = append(hosts, fs.Args()...)

	if len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "usage: netaudit audit <target,...> [flags]")
		fs.PrintDefaults()
		os.Exit(1)
	}

	pOpts := ping.DefaultOptions()
	pOpts.Count = *pingCount
	tOpts := traceroute.DefaultOptions()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sem := make(chan struct{}, *concurrency)
	type indexedAudit struct {
		idx   int
		audit report.TargetAudit
	}
	ch := make(chan indexedAudit, len(hosts))

	start := time.Now()

	for i, target := range hosts {
		go func(idx int, tgt string) {
			sem <- struct{}{}
			defer func() { <-sem }()

			a := report.TargetAudit{Target: tgt}
			pr := ping.Run(ctx, tgt, pOpts)
			a.Ping = &pr

			if *doTrace {
				tr := traceroute.Run(ctx, tgt, tOpts)
				a.Traceroute = &tr
			}

			ch <- indexedAudit{idx: idx, audit: a}
		}(i, target)
	}

	audits := make([]report.TargetAudit, len(hosts))
	for range hosts {
		ia := <-ch
		audits[ia.idx] = ia.audit
	}

	rpt := report.New(audits, time.Since(start))

	// Output destination
	out := os.Stdout
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			logger.Fatalf("cannot create output file: %v", err)
		}
		defer f.Close()
		out = f
	}

	if *jsonOut {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(rpt); err != nil {
			logger.Fatalf("json encode: %v", err)
		}
		return
	}

	rpt.WriteText(out)
}

// ─── serve subcommand ────────────────────────────────────────────────────────

func runServe(args []string, logger *log.Logger) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", ":8080", "listen address")
	_ = fs.Parse(args)

	srv := server.New(*addr, logger)

	// Graceful shutdown on SIGINT / SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Printf("server stopped: %v", err)
		}
	}()

	<-quit
	logger.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("shutdown error: %v", err)
	}
	logger.Println("bye")
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func printUsage() {
	fmt.Fprintf(os.Stderr, `
Wyvern NetAudit — Network Diagnostic Tool  v%s

Usage:
  netaudit ping     <target> [-c count] [-t timeout] [--json]
  netaudit trace    <target> [-m maxhops] [--json]
  netaudit audit    <t1> <t2> ... [-trace] [-c count] [-p concurrency] [--json] [-o file]
  netaudit serve    [-addr :8080]
  netaudit version

API endpoints (serve mode):
  POST /api/ping        {"target":"8.8.8.8","count":4}
  POST /api/traceroute  {"target":"8.8.8.8","max_hops":30}
  POST /api/audit       {"targets":["8.8.8.8","1.1.1.1"],"ping":true,"traceroute":false}
  GET  /health
`, version)
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(os.Stderr, "json encode error: %v\n", err)
		os.Exit(1)
	}
}

func splitTargets(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
