use toml_edit::{Document, value, Value};
use std::fs::File;
use std::io::prelude::*;
use failure::Error;
use std::env::current_dir;
use curse::Mod;

static MANIFEST_TEMPLATE: &str = include_str!("./minepkg-template.toml");

use semver::Version;
use semver::VersionReq;

#[derive(Debug)]
pub struct Manifest {
    doc: Document,
}

#[derive(Debug)]
pub enum Provider {
    Curse,
}

#[derive(Debug)]
pub struct Dependency<'a> {
    pub provider: Provider,
    pub name: &'a str,
}

impl Manifest {
    pub fn add_dependency(&mut self, mc_mod: &Mod) {
        let slug = &mc_mod.slug();
        self.doc["dependencies"][slug] = value(format!("curse:{}", slug));
    }
    pub fn set_mc_version(&mut self, version: &str) {
        self.doc["requirements"]["minecraft-version"] = value(version.clone());
    }
    pub fn dependencies(&self) -> Vec<Dependency> {
        let iter = self.doc["dependencies"].as_table()
            .expect("invalid modpack dependencies")
            .iter();
        iter.map(|entry| {
            let mut value = entry.1.as_str().unwrap().split(':');
            let provider = match value.next().unwrap() {
                "curse" => Provider::Curse,
                _ => panic!("unsupported provider"),
            };
            Dependency { provider, name: value.next().unwrap() }
        }).collect()
    }
    pub fn name(&self) -> &str {
        self.doc["package"]["name"].as_str().expect("modpack has no name")
    }
    pub fn required_mc_version(&self) -> VersionReq {
        self.doc["requirements"]["minecraft-version"].as_str()
            .expect("minepkg: missing requirements.minecraft-version")
            .parse()
            .expect("minepkg: invalid requirements.minecraft-version")
    }
    pub fn version(&self) -> Version {
        self.doc["version"].as_str().unwrap().parse().expect("minepkg: invalid version")
    }
    pub fn save(&self) -> Result<(), Error> {
        let mut file = File::create("./minepkg.toml")?;
        file.write_fmt(format_args!("{}", self.doc))?;
        Ok(())
    }

    pub fn default() -> Manifest {
        let mut doc = MANIFEST_TEMPLATE.parse::<Document>().expect("invalid doc");
        let name = current_dir().unwrap().file_name().unwrap().to_string_lossy().into_owned();
        doc["package"]["name"] = value(name);
        Manifest { doc }
    }
}

// pub fn read_or_create() -> Manifest {
//     if let Ok(mut file) = File::open("./minepkg.toml") {
//         println!("got a file");
//         let mut content = String::new();
//         file.read_to_string(&mut content).expect("reading failed");
//         let mut doc = content.parse::<Document>().expect("invalid doc");
//         Manifest { manifest: doc }
//     } else {
//         Manifest::default()
//     }

// }

pub fn read_local() -> Result<Manifest, Error> {
    let mut file = File::open("./minepkg.toml")?;
    let mut content = String::new();
    file.read_to_string(&mut content)?;
    let doc = content.parse::<Document>()?;
    Ok(Manifest { doc })
}
