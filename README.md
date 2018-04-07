<p align="center">
  <img width="720" src="https://i.imgur.com/Z9ctRZH.gif" alt="mmm install preview" />
</p>


## Usage

```
❯ mmm --help
mmm 0.1.0
Filip Weiss. <mmm+me@fiws.net>
Minecraft Mod Manager at your service.

USAGE:
    mmm [SUBCOMMAND]

FLAGS:
    -h, --help       Prints help information
    -V, --version    Prints version information

SUBCOMMANDS:
    help       Prints this message or the help of the given subcommand(s)
    install    Installs a new mod with the required dependencies
    refresh    Fetches all mods that are available
    search     Search for a mod in the local database
    show       Find a single mod, and display info about it

EXAMPLES:
    mmm install ender io
    mmm install https://minecraft.curseforge.com/projects/journeymap
```

## Building

You will need Rust nigtly to compile mmm. 
The easiest way to get Rust nightly is through [rustup](https://www.rustup.rs/).
Simply run `rustup default nightly` to always use nightly.

Or run `rustup update nightly` and `cargo +nightly <command>` if you do not want to change your default rust compiler version.

After that you can build the project using `cargo build` or build and directly run it using `cargo run -- install journeymap`.

### TLDR:

```
❯ rustup default nightly
❯ git clone https://github.com/fiws/mmm.git
❯ cd mmm
❯ cargo build
❯ ./target/debug/mmm --help
```
