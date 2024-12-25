
# Create your views here.
import re
import subprocess
from django.http import JsonResponse
from django.shortcuts import render
from .models import PingHistory

def dashboard(request):
    return render(request, 'dashboard.html')

#the ping details showing but flushing very fast.
def parse_ping_output(output):
    """Parse ping command output to extract detailed statistics."""
    stats = {
        'min_latency': None,
        'avg_latency': None,
        'max_latency': None,
        'packet_loss': None,
        'packets_transmitted': None,
        'packets_received': None
    }
    
    # Extract packet statistics
    if "packets transmitted" in output:
        match = re.search(r'(\d+) packets transmitted, (\d+) received, (\d+\.?\d*)% packet loss', output)
        if match:
            stats['packets_transmitted'] = int(match.group(1))
            stats['packets_received'] = int(match.group(2))
            stats['packet_loss'] = float(match.group(3))
    
    # Extract latency statistics
    if "min/avg/max" in output:
        match = re.search(r'min/avg/max\S*\s=\s(\d+\.?\d*)/(\d+\.?\d*)/(\d+\.?\d*)', output)
        if match:
            stats['min_latency'] = float(match.group(1))
            stats['avg_latency'] = float(match.group(2))
            stats['max_latency'] = float(match.group(3))
    
    return stats

def ping_device(ip_address, count=4, timeout=1):
    try:
        result = subprocess.run(
            ["ping", "-c", str(count), "-W", str(timeout), ip_address],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        
        is_reachable = result.returncode == 0
        stats = parse_ping_output(result.stdout)
        
        # Store ping result in database
        PingHistory.objects.create(
            ip_address=ip_address,
            is_successful=is_reachable,
            **stats
        )
        
        return {
            'success': is_reachable,
            'message': f"Device {ip_address} is {'reachable' if is_reachable else 'not reachable'}.",
            'details': stats,
            'raw_output': result.stdout
        }
    except Exception as e:
        return {
            'success': False,
            'message': f"An error occurred: {e}",
            'details': None,
            'raw_output': None
        }

def ping_view(request):
    if request.method == "POST":
        target_ip = request.POST.get("target_ip")
        if not target_ip:
            return JsonResponse({"success": False, "message": "Please provide a valid IP address."})
        
        result = ping_device(target_ip)
        return JsonResponse(result)
    
    # Get recent ping history
    recent_pings = PingHistory.objects.all()[:10]
    return render(request, "ping.html", {'recent_pings': recent_pings})

import re
import time
import subprocess
from django.http import JsonResponse
from django.shortcuts import render
from .models import TracerouteHistory, TracerouteHop

def parse_traceroute_output(output):
    """Parse traceroute command output to extract hop information."""
    hops = []
    lines = output.strip().split('\n')
    
    for line in lines[1:]:  # Skip the first line (header)
        hop_info = {
            'hop_number': None,
            'hostname': None,
            'ip_address': None,
            'rtt1': None,
            'rtt2': None,
            'rtt3': None
        }
        
        # Extract hop number
        hop_match = re.match(r'\s*(\d+)', line)
        if hop_match:
            hop_info['hop_number'] = int(hop_match.group(1))
        else:
            continue

        # Extract IP addresses
        ip_matches = re.findall(r'\(([\d.]+)\)', line)
        if ip_matches:
            hop_info['ip_address'] = ip_matches[0]
        else:
            # Fallback: Check for inline IPs
            inline_ip_match = re.search(r'(\d{1,3}(?:\.\d{1,3}){3})', line)
            if inline_ip_match:
                hop_info['ip_address'] = inline_ip_match.group(1)
        
        # Extract RTT values
        rtt_matches = re.findall(r'(\d+\.\d+) ms', line)
        for idx, rtt in enumerate(rtt_matches[:3]):
            hop_info[f'rtt{idx+1}'] = float(rtt)

        # Check hostname
        host_match = re.search(r'(?<=\s)([a-zA-Z0-9.-]+)(?=\s+\()', line)
        if host_match:
            hop_info['hostname'] = host_match.group(1)
        
        # Detect unreachable hops
        if not ip_matches and not rtt_matches:
            hop_info['hostname'] = None
            hop_info['ip_address'] = None

        hops.append(hop_info)
    
    return hops


def traceroute_device(ip_address, max_hops=30, timeout=5):
    """Perform a traceroute to the given IP address with improved error handling."""
    try:
        start_time = time.time()
        
        # Run traceroute command with additional parameters for better output
        result = subprocess.run(
            ["traceroute", "-n", "-m", str(max_hops), "-w", str(timeout), ip_address],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            timeout=60  # Add overall timeout
        )
        
        completion_time = time.time() - start_time
        
        # Check if traceroute was successful
        if result.returncode == 0:
            # Parse the output
            parsed_hops = parse_traceroute_output(result.stdout)
            
            # Create history entry
            history = TracerouteHistory.objects.create(
                ip_address=ip_address,
                is_successful=True,
                total_hops=len(parsed_hops),
                completion_time=completion_time
            )
            
            # Create hop entries
            for hop in parsed_hops:
                TracerouteHop.objects.create(
                    traceroute=history,
                    **hop
                )
            
            return {
                "success": True,
                "message": "Traceroute completed successfully",
                "output": result.stdout,
                "parsed_hops": parsed_hops,
                "total_hops": len(parsed_hops),
                "completion_time": completion_time,
                "raw_output": result.stdout
            }
        else:
            # Handle unsuccessful traceroute
            TracerouteHistory.objects.create(
                ip_address=ip_address,
                is_successful=False,
                completion_time=completion_time
            )
            return {
                "success": False,
                "message": "Traceroute failed",
                "output": result.stderr or "No error message available"
            }
    except subprocess.TimeoutExpired:
        return {
            "success": False,
            "message": "Traceroute operation timed out"
        }
    except Exception as e:
        return {
            "success": False,
            "message": f"An error occurred: {str(e)}"
        }

def traceroute_view(request):
    if request.method == "POST":
        target_ip = request.POST.get("target_ip")
        if not target_ip:
            return JsonResponse({
                "success": False,
                "message": "Please provide a valid IP address."
            })
        
        result = traceroute_device(target_ip)
        return JsonResponse(result)
    
    recent_traceroutes = TracerouteHistory.objects.select_related().prefetch_related('hops').all()[:10]
    return render(request, "traceroute.html", {'recent_traceroutes': recent_traceroutes})