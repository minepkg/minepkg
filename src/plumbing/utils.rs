use hyper_tls::HttpsConnector;
use hyper::Client as HyperClient;
use hyper::client::HttpConnector;
use reqwest::unstable::async::Client;
use std;
use tokio_core::reactor::Core;

pub type CliResult = Result<(), Box<std::error::Error>>;

pub struct AsyncToolbox {
    pub core: Core,
    pub hyper: HyperClient<HttpsConnector<HttpConnector>>,
    pub reqwest: Client,
}

impl AsyncToolbox {
    pub fn new() -> Self {
        let core = Core::new().expect("Failed to create event loop");
        let handle = core.handle();
        let client = HyperClient::configure()
            .connector(HttpsConnector::new(4, &handle).expect("sure"))
            .build(&handle);
        let reqwest = Client::new(&handle);
        AsyncToolbox {
            core,
            hyper: client,
            reqwest,
        }
    }
}

pub fn shorten_name(name: &mut String) -> &mut String {
    if name.len() < 40 {
        return name;
    }
    name.trim();
    name.truncate(38);
    name.push_str("…");
    name
}

pub fn simplify_number(num: u32) -> String {
    if num >= 1_000_000_000 {
        return format!("{} B", num / 1_000_000_000);
    } else if num >= 1_000_000 {
        return format!("{} M", num / 1_000_000);
    } else if num >= 1000 {
        return format!("{} K", num / 1000);
    }
    format!("{}", num)
}
