// Package ping executes ICMP ping via OS subprocess and parses output.
// Design decision: We shell out to the system `ping` binary rather than
// crafting raw ICMP packets — this keeps the binary unprivileged (no
// CAP_NET_RAW needed) while still being fully functional.
package ping

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Result holds the structured outcome of a single ping session.
type Result struct {
	Target      string        `json:"target"`
	Reachable   bool          `json:"reachable"`
	PacketsSent int           `json:"packets_sent"`
	PacketsRecv int           `json:"packets_recv"`
	PacketLoss  float64       `json:"packet_loss_percent"`
	MinRTT      float64       `json:"min_rtt_ms"`
	AvgRTT      float64       `json:"avg_rtt_ms"`
	MaxRTT      float64       `json:"max_rtt_ms"`
	StdDevRTT   float64       `json:"stddev_rtt_ms"`
	RawOutput   string        `json:"raw_output"`
	Duration    time.Duration `json:"duration_ns"`
	Error       string        `json:"error,omitempty"`
}

// Options configures a ping run.
type Options struct {
	Count   int           // number of echo requests
	Timeout time.Duration // per-probe deadline
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		Count:   4,
		Timeout: 5 * time.Second,
	}
}

// Run executes a ping against target and returns a structured Result.
// It respects ctx cancellation — pass context.WithTimeout for hard deadlines.
func Run(ctx context.Context, target string, opts Options) Result {
	start := time.Now()
	res := Result{Target: target, PacketsSent: opts.Count}

	args := buildArgs(target, opts)
	cmd := exec.CommandContext(ctx, "ping", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res.Duration = time.Since(start)
	res.RawOutput = strings.TrimSpace(stdout.String())

	if stderr.Len() > 0 {
		res.Error = strings.TrimSpace(stderr.String())
	}

	// A non-zero exit code usually means unreachable, but we still parse
	// whatever output we got to extract partial stats.
	parseStats(&res, res.RawOutput)

	// If we received at least one reply → reachable
	if err == nil || res.PacketsRecv > 0 {
		res.Reachable = res.PacketsRecv > 0
	}

	return res
}

// buildArgs constructs OS-appropriate ping arguments.
func buildArgs(target string, opts Options) []string {
	count := strconv.Itoa(opts.Count)
	timeoutSec := strconv.Itoa(int(opts.Timeout.Seconds()))

	switch runtime.GOOS {
	case "darwin":
		// macOS: -c count, -t timeout (seconds), -q quiet summary
		return []string{"-c", count, "-t", timeoutSec, "-q", target}
	case "windows":
		// Windows: -n count (no native timeout-per-probe in ms easily)
		return []string{"-n", count, target}
	default:
		// Linux: -c count, -W timeout (seconds), -q quiet
		return []string{"-c", count, "-W", timeoutSec, "-q", target}
	}
}

// --- output parsers -------------------------------------------------------

// rttLineRe matches Linux/macOS summary lines like:
//
//	rtt min/avg/max/mdev = 1.234/2.345/3.456/0.567 ms
var rttLineRe = regexp.MustCompile(
	`(?i)(?:rtt|round-trip)\s+min/avg/max/(?:mdev|stddev)\s*=\s*([\d.]+)/([\d.]+)/([\d.]+)/([\d.]+)\s*ms`,
)

// lossRe matches "X% packet loss" anywhere in the output.
var lossRe = regexp.MustCompile(`([\d.]+)%\s+packet loss`)

// recvRe matches "X received" or "X packets received".
var recvRe = regexp.MustCompile(`(\d+)\s+(?:packets?\s+)?received`)

// sentRe matches "X packets transmitted".
var sentRe = regexp.MustCompile(`(\d+)\s+packets?\s+transmitted`)

func parseStats(res *Result, output string) {
	if m := rttLineRe.FindStringSubmatch(output); len(m) == 5 {
		res.MinRTT, _ = strconv.ParseFloat(m[1], 64)
		res.AvgRTT, _ = strconv.ParseFloat(m[2], 64)
		res.MaxRTT, _ = strconv.ParseFloat(m[3], 64)
		res.StdDevRTT, _ = strconv.ParseFloat(m[4], 64)
	}

	if m := lossRe.FindStringSubmatch(output); len(m) == 2 {
		res.PacketLoss, _ = strconv.ParseFloat(m[1], 64)
	}

	if m := recvRe.FindStringSubmatch(output); len(m) == 2 {
		res.PacketsRecv, _ = strconv.Atoi(m[1])
	}

	if m := sentRe.FindStringSubmatch(output); len(m) == 2 {
		res.PacketsSent, _ = strconv.Atoi(m[1])
	}
}

// Summary returns a human-readable one-liner.
func (r Result) Summary() string {
	if r.Error != "" && !r.Reachable {
		return fmt.Sprintf("[UNREACHABLE] %s — %s", r.Target, r.Error)
	}
	if !r.Reachable {
		return fmt.Sprintf("[UNREACHABLE] %s — 100%% packet loss", r.Target)
	}
	return fmt.Sprintf("[OK] %s — loss=%.0f%% rtt min/avg/max=%.2f/%.2f/%.2f ms",
		r.Target, r.PacketLoss, r.MinRTT, r.AvgRTT, r.MaxRTT)
}
