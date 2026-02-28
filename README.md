# Enveil

Enveil keeps your environment variables encrypted and out of your filesystem.

Instead of storing secrets in `.env` files that can be accidentally committed, leaked, or read by anyone with filesystem access, Enveil stores everything in an encrypted vault and injects variables directly into processes at runtime — without ever writing them to disk.

## How it works

Enveil uses a SQLCipher-encrypted SQLite vault stored at `~/.enveil/vault.db`. The master key never touches disk — it lives only in memory while the daemon is running, or is derived fresh from your password on each command. Variables are organized by project and environment, making it easy to manage development, staging, and production secrets separately.

## Installation

### Requirements

- Go 1.22 or later
- `libsqlcipher-dev` (Ubuntu/Debian) or `sqlcipher` (macOS)

### Ubuntu / Debian
```bash
sudo apt-get install libsqlcipher-dev
go install github.com/maximodev/enveil/cmd/enveil@latest
```

### macOS
```bash
brew install sqlcipher
go install github.com/maximodev/enveil/cmd/enveil@latest
```

### Shell integration (zsh)

Add this to your `~/.zshrc` to automatically activate projects when you navigate to their directories:
```zsh
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

### Running commands with variables injected
```bash
enveil run npm run dev
enveil run python manage.py runserver
enveil run printenv DATABASE_URL
```

Variables are injected into the process environment. No `.env` file is created.

### Listing variables
```bash
enveil list
```

Shows variable names in the active environment. Values are never displayed.

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

### Git hook
```bash
enveil hook install
```

Installs a pre-commit hook that scans staged files for secrets before every commit. Detects known secret formats (AWS keys, Stripe keys, GitHub tokens, connection strings, private keys) and high-entropy strings using Shannon entropy analysis.

To bypass in an emergency:
```bash
git commit --no-verify
```

### Daemon

The daemon keeps your master key in memory so you don't have to type your password on every command.
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