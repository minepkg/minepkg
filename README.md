<p align="center">
  <img width="220" src="./assets/logo.svg" alt="minepkg" />
  <div align="center">
    minepkg is a package manager designed to install Minecraft mods and modpacks.
  </div>
</p>

---

[![pipeline status](https://gitlab.com/fiws/minepkg/badges/master/pipeline.svg)](https://gitlab.com/fiws/minepkg/commits/master)
[![Maintainability](https://api.codeclimate.com/v1/badges/74d43859d907238c4836/maintainability)](https://codeclimate.com/github/fiws/minepkg/maintainability)
[![Discord](https://img.shields.io/discord/517070108191883266.svg?logo=discord)](https://discord.gg/6tjBR5t)


## Features

* Installs mods from [minepkg](https://minepkg.io/) (with dependency resolution)
* Can launch Minecraft
* Saves your installed mods in a sharable `minepkg.toml`
  * `minepkg.toml` modpacks can extend other modpacks

## Installation

### Linux & MacOS

Run this in your favourite shell. The installation will require sudo rights.

```bash
source <(curl -s https://storage.googleapis.com/minepkg-client/linux-installer.sh)
```

### Windows

Run this in **powershell**

```
. { iwr -useb https://storage.googleapis.com/minepkg-client/windows-installer.ps1} | iex; minepkg
```

## Usage

```
❯ minepkg --help
Manage Minecraft mods with ease.

Examples:
  minepkg install rftools
  minepkg install https://minecraft.curseforge.com/projects/storage-drawers

Usage:
  minepkg [command]

Available Commands:
  completion  Output shell completion code for bash
  help        Help about any command
  install     installz packages
  refresh     Fetches all mods that are available

Flags:
  -h, --help            help for minepkg

Use "minepkg [command] --help" for more information about a command.

```

## Demo

<p align="center">
  <img width="720" src="https://i.imgur.com/BRfIa9b.gif" alt="minepkg install preview" />
</p>

## Building

Requires go 1.11+. Don't use your GOPATH.
Just `go run main.go [commands]` or `go build -o out/minepkg`

```
git clone https://github.com/fiws/minepkg.git
cd minepkg
go run main.go --help
```

