use crate::routes::{router::Anchor, router::AppRoute};
use crate::services::requests::{Image, Images, RegistryCatalog};
use yew::services::ConsoleService;
use serde::Deserialize;
use yew::{
    format::{Json, Nothing},
    html,
    services::fetch::{FetchService, FetchTask, Request, Response},
    Component, ComponentLink, Html, ShouldRender,
};

#[derive(Debug)]
pub enum Msg {
    GetRegistryCatelog(Result<ISS, anyhow::Error>),
}

#[derive(Deserialize, Debug, Clone)]
pub struct ISSPosition {
    latitude: String,
    longitude: String,
}

#[derive(Deserialize, Debug, Clone)]
pub struct ISS {
    message: String,
    timestamp: i32,
    iss_position: ISSPosition,
}

enum Class {
    Provider,
    Categories,
    OperatingSystems,
    Architectrues,
}

enum Label {
    ThirdPart(Class, String),
    Official(Class, String),

    Analytics(Class, String),
    ApplicationRuntime(Class, String),
    BaseImages(Class, String),
    Databases(Class, String),
    DevOps(Class, String),
    Messaging(Class, String),
    Monitoring(Class, String),
    OperatingSystem(Class, String),
    Storage(Class, String),
    Networking(Class, String),

    Linux(Class, String),
    Windows(Class, String),

    ARM64(Class, String),
    AMD64(Class, String),
}

impl Component for Images {
    type Message = Msg;
    type Properties = ();

    fn create(props: Self::Properties, link: ComponentLink<Self>) -> Self {
        ConsoleService::info("create app");
        Self {
            repos: None,
            link,
            error: None,
        }
    }

    fn update(&mut self, msg: Self::Message) -> ShouldRender {
        use Msg::*;
        match msg {
            GetRegistryCatelog(response) => match response {
                Ok(repos) => {
                    //self.repos = Some(repos.repositories);
                        ConsoleService::info(&format!("info {:?}", repos)); 
                }
                Err(error) => {
                    ConsoleService::info(&format!("info {:?}", error.to_string())); 
                },
            },
        }
        true
    }

    fn rendered(&mut self, first_render: bool) {
        if first_render {
            ConsoleService::info("view app");
            let request = Request::get("http://api.open-notify.org/iss-now.json")
                .body(Nothing)
                .expect("could not build request.");
            let callback = self.link.callback(
                |response: Response<Json<Result<ISS, anyhow::Error>>>| {
                    let Json(data) = response.into_body();
                    Msg::GetRegistryCatelog(data)
                },
            );
            let task = FetchService::fetch(request, callback).expect("failed to start request");
        }
    }

    fn change(&mut self, props: Self::Properties) -> ShouldRender {
        true
    }

    fn view(&self) -> Html {
        html! {
          <div>
          <div class="columns is-multiline">
            <div class="container column is-1">
              { self.filter() }
            </div>
            <div class="container column is-10">
            { self.image_list() }
            </div>
          </div>
          </div>
        }
    }

    fn destroy(&mut self) {}
}

impl Images {
    fn filter(&self) -> Html {
        html! {
         <aside class="menu">
             <p class="menu-label">
              { "Providor" }
             </p>
             <ul class="menu-list">
             <li><a>{ "Official" }</a></li>
             <li><a>{ "ThirdPart" }</a></li>
             </ul>
             <p class="menu-label">
               { "Categories" }
             </p>
             <ul class="menu-list">
             <li><a>{ "BaseImage" }</a></li>
             <li><a>{ "DataBases" }</a></li>
             <li><a>{ "Messaging" }</a></li>
             <li><a>{ "Monitoring" }</a></li>
             </ul>
             <p class="menu-label">
               { "Architecutures" }
             </p>
             <ul class="menu-list">
             <li><a>{ "ARM64" }</a></li>
             <li><a>{ "AMD64" }</a></li>
             </ul>
         </aside>
        }
    }
    fn image_list(&self) -> Html {
        match &self.repos {
            Some(images) => {
                html! {
                 <div class="columns is-multiline">
                   {
                       for images.iter().map(|image|{
                           self.image_info(image)
                       })
                   }
                 </div>
                }
            }
            None => {
                html! {
                  <p> {"image not found"} </p>
                }
            }
        }
    }

    fn image_info(&self, image: &String) -> Html {
        html! {
         <div class="column is-6">
           <div class="card">
           <Anchor route=AppRoute::ImageDetail(image.to_string())>
             <header class="card-header">
               <p class="card-header-title">
                 { image.to_string() }
               </p>
               <button class="card-header-icon" aria-label="more options">
               <span class="icon">
                 <i class="fal fa-expand" aria-hidden="true"></i>
               </span>
             </button>
             </header>
             </Anchor>
               <div class="card-content">
               <div class="content">
                 { "describe" }
                  <br />
                 <time datetime="2016-1-1">{ "11:09 PM - 1 Jan 2016" }</time>
               </div>
               </div>
            </div>
         </div>
        }
    }
}
