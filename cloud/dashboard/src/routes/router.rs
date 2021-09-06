use yew_router::prelude::*;

#[derive(Switch,Clone)]
pub enum AppRoute {
    #[to = "/images/{name}"]
    ImageDetail(String),
    #[to = "/images"]
    Images
}

pub type Anchor = RouterAnchor<AppRoute>;