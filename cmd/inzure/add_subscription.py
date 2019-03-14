#!/usr/bin/env python3

import sys
import json
import subprocess

try:
    out = subprocess.check_output(
        ["az", "account", "list"]
    )
except OSError:
    print("The Azure CLI does not appear to be installed")
    sys.exit(1)

subscriptions = [
    "/subscriptions/{}".format(e["id"]) for e in json.loads(out)
    if e["state"] == "Enabled"
]

role = None
try:
    with open("role.json", "r", encoding="utf-8") as f:
        role = json.load(f)
except IOError:
    print("This needs to be run in the same directory as role.json file")
    sys.exit(1)
except subprocess.CalledProcessError:
    print("Couldn't get subscriptions")
    sys.exit(1)

role["AssignableScopes"] = subscriptions

with open("role.json", "w") as f:
    json.dump(role, f, sort_keys=True, indent=3)
