# Enveil

Enveil keeps your environment variables encrypted and out of your filesystem.

Instead of storing secrets in `.env` files that can be accidentally committed, leaked, or read by anyone with filesystem access, Enveil stores everything in an encrypted vault and injects variables directly into processes at runtime — without ever writing them to disk.

## How it works

Enveil uses a SQLCipher-encrypted SQLite vault stored at `~/.enveil/vault.db`. The master key never touches disk — it lives only in memory while the daemon is running, or is derived fresh from your password on each command. Variables are organized by project and environment, making it easy to manage development, staging, and production secrets separately.

For teams, Enveil includes a self-hosted server that centralizes secrets across developers. Variables are encrypted on the client before being sent to the server — the server never sees plaintext values.

## Installation

### Linux and macOS
```bash
curl -fsSL https://raw.githubusercontent.com/MaximoCoder/Enveil/main/install.sh | sh
```

The installer automatically detects your OS and architecture, downloads the correct binary, verifies its checksum, and installs it to `/usr/local/bin`.

### Windows

Use [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install) and run the Linux installer inside it.

### Install from source

Requires Go 1.22 or later and `libsqlcipher-dev` (Ubuntu/Debian) or `sqlcipher` (macOS).
```bash
go install github.com/MaximoCoder/Enveil/cli/cmd/enveil@latest
```

### Shell integration

Add this to your `~/.zshrc` or `~/.bashrc` to automatically show the active project in your prompt when you navigate to a registered directory:
```bash
eval "$(enveil shell-init)"
```

## Usage

### First time setup
```bash
enveil init
```

Run this once to create your vault and set your master password. Then run it again inside any project directory to register it.

### Saving variables
```bash
enveil set DATABASE_URL=postgres://localhost/mydb
enveil set API_KEY=supersecret123
```

### Importing from an existing .env file
```bash
enveil import .env
enveil import .env.local
```

Imports all variables from the file into the active environment. Optionally deletes the original file after importing.

### Running commands with variables injected
```bash
enveil run npm run dev
enveil run python manage.py runserver
enveil run printenv DATABASE_URL
```

Variables are injected into the process environment. No `.env` file is created or written to disk.

### Getting and listing variables
```bash
enveil list              # show all variable names (values are masked)
enveil get DATABASE_URL  # get the value of a specific variable
```

### Deleting variables
```bash
enveil delete API_KEY
```

Asks for confirmation before deleting.

### Managing environments
```bash
enveil env list              # list all environments
enveil env add staging       # create a new environment
enveil env use staging       # switch active environment
```

### Comparing environments
```bash
enveil diff development staging
```

Shows which variables are missing, extra, or have different values between two environments — without revealing the values.

### Exporting a temporary .env file
```bash
enveil export
```

For tools that require a physical `.env` file. Automatically adds `.env` to `.gitignore`. Delete it when done.

### Managing projects
```bash
enveil projects     # list all registered projects
enveil unregister   # remove the current project from the vault
```

### Git hook
```bash
enveil hook install
```

Installs a pre-commit hook that scans staged files for secrets before every commit. Detects known secret formats (AWS keys, Stripe keys, GitHub tokens, connection strings, private keys) and high-entropy strings using Shannon entropy analysis.

Files like `.env` and `.env.local` are always blocked. Files like `.env.example` and `.env.template` are allowed since they are intended to contain placeholder values.

To bypass the hook when you are sure a file is safe:
```bash
ENVEIL_SKIP=1 git commit
```

To bypass all hooks entirely:
```bash
git commit --no-verify
```

### Daemon

The daemon keeps your master key in memory so you do not have to type your password on every command.
```bash
enveil daemon start    # start daemon, enter password once
enveil daemon status   # check if daemon is running
enveil daemon stop     # stop daemon, key is removed from memory
```

The daemon is optional. Without it, Enveil asks for your password on each command.

## Team server

The Enveil server allows teams to share encrypted secrets across developers without relying on `.env` files, chat messages, or shared drives.

### How it works

The server stores all variables encrypted. Values are encrypted on the client before being sent over the network — the server never sees plaintext values. Even if the server is compromised, secrets remain unreadable without the API key.

### Installing the server

Download the server binary from the [latest release](https://github.com/MaximoCoder/Enveil/releases/latest) for your platform (`enveil-server-linux-amd64`, `enveil-server-darwin-arm64`, etc.) and place it in your PATH.

### Running the server
```bash
ENVEIL_API_KEY=your-secret-key \
ENVEIL_VAULT_PASSWORD=your-vault-password \
ENVEIL_PORT=8080 \
enveil-server
```

The server stores its vault at `~/.enveil-server/vault.db` by default. You can override this with `ENVEIL_VAULT_PATH`.

For production, put the server behind a reverse proxy like nginx with HTTPS enabled.

### Connecting the CLI to the server
```bash
enveil server connect http://your-server:8080 --key your-secret-key
```

Once connected, all CLI commands use the server instead of the local vault. The connection settings are saved in `~/.enveil/config.json`.
```bash
enveil server status      # check connection
enveil server disconnect  # switch back to local vault
```

### Team workflow

The admin sets up the server once and shares the server URL and API key with the team. Each developer runs:
```bash
enveil server connect http://192.168.1.100:8080 --key shared-api-key
enveil init
```

From that point, all `set`, `get`, `list`, `run`, `import`, `export`, `diff`, and `env` commands operate against the shared server. Variables set by one developer are immediately available to all others.

## Project structure
```
~/.enveil/
  vault.db       # SQLCipher encrypted database (local mode)
  config.json    # Active project, environment, and server connection
  daemon.sock    # Unix socket (only while daemon is running)
  daemon.pid     # Daemon process ID (only while daemon is running)

~/.enveil-server/
  vault.db       # Server vault (on the machine running enveil-server)
  salt           # Key derivation salt
```

## Security model

- **Vault encryption**: AES-256 via SQLCipher. The entire database file is encrypted, including table names, project names, and variable names.
- **Key derivation**: Argon2id with 64MB memory, 4 threads. Resistant to GPU and ASIC brute-force attacks.
- **Master key**: Never written to disk. Lives in memory only while the daemon is running.
- **Transport encryption**: Values are encrypted with AES-GCM on the client before being sent to the server. The server stores and returns ciphertext only.
- **File permissions**: Vault and config files are created with `0600` permissions (owner read/write only).
- **Process injection**: Variables are passed directly to child process environments via `syscall.Exec`, never written to temporary files.
- **Secret scanning**: Pre-commit hook combines pattern matching and Shannon entropy analysis to catch secrets before they reach version control.

## License

MIT
