#!/usr/bin/env python3
"""
Orion MCP Bridge Server
Exposes Orion Control Plane capabilities as MCP tools for Pi.
Run: python3 orion_mcp_bridge.py
"""
import asyncio
import json
import os
import httpx
from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse

app = FastAPI(title="Orion MCP Bridge", version="0.1.0")

ORION_BASE = os.environ.get("ORION_BASE", "http://localhost:8000")
HTTP_CLIENT = httpx.AsyncClient(timeout=30.0)

MCP_SERVER_INFO = {
    "name": "orion-mcp-bridge",
    "version": "0.1.0",
    "description": "LoveLogicAI Orion Control Plane exposed via MCP",
    "tools": [
        {
            "name": "orion/route_inference",
            "description": "Route a prompt through the optimal free provider",
            "parameters": {
                "type": "object",
                "properties": {
                    "prompt": {"type": "string"},
                    "capability_hints": {"type": "array", "items": {"type": "string"}},
                    "latency_class": {"type": "string", "enum": ["realtime", "interactive", "background"]},
                    "cost_policy": {"type": "string", "enum": ["free_only", "free_preferred", "paid_allowed"]}
                },
                "required": ["prompt"]
            }
        },
        {
            "name": "orion/list_models",
            "description": "List available models across all registered providers",
            "parameters": {"type": "object", "properties": {}}
        },
        {
            "name": "orion/get_topology",
            "description": "Get current mesh agent topology with capabilities",
            "parameters": {"type": "object", "properties": {}}
        },
        {
            "name": "orion/execute_task",
            "description": "Delegate a task to a specific mesh node",
            "parameters": {
                "type": "object",
                "properties": {
                    "target_agent": {"type": "string"},
                    "task_type": {"type": "string"},
                    "payload": {"type": "object"}
                },
                "required": ["target_agent", "task_type", "payload"]
            }
        },
        {
            "name": "orion/get_task_state",
            "description": "Query A2A task state by ID",
            "parameters": {
                "type": "object",
                "properties": {"task_id": {"type": "string"}},
                "required": ["task_id"]
            }
        },
        {
            "name": "orion/web_search",
            "description": "Free-first web search via Ollama pi-web-search",
            "parameters": {
                "type": "object",
                "properties": {"query": {"type": "string"}},
                "required": ["query"]
            }
        },
        {
            "name": "orion/research",
            "description": "Trigger Feynman literature review",
            "parameters": {
                "type": "object",
                "properties": {
                    "topic": {"type": "string"},
                    "max_sources": {"type": "integer", "default": 10}
                },
                "required": ["topic"]
            }
        }
    ]
}

@app.get("/mcp/info")
async def mcp_info():
    return MCP_SERVER_INFO

@app.post("/mcp/tools/{tool_name}")
async def mcp_tool(tool_name: str, request: Request):
    body = await request.json()
    
    if tool_name == "orion/route_inference":
        prompt = body.get("prompt", "")
        hints = body.get("capability_hints", [])
        latency = body.get("latency_class", "interactive")
        cost = body.get("cost_policy", "free_preferred")
        try:
            r = await HTTP_CLIENT.post(
                f"{ORION_BASE}/v1/route",
                json={"prompt": prompt, "hints": hints, "latency_class": latency, "cost_policy": cost}
            )
            return JSONResponse({"result": r.json()})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    elif tool_name == "orion/list_models":
        try:
            r = await HTTP_CLIENT.get(f"{ORION_BASE}/v1/models")
            return JSONResponse({"result": r.json()})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    elif tool_name == "orion/get_topology":
        try:
            r = await HTTP_CLIENT.get(f"{ORION_BASE}/topology")
            return JSONResponse({"result": r.json()})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    elif tool_name == "orion/execute_task":
        try:
            r = await HTTP_CLIENT.post(f"{ORION_BASE}/v1/tasks", json=body)
            return JSONResponse({"result": r.json()})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    elif tool_name == "orion/get_task_state":
        task_id = body.get("task_id", "")
        try:
            r = await HTTP_CLIENT.get(f"{ORION_BASE}/v1/tasks/{task_id}")
            return JSONResponse({"result": r.json()})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    elif tool_name == "orion/web_search":
        query = body.get("query", "")
        try:
            r = await HTTP_CLIENT.post(
                "http://localhost:11434/api/experimental/web_search",
                json={"query": query}
            )
            return JSONResponse({"result": r.json()})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    elif tool_name == "orion/research":
        topic = body.get("topic", "")
        max_sources = body.get("max_sources", 10)
        try:
            import subprocess
            result = subprocess.run(
                ["feynman", "research", topic, "--max-sources", str(max_sources), "--output", "json"],
                capture_output=True, text=True, timeout=300
            )
            return JSONResponse({"result": json.loads(result.stdout) if result.stdout else {"stdout": result.stdout, "stderr": result.stderr}})
        except Exception as e:
            return JSONResponse({"error": str(e)}, status_code=500)
    
    return JSONResponse({"error": f"Unknown tool: {tool_name}"}, status_code=404)

@app.on_event("shutdown")
async def shutdown():
    await HTTP_CLIENT.aclose()

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8766)
