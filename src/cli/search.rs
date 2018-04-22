use cli::local_db;
use minepkg::curse::{Mod, PrintAsTable};
use minepkg::utils::CliResult;

/// searched the local mod db for a given string
pub fn search(name: &str) -> CliResult {
    let db = local_db::read_or_download().expect("Problems reading mod db");
    let name = name.to_lowercase();
    println!("Mod db contains {:?} packages", db.mods.len());
    let mut rows: Vec<Mod> = db.mods
        .into_iter()
        .filter(|m| m.name.to_lowercase().contains(&name))
        .collect();
    let rows = &mut rows[..];
    rows.sort_unstable_by(|a, b| a.download_count.cmp(&b.download_count));
    rows.reverse();

    rows.print_as_table();
    Ok(())
}
