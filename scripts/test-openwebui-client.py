"""Test the v0.12.3 MCP client against a locally built server and fake accounts.

Requires Python 3.12, mcp==1.27.2 and httpx==0.28.1 (Open WebUI's pins).
Downloads the tagged client at runtime; does not install or run the WebUI app.
Only environment settings and JSON codec imports are supplied by this harness.
No production credentials or mail server connections are used.
"""

import argparse
import asyncio
import hashlib
import importlib.metadata
import json
import os
from pathlib import Path
import secrets
import socket
import subprocess
import sys
import tempfile
import time
import types
from urllib.request import urlopen

import httpx

VERSION = "v0.12.3"
SOURCE = (
    "https://raw.githubusercontent.com/open-webui/open-webui/"
    f"{VERSION}/backend/open_webui/utils/mcp/client.py"
)


def load_client():
    assert importlib.metadata.version("mcp") == "1.27.2"
    assert importlib.metadata.version("httpx") == "0.28.1"
    with urlopen(SOURCE, timeout=30) as response:
        source = response.read()
    assert hashlib.sha256(source).hexdigest() == "3969ca30bd5a4a3a674a6582d5120790efe0824d5fbdcad1ee71c273aa3ec5eb"
    env = types.ModuleType("open_webui.env")
    env.AIOHTTP_CLIENT_SESSION_TOOL_SERVER_SSL = True
    env.AIOHTTP_CLIENT_TIMEOUT_TOOL_SERVER = 10
    env.MCP_INITIALIZE_TIMEOUT = 5
    codec = types.ModuleType("open_webui.utils.json_codec")
    codec.JSONCodec = json
    sys.modules[env.__name__] = env
    sys.modules[codec.__name__] = codec
    module = types.ModuleType("openwebui_pinned_mcp_client")
    exec(compile(source, SOURCE, "exec"), module.__dict__)
    print(f"Open WebUI {VERSION} client sha256={hashlib.sha256(source).hexdigest()}")
    return module.MCPClient


async def exercise(url, token, client_type):
    headers = {"Authorization": f"Bearer {token}"}
    async with httpx.AsyncClient(timeout=10) as raw:
        assert (await raw.get(url)).status_code == 401
        assert (await raw.get(url, headers=headers)).status_code == 405
        assert (await raw.delete(url, headers=headers)).status_code == 200
        assert (await raw.post(url, headers={"Authorization": "Bearer wrong"}, json={})).status_code == 401
    client = client_type()
    for _ in range(2):
        try:
            await client.connect(url, headers=headers)
            tools = await client.list_tool_specs()
            assert len(tools) == 22, "tool pagination lost entries"
            result = await client.call_tool("email_accounts", {})
            assert "fake" in json.dumps(result)
            resources = await client.list_resources()
            assert any(str(item["uri"]).rstrip("/") == "email://status" for item in resources), resources
            status = await client.read_resource("email://status")
            assert "fake" in json.dumps(status, default=str)
        finally:
            await client.disconnect()
    print("PASS: initialize, paginated tools, call, resources, GET 405, DELETE, authentication, reconnect")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--server-binary", required=True)
    args = parser.parse_args()
    client_type = load_client()
    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        port = reservation.getsockname()[1]
    with tempfile.TemporaryDirectory() as directory:
        token = secrets.token_urlsafe(32)
        token_path = Path(directory) / "token"
        token_path.write_text(token)
        token_path.chmod(0o600)
        environment = {key: value for key, value in os.environ.items() if not key.startswith("EMAIL_")}
        environment.update(
            EMAIL_TRANSPORT="http",
            EMAIL_HTTP_ADDR=f"127.0.0.1:{port}",
            EMAIL_HTTP_ALLOWED_HOSTS=f"127.0.0.1:{port}",
            EMAIL_HTTP_TOKEN_FILE=str(token_path),
            EMAIL_ACCOUNTS=json.dumps([{
                "id": "fake", "email": "fake@example.com",
                "imap_host": "invalid.example", "smtp_host": "invalid.example",
            }]),
        )
        with tempfile.TemporaryFile() as stdout, tempfile.TemporaryFile() as stderr:
            process = subprocess.Popen(
                [args.server_binary], env=environment, stdin=subprocess.DEVNULL, stdout=stdout, stderr=stderr,
            )
            url = f"http://127.0.0.1:{port}"
            try:
                deadline = time.monotonic() + 10
                while True:
                    if process.poll() is not None:
                        raise AssertionError("server exited before readiness")
                    try:
                        if httpx.get(url + "/healthz", timeout=1).status_code == 200:
                            break
                    except httpx.TransportError:
                        pass
                    if time.monotonic() > deadline:
                        raise AssertionError("server failed to become ready")
                    time.sleep(0.05)
                asyncio.run(exercise(url + "/mcp", token, client_type))
            finally:
                process.terminate()
                try:
                    process.wait(timeout=7)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
                    raise AssertionError("shutdown exceeded deadline") from None
            assert process.returncode == 0, "signal shutdown failed"
            stdout.seek(0)
            stderr.seek(0)
            assert stdout.read() == b"", "HTTP emitted protocol bytes to stdout"
            logs = stderr.read()
            assert token.encode() not in logs, "token leaked to logs"
            assert b"fake@example.com" not in logs, "account email leaked to logs"
            print("PASS: process shutdown, protocol-safe stdout, log canaries")


if __name__ == "__main__":
    main()
