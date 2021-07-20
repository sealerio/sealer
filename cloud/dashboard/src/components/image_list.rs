use yew::{html, Component,ComponentLink,Html,ShouldRender};

pub struct Image {
    name: String,
    body: String, 
}

pub struct Images{
    // props: Props,
    images: Vec<Image>
}

pub enum Msg {}

enum Class {
    Providor,
    Categories,
    OperatingSystems,
    Architectrues
}

enum Label {
   ThirdPart(Class),
   Official(Class),

   Analytics(Class),
   ApplicationRuntime(Class),
   BaseImages(Class),
   Databases(Class),
   DevOps(Class),
   Messaging(Class),
   Monitoring(Class),
   OperatingSystem(Class),
   Storage(Class),
   Networking(Class),

   Linux(Class),
   Windows(Class),

   ARM64(Class),
   AMD64(Class)
}

impl Component for Images{
    type Message = Msg;
    type Properties = ();
    
    fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
        Images{
            images: vec![
                Image {
                  name: String::from("kubernetes:v1.19.9"),
                  body: String::from("sealer base image, kuberntes alpine, without calico")
                },
                Image {
                  name: String::from("mysql:v1.19.9"),
                  body: String::from("sealer base image, kuberntes alpine, without calico")
                },
                Image {
                  name: String::from("redis:v1.19.9"),
                  body: String::from("sealer base image, kuberntes alpine, without calico")
                },
                Image {
                  name: String::from("prometheus:v1.19.9"),
                  body: String::from("sealer base image, kuberntes alpine, without calico")
                },
                Image {
                  name: String::from("elk:v1.19.9"),
                  body: String::from("sealer base image, kuberntes alpine, without calico")
                }
            ]
        }
    }
    
    fn update(&mut self, _msg: Self::Message) -> ShouldRender {
        true
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
}

impl Images{
   fn filter(&self) -> Html {
       html!{
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
       html! {
          <div class="columns is-multiline">
            {
                for self.images.iter().map(|image|{
                    self.image_info(image)
                })
            }
          </div>
       }
   }

   fn image_info(&self,image: &Image) -> Html {
       html! {
        <div class="column is-6">
          <div class="card">
            <header class="card-header">
              <p class="card-header-title">
                { image.name.to_string() }
              </p>
              <button class="card-header-icon" aria-label="more options">
              <span class="icon">
                <i class="fal fa-expand" aria-hidden="true"></i>
              </span>
            </button>
            </header>
              <div class="card-content">
              <div class="content">
                { image.body.to_string() }
                 <br />
                <time datetime="2016-1-1">{ "11:09 PM - 1 Jan 2016" }</time>
              </div>
              </div>
           </div>
        </div>
       }
   }
}