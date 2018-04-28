use std::sync::{RwLock};
use std::fmt::Display;
use console::style;

// lazy_static! {
//     #[derive(Debug)]
//     pub static ref level: RwLock<u8> = RwLock::new(0);
// }

pub struct Logger {
    indention: u8,
}

impl Logger {
    pub fn log<T: Display>(&self, s: T) {
        log_indented(s, self.indention)
    }
    pub fn log_success<T: Display>(&self, s: T) {
        self.log(style(s).green())
    }
    pub fn success_headline<T: Display>(&self, s: T) {
        self.headline(style(s).green())
    }
    pub fn emoji_success_headline<T: Display>(&self, emoji: &str, s: T) {
        self.emoji_headline(emoji, style(s).green())
    }
    pub fn headline<T: Display>(&self, s: T) {
        let s = style(s).bold();
        println!("{}", s);
    }
    pub fn emoji_headline<T: Display>(&self, emoji: &str, s: T) {
        let new_line = format!(" {} {}", emoji, s);
        self.headline(new_line);
    }
    pub fn empty_line(&self,) {
        println!();
    }
    pub fn new() -> Logger {
        Logger { indention: 2 }
    }
    pub fn with_indention(spaces: u8) -> Logger {
        Logger { indention: spaces }
    }
    pub fn with_headline<T: Display>(headline: T) -> Logger {
        let logger = Logger::new();
        logger.headline(headline);
        logger
    }
    pub fn with_emoji_headline<T: Display>(emoji: &str, headline: T) -> Logger {
        let logger = Logger::with_indention(4);
        logger.emoji_headline(emoji, headline);
        logger
    }
}

fn log_indented<T: Display>(s: T, spaces: u8) {
    let spaces = String::from(" ").repeat(spaces as usize);
    println!("{}{}", spaces, s);
}
