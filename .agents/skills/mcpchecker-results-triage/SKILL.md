---
name: mcpchecker-results-triage
description: >
  Triage the results mcpchecker evaluation, classify the results
  and propose fixes for false positives.
---

# mcpchecker results triage

Analyze the results file (refered in the doc as {mcp_results_json_file}).
If not specified in the request, use  `evals/mcpchecker/mcpchecker-obs-mcp-tools-out.json`.

## Steps

1. run `{skill_dir}/scripts/failed-tasks-with-details.py {mcp_results_json_file}` to list the failed results
2. for each task, classify if it's true positive or false positive
3. for false positives, assess whether the reason is caused by:
  - `environment` - missed assumption in the targeted system
  - `judge` - incorrect or insufficient judge definition
4. PAUSE: present the user a table with the results and wait for confiramtion or clarification.
5. Once results confirmed, propose changes in the task definitions to address failures marked as `judge`.

## Available scripts

- **`scripts/failed-tasks-with-details.py`** - script to prepare the info about failed tasks enriched with data from the task definition.
