# Update Flow

`holt update --check` checks the latest GitHub release and compares it with the
local CLI version.

```bash
holt update --check
holt update --check --json
holt update --check --repo askspecter/holt
holt update --check --target windows-x64
```

The command is intentionally check-only:

- It does not replace the running binary.
- It exits with code `0` when the check succeeds, even when an update is
  available.
- It exits with code `1` when the release check cannot be completed.
- `--json` prints the same result in a machine-readable format for scripts and
  CI.

Useful flags:

| Flag | Purpose |
|---|---|
| `--repo <owner/repo>` | Check another GitHub repository. |
| `--endpoint <url|owner/repo>` | Check a specific release API URL or repository slug. |
| `--timeout <duration>` | Override the default release check timeout. |
| `--target <platform-arch>` | Validate release metadata for another supported target. |

Supported targets are `linux-x64`, `linux-arm64`, `macos-x64`, `macos-arm64`,
`windows-x64`, and `windows-arm64`. Without `--target`, Holt checks the current
platform.

Endpoint resolution order:

1. `--endpoint`
2. `HOLT_UPDATE_RELEASE_URL`
3. `--repo`
4. `https://api.github.com/repos/askspecter/holt/releases/latest`

Installer scripts download the matching release asset for the local platform and
verify its `.sha256` file. If Holt is already installed, run `holt update --check`
before reinstalling.
