use app_dirs2::*;
use std::path::PathBuf;

const APP_INFO: AppInfo = AppInfo{ name: "mmm", author: "fiws" };

pub fn ensure_data_dir() -> Result<PathBuf, AppDirsError> {
    app_root(AppDataType::UserData, &APP_INFO)
}

pub fn get_data_dir() -> Result<PathBuf, AppDirsError> {
    get_app_root(AppDataType::UserData, &APP_INFO)
}
