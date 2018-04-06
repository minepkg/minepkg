use utils::*;
use mc_instance;
use dep_resolver;

use std::fs::File;
use std::io::Write;
use std::io::Read;
use futures::{Future, Stream};
use futures;
use console::style;
use indicatif;
use std;
use reqwest;
use local_db;

pub fn install(name: &str) -> CliResult {
    println!("{}", style(" 📚 [1 / 3] Searching local mod DB").bold());
    let name = name.to_lowercase();
    let db = local_db::read_or_download().expect("Problems reading mod db");
    let found = &db.wonky_find(&name).ok_or("No mod found")?;

    // prompt user to confirm installation
    print!("\n    Install {} from CurseForge? [Y/n] ", style(&found.name).bold());
    std::io::stdout().flush().unwrap();
    let input: u8 = std::io::stdin()
        .bytes()
        .next()
        .and_then(|result| result.ok()).expect("What the hell did you type in there?");

    match input {
        10 | 121 | 89 => (), // Y y and Enter
        _ => std::process::exit(1), // everything else aborts
    }

    install_id(&found.id.to_string())?;
    println!("{}", style(format!("  ✔ Successfully installed {}", found.name)).green());
    Ok(())
}

/// installs a mod by curse id
pub fn install_id(id: &str) -> CliResult {
    let instance = mc_instance::detect_instance().map_err(|_| "No Minecraft instance found")?;
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

            // add them to the oter progress bars, and setup style
            let pb = progress.add(pb);
            &pb.set_style(
                indicatif::ProgressStyle::default_bar()
                    .template(" {spinner}  {prefix:20!} {wide_bar} 📦"),
            );
            &pb.set_prefix(&mc_mod.file_name);
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
                    let file_name = mods_dir.clone().join(&mc_mod.file_name);
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
    Ok(())
}
