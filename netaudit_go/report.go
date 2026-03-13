// Package report assembles audit results and renders them as structured
// JSON or human-readable text.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/wyvern/netaudit/internal/ping"
	"github.com/wyvern/netaudit/internal/traceroute"
)

// AuditReport is the top-level output structure for one or more targets.
type AuditReport struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Duration    time.Duration `json:"total_duration_ns"`
	Targets     []TargetAudit `json:"targets"`
	Summary     AuditSummary  `json:"summary"`
}

// TargetAudit groups all tests for a single target host.
type TargetAudit struct {
	Target     string             `json:"target"`
	Ping       *ping.Result       `json:"ping,omitempty"`
	Traceroute *traceroute.Result `json:"traceroute,omitempty"`
}

// AuditSummary contains aggregate statistics.
type AuditSummary struct {
	TotalTargets    int `json:"total_targets"`
	Reachable       int `json:"reachable"`
	Unreachable     int `json:"unreachable"`
	TracerouteRan   int `json:"traceroute_ran"`
	DestinationHit  int `json:"destination_reached"`
}

// New creates an AuditReport and computes summary stats.
func New(audits []TargetAudit, duration time.Duration) AuditReport {
	r := AuditReport{
		GeneratedAt: time.Now().UTC(),
		Duration:    duration,
		Targets:     audits,
	}

	s := &r.Summary
	s.TotalTargets = len(audits)
	for _, a := range audits {
		if a.Ping != nil {
			if a.Ping.Reachable {
				s.Reachable++
			} else {
				s.Unreachable++
			}
		}
		if a.Traceroute != nil {
			s.TracerouteRan++
			if a.Traceroute.Reached {
				s.DestinationHit++
			}
		}
	}

	return r
}

// WriteJSON serialises the report as indented JSON to w.
func (r AuditReport) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// WriteText renders a human-readable report to w.
func (r AuditReport) WriteText(w io.Writer) {
	sep := strings.Repeat("─", 60)
	fmt.Fprintln(w, sep)
	fmt.Fprintf(w, "  WYVERN NETWORK AUDIT REPORT\n")
	fmt.Fprintf(w, "  Generated : %s\n", r.GeneratedAt.Format(time.RFC1123))
	fmt.Fprintf(w, "  Duration  : %s\n", r.Duration.Round(time.Millisecond))
	fmt.Fprintln(w, sep)

	// Summary box
	fmt.Fprintf(w, "  Targets: %d  |  Reachable: %d  |  Unreachable: %d\n",
		r.Summary.TotalTargets, r.Summary.Reachable, r.Summary.Unreachable)
	if r.Summary.TracerouteRan > 0 {
		fmt.Fprintf(w, "  Traceroutes: %d ran  |  Destination hit: %d\n",
			r.Summary.TracerouteRan, r.Summary.DestinationHit)
	}
	fmt.Fprintln(w, sep)

	for _, t := range r.Targets {
		fmt.Fprintf(w, "\n▶  %s\n", t.Target)

		if t.Ping != nil {
			p := t.Ping
			status := "✓ REACHABLE"
			if !p.Reachable {
				status = "✗ UNREACHABLE"
			}
			fmt.Fprintf(w, "   Ping   : %s\n", status)
			fmt.Fprintf(w, "            sent=%d recv=%d loss=%.0f%%\n",
				p.PacketsSent, p.PacketsRecv, p.PacketLoss)
			if p.Reachable {
				fmt.Fprintf(w, "            rtt min/avg/max = %.2f/%.2f/%.2f ms\n",
					p.MinRTT, p.AvgRTT, p.MaxRTT)
			}
			if p.Error != "" {
				fmt.Fprintf(w, "            error: %s\n", p.Error)
			}
		}

		if t.Traceroute != nil {
			tr := t.Traceroute
			reached := "✗ NOT REACHED"
			if tr.Reached {
				reached = "✓ REACHED"
			}
			fmt.Fprintf(w, "   Trace  : %s  (%d hops)\n", reached, tr.HopCount)
			for _, h := range tr.Hops {
				if h.TimedOut {
					fmt.Fprintf(w, "   %3d  * * *\n", h.Number)
					continue
				}
				label := h.IP
				if h.Hostname != "" && h.Hostname != h.IP {
					label = fmt.Sprintf("%s (%s)", h.Hostname, h.IP)
				}
				fmt.Fprintf(w, "   %3d  %-40s  avg %.2f ms\n",
					h.Number, label, h.AvgRTT)
			}
			if tr.Error != "" {
				fmt.Fprintf(w, "            error: %s\n", tr.Error)
			}
		}
	}

	fmt.Fprintln(w, "\n"+sep)
}
