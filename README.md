# Term Chess

Term Chess is an SSH-native terminal chess server built in Go. Players connect over SSH, land directly inside a Bubble Tea interface, create time-controlled multiplayer games, or play untimed bot matches backed by [`elo-based-rec`](https://github.com/Ashutoshbind15/elo-based-rec).



https://github.com/user-attachments/assets/fa54412a-5fb7-4ed3-9b15-69c7f5b00817



## Features

- SSH-hosted terminal UI built with `Wish` + `Bubble Tea`
- Fingerprint-based player identity with auto-assigned pet names
- Multiplayer chess with 1, 3, and 5 minute time controls
- Live clock updates and time-forfeit handling
- Bot games against [`elo-based-rec`](https://github.com/Ashutoshbind15/elo-based-rec)
- Postgres-backed player and game history storage

## Legal

Term Chess is operated by Bind Ashutosh Parasnath as an individual side project. See [LEGAL.md](./LEGAL.md) or the canonical site pages:

- [Terms of Service](https://termchess.ashutoshbind.com/terms)
- [Privacy Policy](https://termchess.ashutoshbind.com/privacy)

## Architecture

### High-Level View

```text
SSH client
   |
   v
Wish SSH server
   |
   v
Bubble Tea program per session
   |
   +--> UI model / page routing
   |
   +--> SessionManager   -> active Tea programs by fingerprint
   +--> GameManager      -> live multiplayer state in memory
   +--> ClockManager     -> periodic clock ticks + time forfeits
   +--> BotGameManager   -> live bot games in memory
   +--> DataManager      -> Postgres via GORM
   +--> BotAPIManager    -> external HTTP chess engine service
```

### Runtime Design

The system is structured around one `Bubble Tea` model per SSH session. Each connected user gets an isolated UI state machine, while shared managers coordinate cross-session behavior such as multiplayer games and live clocks.

That split is the core architectural idea in this project:

- Session-local state lives inside the Bubble Tea model: active page, selected squares, inputs, notices, table contents, and current view state.
- Shared live domain state lives in manager types: active players, in-progress games, bot games, and session program references.
- Durable state lives in Postgres: player profiles, completed multiplayer games, and completed bot games.

This creates a clean boundary between presentation state, process-local domain state, and persisted history.

### Main Components

#### `main` package

The `main` package owns transport and UI orchestration: it starts the SSH server, creates the shared managers, creates one Bubble Tea program per session, and routes messages into page-specific update functions.

#### `SessionManager`

`SessionManager` maps player fingerprints to active Bubble Tea programs so the server can push opponent, clock, and forfeit updates into the right sessions.

#### `GameManager`

`GameManager` owns human-vs-human games: player registration, game creation and joining, move validation through `notnil/chess`, turn enforcement, clock state, and building persistence records when a game ends.

#### `ClockManager`

`ClockManager` runs a periodic tick loop, pushes live time updates into player sessions, detects expired clocks, and finalizes time-forfeit results.

#### `BotGameManager`

`BotGameManager` owns the human-vs-bot flow separately from multiplayer, which keeps bot-specific rules such as single-player state, no clocks, and engine-driven moves out of the main game path.

#### `DataManager`

`DataManager` is the persistence boundary. It connects to Postgres through GORM and handles player profiles plus completed multiplayer and bot game history.

#### `BotAPIManager`

`BotAPIManager` is a thin HTTP client for the external [`elo-based-rec`](https://github.com/Ashutoshbind15/elo-based-rec) service. It sends `FEN + level`, receives a UCI move, and keeps engine concerns outside the main application process.

### Request and State Flow

#### Human vs human

1. A user connects over SSH and gets a Bubble Tea session.
2. The server identifies the player by SSH public key fingerprint.
3. The player profile is loaded from Postgres or auto-created with a deterministic pet name on first use.
4. The player creates or joins a live in-memory game.
5. Moves are validated through `notnil/chess`.
6. Opponent sessions receive update messages through `SessionManager`.
7. `ClockManager` emits clock updates while the game is active.
8. When the game ends, a record is persisted and the in-memory game is removed.

#### Human vs bot

1. A user starts a bot game with a selected color and difficulty.
2. `BotGameManager` creates an in-memory game.
3. Player moves are applied locally.
4. The current FEN is sent to the bot API.
5. The returned move is validated and applied.
6. Finished games are persisted and removed from live memory.

## Tech Stack

- Go
- `Wish` for SSH app hosting
- `Bubble Tea`, `Bubbles`, and `Lip Gloss` for the terminal UI
- `notnil/chess` for chess rules and move validation
- Postgres + GORM for persistence
- [`elo-based-rec`](https://github.com/Ashutoshbind15/elo-based-rec) for engine-backed bot play
- Docker / Docker Compose for local infrastructure

## Running Locally

### Prerequisites

- Go `1.25+`
- Docker and Docker Compose
- The [`elo-based-rec`](https://github.com/Ashutoshbind15/elo-based-rec) service available at `BOT_API_URL` for bot games

### Option 1: Run with Docker Compose

1. Start the stack:

   ```bash
   docker compose up --build
   ```

2. Connect to the app:

   ```bash
   ssh localhost -p 23234
   ```

This starts:

- Postgres on `localhost:5432`
- the SSH chess server on `localhost:23234`

### Option 2: Run the app directly

1. Create an env file from the example and provide a working Postgres instance.
2. Start Postgres separately.
3. Run the server:

   ```bash
   go run .
   ```

4. Connect over SSH:

   ```bash
   ssh localhost -p 23234
   ```

### Environment Variables

Example values are provided in `.env.example`.

- `TERN_CHESS_ENV`
  - `development` binds SSH to `127.0.0.1`
  - `production` binds SSH to `0.0.0.0`
- `DB_URL`
  - Postgres connection string
- `BOT_API_URL`
  - base URL for the [`elo-based-rec`](https://github.com/Ashutoshbind15/elo-based-rec) service (must expose `GET /health` before the bot page loads)
- `DEBUG`
  - when set, message traffic is dumped to `messages.log`

## Repository Structure

```text
.
├── main.go              # SSH server bootstrap and top-level Bubble Tea model
├── intro.go             # profile loading and historical game list
├── game.go              # multiplayer game UI and board interaction
├── bot.go               # bot game UI and engine-driven play
├── menu.go              # page selection UI
├── session.go           # active session registry
├── managers/
│   ├── game.go          # multiplayer domain logic
│   ├── clock.go         # live clock updates and forfeits
│   ├── bot_game.go      # bot game domain logic
│   ├── bot_api.go       # external engine client
│   └── data.go          # Postgres persistence
├── common/
│   └── datamodel.go     # persistent models
├── compose.yaml         # local stack
└── Dockerfile           # container build
```
