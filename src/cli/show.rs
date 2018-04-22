use cli::local_db;
use minepkg::utils::CliResult;

pub fn show(name: &str) -> CliResult {
    let db = local_db::read_or_download().expect("Problems reading mod db");
    let found = &db.wonky_find(&name).ok_or("No mod found")?;

    found.pretty_print();
    Ok(())
}
