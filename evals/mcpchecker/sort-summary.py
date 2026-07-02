#!/usr/bin/env python3
"""Sort mcpchecker summary task blocks alphabetically, keeping details attached."""
import sys
import re

TASK_RE = re.compile(r"^  [✓✗~] ")

lines = sys.stdin.read().splitlines()

header, tasks, footer = [], [], []
state = "header"

for line in lines:
    if state == "header":
        if TASK_RE.match(line):
            state = "tasks"
            tasks.append(line)
        else:
            header.append(line)
    elif state == "tasks":
        if TASK_RE.match(line):
            tasks.append(line)
        elif line.startswith("   "):
            tasks[-1] += "\n" + line
        else:
            state = "footer"
            footer.append(line)
    else:
        footer.append(line)

tasks.sort(key=lambda t: re.match(r"^  [✓✗~] (\S+)", t).group(1))

print("\n".join(header + tasks + footer))
