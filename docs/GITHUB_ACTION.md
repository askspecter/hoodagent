# HOLT GitHub Action

Run HOLT headlessly inside a GitHub workflow. The action is a thin wrapper around
`holt exec`: it installs a pinned HOLT release, runs your prompt in the checked-out
repository, surfaces HOLT's exit code as the step status, captures the output as a
file, and can optionally post a summary to the triggering pull request or to Slack.

HOLT is model- and provider-agnostic, and so is this action: **you** choose the
provider and supply the API key. Nothing is hardcoded to any single provider.

## Quick start

```yaml
# .github/workflows/holt.yml
name: HOLT
on:
  workflow_dispatch:
    inputs:
      task:
        description: What should HOLT do?
        required: true

permissions:
  contents: write

jobs:
  run:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: askspecter/holt@v1
        with:
          prompt: ${{ inputs.task }}
          provider: openai
          api-key-env: OPENAI_API_KEY
          api-key: ${{ secrets.OPENAI_API_KEY }}
          model: gpt-4.1
```

> The `provider`, `api-key-env`, and `api-key` trio is how you stay
> provider-neutral: `api-key-env` is the environment variable name the chosen
> provider reads its key from (for example `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`,
> `OPENROUTER_API_KEY`), and `api-key` is the secret value. The action exports
> that variable for the run and never prints it. If your repository already
> commits a `.holt/config.json` with an active provider, you can omit `provider`.

## Inputs

| Input | Required | Default | Description |
| --- | --- | --- | --- |
| `prompt` | one of `prompt`/`prompt-file` | `""` | The instruction for HOLT to execute. |
| `prompt-file` | one of `prompt`/`prompt-file` | `""` | Path (relative to `working-directory`) to a file whose contents are the prompt. |
| `provider` | no | `""` | Provider id to activate (e.g. `openai`, `anthropic`, `gemini`, `ollama`, or any compatible endpoint). |
| `api-key` | no | `""` | The provider API key. Pass from a secret; never logged. |
| `api-key-env` | no | `""` | Env var name the provider reads its key from (e.g. `OPENAI_API_KEY`). Exported with `api-key`. |
| `model` | no | `""` | Model id. Defaults to the resolved provider's default. |
| `mode` | no | `""` | Run mode (`holt exec --mode`), e.g. `smart`, `deep`, `fast`. |
| `auto` | no | `low` | Autonomy ceiling (`holt exec --auto`): `low`, `medium`, or `high`. Conservative by default. |
| `self-correct` | no | `false` | Allow mid-run model escalation (`holt exec --allow-escalation`). |
| `add-dir` | no | `""` | Newline-/comma-separated extra write roots (`holt exec --add-dir`). |
| `worktree` | no | `false` | Run in an isolated git worktree (`holt exec --worktree`). |
| `output-format` | no | `stream-json` | `text`, `json`, or `stream-json`. Captured to a file. |
| `post-to` | no | `none` | `pr-comment`, `slack`, or `none`. Where to post a summary after the run. |
| `slack-webhook-url` | no | `""` | Slack incoming-webhook (or generic webhook) URL for `post-to: slack`. Pass from a secret. |
| `github-token` | no | `${{ github.token }}` | Token used to post a PR comment. Requires `pull-requests: write`. |
| `working-directory` | no | `${{ github.workspace }}` | Directory to run HOLT in. |
| `holt-version` | no | (action ref → `latest`) | HOLT release version/tag to install, e.g. `v1.2.3` or `latest`. |
| `holt-repo` | no | `askspecter/holt` | Repository to install the HOLT release from. |

## Outputs

| Output | Description |
| --- | --- |
| `exit-code` | HOLT's exit code (`0` success, `2` usage, `3` provider, non-zero otherwise). |
| `output-file` | Path to the captured HOLT stdout (the raw `output-format` stream). |
| `summary` | A short, single-line summary parsed from the run, when available. |

The step **fails when HOLT returns a non-zero exit code**, so a failed run fails
the job by default. Use `continue-on-error: true` on the step (and read
`exit-code`) if you want to handle failures yourself.

