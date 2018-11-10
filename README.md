<p align="center">
  <img width="720" src="https://i.imgur.com/BRfIa9b.gif" alt="minepkg install preview" />
</p>


## Usage

```
❯ minepkg --help
Manage Minecraft mods with ease.

Examples:
  minepkg install rftools
  minepkg install https://minecraft.curseforge.com/projects/storage-drawers.

Usage:
  minepkg [command]

Available Commands:
  completion  Output shell completion code for bash
  help        Help about any command
  install     installz packages
  refresh     Fetches all mods that are available

Flags:
      --config string   config file (default is $HOME/.minepkg-config.toml)
  -h, --help            help for minepkg

Use "minepkg [command] --help" for more information about a command.
```

## Building

Requires go 1.11+. Don't use your GOPATH.
Just `go run main.go [commands]` or `go build -o out/minepkg`

### Example

```
git clone https://github.com/fiws/minepkg.git
cd minepkg
go run main.go --help
```
