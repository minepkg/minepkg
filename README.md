<p align="center">
  <img align="center" width="100" src="./assets/logo.svg" alt="minepkg" />
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
* Join compatible modded Servers with one command (Installs modpack and launches Minecraft for you)
* Saves your installed mods in a sharable `minepkg.toml`
  * `minepkg.toml` modpacks can extend other modpacks
* Publish mods and modpacks to minepkg.io

## Installation

### Linux & MacOS

Run this in your favourite shell.

```bash
curl -s https://preview.minepkg.io/install.sh | bash
```

### Windows

Run this in **powershell**

```
. { iwr -useb https://preview.minepkg.io/install.ps1} | iex
```

## Usage

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
  build               Runs the set buildCmd (falls back to gradle build)
  help                Help about any command
  init                Creates a new mod or modpack in the current directory
  install             Installs one or more packages in your current modpack or mod
  join                Joins a compatible server without any setup
  launch              Launch the given or local modpack.
  login               Sign in to Mojang in order to start Minecraft
  minepkg-login       Sign in to minepkg.io (mainly for publishing)
  publish             Publishes the local package in the current directory
  remove              Removes specified package from the current package
  selfupdate          Updates minepkg to the latest version
  try                 Lets you try a mod or modpack in Minecraft
  update              Updates all installed dependencies
  update-requirements Updates installed requirements (minecraft & loader version)

Flags:
  -a, --accept-minecraft-eula   Accept Minecraft's eula. […]
      --config string           config file (default is $HOME/.minepkg/config.toml)
  -h, --help                    help for minepkg
      --no-color                disable color output
      --system-java             Use system java instead […]
      --verbose                 More verbose logging. Not really implented yet
  -v, --version                 version for minepkg

Use "minepkg [command] --help" for more information about a command.

```

## Demo

<p align="center">
  <img width="720" src="https://i.imgur.com/BRfIa9b.gif" alt="minepkg install preview" />
</p>

## Building

Requires go ~ 1.16+. Could also work with older go versions.
Just `go run main.go [commands]` or `go build -o out/minepkg`

```
git clone https://github.com/fiws/minepkg.git
cd minepkg
go run main.go --help
```

