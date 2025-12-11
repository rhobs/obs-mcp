# obs-mcp evals

The evaluations testset for the obs-mcp.

## Quickstart


Prerequisites: [uv](https://docs.astral.sh/uv/)

1. Setup the venv
``` sh
uv sync
```

2. Setup the environment variables
``` sh
export OPENAI_API_KEY="...."
```

3. Run the lightspeed-service with obs-mcp connected (expected to listen on localhost:8080)

4. Run the evaluations

``` sh
# Deleting the .caches to avoid using old data: might be helpful to keep when
# tweaking the evaluation criteria.
rm -rf .caches; uv run lightspeed-eval --system-config system.yaml --eval-data evals.yaml
```

