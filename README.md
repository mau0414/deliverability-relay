# Deliverability Relay

An outbound email relay (MTA) built in Go, modeled after how real-world providers operate: clients submit send requests via API; the relay resolves the recipient's MX records and delivers over outbound SMTP directly. The relay is never the final destination for email — it only sends, mirroring the architecture of a real transactional email API.

Built as a learning and portfolio project, with a focus on understanding the real infrastructure problems behind reliable email delivery: SMTP protocol details, DNS-based sender authentication (SPF/DKIM/DMARC), retry/failure classification, caching strategy, and observability.

## Status

The core relay, domain authentication validation, and the metrics dashboard are implemented and working end-to-end. Backoff and connection-timeout hardening are in progress — see [Roadmap](#roadmap).

## Architecture

```text
### Domain registration flow (POST /domains)

  POST /domains
         │
         ▼
   SPF/DKIM/DMARC
   DNS validation
   (skipped if --env-test)
         │
         ▼
   ┌──────────┐        ┌───────────┐
   │PostgreSQL│───────▶│   Redis   │
   │(source of│  write │  (cache)  │
   │  truth)  │        └───────────┘
   └──────────┘


### Email send flow (POST /send)

  POST /send
      │
      ▼
  Validate request
  (required fields, from/to format)
      │
      ▼
┌──────────┐
│PostgreSQL│  (persist, status = queued)
└────┬─────┘
     ▼
┌────────────────────┐
│      Preflight      │   (runs in the HTTP handler, before enqueueing)
│  (sender domain      │
│   verified?)         │
│                      │
│  ┌───────────┐  hit  │
│  │   Redis   │───────┼──▶ verified?
│  │  (cache)  │       │
│  └─────┬─────┘       │
│        │ miss        │
│        ▼             │
│  ┌──────────┐        │
│  │PostgreSQL│──▶ populate Redis
│  └──────────┘        │
│                      │
│  (skipped if --env-test)
└──────────┬───────────┘
           │
    fails  │  passes
  ┌────────┴────────┐
  ▼                 ▼
PostgreSQL        Queue
(status =            │
 failed,             ▼
 HTTP error,     ┌──────────┐
 never queued)   │  Worker  │
                 └────┬─────┘
                      │
              ┌───────┴────────┐
              ▼                ▼
         MX resolve       SMTP delivery
          (DNS)          (recipient server)
                      │
                      ▼
               ┌──────────┐
               │PostgreSQL│  (status update: sent / bounced / failed)
               └──────────┘
```

## Main Features

### Email submission and delivery

- `POST /send` accepts a JSON payload (`from`, `to`, `subject`, `html`), validates it, persists it to PostgreSQL with `queued` status, runs a domain preflight check (see below), and — if the preflight passes — enqueues it for delivery.
- A background worker dequeues emails, resolves the recipient's MX records via DNS, and delivers over SMTP directly to the destination server — no intermediate relay.
- STARTTLS is attempted opportunistically (only if the destination server advertises support), matching how production MTAs behave in practice.
- Delivery outcomes are persisted back to PostgreSQL, updating the email's status through its lifecycle: `queued → sending → sent | bounced | failed`.

### Invalid addresses are never silently accepted

- A malformed `from` address is rejected at request validation time — before the email is even persisted — regardless of whether `--env-test` is active. This check is independent of domain authentication, so bypassing DNS validation with `--env-test` never bypasses basic address-format validation.
- A malformed `to` address is treated as a permanent failure for that recipient (bounced, not retried) — the same way any other unrecoverable delivery error is handled — rather than being silently skipped. An email is never left indefinitely without a final status because of a bad recipient address.

### Domain registration and validation

Domains are registered explicitly via `POST /domains`, separate from sending — a client verifies a domain once, then sends any number of emails from it, rather than re-validating on every send.

On registration, the relay performs DNS checks:

- **SPF**: looks up the domain's TXT records for one starting with `v=spf1`. Missing SPF is recorded as `HasSPF = false`, not an error — the domain can still be registered.
- **DKIM**: looks up `<selector>._domainkey.<domain>` (a fixed selector controlled by this relay — `mta` — avoiding the ambiguity of guessing arbitrary third-party selectors) and checks for a non-empty public key (`p=`). This validates DNS configuration presence only — it does not perform cryptographic signature verification.
- **DMARC**: looks up `_dmarc.<domain>` for a `v=DMARC1` record. DMARC is informational only — it does not block registration or sending, it's just recorded.

The validator distinguishes a domain that **does not exist** (`NXDOMAIN` — registration fails) from a domain that exists but is simply missing one of these records (registration succeeds, the missing record is reflected in the stored flags). A DNS timeout or resolver error is treated as a third, distinct case: the validation couldn't be performed at all, not a definitive "missing" result.

Stored fields: `HasSPF`, `SPFRecord`, `HasDKIM`, `HasDMARC`.

### Redis cache (cache-aside)

Domain validation results are cached in Redis under a key of the form `domain:<name>`, following a cache-aside pattern:

```text
Read path:
  Redis (cache key: domain:<name>)
    │
    ├── hit  ───────────────► return cached result
    └── miss ─► PostgreSQL ─► populate Redis ─► return result
```

- On `POST /domains`, the validation result is written to Redis immediately after being persisted to PostgreSQL.
- PostgreSQL remains the source of truth. If Redis is unavailable at write time, the domain record in PostgreSQL still succeeds — the cache is a performance layer, never a requirement for correctness.

### Delivery preflight

Before an email is enqueued for delivery, the `/send` handler checks the sender's domain against the cache-aside lookup described above. If the domain isn't verified (missing required SPF or DKIM — DMARC is non-blocking), the email is marked `failed` immediately and the request returns an error — the email is never enqueued, and no MX resolution or SMTP connection is ever attempted for it. There's no retry for this case: a domain that isn't authenticated won't become authenticated by trying again a moment later.

### Failure classification and retry

- SMTP errors are classified as permanent (5xx, e.g. invalid recipient) vs. temporary (4xx, network/timeout errors), using the SMTP response code returned by the destination server.
- Permanent failures are marked `bounced` immediately — retrying them would never succeed.
- Temporary failures are requeued for retry, up to a configured maximum number of attempts, after which the email is marked `failed`.

### Status tracking and metrics dashboard

- Every email's lifecycle is persisted and queryable in PostgreSQL — not just fire-and-forget.
- A dashboard (`GET /dashboard`) visualizes delivery rate, bounce rate, volume over time, and status breakdown, rendered server-side and charted with Chart.js.

## Real-world validation

This project was tested against two targets:

- **Mailpit** (local SMTP test server): full flow validated — API → queue → worker → MX resolution (mocked to localhost) → SMTP delivery → visible in Mailpit's UI.
- **Gmail**: the SMTP handshake was completed successfully end-to-end — connection, STARTTLS negotiation, `HELO`, `MAIL FROM`, `RCPT TO`, and `DATA` were all accepted. The final send was rejected with `550 ... not authorized to send email directly`, which is Gmail's IP-reputation policy for unrecognized sending IPs — not a protocol or implementation bug. This is treated as a successful protocol validation: the relay correctly speaks SMTP to a real, hardened mail provider.

## API

### `POST /send`

```json
{
  "from": "you@example.com",
  "to": ["someone@example.com"],
  "subject": "hello",
  "html": "<p>hi</p>"
}
```

- `202 Accepted` — the email passed validation and preflight, and was enqueued for delivery. Delivery happens asynchronously; check status via the dashboard (a `GET /v1/emails/{id}` endpoint is planned — see Roadmap).
- `400 Bad Request` — the request is malformed (missing required fields, or an invalid `from` address format).
- `422 Unprocessable Entity` — the sender domain isn't registered, or is registered but missing required SPF/DKIM authentication.

### `POST /domains`

```json
{
  "name": "example.com"
}
```

Validates SPF, DKIM, and DMARC for the domain and stores the result.

- `201 Created` — domain registered (even if SPF/DKIM/DMARC are missing; the flags reflect what was found).
- `422 Unprocessable Entity` — the domain does not exist (`NXDOMAIN`).
- `500 Internal Server Error` — DNS infrastructure could not be reached (timeout, resolver error).

## Tech Stack

- **Go** — `net/http` with the stdlib router (Go 1.22+), no framework
- **PostgreSQL** — source of truth for email and domain state, via `pgx`
- **Redis** — cache-aside layer for domain validation lookups
- **SMTP / DNS** — outbound delivery, MX resolution, and SPF/DKIM/DMARC lookups via Go's standard library
- **Docker** — for local PostgreSQL, Redis, and Mailpit during development

Main Go concepts used in the project:

- Interfaces for infrastructure boundaries (`Queue`, injectable DNS resolver) — enables swapping implementations (e.g. in-memory queue → Redis-backed queue) without touching callers
- Goroutines and channels (buffered, non-blocking enqueue)
- `context` propagation and cancellation, including deliberately re-deriving contexts from a parent rather than `context.Background()`, so shutdown signals propagate correctly
- Error wrapping and classification (`errors.As`, `*textproto.Error`), and distinguishing a meaningful negative result (e.g. `NXDOMAIN`, missing SPF) from an infrastructure failure (timeout) — the two require different HTTP responses and different handling
- Cache-aside pattern with PostgreSQL as source of truth
- Table-driven unit tests

## Project Structure

```text
cmd/api          → entry point, wires dependencies, starts the HTTP server
internal/api      → HTTP handlers, request/response DTOs, preflight check, dashboard rendering
internal/domain    → core entities (Email, Domain, Status, validation)
internal/dns       → MX resolution, recipient domain parsing, SPF/DKIM/DMARC lookups
internal/smtp      → outbound SMTP client (connection, protocol, delivery, error classification)
internal/queue     → delivery queue (in-memory, channel-based)
internal/workers    → background worker: drains the queue, resolves MX, delivers
internal/repository → PostgreSQL persistence for emails, domains, and metrics
internal/cache     → Redis cache-aside layer for domain validation results
```

## Running Locally

### Requirements

- Go 1.22+
- Docker (for PostgreSQL, Redis, and optionally Mailpit)

### 1. Clone the repository

```bash
git clone https://github.com/mau0414/deliverability-relay.git
cd deliverability-relay
```

### 2. Start PostgreSQL

```bash
docker run -d \
  --name mta-postgres \
  -e POSTGRES_USER=mta \
  -e POSTGRES_PASSWORD=mta_dev_password \
  -e POSTGRES_DB=mta \
  -p 5433:5432 \
  postgres:16
```

### 3. Start Redis

```bash
docker run -d --name mta-redis -p 6379:6379 redis:7
```

Check it's up:

```bash
docker exec -it mta-redis redis-cli ping
```

### 4. (Optional) Start Mailpit, for local SMTP testing without sending real email

```bash
docker run -d --name mailpit -p 1025:1025 -p 8025:8025 axllent/mailpit
```

View delivered test emails at `http://localhost:8025`.

### 5. Run the application

For local testing, run with `--env-test`, which skips SPF/DKIM validation on domain registration and skips the delivery preflight on send:

```bash
go run ./cmd/api --env-test
```

**This is the recommended way to run the project locally.** Exercising real domain authentication requires configuring a DKIM DNS record with this relay's fixed selector on a domain you actually control — realistically only practical if you own a domain, which most people trying out this project won't have on hand. See [Testing real domain authentication](#testing-real-domain-authentication-optional) below if you want to do that anyway.

Without `--env-test`, `go run ./cmd/api` performs full DNS validation on every domain registration and every send.

### 6. Register a domain, then send a test email

```bash
curl -X POST http://localhost:8080/domains \
  -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'

curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{"from":"you@example.com","to":["someone@example.com"],"subject":"test","html":"<p>hi</p>"}'
```

View the dashboard at `http://localhost:8080/dashboard`.

### Testing real domain authentication (optional)

If you want to exercise SPF/DKIM validation for real, rather than bypassing it with `--env-test`, you'll need a domain you control, with:

- An SPF TXT record (`v=spf1 ...`)
- A DKIM TXT record at **`mta._domainkey.<your-domain>`** with a public key (`p=...`) — `mta` is this relay's fixed DKIM selector (see [A note on testing domain authentication](#a-note-on-testing-domain-authentication) for why the selector is fixed rather than discovered)

```bash
go run ./cmd/api

curl -X POST http://localhost:8080/domains \
  -H "Content-Type: application/json" \
  -d '{"name":"your-actual-domain.com"}'

curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{"from":"you@your-actual-domain.com","to":["someone@example.com"],"subject":"test","html":"<p>hi</p>"}'
```

If SPF or DKIM is missing or misconfigured, `/domains` will still return `201 Created` (a domain existing without full authentication is a valid state, not an error — see [Design Decisions](#design-decisions)), but `/send` will fail preflight and the email will be marked `failed` without an SMTP attempt.

### Testing email delivery: where to send test emails

Once an email is queued, you'll want somewhere to actually check whether it arrived. Three options, depending on how much setup you want:

**Option 1: Your own Gmail (or similar) inbox — quick, but likely to fail.** Point `to` at your personal address. This will most likely be rejected with `550 ... not authorized to send email directly` — Gmail's IP-reputation policy for an unrecognized sending IP (your machine), not a bug in the relay. Still a useful test: if the SMTP handshake completes (connection, STARTTLS, `HELO`, `MAIL FROM`, `RCPT TO`, `DATA` all accepted) and only the final acceptance is rejected, that confirms the relay speaks SMTP correctly — see [Real-world validation](#real-world-validation).

**Option 2: Mailpit — reliable, requires local setup.**

```bash
docker run -d --name mailpit -p 1025:1025 -p 8025:8025 axllent/mailpit
```

Run with `--mailpit` to deliver to `localhost:1025` instead of resolving the recipient's real MX record:

```bash
go run ./cmd/api --mailpit
```

View delivered mail at `http://localhost:8025`.

**Option 3: A permissive public mailbox — no local setup.** Disposable inboxes designed to accept mail broadly rather than apply strict sender-reputation checks — the opposite of Gmail, which is why this works better from an unrecognized IP:

- **[Mailinator](https://www.mailinator.com/)** — send to any address `@mailinator.com`, check the same inbox name at mailinator.com. No signup.
- **[Guerrilla Mail](https://www.guerrillamail.com/)** — similar disposable-inbox service, also no signup.

Neither guarantees permanent, unfiltered acceptance, but both are a reasonable default without a domain of your own to check.

## Trying it out: every code path, end to end

With the server running (`--env-test` recommended — see above), these calls exercise every outcome the API can produce, along with how to verify the result at each storage layer.

### 1. Register a domain

```bash
curl -i -X POST http://localhost:8080/domains -H "Content-Type: application/json" \
  -d '{"name":"example.com"}'
```
`201 Created`. Check it landed in Postgres and Redis:

```bash
docker exec -it mta-postgres psql -U mta -d mta \
  -c "SELECT name, has_spf, has_dkim, has_dmarc FROM domains WHERE name = 'example.com';"

docker exec -it mta-redis redis-cli GET "domain:example.com"
```
The Redis value should be a JSON blob matching the Postgres row — written there immediately after the Postgres insert, per the cache-aside flow.

### 2. Register a domain that doesn't exist

```bash
curl -i -X POST http://localhost:8080/domains -H "Content-Type: application/json" \
  -d '{"name":"this-domain-does-not-exist-123456.com"}'
```
`422 Unprocessable Entity` — `NXDOMAIN`, nothing is persisted.

### 3. Send a valid email

```bash
curl -i -X POST http://localhost:8080/send -H "Content-Type: application/json" \
  -d '{"from":"you@example.com","to":["someone@example.com"],"subject":"test","html":"<p>hi</p>"}'
```
`202 Accepted`. Confirm it made it through the full lifecycle:

```bash
docker exec -it mta-postgres psql -U mta -d mta \
  -c "SELECT id, from_address, status, attempts, created_at, updated_at FROM emails ORDER BY created_at DESC LIMIT 1;"
```
Status should progress to `sent` (against Mailpit or a real destination) shortly after — see [Testing email delivery](#testing-email-delivery-where-to-send-test-emails) for where to actually check the message arrived.

### 4. Send with a malformed `from` address

```bash
curl -i -X POST http://localhost:8080/send -H "Content-Type: application/json" \
  -d '{"from":"not-an-email","to":["someone@example.com"],"subject":"test","html":"<p>hi</p>"}'
```
`400 Bad Request` — rejected at validation, before persistence, independent of `--env-test`.

### 5. Send from a domain that was never registered

```bash
curl -i -X POST http://localhost:8080/send -H "Content-Type: application/json" \
  -d '{"from":"you@never-registered.com","to":["someone@example.com"],"subject":"test","html":"<p>hi</p>"}'
```
`422 Unprocessable Entity`. The email is still persisted, with a final status:

```bash
docker exec -it mta-postgres psql -U mta -d mta \
  -c "SELECT from_address, status FROM emails WHERE from_address = 'you@never-registered.com';"
```
Should show `failed` — rejected, but not silently dropped.

### 6. Send from a registered domain missing SPF/DKIM

Only reachable **without** `--env-test`, since the flag treats every registered domain as authenticated regardless of real DNS:

```bash
go run ./cmd/api   # no --env-test

curl -i -X POST http://localhost:8080/domains -H "Content-Type: application/json" -d '{"name":"wikipedia.org"}'
curl -i -X POST http://localhost:8080/send -H "Content-Type: application/json" \
  -d '{"from":"you@wikipedia.org","to":["someone@example.com"],"subject":"test","html":"<p>hi</p>"}'
```
`422 Unprocessable Entity` — the domain exists and was registered successfully, but lacks the SPF/DKIM this relay requires.

## Running Tests

```bash
go test ./... -v
```

Covers recipient address parsing, RFC 5322 message formatting (including CRLF line endings), request validation, and SMTP failure classification (permanent vs. retryable).

## Design Decisions

### Standard library over a framework
Chosen deliberately to demonstrate understanding of Go fundamentals rather than relying on a framework's conventions.

### Opportunistic STARTTLS
The relay attempts STARTTLS if the destination server advertises support, rather than requiring it. This mirrors real-world MTA behavior — some servers (like local test servers) don't support TLS — and is a conscious security/compatibility trade-off, documented rather than hidden.

### Preflight lives in the HTTP handler, not the worker
Domain authentication is checked synchronously in `POST /send`, before the email is enqueued — not later in the background worker. This means a client submitting to an unauthenticated domain gets an immediate, informative error (`422`) instead of a `202` followed by silent, asynchronous failure. The email is still persisted first (with the rejection reflected in its final status), so there's a durable record of every request, accepted or not.

### Missing DNS records are not necessarily errors
A domain can exist without SPF, DKIM, or DMARC configured. Because of this, the validator treats "domain does not exist" as a fundamentally different case from "domain exists but SPF is missing" — the first is a hard failure, the second is a valid (if incomplete) domain state worth recording, not rejecting.

### DNS errors vs. DNS results
The validator also distinguishes an actual DNS result (`NXDOMAIN`, "no SPF record found") from an infrastructure failure (timeout, resolver unreachable). These are handled — and reported to the API caller — differently: one is a definitive answer about the domain, the other means no answer was obtained at all.

### PostgreSQL as the source of truth, Redis as a pure optimization
Redis is deliberately not used as the primary data store for domain validation results. Losing Redis doesn't mean losing domain data — the cache can always be rebuilt from PostgreSQL on the next read. If Redis is unavailable at write time, the PostgreSQL write still succeeds; caching failure never blocks a correctness-critical operation.

### In-memory queue, PostgreSQL for status
The delivery queue is ephemeral by design (a Go channel); losing it on process restart is an accepted limitation for the current stage of the project. Email status, by contrast, is transactional data and belongs in PostgreSQL, not in a disposable queue or cache — this distinction was a deliberate choice about which storage tool fits which kind of data.

### No sender-facing API authentication (yet)
`POST /send` and `POST /domains` are currently open — any caller can submit a send or register a domain. In a production system this relay's own premise (only authenticated clients should be able to send) would require it. It's left out of the current scope by deliberate choice, not oversight, to keep focus on the delivery and DNS-authentication mechanics that were the core learning goal of this project.

### `--env-test`: an explicit, opt-in bypass — never a default

Both domain registration and the delivery preflight depend on real DNS by design — that's the entire point of the authentication layer. But that also means the project can't be exercised end-to-end without a domain the developer actually controls, which is a real onboarding friction for anyone just trying to run the code.

Rather than silently relaxing validation in some "dev mode" inferred from an environment variable, `--env-test` is an explicit command-line flag: it has to be typed, it's visible in the process's invocation, and it can't be enabled by accident through a misconfigured `.env` file left over from testing. The bypassed checks are exactly the same ones that matter most in production (DNS-backed domain authentication) — flagging that clearly, rather than hiding it behind a generic "test mode," was a deliberate choice to keep the risk of accidentally shipping a bypassed check to production as visible as possible.

### A note on testing domain authentication

SPF and DKIM validation exist to solve a real problem — without them, anyone could claim to send email as any domain. That's exactly why they're inconvenient to test: configuring a real DKIM record with this relay's selector (`mta._domainkey.<your-domain>`) requires owning a domain and having DNS access to it, which most people trying out this project won't have on hand.

This is expected, not a limitation specific to this project — any real email-sending API (including the ones this project is modeled after) has the same requirement for production use. `--env-test` exists precisely to let you exercise the rest of the system (queue, worker, SMTP delivery, status tracking, dashboard) without that setup cost, while keeping the validation logic itself fully implemented and independently testable (see `internal/dns`'s test suite).

## Roadmap

- **`GET /v1/emails/{id}`** — query a single email's delivery status by ID.
- **Exponential backoff on retry** — retries are currently immediate; backing off (1s, 2s, 4s...) avoids hammering a struggling destination server.
- **Robust SMTP connection timeout** — `net.DialTimeout` + `smtp.NewClient` instead of the current `smtp.Dial`, which has no built-in timeout.

## Scope and Limitations

This project intentionally stops short of production email infrastructure. Out of scope by design:

- API authentication (see Design Decisions above — explicit, deliberate gap)
- Full DKIM cryptographic signature verification (only DNS configuration presence is checked)
- DMARC enforcement (checked and recorded, but never blocks registration or sending)
- IP reputation / warmup management
- Distributed queue / horizontal worker scaling
- Advanced bounce classification beyond permanent vs. retryable

## What I Learned

- How SMTP-based delivery actually works at the protocol level (STARTTLS negotiation, the full `EHLO`/`MAIL`/`RCPT`/`DATA` sequence, and why a technically correct implementation can still be rejected on reputation grounds — validated first-hand against Gmail)
- How SPF, DKIM, and DMARC are actually structured in DNS, and the real-world subtlety of DKIM selectors (a domain can't be scanned for "any" DKIM configuration without knowing the selector — which is why this relay controls and checks a fixed one, the same approach real providers like Resend use)
- Designing for partial failure at multiple layers: DNS results vs. DNS infrastructure errors, and SMTP permanent vs. retryable errors — and why conflating either pair leads to incorrect behavior (retrying something that will never succeed, or failing something that was actually fine)
- Cache-aside as a pattern: keeping a cache as a pure performance layer, never a source of truth, so its unavailability degrades performance but never correctness
- Separating ephemeral work (a queue) from durable state (delivery and domain status) in storage design
- Go idioms: interfaces as seams for testability, context propagation for graceful cancellation, table-driven tests