# Enveil

Enveil keeps your environment variables encrypted and out of your filesystem.

Instead of storing secrets in `.env` files that can be accidentally committed, leaked, or read by anyone with filesystem access, Enveil stores everything in an encrypted vault and injects variables directly into processes at runtime — without ever writing them to disk.

## How it works

Enveil uses a SQLCipher-encrypted SQLite vault stored at `~/.enveil/vault.db`. The master key never touches disk — it lives only in memory while the daemon is running, or is derived fresh from your password on each command. Variables are organized by project and environment, making it easy to manage development, staging, and production secrets separately.

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
go install github.com/MaximoCoder/Enveil/cmd/enveil@latest
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

## Project structure
```
~/.enveil/
  vault.db       # SQLCipher encrypted database
  config.json    # Active project and environment (no secrets)
  daemon.sock    # Unix socket (only while daemon is running)
  daemon.pid     # Daemon process ID (only while daemon is running)
```

## Security model

- **Vault encryption**: AES-256 via SQLCipher. The entire database file is encrypted, including table names, project names, and variable names.
- **Key derivation**: Argon2id with 64MB memory, 4 threads. Resistant to GPU and ASIC brute-force attacks.
- **Master key**: Never written to disk. Lives in memory only while the daemon is running.
- **File permissions**: Vault and config files are created with `0600` permissions (owner read/write only).
- **Process injection**: Variables are passed directly to child process environments via `syscall.Exec`, never written to temporary files.
- **Secret scanning**: Pre-commit hook combines pattern matching and Shannon entropy analysis to catch secrets before they reach version control.

## License

MIT
