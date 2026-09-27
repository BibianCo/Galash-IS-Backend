#!/usr/bin/env python3
"""Black-box HTTP contract checks for the Auth service.

The checks use only the published HTTP contract and standard-library modules.
Set AUTH_BASE_URL when the service is running; no application package is
imported.
"""

import json
import os
import sys
import urllib.error
import urllib.request


BASE_URL = os.environ.get("AUTH_BASE_URL", "http://localhost:8080").rstrip("/")


def request(method, path, body=None, headers=None):
    data = None if body is None else body.encode()
    request = urllib.request.Request(
        BASE_URL + path, data=data, headers=headers or {}, method=method
    )
    try:
        with urllib.request.urlopen(request, timeout=5) as response:
            return response.status, response.read().decode()
    except urllib.error.HTTPError as error:
        return error.code, error.read().decode()


def assert_contract(name, actual, expected_status, expected_body_key=None):
    status, body = actual
    if status != expected_status:
        raise AssertionError(f"{name}: expected HTTP {expected_status}, got {status}: {body}")
    if expected_body_key is not None:
        payload = json.loads(body)
        if expected_body_key not in payload:
            raise AssertionError(f"{name}: response lacks {expected_body_key!r}: {body}")


def main():
    checks = [
        (
            "health contract",
            request("GET", "/health"),
            200,
            "status",
        ),
        (
            "registration rejects missing bearer token",
            request("POST", "/auth/register", "{}", {"Content-Type": "application/json"}),
            401,
            "error",
        ),
        (
            "unknown route is not accepted",
            request("GET", "/not-a-public-route"),
            404,
            None,
        ),
    ]
    for name, actual, expected_status, expected_key in checks:
        assert_contract(name, actual, expected_status, expected_key)
        print(f"PASS {name}")


if __name__ == "__main__":
    try:
        main()
    except (AssertionError, OSError, urllib.error.URLError, json.JSONDecodeError) as error:
        print(f"FAIL {error}", file=sys.stderr)
        sys.exit(1)
