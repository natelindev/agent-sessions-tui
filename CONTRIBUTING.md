# Contributing to Agent Sessions

## Development

Use Go 1.24+. Keep discovery, normalized sessions, resume argument construction, and UI state separate.

```sh
go test -race ./...
go vet ./...
go build -trimpath -o bin/agent-sessions-tui ./cmd/agent-sessions-tui
```

Run `gofmt -w` on edited Go files. Add focused tests for provider parsing, incremental cache behavior, selection, and resume arguments. Provider stores must remain read-only. Use redacted fixtures rather than real session histories in tests.

## Screenshots

The screenshot is actual terminal output captured in a PTY. The script creates and removes a private synthetic home, and never reads personal histories.

```sh
python3 -m venv /tmp/agent-sessions-screenshot-tools
/tmp/agent-sessions-screenshot-tools/bin/pip install -r scripts/requirements-screenshot.txt
go build -o bin/agent-sessions-tui ./cmd/agent-sessions-tui
/tmp/agent-sessions-screenshot-tools/bin/python scripts/capture-screenshot.py
```

Use `--font /path/to/mono.ttf` on systems without a detected monospaced font.

## Documentation site

The public site is https://natelindev-agent-sessions-tui.pages.dev/. Its Cloudflare Pages project is `natelindev-agent-sessions-tui`. Current deployments use manual Direct Upload; automatic Cloudflare deployments are not configured.

Edit `docs/` and check desktop/mobile layouts, navigation, code copying, images, and light/dark appearance. The static HTML remains readable without JavaScript. Preview with `python3 -m http.server 8000 --directory docs`.

To publish, authenticate Wrangler to the account owning the project with Pages write permission:

```sh
npx --yes wrangler@4.148.0 login
npx --yes wrangler@4.148.0 pages deploy docs --project-name natelindev-agent-sessions-tui --branch main
```

Alternatively, upload a ZIP of the contents of `docs/` in the project dashboard, with `index.html` at the archive root. The included `deploy-docs.yml` workflow can publish using repository secrets `CLOUDFLARE_API_TOKEN` (Account → Cloudflare Pages → Edit) and `CLOUDFLARE_ACCOUNT_ID`. These secrets are not currently configured. See [Cloudflare’s direct-upload CI guide](https://developers.cloudflare.com/pages/how-to/use-direct-upload-with-continuous-integration/).

## Pull requests and reports

Describe the user-visible change and relevant verification. Keep private keys, API tokens, personal transcripts, and private hostnames out of commits, screenshots, and reports. Include a small reproduction and OS/tool versions. Brand assets live in `docs/assets/brand/`; marks are path-based SVGs, with transparent PNG exports and dark variants.