## Examples

### Run HOLT on every issue labeled `holt`

```yaml
name: HOLT issue triage
on:
  issues:
    types: [labeled]

permissions:
  contents: write
  issues: write
  pull-requests: write

jobs:
  triage:
    if: github.event.label.name == 'holt'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: askspecter/holt@v1
        with:
          prompt: |
            Investigate this issue and propose a fix.

            Title: ${{ github.event.issue.title }}

            ${{ github.event.issue.body }}
          provider: anthropic
          api-key-env: ANTHROPIC_API_KEY
          api-key: ${{ secrets.ANTHROPIC_API_KEY }}
          auto: low
          post-to: slack
          slack-webhook-url: ${{ secrets.HOLT_SLACK_WEBHOOK_URL }}
```

### Nightly dependency-upgrade PR

```yaml
name: HOLT nightly deps
on:
  schedule:
    - cron: "0 6 * * 1" # Mondays 06:00 UTC

permissions:
  contents: write
  pull-requests: write

jobs:
  deps:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: askspecter/holt@v1
        id: holt
        with:
          prompt-file: .github/holt/upgrade-deps.md
          provider: openai
          api-key-env: OPENAI_API_KEY
          api-key: ${{ secrets.OPENAI_API_KEY }}
          worktree: true
          auto: medium
      - name: Open pull request
        run: |
          git checkout -b holt/deps-$(date +%Y%m%d)
          git commit -am "chore(deps): nightly upgrade via HOLT" || exit 0
          git push -u origin HEAD
          gh pr create --fill --label dependencies
        env:
          GH_TOKEN: ${{ github.token }}
      - name: Upload HOLT output
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: holt-output
          path: ${{ steps.holt.outputs.output-file }}
          if-no-files-found: warn
```

## Security notes

- **Secrets are passed as Action secrets and never logged.** The action exports
  `api-key` (under `api-key-env`) and `slack-webhook-url` into the step
  environment only. They are never echoed, and HOLT redacts secret-shaped
  strings from its own output.
- **The sandbox is always active.** This action never passes
  `--skip-permissions-unsafe`, so writes stay inside the checked-out repository
  (plus any roots you grant with `add-dir`). Unsafe mode is never enabled
  implicitly.
- **Autonomy defaults to `low`.** Raise it deliberately (`auto: medium`/`high`)
  only for tasks you trust to run unattended.
- **Least-privilege tokens.** Grant only the permissions the workflow needs
  (`contents: write` to edit files, `pull-requests: write` to comment). The
  default `GITHUB_TOKEN` is scoped to the repository.
- **Pin the action.** Reference a tag (`askspecter/holt@v1`) or a commit SHA so a
  workflow run uses a known HOLT version. `holt-version` lets you pin the
  installed binary independently of the action ref.
- **Linux and macOS runners** are supported; Windows runners are rejected with a
  clear error.

## Slack / webhook notifier

The action's `post-to: slack` step sends a one-line summary to a Slack incoming
webhook after the run. HOLT also has a built-in webhook notifier sink
(`internal/notify`) that an unattended run can use to report
"finished / needs input / verify failed after N retries" to Slack or any generic
webhook:

- Configure the destination with the `HOLT_SLACK_WEBHOOK_URL` environment variable
  (or settings). A blank URL disables the sink.
- The sink POSTs a JSON body `{ "text", "type", "message", "summary?", "links?" }`.
  The `text` field is what Slack renders; the structured fields carry the
  machine-readable detail.
- **Fail-soft:** a non-2xx response or a transport error is logged (redacted) and
  swallowed — a webhook problem never crashes the run.
- **Redaction:** the message, summary, links, and the webhook URL itself are run
  through HOLT's redaction before being sent or logged, so tokens never leak.
- **Egress/proxy:** the notifier uses the default HTTP transport, which honors
  `HTTP_PROXY`/`HTTPS_PROXY` when a proxy is configured.
