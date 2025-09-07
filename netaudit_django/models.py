from django.db import models

# Create your models here.
class PingHistory(models.Model):
    ip_address = models.CharField(max_length=45)
    timestamp = models.DateTimeField(auto_now_add=True)
    is_successful = models.BooleanField()
    min_latency = models.FloatField(null=True)
    avg_latency = models.FloatField(null=True)
    max_latency = models.FloatField(null=True)
    packet_loss = models.FloatField(null=True)
    packets_transmitted = models.IntegerField(null=True)
    packets_received = models.IntegerField(null=True)

    class Meta:
        ordering = ['-timestamp']

class TracerouteHistory(models.Model):
    ip_address = models.CharField(max_length=45)
    timestamp = models.DateTimeField(auto_now_add=True)
    is_successful = models.BooleanField()
    total_hops = models.IntegerField(null=True)
    completion_time = models.FloatField(null=True)  # in seconds

    class Meta:
        ordering = ['-timestamp']


class TracerouteHop(models.Model):
    traceroute = models.ForeignKey(TracerouteHistory, related_name='hops', on_delete=models.CASCADE)
    hop_number = models.IntegerField()
    hostname = models.CharField(max_length=255, null=True, blank=True)
    ip_address = models.CharField(max_length=45, null=True, blank=True)
    rtt1 = models.FloatField(null=True)  # Round-trip time 1
    rtt2 = models.FloatField(null=True)  # Round-trip time 2
    rtt3 = models.FloatField(null=True)  # Round-trip time 3
    
    class Meta:
        ordering = ['hop_number']

""" from django.db import models

class SQLMapLog(models.Model):
    target_url = models.URLField(max_length=500)
    options = models.TextField()
    output = models.TextField()
    success = models.BooleanField(default=False)
    timestamp = models.DateTimeField(auto_now_add=True)

    def __str__(self):
        return f"SQLMap Log - {self.target_url} ({'Success' if self.success else 'Failed'})"
 """