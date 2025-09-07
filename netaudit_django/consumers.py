""" import asyncio
import json
import subprocess
from channels.generic.websocket import AsyncWebsocketConsumer # type: ignore

class SQLMapConsumer(AsyncWebsocketConsumer):
    async def connect(self):
        await self.accept()
        self.process = None

    async def disconnect(self, close_code):
        if self.process:
            self.process.terminate()

    async def receive(self, text_data):
        try:
            data = json.loads(text_data)
            if data["action"] == "start":
                # Start SQLMap process
                target_url = data["target_url"]
                selected_options = data["selected_options"]
                command = ["sqlmap", "-u", target_url] + selected_options.split()

                self.process = await asyncio.create_subprocess_exec(
                    *command,
                    stdout=subprocess.PIPE,
                    stderr=subprocess.PIPE,
                    stdin=subprocess.PIPE
                )

                # Read output asynchronously
                asyncio.create_task(self.stream_sqlmap_output())
            elif data["action"] == "input" and self.process:
                # Send user input to the process
                user_input = data["input"]
                self.process.stdin.write(user_input.encode() + b"\n") # type: ignore
                await self.process.stdin.drain() # type: ignore

        except Exception as e:
            await self.send(json.dumps({"type": "error", "message": str(e)}))

    async def stream_sqlmap_output(self):
        try:
            while self.process:
                line = await self.process.stdout.readline() # type: ignore
                if not line:
                    break
                await self.send(json.dumps({"type": "output", "message": line.decode()}))
        except Exception as e:
            await self.send(json.dumps({"type": "error", "message": str(e)}))
 """

import json
from channels.generic.websocket import AsyncWebsocketConsumer

class NucleiScanConsumer(AsyncWebsocketConsumer):
    async def connect(self):
        # Accept the WebSocket connection
        self.room_group_name = "nuclei_scan_updates"

        # Join the room group
        await self.channel_layer.group_add(
            self.room_group_name,
            self.channel_name
        )

        await self.accept()

    async def disconnect(self, close_code):
        # Leave the room group
        await self.channel_layer.group_discard(
            self.room_group_name,
            self.channel_name
        )

    # Receive a message from WebSocket
    async def receive(self, text_data):
        data = json.loads(text_data)
        # Handle the received data
        pass

    # Send message to WebSocket
    async def send_scan_update(self, event):
        message = event['message']

        # Send message to WebSocket
        await self.send(text_data=json.dumps({
            'message': message
        }))
