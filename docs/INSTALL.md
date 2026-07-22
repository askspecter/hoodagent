# Installing Holt

Holt is distributed as:

- an npm package, `@askspecter/holt`
- release archives on GitHub Releases
- source builds with Go 1.25+

The npm package and install scripts download a platform-specific release archive.
They require a published GitHub Release for the requested version.

## npm

```bash
npm install -g @askspecter/holt
holt
```

The package supports Linux, macOS, and Windows on x64 and arm64. It installs the
`holt` command and downloads the matching release binary during `postinstall`.

Requirements:

- Node.js 18+
- network access to npm and GitHub Releases

## Bun

Bun is "default-secure" and does not run lifecycle scripts of installed
dependencies (only the installing project's own scripts), so the `postinstall`
that fetches the Holt binary is silently skipped. The first run then fails with
`No native binary found next to the npm wrapper`.

To install with Bun, either run the installer manually after installing:

```bash
bun add @askspecter/holt
node node_modules/@askspecter/holt/scripts/postinstall.mjs
```

Or allow the postinstall to run by adding the package to your project's
`trustedDependencies` before installing:

```json
{
  "trustedDependencies": ["@askspecter/holt"]
}
```

```bash
bun add @askspecter/holt
```

For global installs (`bun add -g @askspecter/holt`), run the installer manually
against the global install path, or use the install scripts below.

Reference: <https://bun.sh/docs/pm/lifecycle>

## Linux And macOS Script

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/askspecter/holt/main/scripts/install.sh | bash
```

From a checkout:

```bash
scripts/install.sh
```

Install a specific version:

```bash
HOLT_VERSION=0.1.0 scripts/install.sh
scripts/install.sh --version 0.1.0
```

Install somewhere else:

```bash
HOLT_INSTALL_DIR="$HOME/bin" scripts/install.sh
scripts/install.sh --install-dir "$HOME/bin"
```

Defaults:

- Repository: `askspecter/holt`
- Version: latest GitHub release
- Install path: `~/.local/bin/holt`

Requirements: Bash, `curl` or `wget`, `tar`, and `shasum` or `sha256sum`.

## Windows PowerShell Script

Install the latest release:

```powershell
irm https://raw.githubusercontent.com/askspecter/holt/main/scripts/install.ps1 | iex
```

From a checkout:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1
```

Install a specific version:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -Version 0.1.0
```

Install somewhere else:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/install.ps1 -InstallDir "$env:USERPROFILE\bin"
```

Defaults:

- Repository: `askspecter/holt`
- Version: latest GitHub release
- Install path: `%LOCALAPPDATA%\holt\bin\holt.exe`

## From Source

```bash
git clone https://github.com/askspecter/holt.git
cd holt
go run ./cmd/holt
```

Build a local binary:

```bash
go build -o holt ./cmd/holt
```

Source builds require Go 1.25+.

### Sandbox Helpers For Source Builds

Release archives include the platform sandbox helpers. If you build directly
from source, build the helpers you need:

Linux:

```bash
go build -o holt ./cmd/holt
go build -o holt-linux-sandbox ./cmd/holt-linux-sandbox
go build -o holt-seccomp ./cmd/holt-seccomp
```

Put `holt` and `holt-linux-sandbox` in the same directory on `PATH`, for example
`~/.local/bin`. `holt-seccomp` is kept as a compatibility wrapper; the sandbox
helper applies the Unix-socket filter itself when that sandbox option is enabled.
Linux native sandboxing also requires Bubblewrap to be installed.

macOS uses the system sandbox and does not need an extra helper binary.

Windows source builds can use the main `holt.exe` as the command runner and setup
helper through Holt's built-in self-dispatch path. If you want a release-style
layout anyway, build the standalone helper executables next to `holt.exe`:

```powershell
go build -o holt.exe ./cmd/holt
go build -o holt-windows-command-runner.exe ./cmd/holt-windows-command-runner
go build -o holt-windows-sandbox-setup.exe ./cmd/holt-windows-sandbox-setup
```

## Release Archive Format

Release archives are named:

- `holt-v<version>-linux-<arch>.tar.gz`
- `holt-v<version>-macos-<arch>.tar.gz`
- `holt-v<version>-windows-<arch>.zip`

Supported targets:

- `linux-x64`
- `linux-arm64`
- `macos-x64`
- `macos-arm64`
- `windows-x64`
- `windows-arm64`

Each archive must have a matching `.sha256` file. The install scripts download
both files, verify the checksum, and then copy the binary into the install
directory.

## Updating

Check for a newer release:

```bash
holt update --check
```

Then reinstall with npm or rerun the install script for the version you want.
