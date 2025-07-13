# SSH Key Manager (sshctl)

A powerful, cross-platform SSH key management tool written in Go that replaces and extends the functionality of bash-based SSH management scripts.

## Features

### Core Functionality (Original)
- 🔑 **Add all private keys** to ssh-agent
- 📋 **Copy public keys** to clipboard with **interactive arrow key navigation**
- ⚙️ **Open SSH config** in your preferred editor
- 📄 **List all SSH keys** from ~/.ssh/ directory
- 🖥️ **List SSH hosts** from config file
- 🧹 **Remove hosts** from known_hosts file
- 📊 **List currently loaded** keys in ssh-agent
- 🔐 **Generate new SSH key pairs** with **interactive menus** (RSA, Ed25519)
- 🧪 **Test SSH connections** with timeout
- 💾 **Backup SSH directory** to compressed archive
- 🔍 **Analyze key security** (strength, type, age)
- 🌈 **Colored output** for better readability
- 🖱️ **Cross-platform clipboard** support (macOS, Linux, Windows)
- ⌨️ **Arrow key navigation** for all interactive menus

## Installation

### Prerequisites
- Go 1.21 or later
- SSH client tools (`ssh`, `ssh-keygen`, `ssh-add`)

### Build from Source

1. **Clone or create the project:**
```bash
mkdir sshctl && cd sshctl
```

2. **Create the files** (copy the Go code and go.mod content)

3. **Initialize and build:**
```bash
go mod tidy
go build -o sshctl main.go
```

4. **Install globally (optional):**
```bash
# On Unix-like systems:
sudo mv sshctl /usr/local/bin/

# On Windows, add to PATH or move to a directory in PATH
```

### Quick Install Script

```bash
#!/bin/bash
git clone <your-repo-url> sshctl
cd sshctl
go mod tidy
go build -o sshctl main.go
sudo mv sshctl /usr/local/bin/
echo "sshctl installed successfully!"
```

## Usage

### Interactive Navigation

`sshctl` features a **modern TUI interface** powered by [Bubbletea](https://github.com/charmbracelet/bubbletea) for all interactive menus:

```bash
# Copy public key to clipboard
sshctl -c
Select SSH public key to copy:
Use ↑↓ (or j/k) to navigate, Enter to select, Esc/q to cancel

  ❯ id_ed25519.pub
    id_rsa.pub
    work_key.pub

Press Ctrl+C to force quit
```

```bash
# Generate new SSH key with interactive prompts
sshctl -G
Enter key name (default: id_rsa_new): mykey

Select key type:
Use ↑↓ (or j/k) to navigate, Enter to select, Esc/q to cancel

  ❯ ed25519 (recommended)
    rsa

Press Ctrl+C to force quit
```

**Navigation Controls:**
- **↑↓** or **j/k** to move selection
- **Enter** to confirm selection
- **Esc/q** to cancel operation
- **Ctrl+C** to force quit

**Robust Terminal Support:**
- Built on [Bubbletea](https://github.com/charmbracelet/bubbletea) for reliable cross-platform TUI
- Automatic detection of terminal capabilities
- Graceful fallback to numbered selection in non-interactive environments
- Works perfectly in SSH sessions, CI/CD, and piped environments

### Basic Commands

```bash
# Show help
sshctl -h

# Add all private keys to ssh-agent
sshctl -a

# Copy a public key to clipboard (interactive with arrow keys)
sshctl -c

# List all SSH keys
sshctl -g

# List currently loaded keys
sshctl -l

# List hosts from SSH config
sshctl -H

# Open SSH config in editor
sshctl -C

# Remove host from known_hosts
sshctl -r old-server.com
```

### Advanced Commands

```bash
# Generate new SSH key pair (interactive prompts)
sshctl -G                         # Fully interactive
sshctl -G mykey                   # Interactive type/size selection
sshctl -G mykey ed25519           # Generate Ed25519 key
sshctl -G mykey rsa 4096          # Generate 4096-bit RSA key

# Test SSH connection
sshctl -t user@example.com
sshctl -t myserver

# Backup SSH directory
sshctl -b                         # Auto-named backup
sshctl -b my_ssh_backup.tar.gz    # Custom name

# Analyze key security
sshctl -A
```

## Command Reference

| Short | Long | Description | Example |
|-------|------|-------------|---------|
| `-a` | `--add` | Add all private keys to ssh-agent | `sshctl -a` |
| `-c` | `--copy` | Copy public key to clipboard | `sshctl -c` |
| `-C` | `--config` | Open SSH config in editor | `sshctl -C` |
| `-g` | `--get` | List all SSH keys | `sshctl -g` |
| `-h` | `--help` | Show help message | `sshctl -h` |
| `-H` | `--host` | List hosts from config | `sshctl -H` |
| `-l` | `--list` | List loaded keys | `sshctl -l` |
| `-r` | `--remove` | Remove host from known_hosts | `sshctl -r host` |
| `-G` | `--generate` | Generate new key pair | `sshctl -G name type bits` |
| `-t` | `--test` | Test SSH connection | `sshctl -t host` |
| `-b` | `--backup` | Backup SSH directory | `sshctl -b [path]` |
| `-A` | `--analyze` | Analyze key security | `sshctl -A` |

## Key Generation Options

### Key Types
- **`rsa`**: Traditional RSA keys (default)
- **`ed25519`**: Modern elliptic curve keys (recommended)

### Key Sizes (RSA only)
- **2048**: Minimum recommended
- **4096**: High security (default)

### Examples
```bash
# Quick Ed25519 key (recommended)
sshctl -G myserver ed25519

# High-security RSA key
sshctl -G backup-key rsa 4096

# Interactive generation
sshctl -G
```

## Security Analysis Features

The analyze command (`-A`) provides:

- **Key type identification** (RSA, Ed25519, etc.)
- **Key strength assessment** (bits for RSA)
- **Security recommendations**
- **Key age information**
- **Weakness warnings** for old/small keys

Example output:
```
SSH Key Analysis:
─────────────────────────────────────────
  id_rsa.pub:
    Type: ssh-rsa
    Created: 2023-01-15 14:30:22
    Bits: 2048 (GOOD)

  id_ed25519.pub:
    Type: ssh-ed25519
    Created: 2024-03-10 09:15:33
    (EXCELLENT - Modern curve)
```

## Platform Support

| Feature | macOS | Linux | Windows |
|---------|-------|-------|---------|
| Core SSH operations | ✅ | ✅ | ✅ |
| Clipboard copy | ✅ | ✅* | ✅ |
| Key generation | ✅ | ✅ | ✅ |
| Connection testing | ✅ | ✅ | ✅ |
| Backup creation | ✅ | ✅ | ⚠️** |

*Linux requires `xclip` or `xsel`  
**Windows backup uses different archival method