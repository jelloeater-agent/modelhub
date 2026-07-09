# ModelHub

[![Test](https://github.com/jelloeater-agent/modelhub/actions/workflows/test.yml/badge.svg)](https://github.com/jelloeater-agent/modelhub/actions/workflows/test.yml)
![coverage](https://raw.githubusercontent.com/jelloeater-agent/modelhub/refs/heads/badges/.badges/main/coverage.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/jelloeater-agent/modelhub)](https://goreportcard.com/report/github.com/jelloeater-agent/modelhub)
![Libraries.io dependency status for GitHub repo](https://img.shields.io/librariesio/github/jelloeater-agent/modelhub)

![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/jelloeater-agent/modelhub/total)
![GitHub Release](https://img.shields.io/github/v/release/jelloeater-agent/modelhub)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jelloeater-agent/modelhub)
![GitHub Release Date](https://img.shields.io/github/release-date/jelloeater-agent/modelhub)

AI model pricing, capabilities, and benchmarks — from Bifrost, models.dev, and Artificial Analysis. **Zero dependencies.**

## Install

```shell
go install github.com/jelloeater-agent/modelhub/cmd/modelhub@latest
```

Or grab a binary from [releases](https://github.com/jelloeater-agent/modelhub/releases).

## Usage

```
modelhub refresh           Fetch latest data from all sources
modelhub list [--table]    List models (JSON default)
modelhub show <id>         Show a single model (JSON)
modelhub stats             Aggregate statistics (JSON + summary)
```

### Examples

```shell
# Refresh data
modelhub refresh

# Pipe JSON to jq for any query
modelhub list | jq '.[] | select(.provider=="openai") | .name'
modelhub list | jq '.[] | select(.context_window > 128000 and .input_price_per_1m < 1)'
modelhub list | jq '.[] | select(.supports_vision) | .id'

# Quick visual scan
modelhub list --table
modelhub list --table | grep gpt-4

# Single model details
modelhub show openai/gpt-4o | jq .context_window

# Stats
modelhub stats

# Search interactively with fzf
modelhub list | jq -r '.[].id' | fzf --preview 'modelhub show {}'
```

### Config

Set `AA_API_KEY` env var for Artificial Analysis benchmarks (optional):

```shell
export AA_API_KEY=your_key_here
```

Or create `~/.modelhub/config.json`:

```json
{
  "aa_api_key": "your_key_here"
}
```

## Build

```shell
go build ./cmd/modelhub
```
