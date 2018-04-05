use bzip2;
use curse::ModDb;
use futures::{Future, Stream};
use hyper;
use indicatif;
use serde_json;
use snap;
use std;
use std::fs::File;
use std::io::prelude::*;
use utils::*;

const DB_URL: &str = "https://clientupdate-v6.cursecdn.com/feed/addons/432/v10/complete.json.bz2";
const APP_PATH: &str = ".mmm/";
const DB_PATH: &str = "complete.json.sz";

lazy_static! {
    static ref APP_DIR: std::path::PathBuf = {
        std::env::home_dir()
            .expect("No idea where your home directory is…")
            .join(APP_PATH)
    };
}

/// db path helper
pub fn db_location() -> std::path::PathBuf {
    APP_DIR.join(DB_PATH)
}

/// refreshed the local mod db
pub fn refresh_db() -> CliResult {
    let AsyncToolbox {
        mut core,
        hyper,
        reqwest,
    } = AsyncToolbox::new();
    let bar = indicatif::ProgressBar::new(100);

    bar.set_message("Fetching DB");
    bar.set_style(
        indicatif::ProgressStyle::default_bar()
            .template("{spinner:.green} {wide_bar} {bytes}/{total_bytes} ({eta})"),
    );
    let file = File::create(db_location()).unwrap(); // write to fs
    let recompressor = snap::Writer::new(file); // recompress with snap
    let mut decompressor = bzip2::write::BzDecoder::new(recompressor); // decompress first

    let uri = DB_URL.parse().unwrap();
    let work = hyper
        .get(uri)
        .and_then(|res| {
            let size: u64 = match res.headers().get::<hyper::header::ContentLength>() {
                Some(length) => length.0,
                None => 0,
            };
            bar.set_length(size);

            res.body().for_each(|chunk| {
                &bar.inc(chunk.len() as u64);
                decompressor.write_all(&chunk).map_err(From::from)
            })
        })
        .then(|res| {
            &bar.finish();
            res
        });
    core.run(work).expect("failed just failed");
    println!("✓ updated local DB");
    Ok(())
}

/// reads the local mod db. If there is none, downloads it
pub fn read_or_download() -> Result<ModDb, std::io::Error> {
    let file = File::open(db_location()).or_else(|e| {
        if e.kind() == std::io::ErrorKind::NotFound {
            println!("There is no mod db yet. Fetching now");
            // TODO: this is bad ↓. better error handling
            refresh_db().expect("Refreshing DB failed");
            File::open(db_location())
        } else {
            Err(e)
        }
    })?;
    // TODO: check if file is old and refresh
    let mut decompressor = snap::Reader::new(file);
    let mut contents: Vec<u8> = Vec::new();
    decompressor.read_to_end(&mut contents)?;
    let db: ModDb = serde_json::from_slice(&contents)?;
    Ok(db)
}
