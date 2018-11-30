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

* Installs mods from CurseForge (with dependency resolution)
* Compiles mods from source zip files (eg. from GitHub) with docker
* Saves your installed mods in a sharable `minepkg.toml`
  * `minepkg.toml` modpacks can extend other modpacks
* Works with vanilla and MultiMC instances (MultiMC support is unofficial)

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

## Thanks to

* dries007 for [cursemeta](https://github.com/dries007/CurseMeta) – used to talk to CurseForge API
