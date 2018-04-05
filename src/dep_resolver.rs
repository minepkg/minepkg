use curse;
use futures;
use futures::{Future, Stream};
use hyper;
use hyper::client::HttpConnector;
use hyper_tls::HttpsConnector;
use serde_json;

use std::cell::RefCell;
use std::collections::HashSet;
use std::error::Error;
use std::rc::Rc;

type HyperClient = hyper::Client<HttpsConnector<HttpConnector>>;

pub struct DepResolver {
    hyper: Rc<HyperClient>,
    /// The minecraft version to query the dependencies for
    pub mc_version: String,
    /// HashSet containing the resolved dependencies.
    /// Will include the mod given to `resolve(mod_id)`
    pub resolved_deps: Rc<RefCell<HashSet<curse::ModFile>>>,
}

impl DepResolver {
    pub fn new(hyper: HyperClient) -> Self {
        DepResolver {
            hyper: Rc::new(hyper),
            mc_version: "1.12.2".to_owned(), // TODO: remove hardcoded version
            resolved_deps: Rc::new(RefCell::new(HashSet::new())),
        }
    }
    /// Sets the minecraft version
    pub fn set_mc_version(&mut self, version: String) {
        self.mc_version = version;
    }
    pub fn fetch_mod<'a>(
        &'a self,
        mod_id: &str,
    ) -> impl Future<Item = curse::Mod, Error = hyper::Error> + 'a {
        // TODO: use reqwest, not hyper!
        let hyper = self.hyper.clone();
        let uri = format!(
            "https://cursemeta.dries007.net/api/v2/direct/GetAddOn/{}",
            mod_id
        ).parse()
            .expect("invalid id");
        hyper.get(uri).and_then(|res| as_curse_mod(res))
    }
    pub fn fetch_file<'a>(
        &'a self,
        add_on_id: u32,
        mod_id: u32,
    ) -> impl Future<Item = curse::ModFile, Error = hyper::Error> + 'a {
        // TODO: use reqwest, not hyper!
        let hyper = self.hyper.clone();
        let uri = format!(
            "https://cursemeta.dries007.net/api/v2/direct/GetAddOnFile/{}/{}",
            add_on_id, mod_id
        ).parse()
            .expect("invalid ids");
        hyper.get(uri).and_then(|res| as_curse_file(res))
    }
    /// Resolve all dependencies for the given mod
    pub fn resolve<'a>(&'a self, mod_id: &str) -> Box<Future<Item = (), Error = Box<Error>> + 'a> {
        let mc_version = &self.mc_version;

        let wrks = self.fetch_mod(mod_id)
            .map_err(|e| e.into())
            .and_then(move |mc_mod| {
                let release = mc_mod.release_for_mc_version(mc_version).ok_or({
                    format!(
                        "The mod {} is not avaiable for your Minecraft Version ({})",
                        mc_mod.name, mc_version
                    )
                })?;

                Ok(self.fetch_file(mc_mod.id, release.file_id)
                    .map_err(|e| e.into()))

            })
            // TODO: find out howto get rid of this ugly `and_then(|f| f)`
             // flatten (?) does not work
            .and_then(|f| f)
            .and_then(move |file| {
                let deps = {
                    let ids: Vec<String> = file.dependencies
                        .iter()
                        .filter(|d| d.req_type == curse::ReqType::Required)
                        .map(|d| d.add_on_id.to_string())
                        .collect();

                    ids.into_iter().map(move |ref d| self.resolve(d))
                };

                self.resolved_deps.borrow_mut().insert(file.clone());

                Ok(deps)
            })
            .and_then(|deps| futures::future::join_all(deps));

        Box::new(wrks.and_then(|_| Ok(())))
    }
}

/// Helper to get a `curse::Mod` out
fn as_curse_mod(res: hyper::Response) -> impl Future<Item = curse::Mod, Error = hyper::Error> {
    let work = res.body().concat2().and_then(move |body| {
        let v: curse::Mod = serde_json::from_slice(&body).unwrap();
        Ok(v)
    });
    work
}

/// Helper to get a `curse::ModFile` out
fn as_curse_file(res: hyper::Response) -> impl Future<Item = curse::ModFile, Error = hyper::Error> {
    let work = res.body().concat2().and_then(move |body| {
        let v: curse::ModFile = serde_json::from_slice(&body).unwrap();
        Ok(v)
    });
    work
}
