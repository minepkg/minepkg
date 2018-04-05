// use std::collections::HashSet;
// use std::hash::{Hash, Hasher};
// use rayon::prelude::*;

use std::error::Error;
use std::fs::{self, File};
use std::io;
use std::io::prelude::*;
use std::path::PathBuf;
use version_compare::VersionCompare;

use serde_json;

mod multi_mc {
    #[derive(Deserialize, Debug)]
    pub struct Pack {
        pub components: Vec<Component>,
    }

    #[derive(Deserialize, Debug)]
    #[serde(rename_all = "camelCase")]
    pub struct Component {
        pub cached_name: String,
        pub uid: String,
        pub cached_version: String,
        // cached_requires: Vec<Requirement>,
    }

    #[derive(Deserialize, Debug)]
    pub struct Requirement {
        equals: String,
    }

    impl Pack {
        pub fn mc_component(&self) -> Option<&Component> {
            self.components.iter().find(|p| p.uid == "net.minecraft")
        }
        pub fn mc_version(&self) -> Option<&str> {
            let mc = self.mc_component()?;
            Some(&mc.cached_version)
        }
    }
}

/// Looks for a MultiMC file structure
fn read_mmc_pack() -> Result<McInstance, io::Error> {
    let mut file = File::open("./mmc-pack.json")?;
    let mut buf: Vec<u8> = Vec::new();
    file.read_to_end(&mut buf)?;
    let pack: multi_mc::Pack = serde_json::from_slice(&buf)?;
    let mods_dir = fs::read_dir("./")?
        .map(|entry| entry.unwrap().path())
        .find(|path| {
            path.ends_with("minecraft") || path.ends_with(".minecraft")
        }).unwrap();
    Ok(McInstance {
        flavour: Flavour::MultiMC,
        version: pack.mc_version().map(|v| String::from(v)),
        mods_dir: mods_dir.join("mods"),
    })
}

/// Looks for a vanilla launcher file structure
pub fn read_vanilla_instance() -> Result<McInstance, Box<Error>> {
    let dir_iter = fs::read_dir("./versions")?;
    // TODO: lots of unwrap here – lots of bad stuff can happen
    let latest = dir_iter
        .map(|entry| entry.unwrap().file_name().into_string().unwrap())
        .max_by(|va, vb| VersionCompare::compare(va, vb).unwrap().ord().unwrap());

    if latest.is_some() {
        println!(
            "🛈 Asuming the latest installed minecraft version {:?}",
            latest
        );
        println!("Run this command with --mc-version <version> to overwrite this behaviour");
    }
    let latest = latest.ok_or("You need to launch minecraft once")?;
    let mods_dir: PathBuf = ["versions", &latest, "mods"].iter().collect();
    println!("{:?}", mods_dir);
    Ok(McInstance {
        flavour: Flavour::Vanilla,
        version: Some(latest),
        mods_dir,
    })
}

pub fn detect_instance() -> Result<McInstance, Box<Error>> {
    read_mmc_pack() // try reading MultiMC instance first
        .or_else(|_| read_vanilla_instance()) // fallback to vanilla instance
}

#[derive(Debug)]
enum Flavour {
    MultiMC,
    Vanilla,
}

#[derive(Debug)]
pub struct McInstance {
    flavour: Flavour,
    version: Option<String>,
    pub mods_dir: PathBuf,
}

impl McInstance {
    pub fn mc_version(&self) -> Option<&str> {
        self.version.as_ref().map(|v| &v[..])
    }
}

// impl From<multi_mc::Pack> for McInstance {
//     fn from(pack: multi_mc::Pack) -> Self {
//         McInstance {
//             version: pack.mc_version().map(|v| String::from(v)),
//             mods_dir: "./minecraft/mods".parse().unwrap(),
//         }
//     }
// }
