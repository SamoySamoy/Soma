# Soma

**A private, self-hosted record of a whole person.**

Soma (Greek for *body*) keeps the records of your life in one place: the people you know, your health, your money and belongings, your diary, your work, and the tasks that keep it all running. Instead of a menu of separate tools, Soma is organised around **you**. You open it and see yourself as a body: your mind in the head, your people in the heart, your work and money in your hands. Click any part to go to that area of your life.

> **Status: early development.** Soma is being built in the open and isn't usable yet. Watch or star the repository to follow progress.

---

## Why Soma

Personal records are scattered across notes apps, spreadsheets, banking apps, health apps and memory. Nothing connects them, and most tools that could want to own your data. Soma is built on a few principles:

- **The person is the interface.** Navigation follows the body, not a list of features.
- **Your data, your machine.** Runs fully offline on your own computer. Everything can be exported in an open, documented format.
- **Private by default.** Nothing is shared unless you share it. Journal, health and finance data get extra protection.
- **Backend first.** Every feature is available through a documented REST API. The web app is just one client.
- **Connected records.** Any record can link to people, tags, files and dates, so your history reads as one life instead of separate lists.

## The body map

```
                 .---.
                ( Mind )  ........  Journal, Library
                ( Self )  ........  Who you are
                 '---'
     Tasks ....  /|♥|\  .......... Heart: People
                / | | \ .......... Body: Health
     Work ....  @ |_| @  ......... Money
                 /   \
     Growth .... |   |
     Journeys .. ^   ^

     [ Home ]            [ Papers ]
     assets              documents
```

Each part of the figure shows whether it needs attention: calm, needs attention, or urgent. A friend you haven't talked to in a while turns the heart amber. A passport about to expire turns the papers red. A second view turns the same figure into a health map: click a knee to see your knee injury and physio visits.

| Body region | Area | What it holds |
|---|---|---|
| Head | Mind | Journal, mood, learning |
| Face | Self | Profile, personality, values, life in weeks |
| Shoulders | Responsibilities | Tasks, reminders, events |
| Heart | Heart | Contacts, relationships, keeping in touch |
| Torso | Body | Health records, medications, metrics |
| Right hand | Work | Jobs, skills, achievements |
| Left hand | Money | Accounts, transactions, budgets, net worth |
| Legs | Growth | Habits and goals |
| Feet | Journeys | Trips and places |
| Beside you | Home, Papers | Belongings and important documents |

## Planned features

The first release (MVP) focuses on daily use on your own machine:

- **Accounts and security:** email and password sign-in, two-factor authentication with recovery codes, session management, brute-force protection, audit log.
- **Self:** profile, personality, values, preferences, and a "life in weeks" grid.
- **People:** contacts, relationships, interaction history, keep-in-touch reminders, birthdays.
- **Journal:** Markdown entries with mood, mentions of people, and full-text search.
- **Tasks and reminders:** projects, recurring tasks, reminders that fire exactly once.
- **Money and assets:** accounts, transactions, transfers, budgets, multiple currencies, an asset register and net worth.
- **Body map and Today:** the figure with live status, plus a daily list of what needs attention.
- **Search, tags, links, attachments, trash.**
- **Export everything** as one encrypted `.soma` file in an open, documented format.

## Roadmap

| Phase | Focus |
|---|---|
| **P1: MVP** | Foundation and daily use, local single-user mode. *(in progress)* |
| **P2** | Health with the anatomical body view, work and skills, documents vault, habits and goals, home maintenance, a graph view of your whole life, import and partial exports, invite-only cloud hosting. |
| **P3** | Shared household spaces, emergency access for trusted contacts, travel, library, passkeys, webhooks, and optional AI features built on your own data, which you can run locally. |

## Tech stack

| Layer | Technology |
|---|---|
| Backend | Go: a modular monolith on `net/http`, an OpenAPI-first contract, background jobs with River |
| Database | PostgreSQL 18, with row-level security for isolation between users |
| Frontend | React, TypeScript, Vite, Mantine, built into the Go binary |
| Security | Argon2id passwords, server-side sessions, TOTP two-factor, envelope encryption with Tink |
| Deployment | One binary plus PostgreSQL; Docker Compose for local and small servers |

Some design choices worth knowing:

- **One database.** Soma's data is connected like a graph, but it lives in PostgreSQL as graph-shaped tables. Every record is a node and every connection is a typed edge. This keeps money and health data under strict transactions and constraints, with one source of truth.
- **Two layers of access control.** Permissions are checked in the application and enforced again by PostgreSQL row-level security, so one missed check can't leak another user's data.
- **No telemetry.** Soma never calls home.

## Getting started

Not available yet. Install instructions will appear here once the first milestone runs end to end. The plan is:

```bash
git clone https://github.com/<you>/soma.git
cd soma
docker compose up
# open http://localhost:8080 and create your account
```

### Development prerequisites

- Go 1.27
- Node.js 24 LTS
- Docker

All other development tools are pinned in `go.mod` and run through `go tool`, so there is nothing else to install. Common commands, once the project skeleton lands:

```bash
go tool task dev       # run the database, API server and web app with live reload
go tool task check     # generate code, lint, and run all tests
```

## Contributing

Soma is in early development and the structure is still settling. Issues with ideas and feedback are welcome. Pull requests will be easier to accept once the first milestones are in place. Project conventions for code style, tests and commits are in [CLAUDE.md](CLAUDE.md).

## License

Soma is released under the [MIT License](LICENSE).
