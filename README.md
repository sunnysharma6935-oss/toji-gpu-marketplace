# GPU Marketplace — Phase 1

Day 1-3 milestone: **you can create a user and register a "host" record via API.**
Nothing else is built yet on purpose — see the 30-day plan for what comes next.

## Run it

```bash
# 1. Start Postgres
docker compose up -d

# 2. Apply the schema
docker exec -i $(docker compose ps -q postgres) \
  psql -U gpumarket -d gpumarket < backend/migrations/001_init.sql

# 3. Get Go deps and run the server
cd backend
go mod tidy
go run ./cmd/server
```

Server listens on `:8080`.

## Test the Day 1-3 milestone

```bash
# Health check
curl localhost:8080/health
# -> ok

# Sign up
curl -X POST localhost:8080/signup \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"testpass123"}'
# -> {"user_id":"...","token":"..."}

# Save the token from above, then register your first host (PC1)
TOKEN="paste-token-here"
curl -X POST localhost:8080/hosts/register \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"machine_label":"PC1"}'
# -> {"host_id":"...","agent_token":"..."}
```

**Done means:** that last call returns a `host_id` and `agent_token`, and you can see
the row in Postgres:

```bash
docker exec -it $(docker compose ps -q postgres) \
  psql -U gpumarket -d gpumarket -c "SELECT id, machine_label, status FROM hosts;"
```

Save the `agent_token` — that's what the Host Agent (Day 4-7, next) will use to
authenticate itself when it registers your actual RTX 5060 GPUs.

## What's next (Day 4-7)
Build the Go host agent: detect GPUs via `nvidia-smi`, POST them to a new
`/agent/gpu-report` endpoint using the `agent_token`, and start sending heartbeats
to `/agent/heartbeat` so the host's `status` flips to `online`.
