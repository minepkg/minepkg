use prettytable;
use prettytable::row::Row;
use prettytable::Table;
use rayon::prelude::*;
use serde::de::{self, Visitor};
use serde::Deserialize;
use serde::Deserializer;
use std::collections::HashSet;
use std::convert::AsRef;
use std::fmt;
use std::hash::{Hash, Hasher};

use utils;

#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct ModDb {
    // timestamp: u64,
    #[serde(rename = "data")]
    pub mods: Vec<Mod>,
}

impl ModDb {
    /// Resolves the dependencies using the local DB ONLY
    /// Returns a `HashSet` including the mod you specified and all its (recursive) dependencies
    // currently not used
    #[allow(dead_code)]
    pub fn resolve_dependencies(
        &self,
        file: &ModFile,
        version: &str,
        optional: bool,
    ) -> HashSet<&Mod> {
        file.dependencies
            .iter()
            .filter(|dep| optional || dep.req_type == ReqType::Required)
            .map(|dep| {
                self.mods
                    .iter()
                    .find(|m| m.id == dep.add_on_id)
                    .expect("Unmet dependency")
            })
            .flat_map(|dep| {
                // let latest_release = dep.release_for_mc_version(version).expect("No release");
                let f = dep.latest_file_for(version).expect("no release");
                let mut deps_from_dep = self.resolve_dependencies(f, version, false);
                deps_from_dep.insert(dep);
                deps_from_dep
            })
            .collect()
    }
    /// Finds a mod by name
    /// Priortizes mods with high download count
    pub fn find_by_name(&self, name: &str) -> Option<&Mod> {
        let name = name.to_lowercase();
        let mut mods: Vec<_> = self.mods
            .iter()
            .filter(|m| m.name.to_lowercase().contains(&name))
            .take(100)
            .collect();
        mods.sort_unstable_by(|a, b| a.download_count.cmp(&b.download_count));
        mods.last().map(|a| *a)
    }
    /// Finds a mod by slug
    pub fn find_by_slug(&self, name: &str) -> Option<&Mod> {
        let name = name.to_lowercase();
        self.mods
            .par_iter()
            .find_any(|m| m.web_site_url.to_lowercase().ends_with(&name))
    }
    /// Finds a mod by id
    pub fn find_by_id(&self, id: u32) -> Option<&Mod> {
        self.mods.par_iter().find_any(|m| m.id == id)
    }
    /// Tries its best to find a mod
    /// 1. curseforge url? – search by slug
    /// 2. id? – find by id
    /// 3. string? – find by name
    pub fn wonky_find(&self, arg: &str) -> Option<&Mod> {
        // is this a curse url?
        if arg.starts_with("https://minecraft.curseforge.com/projects/") {
            return self.find_by_slug(&arg[42..]);
        }
        // is it an id?
        match arg.parse::<u32>() {
            Ok(id) => self.find_by_id(id),
            // fallback to name search
            Err(_) => self.find_by_name(arg),
        }
    }
}

/// Contains a single mod
#[derive(Deserialize, Debug)]
#[serde(rename_all = "PascalCase")]
pub struct Mod {
    pub name: String,
    pub id: u32,
    #[serde(rename = "WebSiteURL")]
    pub web_site_url: String,
    #[serde(deserialize_with = "thanks_curse")]
    pub download_count: u32,
    pub latest_files: Vec<ModFile>,
    game_version_latest_files: Vec<GameVersionRelease>,
}

impl AsRef<[u8]> for Mod {
    fn as_ref(&self) -> &[u8] {
        &self.name.as_bytes()
    }
}

impl Mod {
    /// Gets the (latest) release for the specified minecraft version
    pub fn release_for_mc_version(&self, version: &str) -> Option<&GameVersionRelease> {
        // TODO: filter by release type
        self.game_version_latest_files
            .iter()
            .find(|r| r.game_version == version)
    }
    /// Gets the latest file for the specified minecraft version
    /// These just seem to be the latest *uploads*.
    /// (Could return `None` for the wanted mc version, even if the mod supports it)
    pub fn latest_file_for(&self, version: &str) -> Option<&ModFile> {
        self.latest_files
            .iter()
            .find(|f| f.game_version.iter().any(|v| v == version))
    }
    /// Prints out some info about the mod
    pub fn pretty_print(&self) {
        println!("{}", self.name);
        println!("===============================");
        println!("id: {}", self.id);
        println!("Downloads: {}", self.download_count);
        println!("URL: {}", self.web_site_url);
        println!("Latest Releases: ");
        for gvr in &self.game_version_latest_files {
            print!("· {} ({:?}) ", gvr.game_version, gvr.file_type);
        }
        if let Some(file) = self.latest_files.first() {
            println!("\n\nLatest File: {}", file.file_name);
            println!("Game Version: {:?}", file.game_version);
            println!("Dependencies:");
            let deps = &file.dependencies;
            let types = [ReqType::Required, ReqType::Optional, ReqType::Embedded];
            for ty in types.iter() {
                let deps: Vec<_> = deps.iter().filter(|f| f.req_type == *ty).collect();
                println!("  {:?}: {}", ty, &deps.len());
                let deps = deps.into_iter()
                    .map(|d| d.add_on_id.to_string())
                    .collect::<Vec<_>>()
                    .join(",");
                println!("{:?}", deps);
            }
            // println!("  Optional: {}", deps.iter().filter(|f| f.req_type == ReqType::Optional).count());
            // println!("  Embedded: {}", deps.iter().filter(|f| f.req_type == ReqType::Embedded).count());
        }
    }
}

impl Hash for Mod {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.id.hash(state);
    }
}

