<p align="center">
  <img align="center" width="100" src="./assets/logo.svg" alt="minepkg" />
  <div align="center">
    minepkg is a package manager designed to install Minecraft mods and modpacks.
  </div>
</p>

---

[![main builds](https://github.com/minepkg/minepkg/actions/workflows/main-builds.yml/badge.svg)](https://github.com/minepkg/minepkg/actions/workflows/main-builds.yml)
[![Maintainability](https://api.codeclimate.com/v1/badges/cd2f11d2dd41dee1fcbc/maintainability)](https://codeclimate.com/github/minepkg/minepkg/maintainability)
[![Discord](https://img.shields.io/discord/517070108191883266.svg?logo=discord)](https://discord.gg/6tjBR5t)

## Features

* Installs mods from [minepkg](https://minepkg.io/) (with dependency resolution)
* Can launch Minecraft
* Join compatible modded Servers with one command (Installs modpack and launches Minecraft & joins the server for you)
* Saves your installed mods in a sharable `minepkg.toml`
  * `minepkg.toml` modpacks can extend other modpacks
* Publish mods and modpacks to minepkg.io

## Installation

You can read the [installation docs here](https://minepkg.io/docs/install) for more detailed instructions.

### Linux & MacOS

Run this in your favorite shell.

```bash
curl -s https://minepkg.io/install.sh | bash
```

### Windows

Run this in **powershell**

```
. { iwr -useb https://minepkg.io/install.ps1} | iex
```

### From Source

If you have the go toolchain installed you can (compile &) install minepkg from source:

```bash
go install github.com/minepkg/minepkg@latest
```

## Usage

* [FAQ](https://minepkg.io/docs/faq)
* [Manifest Documentation](https://minepkg.io/docs/manifest)
* [Mod publishing](https://minepkg.io/docs/mod-publishing)

```
$ minepkg --help
Manage Minecraft mods with ease

Usage:
  minepkg [command]

Examples:

  minepkg init -l fabric
  minepkg install modmenu@latest
  minepkg join demo.minepkg.host

Available Commands:
  bump                Bumps the version number of this package
  config              Manage global config options
  dev                 Advanced package dev related tasks (eg. build)
  help                Help about any command
  init                Creates a new mod or modpack in the current directory
  install             Installs one or more packages in your current modpack or mod
  join                Joins a compatible server without any setup
  launch              Launch the given or local modpack.
  publish             Publishes the local package in the current directory to minepkg.io
  remove              Removes supplied package from the current directory & package
  selfupdate          Updates minepkg to the latest version
  try                 Lets you try a mod or modpack in Minecraft
  update              Updates all installed dependencies
  update-requirements Updates installed requirements (minecraft & loader version)

Flags:
  -a, --accept-minecraft-eula   Accept Minecraft's eula. See https://www.minecraft.net/en-us/eula/
      --config string           config file (default is /home/fiws/.config/minepkg/config.toml)
  -h, --help                    help for minepkg
      --non-interactive         Do not prompt for anything (use defaults instead)
      --verbose                 More verbose logging. Not really implemented yet
  -v, --version                 version for minepkg

Use "minepkg [command] --help" for more information about a command.

```

## Demo

<p align="center">
  <img width="720" src="https://i.imgur.com/Sbwlre9.gif" alt="minepkg install preview" />
</p>

## Building

Requires go ~ 1.16+. Could also work with older go versions.
Just `go run main.go [commands]` or `go build -o out/minepkg`

```
git clone https://github.com/minepkg/minepkg.git
cd minepkg
go run main.go --help
```

