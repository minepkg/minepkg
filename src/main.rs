#[macro_use]
extern crate clap;
extern crate console;
use clap::{App, Arg, SubCommand};

extern crate mmm;

fn main() {
    let mod_name_arg = Arg::with_name("MOD")
        .help("The mod name, id or URL")
        .required(true)
        .multiple(true)
        .index(1);

    let matches = App::new("mmm")
        .version(crate_version!())
        .author("Filip Weiss. <mmm+me@fiws.net>")
        .about("Minecraft Mod Manager at your service.")
        .after_help("EXAMPLES:\n    mmm install ender io\n    mmm install https://minecraft.curseforge.com/projects/journeymap")
        // TODO: verbosity setting would be nice
        // .arg(Arg::with_name("v")
        //     .short("v")
        //     .multiple(true)
        //     .help("Sets the level of verbosity"))
        .subcommand(SubCommand::with_name("install")
            .about("Installs a new mod with the required dependencies")
            .arg(&mod_name_arg)
            .alias("add"))
        .subcommand(SubCommand::with_name("search")
            .arg(&mod_name_arg)
            .about("Search for a mod in the local database"))
        .subcommand(SubCommand::with_name("show")
            .arg(&mod_name_arg)
            .about("Find a single mod, and display info about it")
            .alias("info"))
        .subcommand(SubCommand::with_name("refresh")
            .about("Fetches all mods that are available"))
        .subcommand(SubCommand::with_name("list")
            .about("lists all installed mods"))
        .get_matches();
    
    let get_mod_val = |v| {
        matches.subcommand_matches(v).unwrap().values_of_lossy("MOD").unwrap().join(" ")
    };

    mmm::ensure_app_dir();
    // TODO: those functions probably need params and stuff soon
    let result = match matches.subcommand_name() {
        Some("install") => mmm::install(&get_mod_val("install")),
        Some("search") => mmm::search(&get_mod_val("search")),
        Some("show") => mmm::show(&get_mod_val("show")),
        Some("refresh") => mmm::refresh_db(),
        // Some("list") => mmm::list(),
        Some(_) | None => {
            println!("{}", matches.usage());
            Ok(())
        },
    };
    if let Err(e) = result {
        eprintln!("{}: {}", console::style("💣 error").red().bold(), e);
        std::process::exit(1);
    }
}
