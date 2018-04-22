use console::style;
use futures;
use futures::{Future, Stream};
use indicatif;
use reqwest;
use std;
use std::borrow::Cow;
use std::fs::File;
use std::io::Read;
use std::io::Write;

use cli::local_db;
use minepkg::{
    curse::Mod,
    dep_resolver,
    mc_instance,
    utils::*,
};

fn confirm(msg: String) {
    // this is inefficient, but duh
    print!("{}", msg);
    std::io::stdout().flush().unwrap();
    let input: u8 = std::io::stdin()
        .bytes()
        .next()
        .and_then(|result| result.ok()).expect("What the hell did you type in there?");
    
    match input {
        10 | 121 | 89 => (), // Y y and Enter
        _ => std::process::exit(1), // everything else aborts
    }
}

pub fn install(name: Option<String>) -> CliResult {
    if let Some(name) = name { install_single(&name) }
    else { install_modpack() }
}

pub fn install_modpack() -> CliResult {
    let instance = mc_instance::detect_instance().map_err(|_| "No Minecraft instance found")?;
    // read the minepkg.toml and add our new dependency
    let manifest = instance.manifest()?;
    let db = local_db::read_or_download().expect("Problems reading mod db");
    println!("{}", style(" 📔 [1 / 3] Reading local modpack").bold());

    let mut to_be_installed: Vec<&Mod> = Vec::new();
    for dep in manifest.dependencies() {
        let mc_mod = db.find_by_slug(dep.name)
            .ok_or(format!("Mod '{}' not found in local db.", dep.name))?;
        println!("    requires {} from CurseForge", mc_mod.name);
        to_be_installed.push(mc_mod);
    }

    // prompt user to confirm installation
    confirm(format!("\n    Install {} packages? [Y/n] ", style(&to_be_installed.len()).bold()));
    for dep in to_be_installed {
        install_mod(dep)?;
    }

    println!("{}", style(format!("  ✔ Successfully installed {} modpack", manifest.name())).green());
    Ok(())
}

pub fn install_single(name: &str) -> CliResult {
    println!("{}", style(" 📚 [1 / 3] Searching local mod DB").bold());
    let name = name.to_lowercase();
    let db = local_db::read_or_download().expect("Problems reading mod db");
    let found = &db.wonky_find(&name).ok_or("No mod found")?;

    // prompt user to confirm installation
    confirm(format!("\n    Install {} from CurseForge? [Y/n] ", style(&found.name).bold()));

    install_mod(&found)?;
    println!("{}", style(format!("  ✔ Successfully installed {}", found.name)).green());
    Ok(())
}

/// installs a mod
pub fn install_mod(mc_mod: &Mod) -> CliResult {
    let id = mc_mod.id.to_string();
    let instance = mc_instance::detect_instance().map_err(|_| "No Minecraft instance found")?;
    // read the minepkg.toml and add our new dependency
    let mut manifest = instance.manifest()?;
    manifest.add_dependency(mc_mod);

    let mc_version = instance
        .mc_version()
        .ok_or("Your instance does not have minecraft installed (yet)")?;
    let AsyncToolbox {
        mut core,
        hyper,
        reqwest,
    } = AsyncToolbox::new();

    // resolve dependencies
    println!("{}", style(" 🔎 [2 / 3] Resolving Dependencies").bold());
    let mut dep_resolver = dep_resolver::DepResolver::new({ hyper });
    dep_resolver.set_mc_version(mc_version.to_string());
    let work = dep_resolver.resolve(&String::from(id));
    core.run(work)?;

    let to_install = dep_resolver.resolved_deps.borrow();
    for mc_mod in to_install.iter() {
        println!("    requires {}", mc_mod.file_name);
    }

    // install them
    println!("{}", style(format!(
            " 🚚 [3 / 3] Downloading {} mods",
            to_install.len()
        )).bold()
    );
    let progress = indicatif::MultiProgress::new();
    let work: Vec<_> = to_install
        .iter()
        .map(|mc_mod| {
            // new progress bar for each download
            let pb = indicatif::ProgressBar::new(1_100_000);
            let mods_dir = &instance.mods_dir;

            // fix mods not containing jar in the filename
            let mut file_name = Cow::from(&mc_mod.file_name[..]);
            if !file_name.ends_with(".jar") {
                file_name.to_mut().push_str(".jar");
            }

            // add them to the oter progress bars, and setup style
            let pb = progress.add(pb);
            &pb.set_style(
                indicatif::ProgressStyle::default_bar()
                    .template(" {spinner}  {prefix:20!} {wide_bar} 📦"),
            );
            &pb.set_prefix(&file_name);
            // now star the (download) request
            reqwest
                .get(&mc_mod.download_url)
                .send()
                .and_then(move |res| {
                    // we need the length (filesize) to properly display the progress bar
                    let size: u64 = match res.headers().get::<reqwest::header::ContentLength>() {
                        Some(length) => length.0,
                        None => 2_500_000, // we estimate mods to be 2.5MB if there is no header 😅
                    };
                    pb.set_length(size);
                    // build the final file path here
                    let file_name = mods_dir.clone().join(file_name.as_ref());
                    let mut file = File::create(file_name).unwrap();

                    // write the file in chunks and update the progress bar
                    // TODO: this is synchronous! we need a fs threadpool here
                    res.into_body().for_each(move |chunk| {
                        &pb.inc(chunk.len() as u64);
                        file.write_all(&chunk).unwrap();
                        Ok(())
                    })
                })
        })
        .collect();

    // the multiprogress bar needs to be on another thread
    // https://github.com/mitsuhiko/indicatif/issues/33
    let handler = std::thread::spawn(move || {
        progress.join().unwrap();
    });

    // finally start all the downloads in parallel
    // TODO: maybe limit to ~5 at a time or something
    core.run(futures::future::join_all(work))?;
    // all jobs ran, we stop the progress bar thread now
    handler.join().unwrap();
    // last but not least, save the manifest changes
    manifest.save()?;
    Ok(())
}
