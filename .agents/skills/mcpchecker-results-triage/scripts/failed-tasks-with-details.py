#!/usr/bin/env python
# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "pyyaml",
# ]
# ///

# Reads an mcpchecker results JSON file, filters for failed tasks, and outputs
# a combined JSON enriched with each task's prompt, verify spec, output, error,
# and judge reason. Accepts an optional path to the results file as the first
# argument; defaults to the standard mcpchecker output location.
import json
import sys
import yaml

input_file = (
    sys.argv[1]
    if len(sys.argv) > 1
    else "evals/mcpchecker/mcpchecker-obs-mcp-tools-out.json"
)

with open(input_file) as f:
    data = json.load(f)

failed = [r for r in data["results"] if not r["taskPassed"]]

results = []
for r in failed:
    with open(r["taskPath"]) as f:
        task_spec = yaml.safe_load(f)
    spec = task_spec.get("spec", {})
    exclude = {"agentOutput", "callHistory", "judgeTokenUsage", "setupOutput"}
    results.append(
        {
            **{k: v for k, v in r.items() if k not in exclude},
            "prompt": spec.get("prompt", {}),
            "verify": spec.get("verify", {}),
        }
    )

print(json.dumps(results, indent=2))
