use regex::Regex;
use std::process::Stdio;
use std::time::Instant;
use tokio::process::Command;

use crate::models::{PingResult, PingStats, TracerouteHop, TracerouteResult};

pub async fn ping_device(ip_address: &str) -> PingResult {
    let start_time = Instant::now();
    
    let result = Command::new("ping")
        .arg("-c")
        .arg("4")
        .arg("-W")
        .arg("1")
        .arg(ip_address)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .output()
        .await;

    match result {
        Ok(output) => {
            let stdout = String::from_utf8_lossy(&output.stdout);
            let is_successful = output.status.success();
            
            let stats = if is_successful {
                parse_ping_output(&stdout)
            } else {
                None
            };

            PingResult {
                success: is_successful,
                message: format!(
                    "Device {} is {}.",
                    ip_address,
                    if is_successful { "reachable" } else { "not reachable" }
                ),
                details: stats,
                raw_output: Some(stdout.to_string()),
            }
        }
        Err(e) => PingResult {
            success: false,
            message: format!("An error occurred: {}", e),
            details: None,
            raw_output: None,
        },
    }
}

fn parse_ping_output(output: &str) -> Option<PingStats> {
    let mut stats = PingStats {
        min_latency: 0.0,
        avg_latency: 0.0,
        max_latency: 0.0,
        packet_loss: None,
        packets_transmitted: None,
        packets_received: None,
    };

    // Extract packet statistics
    if let Ok(packet_regex) = Regex::new(r"(\d+) packets transmitted, (\d+) received, (\d+\.?\d*)% packet loss") {
        if let Some(captures) = packet_regex.captures(output) {
            stats.packets_transmitted = captures.get(1)?.as_str().parse().ok();
            stats.packets_received = captures.get(2)?.as_str().parse().ok();
            stats.packet_loss = captures.get(3)?.as_str().parse().ok();
        }
    }

    // Extract latency statistics
    if let Ok(latency_regex) = Regex::new(r"min/avg/max\S*\s=\s(\d+\.?\d*)/(\d+\.?\d*)/(\d+\.?\d*)") {
        if let Some(captures) = latency_regex.captures(output) {
            stats.min_latency = captures.get(1)?.as_str().parse().ok()?;
            stats.avg_latency = captures.get(2)?.as_str().parse().ok()?;
            stats.max_latency = captures.get(3)?.as_str().parse().ok()?;
        }
    }

    Some(stats)
}

pub async fn traceroute_device(ip_address: &str) -> TracerouteResult {
    let start_time = Instant::now();
    
    let result = Command::new("traceroute")
        .arg("-n")
        .arg("-m")
        .arg("30")
        .arg("-w")
        .arg("5")
        .arg(ip_address)
        .stdout(Stdio::piped())
        .stderr(Stdio::piped())
        .output()
        .await;

    let completion_time = start_time.elapsed().as_secs_f64();

    match result {
        Ok(output) => {
            let stdout = String::from_utf8_lossy(&output.stdout);
            let is_successful = output.status.success();

            if is_successful {
                let parsed_hops = parse_traceroute_output(&stdout);
                let total_hops = parsed_hops.len() as i32;

                TracerouteResult {
                    success: true,
                    message: "Traceroute completed successfully".to_string(),
                    parsed_hops: Some(parsed_hops),
                    total_hops: Some(total_hops),
                    completion_time,
                    raw_output: Some(stdout.to_string()),
                }
            } else {
                let stderr = String::from_utf8_lossy(&output.stderr);
                TracerouteResult {
                    success: false,
                    message: "Traceroute failed".to_string(),
                    parsed_hops: None,
                    total_hops: None,
                    completion_time,
                    raw_output: Some(stderr.to_string()),
                }
            }
        }
        Err(e) => TracerouteResult {
            success: false,
            message: format!("An error occurred: {}", e),
            parsed_hops: None,
            total_hops: None,
            completion_time,
            raw_output: None,
        },
    }
}

fn parse_traceroute_output(output: &str) -> Vec<TracerouteHop> {
    let mut hops = Vec::new();
    let lines: Vec<&str> = output.trim().split('\n').collect();

    // Skip the first line (header)
    for line in lines.iter().skip(1) {
        let mut hop = TracerouteHop {
            id: 0,
            traceroute_id: 0,
            hop_number: 0,
            hostname: None,
            ip_address: None,
            rtt1: None,
            rtt2: None,
            rtt3: None,
        };

        // Extract hop number
        if let Ok(hop_regex) = Regex::new(r"^\s*(\d+)") {
            if let Some(captures) = hop_regex.captures(line) {
                if let Ok(hop_num) = captures.get(1).unwrap().as_str().parse::<i32>() {
                    hop.hop_number = hop_num;
                } else {
                    continue;
                }
            } else {
                continue;
            }
        }

        // Extract IP addresses
        if let Ok(ip_regex) = Regex::new(r"\(([\d.]+)\)") {
            if let Some(captures) = ip_regex.captures(line) {
                hop.ip_address = Some(captures.get(1).unwrap().as_str().to_string());
            } else {
                // Fallback: Check for inline IPs
                if let Ok(inline_ip_regex) = Regex::new(r"(\d{1,3}(?:\.\d{1,3}){3})") {
                    if let Some(captures) = inline_ip_regex.captures(line) {
                        hop.ip_address = Some(captures.get(1).unwrap().as_str().to_string());
                    }
                }
            }
        }

        // Extract RTT values
        if let Ok(rtt_regex) = Regex::new(r"(\d+\.\d+) ms") {
            let rtt_matches: Vec<_> = rtt_regex.captures_iter(line).collect();
            if !rtt_matches.is_empty() {
                if let Ok(rtt) = rtt_matches[0].get(1).unwrap().as_str().parse::<f64>() {
                    hop.rtt1 = Some(rtt);
                }
                if rtt_matches.len() > 1 {
                    if let Ok(rtt) = rtt_matches[1].get(1).unwrap().as_str().parse::<f64>() {
                        hop.rtt2 = Some(rtt);
                    }
                }
                if rtt_matches.len() > 2 {
                    if let Ok(rtt) = rtt_matches[2].get(1).unwrap().as_str().parse::<f64>() {
                        hop.rtt3 = Some(rtt);
                    }
                }
            }
        }

        // Extract hostname
        if let Ok(hostname_regex) = Regex::new(r"(?<=\s)([a-zA-Z0-9.-]+)(?=\s+\()") {
            if let Some(captures) = hostname_regex.captures(line) {
                hop.hostname = Some(captures.get(1).unwrap().as_str().to_string());
            }
        }

        hops.push(hop);
    }

    hops
}