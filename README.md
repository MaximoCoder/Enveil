# Enveil

Enveil keeps your environment variables encrypted and out of your filesystem — for individuals and teams.

Most developers store secrets in `.env` files. Those files get accidentally committed to version control, shared over Slack, read by AI coding tools with filesystem access, and left behind on old machines. Enveil eliminates the file entirely. Secrets live in an encrypted vault and are injected directly into your process at runtime — they never touch disk as plaintext, not even temporarily.

For teams, Enveil goes further: a self-hosted server lets every developer share the same encrypted secrets without `.env` files, chat messages, or shared drives. One developer sets a variable; everyone else has it immediately.

## How it works

When you run `enveil run npm run dev`, Enveil:

1. Derives a 256-bit key from your master password using **Argon2id** (64MB memory, 4 threads) — resistant to GPU and ASIC brute-force attacks
2. Decrypts the **SQLCipher vault** at `~/.enveil/vault.db` — the entire file is encrypted with AES-256, including table names, project names, and variable names
3. Reads the variables for the active project and environment
4. Spawns your process with those variables injected via `syscall.Exec` — no temporary files, no subshells, no intermediate writes
5. The master key lives only in memory for the duration of the command, then is gone

The vault file is opaque binary. Without the master password, it is indistinguishable from random noise.

For teams using the server, values are encrypted on the client with AES-GCM before being sent over the network. The server stores and returns ciphertext only — it never sees plaintext values, even if you trust the server operator completely.

## Installation

### Linux and macOS

```bash
curl -fsSL https://raw.githubusercontent.com/MaximoCoder/Enveil/main/install.sh | sh
```

The installer detects your OS and architecture, downloads the correct binary, verifies its checksum, and installs it to `/usr/local/bin`.

### Windows

