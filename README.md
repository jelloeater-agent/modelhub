# ModelHub

[![Test](https://github.com/jelloeater-agent/modelhub/actions/workflows/test.yml/badge.svg)](https://github.com/jelloeater-agent/modelhub/actions/workflows/test.yml)
![coverage](https://raw.githubusercontent.com/jelloeater-agent/modelhub/refs/heads/badges/.badges/main/coverage.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/jelloeater-agent/modelhub)](https://goreportcard.com/report/github.com/jelloeater-agent/modelhub)
![Libraries.io dependency status for GitHub repo](https://img.shields.io/librariesio/github/jelloeater-agent/modelhub)

![GitHub Downloads (all assets, all releases)](https://img.shields.io/github/downloads/jelloeater-agent/modelhub/total)
![GitHub Release](https://img.shields.io/github/v/release/jelloeater-agent/modelhub)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/jelloeater-agent/modelhub)
![GitHub Release Date](https://img.shields.io/github/release-date/jelloeater-agent/modelhub)

TUI for browsing LLM pricing, capabilities, and benchmarks — sourced from Bifrost, models.dev, and Artificial Analysis.

![ModelHub](modelhub.gif)

## Install

### Apt (Preferred)

```shell
curl -s https://packagecloud.io/install/repositories/jelloeater/modelhub/script.deb.sh | sudo bash
sudo apt-get install modelhub
```

### Yum (Preferred)

```shell
curl -s https://packagecloud.io/install/repositories/jelloeater/modelhub/script.rpm.sh | sudo bash
sudo yum install modelhub
```

### Binary (eget)

```shell
curl https://zyedidia.github.io/eget.sh | sh
sudo mv eget /usr/local/bin
sudo eget jelloeater-agent/modelhub --to /usr/local/bin --asset=tar.gz
```

### Via Go

```shell
go install github.com/jelloeater-agent/modelhub/cmd/modelhub@latest
```

## Usage

```shell
modelhub
```

### Controls

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate rows |
| `PgUp/PgDn` | Page up/down |
| `Home/End` | Jump to top/bottom |
| `/` | Search |
| `s` | Sort column |
| `f` | Filter by provider |
| `Enter` | View model details |
| `Esc` | Back / Clear |
| `r` | Refresh data |
| `q` / `Ctrl+C` | Quit |

### Settings

| Env Var | Description |
|---------|-------------|
| `AA_API_KEY` | Artificial Analysis API key (for benchmarks) |

## Build

```shell
go mod download
go build -o ./build ./cmd/modelhub
./build/modelhub
```
