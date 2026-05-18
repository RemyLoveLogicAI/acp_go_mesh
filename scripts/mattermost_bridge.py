#!/usr/bin/env python3
"""
Pi-Intercom -> Mattermost Bridge
Forwards Pi session events to Mattermost channels.
Receives Mattermost commands and triggers Pi tasks.
"""
import argparse
import asyncio
import json
import os
import sys
import websockets
import httpx
from datetime import datetime

MATTERMOST_URL = os.environ.get("MATTERMOST_URL", "http://localhost:8065")
BOT_TOKEN = os.environ.get("MATTERMOST_BOT_TOKEN", "")
TEAM_NAME = os.environ.get("MATTERMOST_TEAM", "orion-fabric")
DEFAULT_CHANNEL = os.environ.get("MATTERMOST_DEFAULT_CHANNEL", "peer-coordination")

HTTP = httpx.AsyncClient(timeout=30.0)

async def get_channel_id(channel_name: str) -> str:
    headers = {"Authorization": f"Bearer {BOT_TOKEN}"}
    r = await HTTP.get(f"{MATTERMOST_URL}/api/v4/teams/name/{TEAM_NAME}/channels/name/{channel_name}", headers=headers)
    r.raise_for_status()
    return r.json()["id"]

async def post_message(channel_id: str, message: str, props: dict = None):
    headers = {"Authorization": f"Bearer {BOT_TOKEN}"}
    payload = {"channel_id": channel_id, "message": message}
    if props:
        payload["props"] = props
    r = await HTTP.post(f"{MATTERMOST_URL}/api/v4/posts", headers=headers, json=payload)
    r.raise_for_status()
    return r.json()

async def broadcast_to_channel(channel_name: str, event_type: str, data: dict):
    try:
        channel_id = await get_channel_id(channel_name)
        ts = datetime.utcnow().isoformat() + "Z"
        message = f"**[{event_type}]** `{ts}`\n```json\n{json.dumps(data, indent=2)}\n```"
        await post_message(channel_id, message)
    except Exception as e:
        print(f"[Bridge] Failed to broadcast: {e}", file=sys.stderr)

async def listen_websocket():
    ws_url = MATTERMOST_URL.replace("http://", "ws://").replace("https://", "wss://") + "/api/v4/websocket"
    headers = {"Authorization": f"Bearer {BOT_TOKEN}"}
    
    async with websockets.connect(ws_url, extra_headers=headers) as ws:
        await ws.send(json.dumps({"seq": 1, "action": "authentication_challenge", "data": {"token": BOT_TOKEN}}))
        
        async for message in ws:
            event = json.loads(message)
            if event.get("event") == "posted":
                post_data = json.loads(event["data"]["post"])
                message_text = post_data.get("message", "")
                channel_id = post_data.get("channel_id", "")
                
                if post_data.get("props", {}).get("from_bot") == "true":
                    continue
                
                if message_text.startswith("/pi "):
                    cmd = message_text[4:].strip()
                    print(f"[Bridge] Received Pi command: {cmd}")
                    await post_message(channel_id, f"🚀 Triggering Pi task: `{cmd}`")

async def main():
    parser = argparse.ArgumentParser(description="Pi-Intercom Mattermost Bridge")
    parser.add_argument("--channel", default=DEFAULT_CHANNEL, help="Target Mattermost channel")
    parser.add_argument("--message", help="One-shot message to post")
    parser.add_argument("--event-type", default="mesh_event", help="Event type for structured posts")
    parser.add_argument("--daemon", action="store_true", help="Run as persistent WebSocket listener")
    parser.add_argument("--json-data", help="JSON payload for structured event")
    args = parser.parse_args()
    
    if args.message:
        data = json.loads(args.json_data) if args.json_data else {"message": args.message}
        await broadcast_to_channel(args.channel, args.event_type, data)
    elif args.daemon:
        print("[Bridge] Starting daemon mode...")
        await listen_websocket()
    else:
        parser.print_help()

if __name__ == "__main__":
    asyncio.run(main())
