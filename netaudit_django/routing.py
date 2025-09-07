""" from django.urls import path
from netaudit import consumers

websocket_urlpatterns = [
    path("ws/sqlmap/", consumers.SQLMapConsumer.as_asgi()),
]
 """

from django.urls import path
from .consumers import NucleiScanConsumer

websocket_urlpatterns = [
    path("ws/nuclei-scan/", NucleiScanConsumer.as_asgi()),
]
