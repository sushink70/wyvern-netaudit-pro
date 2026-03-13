// Package traceroute shells out to the system `traceroute` (or `tracert` on
// Windows) and parses each hop into a structured slice.
package traceroute

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

// Hop represents a single TTL hop in the route.
type Hop struct {
	Number   int      `json:"hop"`
	Hostname string   `json:"hostname,omitempty"`
	IP       string   `json:"ip,omitempty"`
	RTTs     []string `json:"rtts_ms"`       // raw strings; "*" means timeout
	AvgRTT   float64  `json:"avg_rtt_ms"`    // 0 if all probes timed out
	TimedOut bool     `json:"timed_out"`     // true when all probes are "*"
}

// Result holds the full traceroute outcome.
type Result struct {
	Target    string        `json:"target"`
	Hops      []Hop         `json:"hops"`
	HopCount  int           `json:"hop_count"`
	Reached   bool          `json:"destination_reached"`
	RawOutput string        `json:"raw_output"`
	Duration  time.Duration `json:"duration_ns"`
	Error     string        `json:"error,omitempty"`
}

// Options configures a traceroute run.
type Options struct {
	MaxHops int
	Timeout time.Duration // total execution timeout
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		MaxHops: 30,
		Timeout: 60 * time.Second,
	}
}

// Run executes traceroute against target and returns structured output.
func Run(ctx context.Context, target string, opts Options) Result {
	start := time.Now()
	res := Result{Target: target}

	args := buildArgs(target, opts)
	bin := binary()
	cmd := exec.CommandContext(ctx, bin, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	_ = cmd.Run() // ignore exit code; partial output is still useful
	res.Duration = time.Since(start)
	res.RawOutput = strings.TrimSpace(stdout.String())

	if stderr.Len() > 0 {
		res.Error = strings.TrimSpace(stderr.String())
	}

	res.Hops = parseHops(res.RawOutput)
	res.HopCount = len(res.Hops)

	// Destination reached if the last non-timeout hop mentions target
	if res.HopCount > 0 {
		last := res.Hops[res.HopCount-1]
		if !last.TimedOut && (strings.Contains(last.IP, target) ||
			strings.Contains(last.Hostname, target)) {
			res.Reached = true
		}
	}

	return res
}

func binary() string {
	if runtime.GOOS == "windows" {
		return "tracert"
	}
	return "traceroute"
}

func buildArgs(target string, opts Options) []string {
	maxHops := strconv.Itoa(opts.MaxHops)
	switch runtime.GOOS {
	case "windows":
		return []string{"-h", maxHops, target}
	case "darwin":
		return []string{"-m", maxHops, "-q", "3", target}
	default:
		// Linux: -m maxhops, -q probes per hop, -w wait per probe (sec)
		return []string{"-m", maxHops, "-q", "3", "-w", "2", target}
	}
}

// --- hop line parser -------------------------------------------------------
//
// Linux traceroute output per hop:
//   1  gateway (192.168.1.1)  0.512 ms  0.423 ms  0.401 ms
//   2  * * *
//   3  203.0.113.1 (203.0.113.1)  10.1 ms  9.8 ms  9.9 ms
//
// The regex captures:
//   group 1 = hop number
//   group 2 = rest of the line (hostname/ip + rtts)

var hopLineRe = regexp.MustCompile(`^\s*(\d+)\s+(.+)$`)

// ipAddrRe matches bare IP addresses or (IP) in the hop line.
var ipAddrRe = regexp.MustCompile(`\(?(\d{1,3}(?:\.\d{1,3}){3})\)?`)

// rttValueRe matches numeric RTT values (e.g. "10.512 ms").
var rttValueRe = regexp.MustCompile(`([\d.]+)\s*ms`)

func parseHops(output string) []Hop {
	lines := strings.Split(output, "\n")
	hops := make([]Hop, 0, 30)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		m := hopLineRe.FindStringSubmatch(line)
		if len(m) != 3 {
			continue
		}

		hopNum, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}

		rest := m[2]
		hop := Hop{Number: hopNum}

		// Check for total timeout line ("* * *")
		if isAllStars(rest) {
			hop.TimedOut = true
			hop.RTTs = []string{"*", "*", "*"}
			hops = append(hops, hop)
			continue
		}

		// Extract IP
		if ips := ipAddrRe.FindStringSubmatch(rest); len(ips) > 1 {
			hop.IP = ips[1]
		}

		// Extract hostname (first token before the IP or parenthesis)
		tokens := strings.Fields(rest)
		if len(tokens) > 0 && !strings.HasPrefix(tokens[0], "(") {
			first := tokens[0]
			// If it doesn't look like an IP, treat it as hostname
			if !ipAddrRe.MatchString(first) {
				hop.Hostname = first
			}
		}

		// Extract RTT values
		rttMatches := rttValueRe.FindAllStringSubmatch(rest, -1)
		sum := 0.0
		count := 0
		for _, rm := range rttMatches {
			v, _ := strconv.ParseFloat(rm[1], 64)
			hop.RTTs = append(hop.RTTs, rm[1]+" ms")
			sum += v
			count++
		}
		// Fill missing probes with "*"
		for len(hop.RTTs) < 3 {
			hop.RTTs = append(hop.RTTs, "*")
		}

		if count > 0 {
			hop.AvgRTT = sum / float64(count)
		}

		hops = append(hops, hop)
	}

	return hops
}

func isAllStars(s string) bool {
	for _, f := range strings.Fields(s) {
		if f != "*" {
			return false
		}
	}
	return len(strings.Fields(s)) > 0
}

// Summary returns a concise human-readable overview.
func (r Result) Summary() string {
	reached := "NOT REACHED"
	if r.Reached {
		reached = "REACHED"
	}
	return fmt.Sprintf("[%s] %s — %d hops in %s",
		reached, r.Target, r.HopCount, r.Duration.Round(time.Millisecond))
}
