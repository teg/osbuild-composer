#!/usr/bin/python3
import urllib.request
import json
import sys
import time
from pprint import pprint


def compose_request(distro, koji):
    req = {
        "distribution": distro,
        "koji": {
            "server": koji
        },
        "image_requests": [{
            "architecture": "x86_64",
            "image_type": "qcow2",
            "repositories": [{
                "baseurl": "http://download.fedoraproject.org/pub/fedora/linux/releases/32/Everything/x86_64/os/"
            }]
        }]
    }

    return req


def main():
    cr = compose_request("fedora-32", "https://localhost/kojihub")
    data = json.dumps(cr)
    print(data)

    req = urllib.request.Request("http://localhost:8701/compose")
    req.add_header('Content-Type', 'application/json')
    raw = data.encode('utf-8')
    req.add_header('Content-Length', len(raw))
    with urllib.request.urlopen(req, raw) as res:
        payload = res.read().decode('utf-8')
        if res.status != 201:
            print("Failed to create compose")
            print(payload)
            sys.exit(1)
    ps = json.loads(payload)
    compose_id = ps["id"]

    req = urllib.request.Request(f"http://localhost:8701/compose/{compose_id}")
    while True:
        with urllib.request.urlopen(req) as res:
            payload = res.read().decode('utf-8')
            if res.status != 200:
                print("Failed to get compose status")
                sys.exit(1)
        ps = json.loads(payload)
        status = ps["status"]
        if status != "RUNNING":
            break
        time.sleep(2)

    if status == "FAILED":
        print("compose failed!")
        sys.exit(1)

    print("Compose worked!")


if __name__ == "__main__":
    main()
