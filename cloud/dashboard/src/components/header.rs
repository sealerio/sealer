use yew::{html, Component,ComponentLink,Html,ShouldRender};

pub struct Header {
    // props: Props,
}

pub enum Msg {}

impl Component for Header {
    type Message = Msg;
    type Properties = ();
    
    fn create(props: Self::Properties, _: ComponentLink<Self>) -> Self {
        Header {}
    }
    
    fn update(&mut self, _msg: Self::Message) -> ShouldRender {
        true
    }
    
    fn change(&mut self, props: Self::Properties) -> ShouldRender {
        true
    }
    
    fn view(&self) -> Html {
        html! {
            <nav class="navbar is-primary block" role="navigation" aria-label="main navigation">
                { self.logo_name() }
                { self.search() }
                { self.login() }
            </nav>
        }
    }
}

impl Header {
   fn logo_name(&self) -> Html {
       html! {
           <div class="navbar-brand">
            <div class="navbar-item">
                <i class="far fa-cloud fa-2x fa-pull-left"></i>
                <strong> { "Sealer Cloud" }</strong>
            </div>
           </div>
       }
   }

   fn login(&self) -> Html {
       html! {
           <div class="navbar-end">
             <div class="navbar-item">
               <div class="botton" >
                  <i class="fab fa-github fa-2x"></i>
               </div>
             </div>
           </div>
       }
   }

   fn search(&self) -> Html {
       html! {
        <div class="nav-brand">
            <div class="navbar-item">
                <div class="control has-icons-left has-icons-right">
                    <input class="input is-success" type="text" placeholder="image name" value="" />
                    <span class="icon is-small is-left">
                    <i class="fas fa-search"></i>
                    </span>
                </div>
            </div>
         </div>
       }
   }
}