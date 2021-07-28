use anyhow::Error;
use serde::Deserialize;
use yew::{callback::Callback, format::Nothing, services::fetch::Request, ComponentLink};

pub struct Image {
    pub name: String,
    pub body: String,
}

#[derive(Deserialize, Debug, Clone)]
pub struct RegistryCatalog {
    pub repositories: Vec<String>,
}

pub struct Images {
    // props: Props,
    pub repos: Option<Vec<String>>,
    pub error: Option<String>,
    pub link: ComponentLink<Self>,
}

pub fn get_image_list(callback: Callback<Result<String, Error>>) {
    let images_list = Request::get("https://localhost:5000/v2/_catalog")
        .body(Nothing)
        .expect("Could not build that request");
}