Use [WSL2](https://learn.microsoft.com/en-us/windows/wsl/install) and run the Linux installer inside it.

### From source

Requires Go 1.22+ and `libsqlcipher-dev` (Ubuntu/Debian) or `sqlcipher` (macOS).

```bash
go install github.com/MaximoCoder/Enveil/cli/cmd/enveil@latest
```

### Shell integration

Add this to your `~/.zshrc` or `~/.bashrc` to automatically show the active project and environment in your prompt when you navigate to a registered directory:

```bash
eval "$(enveil shell-init)"
```

## Quickstart

```bash
# 1. Create your vault and set a master password (run once)
enveil init

# 2. Register your project (run once per project directory)
cd ~/projects/myapp
enveil init

# 3. Import your existing .env file
enveil import .env

# 4. Delete the .env file — you no longer need it
rm .env

# 5. Run your app normally
enveil run npm run dev
```

From this point on, your secrets exist only in the encrypted vault.

## Usage

### First time setup

```bash
enveil init
```

Run this once globally to create your vault and set your master password. Run it again inside any project directory to register that project.

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

Imports all variables from the file into the active environment. Enveil will ask if you want to delete the original file after importing.

### Running commands with variables injected

```bash
enveil run npm run dev
enveil run python manage.py runserver
enveil run php artisan serve
enveil run printenv DATABASE_URL
```

Variables are injected directly into the process environment. No `.env` file is created or written to disk at any point.

### Getting and listing variables

```bash
enveil list              # show all variable names (values are masked by default)
enveil get DATABASE_URL  # get the value of a specific variable
```

### Deleting variables

```bash
enveil delete API_KEY
```

Asks for confirmation before deleting.

### Managing environments

Each project can have multiple environments. Enveil creates `development` by default.

```bash
enveil env list              # list all environments in the current project
enveil env add staging       # create a new environment
enveil env add production
enveil env use staging       # switch the active environment
```

All `set`, `get`, `list`, `run`, and `import` commands operate on the active environment. Switching environments is instant — no files to copy or rename.

### Comparing environments

```bash
enveil diff development staging
```

Shows which variables are missing, extra, or have different values between two environments — without revealing the actual values. Useful for catching configuration drift before a deployment.

### Exporting a temporary .env file

```bash
enveil export
```

For tools that require a physical `.env` file. Automatically adds `.env` to `.gitignore`. Delete it when done — the vault is your source of truth.

### Managing projects

```bash
enveil projects     # list all registered project directories
enveil unregister   # remove the current directory from the vault
```

### Git hook

```bash
enveil hook install
```

Installs a pre-commit hook that scans staged files for secrets before every commit. It detects known secret formats (AWS keys, Stripe keys, GitHub tokens, connection strings, private keys) and high-entropy strings using Shannon entropy analysis.

Files like `.env` and `.env.local` are always blocked. Files like `.env.example` and `.env.template` are allowed since they are intended to contain placeholder values.

To bypass the hook when you are sure a file is safe:

```bash
ENVEIL_SKIP=1 git commit
```

To bypass all hooks:

```bash
git commit --no-verify
```

### Daemon

The daemon keeps your master key in memory so you do not have to type your password on every command.

```bash
enveil daemon start    # start the daemon, enter your password once
enveil daemon status   # check if the daemon is running
enveil daemon stop     # stop the daemon — the key is removed from memory immediately
```

The daemon is optional. Without it, Enveil derives the key fresh from your password on each command. With it, you type your password once per session.

The key is never written to disk — the daemon holds it in a Unix socket at `~/.enveil/daemon.sock`, accessible only to your user.

## Security verification

You can verify Enveil's security properties manually without trusting the source code.

### 1. Confirm the vault is opaque

After running `enveil init` and setting a variable:

```bash
xxd ~/.enveil/vault.db | head -5
strings ~/.enveil/vault.db
```

`xxd` will show binary data. `strings` will return nothing — there are no readable strings to extract. Every byte, including table names and variable names, is encrypted.

### 2. Confirm the wrong password is rejected

```bash
enveil daemon stop   # make sure the daemon is not caching your key
enveil list          # enter the wrong password
```

Enveil will refuse to open the vault and return an error. The data is never partially exposed.

### 3. Confirm variables are never written to disk during injection

```bash
# Run a command and check that no .env file was created
enveil run printenv DATABASE_URL
ls -la .env 2>/dev/null || echo "no .env file — correct"
```

### 4. Confirm file permissions

```bash
ls -la ~/.enveil/vault.db
```

The vault is created with `0600` permissions — readable and writable only by your user, not by other users on the same machine.

## Team server

The Enveil server lets teams share encrypted secrets across developers without relying on `.env` files, chat messages, or shared drives. It is self-hosted — your secrets never leave your infrastructure.

### How it works

Values are encrypted on the client with AES-GCM before being sent to the server. The server stores and returns ciphertext only. Even if someone gains access to the server machine or the server vault file, they cannot read the secrets without the API key used to encrypt them.

### Setting up the server

Download the server binary from the [latest release](https://github.com/MaximoCoder/Enveil/releases/latest) for your platform (`enveil-server-linux-amd64`, `enveil-server-darwin-arm64`, etc.) and place it in your PATH.

```bash
ENVEIL_API_KEY=your-secret-key \
ENVEIL_VAULT_PASSWORD=your-vault-password \
ENVEIL_PORT=8080 \
enveil-server
```

The server stores its vault at `~/.enveil-server/vault.db` by default. Override with `ENVEIL_VAULT_PATH`.

For production, put the server behind a reverse proxy like nginx with HTTPS enabled.

### Team workflow

**Admin (one time):**

```bash
# Start the server
ENVEIL_API_KEY=shared-api-key ENVEIL_VAULT_PASSWORD=vault-pass ENVEIL_PORT=8080 enveil-server

# Connect your own CLI to it
enveil server connect http://your-server:8080 --key shared-api-key

# Register your project on the server
cd ~/projects/myapp
enveil init

# Import your existing secrets
enveil import .env
```

**Each developer (one time):**

```bash
# Connect to the server
enveil server connect http://your-server:8080 --key shared-api-key

# Associate their local directory with the shared project
cd ~/projects/myapp
enveil server use-project myapp
```

From that point, all `set`, `get`, `list`, `run`, `import`, `export`, `diff`, and `env` commands operate against the shared server. Variables set by one developer are immediately available to all others. No `.env` files are shared, committed, or sent over chat.

**Checking server status:**

```bash
enveil server status      # verify connection
enveil server disconnect  # switch back to local vault
```

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

| Property | Implementation |
|---|---|
| Vault encryption | AES-256 via SQLCipher. The entire database is encrypted, including metadata. |
| Key derivation | Argon2id, 64MB memory, 4 threads. Resistant to GPU and ASIC attacks. |
| Master key at rest | Never written to disk. Held in memory by the daemon, or derived per-command and discarded. |
| Transport encryption | AES-GCM on the client before transmission. The server never sees plaintext. |
| File permissions | Vault and config created with `0600` — owner only. |
| Process injection | Variables passed via `syscall.Exec`, never through temporary files or environment exports. |
| Secret scanning | Pre-commit hook uses pattern matching and Shannon entropy analysis. |

## License

MIT
