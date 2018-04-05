use curse;
use futures::Future;
use reqwest::unstable::async::Client;
use reqwest::unstable::async::Response;
use std::error::Error;

/// Big construction site here 

const API_URL: &str = "https://cursemeta.dries007.net";

pub trait CurseMeta {
    fn download_mod<'a>(
        &self,
        mod_file: &curse::ModFile,
    ) -> Box<Future<Item = Response, Error = Box<Error>> + 'a>;
}

impl CurseMeta for Client {
    fn download_mod<'a>(
        &self,
        mod_file: &curse::ModFile,
    ) -> Box<Future<Item = Response, Error = Box<Error>> + 'a> {
        let work = self.get("http://localhost:1111")
            .send()
            .map_err(|e| e.into());
        Box::new(work)
    }
}
