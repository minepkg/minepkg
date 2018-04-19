#![feature(box_patterns)]

#[macro_use]
extern crate prettytable;
#[macro_use]
extern crate serde_derive;
// #[macro_use] extern crate failure_derive;

extern crate app_dirs2;
extern crate bzip2;
extern crate clap;
extern crate console;
extern crate failure;
extern crate futures;
extern crate hyper_tls;
extern crate hyper;
extern crate indicatif;
extern crate petgraph;
extern crate rayon;
extern crate reqwest;
extern crate semver;
extern crate serde_json;
extern crate serde;
extern crate snap;
extern crate tokio_core;
extern crate toml_edit;
extern crate version_compare;

mod cli;
mod curse;
mod dep_resolver;
mod local_db;
mod mc_instance;
mod utils;
mod manifest;
pub mod cli_config;
pub use cli::*;
pub use local_db::refresh_db;
