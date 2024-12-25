

# Create your views here.
# views.py
from django.shortcuts import render
from django.http import JsonResponse
import subprocess

def ping_device(ip_address, count=4, timeout=1):
    try:
        result = subprocess.run(
            ["ping", "-c", str(count), "-W", str(timeout), ip_address],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        if result.returncode == 0:
            return True, f"Device {ip_address} is reachable."
        else:
            return False, f"Device {ip_address} is not reachable."
    except Exception as e:
        return False, f"An error occurred: {e}"

def ping_view(request):
    if request.method == "POST":
        target_ip = request.POST.get("target_ip")
        if not target_ip:
            return JsonResponse({"success": False, "message": "Please provide a valid IP address."})
        is_reachable, message = ping_device(target_ip)
        return JsonResponse({"success": is_reachable, "message": message})
    return render(request, "ping.html")


def traceroute_device(ip_address, max_hops=30, timeout=5):
    """
    Perform a traceroute to the given IP address or hostname.
    
    Args:
        ip_address (str): The target IP address or hostname.
        max_hops (int): Maximum number of hops to trace. Default is 30.
        timeout (int): Timeout for each hop in seconds. Default is 5.
    
    Returns:
        dict: A dictionary containing the success status and traceroute result.
    """
    try:
        # Run the traceroute command
        result = subprocess.run(
            ["traceroute", "-m", str(max_hops), "-w", str(timeout), ip_address],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True
        )
        if result.returncode == 0:
            return {"success": True, "output": result.stdout}
        else:
            return {"success": False, "output": result.stderr}
    except Exception as e:
        return {"success": False, "output": str(e)}

def traceroute_view(request):
    if request.method == "POST":
        target_ip = request.POST.get("target_ip")
        if not target_ip:
            return JsonResponse({"success": False, "message": "Please provide a valid IP address."})
        result = traceroute_device(target_ip)
        return JsonResponse(result)
    return render(request, "traceroute.html")
