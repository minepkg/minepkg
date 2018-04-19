use toml_edit::{Document, value};
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

impl Manifest {
    pub fn add_dependency(&mut self, mc_mod: &Mod) {
        let slug = &mc_mod.slug();
        self.doc["dependencies"][slug] = value(format!("curse:{}", slug));
    }
    pub fn set_mc_version(&mut self, version: &str) {
        self.doc["requirements"]["minecraft-version"] = value(version.clone());
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
    println!("got a file");
    let mut content = String::new();
    file.read_to_string(&mut content)?;
    let doc = content.parse::<Document>()?;
    Ok(Manifest { doc })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn it_works() {
        let source = include_str!("../tests/fixtures/example1.minepkg.toml");
        let mut doc = source.parse::<Document>().expect("invalid doc");
        println!("{:?}", doc["dependencies"]);
        doc["dependencies"]["grainerIO"] = value("curse:garinerIO");
        doc["dependencies"]["enderio"] = value("github:enderio");
        for idk in doc["dependencies"].as_table().unwrap().iter() {
            println!("{:?}", idk);
            println!("----------------");
        }
        println!("{}", doc);
    }
}
