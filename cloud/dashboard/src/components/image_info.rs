
use yew::{html, Component,ComponentLink,Html,ShouldRender,Properties};

pub struct ImageDetail{
   props: Props,
}

#[derive(Properties, Clone)]
pub struct Props {
    pub image_name: String,
}

pub enum Msg {}

impl Component for ImageDetail{
    type Message = Msg;
    type Properties = Props;
    
    fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
        ImageDetail{
            props,
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
            { "this is image info" }
            { self.props.image_name.to_string() }
            </div>
        }
    }
}

impl ImageDetail{
   fn detail(&self) -> Html {
       html! {
           <div class="navbar-brand">
           </div>
       }
   }
}