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

Run this in your favourite shell. The installation will require/ask for sudo rights.

```bash
bash <(curl -s https://storage.googleapis.com/minepkg-client/installer.sh)
```

### Windows

Run this in **powershell**

```
. { iwr -useb https://storage.googleapis.com/minepkg-client/windows-installer.ps1} | iex; minepkg
```

## Usage

```
❯ minepkg --help
Manage Minecraft mods with ease

Usage:
  minepkg [command]

Examples:

  minepkg init -l fabric
  minepkg install modmenu@latest
  minepkg install https://minepkg.io/projects/desire-paths

Available Commands:
  build               Runs the build hook (falls back to gradle build)
  help                Help about any command
  init                Creates a new mod or modpack in the current directory
  install             installs one or more packages
  join                Joins a compatible server without any setup
  launch              Launch a local or remote modpack.
  login               Sign in to Mojang in order to start Minecraft
  minepkg-login       Sign in to minepkg.io
  publish             Publishes the local package in the current directory
  remove              removes specified package from the manifest
  selfupdate          Updates minepkg to the latest version
  try                 Lets you try a mod or modpack in Minecraft
  update              updates all installed dependencies
  update-requirements updates installed requirements (minecraft & loader version)

Flags:
  -a, --accept-minecraft-eula   Accept Mojang's Minecraft eula. […]
      --config string           config file (default is $HOME/.minepkg/config.toml)
  -h, --help                    help for minepkg
      --no-color                disable color output
      --system-java             Use system java […]
      --verbose                 More verbose logging. Not really implented yet

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

