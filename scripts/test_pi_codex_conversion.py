#!/usr/bin/env python3
"""
Validate @howaboua/pi-codex-conversion bridge.
Tests round-trip context preservation between Codex and Pi.
"""
import json
import os
import subprocess
import tempfile

def test_roundtrip():
    codex_artifact = {
        "version": "1.0",
        "session_id": "test-session-001",
        "project": "orion-control-plane",
        "messages": [
            {"role": "user", "content": "Build a FastAPI gateway for multi-provider inference"},
            {"role": "assistant", "content": "I'll build Orion Control Plane with provider routing..."}
        ],
        "files_modified": ["src/main.py", "src/router.py"],
        "context": {
            "framework": "FastAPI",
            "providers": ["groq", "gemini", "together", "ollama"]
        }
    }
    
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(codex_artifact, f)
        codex_path = f.name
    
    pi_context_path = codex_path.replace('.json', '_pi.json')
    codex_roundtrip_path = codex_path.replace('.json', '_rt.json')
    
    try:
        result = subprocess.run(
            ["node", "/opt/homebrew/lib/node_modules/@howaboua/pi-codex-conversion/dist/convert.js",
             "--input", codex_path, "--output", pi_context_path, "--to", "pi"],
            capture_output=True, text=True, timeout=30
        )
        if result.returncode != 0:
            print(f"Codex -> Pi conversion failed: {result.stderr}")
            return False
        
        with open(pi_context_path) as f:
            pi_context = json.load(f)
        assert "messages" in pi_context
        assert pi_context["messages"][0]["role"] == "user"
        print("Codex -> Pi conversion: PASSED")
        
        result = subprocess.run(
            ["node", "/opt/homebrew/lib/node_modules/@howaboua/pi-codex-conversion/dist/convert.js",
             "--input", pi_context_path, "--output", codex_roundtrip_path, "--to", "codex"],
            capture_output=True, text=True, timeout=30
        )
        if result.returncode != 0:
            print(f"Pi -> Codex conversion failed: {result.stderr}")
            return False
        
        with open(codex_roundtrip_path) as f:
            roundtrip = json.load(f)
        
        assert roundtrip["session_id"] == codex_artifact["session_id"]
        assert len(roundtrip["messages"]) == len(codex_artifact["messages"])
        print("Pi -> Codex roundtrip: PASSED")
        
        assert roundtrip["context"]["framework"] == "FastAPI"
        assert "ollama" in roundtrip["context"]["providers"]
        print("Context preservation: PASSED")
        
        print("\nAll pi-codex-conversion tests PASSED")
        return True
        
    finally:
        for p in [codex_path, pi_context_path, codex_roundtrip_path]:
            if os.path.exists(p):
                os.unlink(p)

if __name__ == "__main__":
    import sys
    ok = test_roundtrip()
    sys.exit(0 if ok else 1)
