# Timeless

AI-powered Sponsorship Intelligence & Operations Platform. Multi-tenant SaaS that automates sponsor research, qualification, outreach, proposal generation, and pipeline management.

## Architecture

```
├── backend/          Go (Fiber v3 + GORM + PostgreSQL + Redis + Asynq)
├── frontend/         Next.js 14 (App Router + TanStack Query + Tailwind CSS v4)
├── docker/           Dockerfiles for production builds
└── .github/          CI/CD workflows
```

**Backend stack:** Go, Fiber v3, GORM, PostgreSQL (pgvector), Redis, Asynq workers  
**Frontend stack:** Next.js App Router, TypeScript, Tailwind CSS v4, shadcn/ui, TanStack Query, Zustand  
**AI layer:** Multi-provider (OpenAI, Anthropic, Gemini, Groq, OpenRouter) with 11 specialized agents  

## Quick Start

```bash
# 1. Clone and copy env
cp .env.example .env
cp frontend/.env.example frontend/.env.local

# 2. Start infrastructure
make dev-infra

# 3. Start backend + frontend
make dev

# 4. Or start everything via Docker
docker compose up
```

The app runs at:
- Frontend: http://localhost:3000
- Backend API: http://localhost:8080
- MinIO Console: http://localhost:9001

## Key Features

- **Pipeline Management** — Kanban board with customizable stages per campaign
- **AI Proposal Generation** — One-click AI-written sponsorship proposals
- **Multi-Agent System** — Research, qualification, outreach drafting, company intel, decision-maker finding
- **Outreach Sequences** — Multi-step email/call sequences with enrollment tracking
- **Automations** — Trigger-based workflows (e.g., auto-qualify on create, stage change notifications)
- **Webhooks** — Event-driven integrations with retry logic and delivery tracking
- **Integrations** — Zapier (primary gateway), Notion, and Apollo, with real OAuth/API-key connect, incremental sync, auto-retry, and duplicate merging — see [backend/docs/INTEGRATIONS.md](backend/docs/INTEGRATIONS.md)
- **Real-time** — WebSocket + SSE for live pipeline updates
- **Analytics** — KPI dashboard, pipeline funnel, activity timeline
- **Multi-tenant** — Organization-scoped data isolation, RBAC-ready

## API Endpoints

All tenant-scoped routes require JWT auth and are prefixed with `/api/v1`.

| Resource | Routes |
|----------|--------|
| Auth | `POST /auth/register, /login, /refresh, /logout` |
| Campaigns | CRUD `/campaigns` |
| Sponsors | CRUD `/sponsors` + `PATCH /:id/stage` |
| Companies | CRUD `/companies` |
| Contacts | CRUD `/contacts` |
| Proposals | CRUD `/proposals` + `POST /proposals/generate` |
| Sequences | CRUD `/sequences` + `POST /:id/enroll` |
| Communications | CRUD `/communications` + `GET /stats` |
| Activities | `GET /activities`, `POST /activities` |
| Automations | CRUD `/automations` + `PATCH /:id/toggle` |
| Integrations | CRUD `/integrations` |
| Webhooks | CRUD `/webhooks` + rotate-secret, test, deliveries |
| Analytics | `GET /analytics/dashboard, /pipeline, /activity` |
| AI | `POST /ai/query`, `GET /ai/agents` |
| WebSocket | `GET /ws` |
| SSE | `GET /events` |

## Development

```bash
make help              # List all commands
make dev               # Start full dev stack
make dev-worker        # Start background worker
make test              # Run all tests
make lint              # Run linters
make build-backend     # Build Go binaries
make build-frontend    # Build Next.js
```

## Environment Variables

See `.env.example` for all configuration options. Required:
- `DATABASE_URL` — PostgreSQL connection string
- `JWT_SECRET` — Secret for JWT signing
- At least one AI provider key (`OPENAI_API_KEY`, `OPENROUTER_API_KEY`, etc.)
