# MediaKit CLI

`mediakit-cli` is the MediaKit command-line interface. It supports local and cloud execution modes, generated domain commands, and multi-platform binary distribution.

## Install

### npm

```bash
npm install -g @volcengine/mediakit-cli
mediakit-cli version
```

### npx

```bash
npx @volcengine/mediakit-cli version
```

### curl

```bash
curl -fsSL https://raw.githubusercontent.com/volcengine/mediakit-cli/main/scripts/install.sh | bash
mediakit-cli version
```

Install a specific version or custom path:

```bash
VERSION=v0.1.0 INSTALL_DIR="$HOME/.local/bin" \
curl -fsSL https://raw.githubusercontent.com/volcengine/mediakit-cli/main/scripts/install.sh | bash
```

## AI Agent Skills

`mediakit-cli` pairs with AI agent skills (Claude Code, etc.) that
teach the agent MediaKit CLI patterns, best practices, and workflows.

Install all skills:

```bash
npx skills add volcengine/mediakit-cli -g -y
```

Or pick specific domains:

```bash
npx skills add volcengine/mediakit-cli -s byted-mediakit-editing -y
npx skills add volcengine/mediakit-cli -s byted-mediakit-video -y
npx skills add volcengine/mediakit-cli -s byted-mediakit-shared -y
```

## Commands

```bash
mediakit-cli version
mediakit-cli init
mediakit-cli doctor
mediakit-cli config show
mediakit-cli --domains
```

## Development

### Build

```bash
make build
```

### Build All Platforms

```bash
make build-all
```

### Snapshot Release

```bash
make snapshot
```

The local build writes the binary to `.mediakit/build/dev/mediakit-cli`.

## Release

- GitHub Releases are produced with `.goreleaser.yml`
- npm distribution uses `package.json`, `scripts/install.js`, and `scripts/run.js`
- curl installation uses `scripts/install.sh`
- the GitHub Actions entry is `.github/workflows/mediakit-cli-release.yml`

## Local Tool Admission

Current local tool allowlist:

- `ffmpeg`: required, `5.1.x`, `LGPL v2.1 or later`, commercial use allowed
- `ffprobe`: required, `5.1.x`, `LGPL v2.1 or later`, commercial use allowed
- optional FFmpeg features: `openh264`, `demuxer`, `libmp3lame`, `libass`, `libfreetype`, `libfontconfig`, `libfribidi`, `libharfbuzz`, `zlib`, `libpng`, `libjpeg-turbo`
- approved optional feature version anchors: `libass 0.17.3`, `libfreetype 2.13.3`, `libfontconfig 2.15.0`, `libfribidi 1.0.16`, `libharfbuzz 10.2.0`
- built-in local helper: `fetch-file` downloads HTTP/HTTPS URLs to `MEDIAKIT_OUTPUT_PATH` and returns local paths unchanged

Admission boundary:

- Only use external process execution via `os/exec`
- Do not statically link or dynamically link local tools into the Go binary
- Do not modify upstream source code
- Keep FFmpeg in LGPL mode by default
- Do not introduce `non-free` components by default
- Do not keep local intermediate artifacts; only final user outputs and `fetch-file` downloads are retained

## Local File Output

- `MEDIAKIT_OUTPUT_PATH` overrides the local file output directory
- Default output directory: `~/.mediakit/temp`