impl PartialEq for Mod {
    fn eq(&self, other: &Mod) -> bool {
        self.id == other.id
    }
}
impl Eq for Mod {}

#[derive(Deserialize, Debug)]
pub enum ReleaseType {
    Release,
    Beta,
    Alpha,
}

#[derive(Deserialize, Debug)]
#[serde(rename_all = "PascalCase")]
pub struct GameVersionRelease {
    #[serde(deserialize_with = "map_release")]
    pub file_type: ReleaseType,
    #[serde(rename = "ProjectFileID")]
    pub file_id: u32,
    #[serde(rename = "GameVesion")] // YES VESION
    pub game_version: String,
}

#[derive(Deserialize, Debug, Clone, PartialEq, Eq)]
pub enum ReqType {
    Required,
    Optional,
    Embedded,
}

#[derive(Deserialize, Debug, Clone)]
#[serde(rename_all = "PascalCase")]
/// Defines a dependency on another mod
pub struct ModDependency {
    /// id of the dependency
    pub add_on_id: u32,
    #[serde(rename = "Type")]
    #[serde(deserialize_with = "map_req_type")]
    /// required or optional or the only values we care about
    pub req_type: ReqType, // TODO: bool out if this
}

#[derive(Deserialize, Debug, Clone)]
#[serde(rename_all = "PascalCase")]
/// A downloadable file inside a mod
pub struct ModFile {
    #[serde(rename = "DownloadURL")]
    pub download_url: String,
    pub id: u32,
    pub file_name: String,
    pub game_version: Vec<String>,
    #[serde(deserialize_with = "null_to_empty_vec")]
    pub dependencies: Vec<ModDependency>,
}

impl Hash for ModFile {
    fn hash<H: Hasher>(&self, state: &mut H) {
        self.id.hash(state);
    }
}

impl PartialEq for ModFile {
    fn eq(&self, other: &ModFile) -> bool {
        self.id == other.id
    }
}
impl Eq for ModFile {}

pub trait PrintAsTable {
    fn print_as_table(&self);
}

/// Prints a pretty table for any mod slice
impl PrintAsTable for [Mod] {
    fn print_as_table(&self) {
        let head = row!["Name", "Downloads", "Id"];
        let rows: Vec<Row> = self.into_iter()
            .map(|m| {
                row![
                    utils::shorten_name(&mut m.name.clone()),
                    // m.latest_files[0].game_version.unwrap()[0],
                    utils::simplify_number(m.download_count),
                    m.id
                ]
            })
            .collect();
        let mut table = Table::init(rows);
        table.set_titles(head);
        table.set_format(*prettytable::format::consts::FORMAT_NO_LINESEP);
        table.print_tty(false);
    }
}

// Don't look at this java code
struct ReleaseTypeVisitor;
impl<'de> Visitor<'de> for ReleaseTypeVisitor {
    type Value = ReleaseType;

    fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
        formatter.write_str("an u8 or String (Release, Beta or Alpha)")
    }

    fn visit_str<E>(self, value: &str) -> Result<ReleaseType, E>
    where
        E: de::Error,
    {
        let release = match value.as_ref() {
            r"Release" => ReleaseType::Release,
            r"Beta" => ReleaseType::Beta,
            _ => ReleaseType::Alpha,
        };
        Ok(release)
    }

    fn visit_u64<E>(self, value: u64) -> Result<ReleaseType, E>
    where
        E: de::Error,
    {
        let release = match value {
            1 => ReleaseType::Release,
            2 => ReleaseType::Beta,
            _ => ReleaseType::Alpha,
        };
        Ok(release)
    }
}

struct ReqTypeVisitor;
impl<'de> Visitor<'de> for ReqTypeVisitor {
    type Value = ReqType;

    fn expecting(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
        formatter.write_str("an u8 or str (Required, Optional, Embedded)")
    }

    fn visit_str<E>(self, value: &str) -> Result<ReqType, E>
    where
        E: de::Error,
    {
        let release = match value.as_ref() {
            r"Required" => ReqType::Required,
            r"Optional" => ReqType::Optional,
            _ => ReqType::Embedded,
        };
        Ok(release)
    }

    fn visit_u64<E>(self, value: u64) -> Result<ReqType, E>
    where
        E: de::Error,
    {
        let release = match value {
            1 => ReqType::Required,
            2 => ReqType::Optional,
            _ => ReqType::Embedded,
        };
        Ok(release)
    }
}

/// Maps require type to our enum (curse SOAP and local_db json have different formats)
fn map_req_type<'de, D>(dese: D) -> Result<ReqType, D::Error>
where
    D: Deserializer<'de>,
{
    dese.deserialize_any(ReqTypeVisitor)
}

// Maps release to our enum (curse SOAP and local_db json have different formats)
fn map_release<'de, D>(dese: D) -> Result<ReleaseType, D::Error>
where
    D: Deserializer<'de>,
{
    dese.deserialize_any(ReleaseTypeVisitor)
}

/// Helper to convert f32 download count as u32 (there are no half downloads)
// should probably rename this
fn thanks_curse<'de, D>(dese: D) -> Result<u32, D::Error>
where
    D: Deserializer<'de>,
{
    let f = f32::deserialize(dese)?;

    Ok(f as u32)
}

/// Helper that returns a empty vec if deserialization fails
// thanks curse again
fn null_to_empty_vec<'de, D>(dese: D) -> Result<Vec<ModDependency>, D::Error>
where
    D: Deserializer<'de>,
{
    Deserialize::deserialize(dese).map(|x: Option<_>| x.unwrap_or(Vec::new()))
}
