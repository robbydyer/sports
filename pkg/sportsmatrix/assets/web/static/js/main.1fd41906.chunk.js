(this.webpackJsonpweb=this.webpackJsonpweb||[]).push([[0],{61:function(t,e,n){"use strict";n.r(e);var a=n(1),c=n(0),s=n.n(c),r=n(32),o=n.n(r),i=n(11),l=n(12),u=n(14),h=n(13),d=(n(23),n(8)),j=n.n(d),b=n(10),p=n(20),f=n(16),O=n(7),x=n(5),m=n(9),w="http://"+window.location.host;function v(t,e){return k.apply(this,arguments)}function k(){return(k=Object(b.a)(j.a.mark((function t(e,n){var a,c;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,fetch("".concat(w,"/api/").concat(e),{method:"GET",mode:"cors"});case 2:return a=t.sent,t.next=5,a.text();case 5:c=t.sent,a.ok&&n("true"===c);case 7:case"end":return t.stop()}}),t)})))).apply(this,arguments)}function g(t){console.log("Calling matrix API ".concat(t));var e=fetch("".concat(w,"/api/").concat(t),{method:"GET",mode:"cors"});console.log("Response",e.ok)}function y(t,e){var n={method:"POST",headers:{"Content-Type":"application/json"},body:e};console.log("Matrix POST ".concat(e)),fetch("".concat(w,"/api/").concat(t),n)}function S(){return(S=Object(b.a)(j.a.mark((function t(e){var n,a;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,fetch("".concat(w,"/api/version"),{method:"GET",mode:"cors"});case 2:return n=t.sent,t.next=5,n.text();case 5:a=t.sent,e(a);case 7:case"end":return t.stop()}}),t)})))).apply(this,arguments)}var C=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.state={screen:!1,webboard:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("status",(function(t){e.setState({screen:t})}));case 2:return t.next=4,v("webboardstatus",(function(t){e.setState({webboard:t})}));case 4:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"render",value:function(){var t=this;return Object(a.jsx)(f.a,{fluid:!0,children:Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsxs)(x.a,{children:[Object(a.jsx)(m.a.Switch,{id:"screen",label:"Screen On/Off",checked:this.state.screen,onChange:function(){return t.handleSwitch("screenon","screenoff","screen")}}),Object(a.jsx)(m.a.Switch,{id:"webboard",label:"Web Board On/Off",checked:this.state.webboard,onChange:function(){return t.handleSwitch("webboardon","webboardoff","webboard")}})]})})})}}]),n}(s.a.Component),N=n(24),T=n(21),A=n.p+"static/media/nhllogo.ba9e188b.jpeg",E=n.p+"static/media/mlb.c8036288.png",J=n.p+"static/media/ncaam.2b001451.png",L=n.p+"static/media/nba.5e388b8b.png",M=n.p+"static/media/nfl.6d694f80.png",Q=n.p+"static/media/mls.9317ef90.png",I=n.p+"static/media/ncaaf.903eccc0.png",U=n.p+"static/media/epl.49b03d71.png",W=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.handleJump=function(t){y("jump",'{"board":"'.concat(t,'"}')),a.updateStatus()},a.state={enabled:!1,hideFavorite:!1,stickyFavorite:!1,stats:!1,scroll:!1,statscroll:!1,tightscroll:!1,record:!1,odds:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("".concat(this.props.sport,"/status"),(function(t){e.setState({enabled:t})}));case 2:return t.next=4,v("".concat(this.props.sport,"/scrollstatus"),(function(t){e.setState({scroll:t})}));case 4:return t.next=6,v("".concat(this.props.sport,"/tightscrollstatus"),(function(t){e.setState({tightscroll:t})}));case 6:return t.next=8,v("".concat(this.props.sport,"/stats/status"),(function(t){e.setState({stats:t})}));case 8:return t.next=10,v("".concat(this.props.sport,"/stats/scrollstatus"),(function(t){e.setState({statscroll:t})}));case 10:return t.next=12,v("".concat(this.props.sport,"/favoritescorestatus"),(function(t){e.setState({hideFavorite:t})}));case 12:return t.next=14,v("".concat(this.props.sport,"/favoritestickystatus"),(function(t){e.setState({stickyFavorite:t})}));case 14:return t.next=16,v("".concat(this.props.sport,"/oddsstatus"),(function(t){e.setState({odds:t})}));case 16:return t.next=18,v("".concat(this.props.sport,"/recordrankstatus"),(function(t){e.setState({record:t})}));case 18:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"logosrc",value:function(){return"nhl"==this.props.sport?A:"ncaam"==this.props.sport?J:"ncaaf"==this.props.sport?I:"nba"==this.props.sport?L:"nfl"==this.props.sport?M:"mls"==this.props.sport?Q:"epl"==this.props.sport?U:E}},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:this.logosrc(),style:{height:"100px",width:"auto"},onClick:function(){return t.handleJump(t.props.sport)},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"enabler",label:"Enable/Disable",checked:this.state.enabled,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/enable"),"".concat(t.props.sport,"/disable"),"enabled")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"scroller",label:"Scroll Mode",checked:this.state.scroll,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/scrollon"),"".concat(t.props.sport,"/scrolloff"),"scroll")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"tightscroller",label:"Back-to-back Scroll Mode",checked:this.state.tightscroll,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/tightscrollon"),"".concat(t.props.sport,"/tightscrolloff"),"tightscroll")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"stats",label:"Stats",checked:this.state.stats,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/stats/enable"),"".concat(t.props.sport,"/stats/disable"),"stats")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"statscroll",label:"Stats Scroll Mode",checked:this.state.statscroll,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/stats/scrollon"),"".concat(t.props.sport,"/stats/scrolloff"),"statscroll")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"favscore",label:"Hide Favorite Scores",checked:this.state.hideFavorite,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/hidefavoritescore"),"".concat(t.props.sport,"/showfavoritescore"),"hideFavorite")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"odds",label:"Show Odds",checked:this.state.odds,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/oddson"),"".concat(t.props.sport,"/oddsoff"),"odds")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"record",label:"Record + Rank",checked:this.state.record,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/recordrankon"),"".concat(t.props.sport,"/recordrankoff"),"record")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:this.props.sport+"favstick",label:"Stick Favorite Live Games",checked:this.state.stickyFavorite,onChange:function(){return t.handleSwitch("".concat(t.props.sport,"/favoritesticky"),"".concat(t.props.sport,"/favoriteunstick"),"stickyFavorite")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(N.a,{variant:"primary",onClick:function(){return t.handleJump(t.props.sport)},children:"Jump"})})})]})}}]),n}(s.a.Component),G=n.p+"static/media/pga.f4df3969.png",H=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.state={stats:!1,scroll:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("pga/stats/status",(function(t){e.setState({stats:t})}));case 2:return t.next=4,v("pga/stats/scrollstatus",(function(t){e.setState({scroll:t})}));case 4:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"logosrc",value:function(){return G}},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:this.logosrc(),style:{height:"100px",width:"auto"},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"pgastats",label:"Enable/Disable",checked:this.state.stats,onChange:function(){return t.handleSwitch("pga/stats/enable","pga/stats/disable","stats")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"pgascroll",label:"Scroll Mode",checked:this.state.scroll,onChange:function(){return t.handleSwitch("pga/stats/scrollon","pga/stats/scrolloff","scroll")}})})})]})}}]),n}(s.a.Component),D=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.handleJump=function(t){y("jump",'{"board":"'.concat(t,'"}')),a.updateStatus()},a.state={enabled:!1,memcache:!1,diskcache:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("img/status",(function(t){e.setState({enabled:t})}));case 2:return t.next=4,v("img/memcachestatus",(function(t){e.setState({memcache:t})}));case 4:return t.next=6,v("img/diskcachestatus",(function(t){e.setState({diskcache:t})}));case 6:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:"data:image/png;base64,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",style:{height:"100px",width:"auto"},onClick:function(){return t.handleJump("img")},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"imgenabler",label:"Enable/Disable",checked:this.state.enabled,onChange:function(){return t.handleSwitch("img/enable","img/disable","enabled")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"imgmem",label:"Enable Memory Cache",checked:this.state.memcache,onChange:function(){return t.handleSwitch("img/enablememcache","img/disablememcache","memcache")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"imgdisk",label:"Enable Disk Cache",checked:this.state.diskcache,onChange:function(){return t.handleSwitch("img/enablediskcache","img/disablediskcache","diskcache")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(N.a,{variant:"primary",onClick:function(){return t.handleJump("img")},children:"Jump"})})})]})}}]),n}(s.a.Component),X=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.handleJump=function(t){y("jump",'{"board":"'.concat(t,'"}')),a.updateStatus()},a.state={enabled:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("clock/status",(function(t){e.setState({enabled:t})}));case 2:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:"data:image/png;base64,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",style:{height:"100px",width:"auto"},onClick:function(){return t.handleJump("clock")},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"clockenabler",label:"Enable/Disable",checked:this.state.enabled,onChange:function(){return t.handleSwitch("clock/enable","clock/disable","enabled")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(N.a,{variant:"primary",onClick:function(){return t.handleJump("clock")},children:"Jump"})})})]})}}]),n}(s.a.Component),K="http://"+window.location.host,q=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).state={t:Date.now()},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=this;document.body.style.backgroundColor="black",this.interval=setInterval((function(){return t.setState({t:Date.now()})}),2e3)}},{key:"componentWillUnmount",value:function(){clearInterval(this.interval),fetch("".concat(K,"/api/imgcanvas/disable"),{method:"GET",mode:"cors"}),document.body.style.backgroundColor="white"}},{key:"render",value:function(){return Object(a.jsxs)(a.Fragment,{children:[Object(a.jsx)("div",{style:{backgroundColor:"black"}}),Object(a.jsx)(f.a,{fluid:!0,children:Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:"".concat(K,"/api/imgcanvas/board?").concat(this.state.t),style:{height:"auto",width:"auto"},name:this.state.t,fluid:!0})})})})]})}}]),n}(s.a.Component),V=n.p+"static/media/server.53a70363.png",Z=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.handleJump=function(t){y("jump",'{"board":"'.concat(t,'"}')),a.updateStatus()},a.state={enabled:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("sys/status",(function(t){e.setState({enabled:t})}));case 2:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:V,style:{height:"100px",width:"auto"},onClick:function(){return t.handleJump("sys")},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"sysenabler",label:"Enable/Disable",checked:this.state.enabled,onChange:function(){return t.handleSwitch("sys/enable","sys/disable","enabled")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(N.a,{variant:"primary",onClick:function(){return t.handleJump("sys")},children:"Jump"})})})]})}}]),n}(s.a.Component),F=n.p+"static/media/stock.2479f885.png",R=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.handleJump=function(t){y("jump",'{"board":"'.concat(t,'"}')),a.updateStatus()},a.state={enabled:!1,scroll:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("stocks/status",(function(t){e.setState({enabled:t})}));case 2:return t.next=4,v("stocks/scrollstatus",(function(t){e.setState({scroll:t})}));case 4:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:F,style:{height:"100px",width:"auto"},onClick:function(){return t.handleJump("stocks")},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"stocksenabler",label:"Enable/Disable",checked:this.state.enabled,onChange:function(){return t.handleSwitch("stocks/enable","stocks/disable","enabled")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"stocksscroller",label:"Scroll Mode",checked:this.state.scroll,onChange:function(){return t.handleSwitch("stocks/scrollon","stocks/scrolloff","scroll")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(N.a,{variant:"primary",onClick:function(){return t.handleJump("stocks")},children:"Jump"})})})]})}}]),n}(s.a.Component),Y=n.p+"static/media/weather.0b8414b1.png",B=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).handleSwitch=function(t,e,n){var c=a.state[n];console.log("handle switch",c),c?(console.log("Turn off",e),g(e)):(console.log("Turn on",t),g(t)),a.setState((function(t){return Object(p.a)({},n,!t[n])}))},a.handleJump=function(t){y("jump",'{"board":"'.concat(t,'"}')),a.updateStatus()},a.state={enabled:!1,scroll:!1,daily:!1,hourly:!1},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=Object(b.a)(j.a.mark((function t(){return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:this.updateStatus();case 1:case"end":return t.stop()}}),t,this)})));return function(){return t.apply(this,arguments)}}()},{key:"updateStatus",value:function(){var t=Object(b.a)(j.a.mark((function t(){var e=this;return j.a.wrap((function(t){for(;;)switch(t.prev=t.next){case 0:return t.next=2,v("weather/status",(function(t){e.setState({enabled:t})}));case 2:return t.next=4,v("weather/scrollstatus",(function(t){e.setState({scroll:t})}));case 4:return t.next=6,v("weather/dailystatus",(function(t){e.setState({daily:t})}));case 6:return t.next=8,v("weather/hourlystatus",(function(t){e.setState({hourly:t})}));case 8:case"end":return t.stop()}}),t)})));return function(){return t.apply(this,arguments)}}()},{key:"render",value:function(){var t=this;return Object(a.jsxs)(f.a,{fluid:!0,children:[Object(a.jsx)(O.a,{className:"text-center",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(T.a,{src:Y,style:{height:"100px",width:"auto"},onClick:function(){return t.handleJump("weather")},fluid:!0})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"weatherenabler",label:"Enable/Disable",checked:this.state.enabled,onChange:function(){return t.handleSwitch("weather/enable","weather/disable","enabled")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"weatherscroller",label:"Scroll Mode",checked:this.state.scroll,onChange:function(){return t.handleSwitch("weather/scrollon","weather/scrolloff","scroll")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"dailyenabler",label:"Daily Forecast",checked:this.state.daily,onChange:function(){return t.handleSwitch("weather/dailyenable","weather/dailydisable","daily")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(m.a.Switch,{id:"hourlyenabler",label:"Hourly Forecast",checked:this.state.hourly,onChange:function(){return t.handleSwitch("weather/hourlyenable","weather/hourlydisable","hourly")}})})}),Object(a.jsx)(O.a,{className:"text-left",children:Object(a.jsx)(x.a,{children:Object(a.jsx)(N.a,{variant:"primary",onClick:function(){return t.handleJump("weather")},children:"Jump"})})})]})}}]),n}(s.a.Component),P=n(33),z=n(18),_=n(17),$=n(15),tt=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(t){var a;return Object(i.a)(this,n),(a=e.call(this,t)).state={version:""},a}return Object(l.a)(n,[{key:"componentDidMount",value:function(){var t=this;""===this.state.version&&(console.log("fetching version"),function(t){S.apply(this,arguments)}((function(e){t.setState({version:e})})))}},{key:"render",value:function(){return Object(a.jsx)(f.a,{fluid:!0,children:Object(a.jsxs)(P.a,{expand:"sm",bg:"dark",variant:"dark",hidden:"/board"===this.props.location.pathname,children:[Object(a.jsx)(P.a.Brand,{children:"SportsMatrix"}),Object(a.jsx)(P.a.Toggle,{"aria-controls":"basic-navbar-nav"}),Object(a.jsxs)(P.a.Collapse,{id:"basic-navbar-nav",children:[Object(a.jsxs)(z.a,{className:"mr-auto",children:[Object(a.jsx)(z.a.Link,{as:_.b,to:"/",children:"Home"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/nhl",children:"NHL"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/ncaaf",children:"NCAA Football"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/mlb",children:"MLB"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/pga",children:"PGA"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/ncaam",children:"NCAA Men Basketball"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/nfl",children:"NFL"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/nba",children:"NBA"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/mls",children:"MLS"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/epl",children:"EPL"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/img",children:"Image Board"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/clock",children:"Clock"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/sys",children:"System Info"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/stocks",children:"Stocks"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/weather",children:"Weather"}),Object(a.jsx)(z.a.Link,{as:_.b,to:"/board",children:"Live Board"})]}),Object(a.jsx)(P.a.Text,{children:this.state.version})]})]})})}}]),n}(s.a.Component),et=Object($.e)(tt),nt=n(29),at={row:{marginTop:10},col:{paddingTop:"20px"}},ct="18rem",st=["ncaaf","nhl","mlb","ncaam","nfl","nba","mls","epl"].map((function(t){return Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(W,{sport:t,id:t},t)})})})),rt=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(){return Object(i.a)(this,n),e.apply(this,arguments)}return Object(l.a)(n,[{key:"render",value:function(){return Object(a.jsx)(f.a,{fluid:"xl",children:Object(a.jsxs)(O.a,{className:"justify-content-md-space-between",sm:1,lg:2,xl:3,style:at.row,children:[Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(C,{})})}),st,Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(H,{})})}),Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(B,{id:"weatherboard"})})}),Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(D,{id:"imgboard"})})}),Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(R,{id:"stocks"})})}),Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(X,{id:"clock"})})}),Object(a.jsx)(x.a,{lg:"auto",style:at.col,children:Object(a.jsx)(nt.a,{style:{width:{card_border:ct}},children:Object(a.jsx)(Z,{id:"sys"})})})]})})}}]),n}(s.a.Component),ot="http://"+window.location.host,it=function(t){Object(u.a)(n,t);var e=Object(h.a)(n);function n(){return Object(i.a)(this,n),e.apply(this,arguments)}return Object(l.a)(n,[{key:"screenOn",value:function(){console.log("Turning screen on"),fetch("".concat(ot,"/api/screenon"),{method:"GET",mode:"cors"})}},{key:"screenOff",value:function(){console.log("Turning screen off"),fetch("".concat(ot,"/api/screenoff"),{method:"GET",mode:"cors"})}},{key:"render",value:function(){return Object(a.jsxs)(a.Fragment,{children:[Object(a.jsxs)(_.a,{children:[Object(a.jsx)(et,{}),Object(a.jsx)($.a,{path:"/",exact:!0,component:rt}),Object(a.jsx)($.a,{path:"/nhl",render:function(){return Object(a.jsx)(W,{sport:"nhl"})}}),Object(a.jsx)($.a,{path:"/mlb",render:function(){return Object(a.jsx)(W,{sport:"mlb"})}}),Object(a.jsx)($.a,{path:"/pga",render:function(){return Object(a.jsx)(H,{})}}),Object(a.jsx)($.a,{path:"/ncaam",render:function(){return Object(a.jsx)(W,{sport:"ncaam"})}}),Object(a.jsx)($.a,{path:"/ncaaf",render:function(){return Object(a.jsx)(W,{sport:"ncaaf"})}}),Object(a.jsx)($.a,{path:"/nba",render:function(){return Object(a.jsx)(W,{sport:"nba"})}}),Object(a.jsx)($.a,{path:"/nfl",render:function(){return Object(a.jsx)(W,{sport:"nfl"})}}),Object(a.jsx)($.a,{path:"/mls",render:function(){return Object(a.jsx)(W,{sport:"mls"})}}),Object(a.jsx)($.a,{path:"/epl",render:function(){return Object(a.jsx)(W,{sport:"epl"})}}),Object(a.jsx)($.a,{path:"/img",exact:!0,component:D}),Object(a.jsx)($.a,{path:"/clock",exact:!0,component:X}),Object(a.jsx)($.a,{path:"/sys",exact:!0,component:Z}),Object(a.jsx)($.a,{path:"/stocks",exact:!0,component:R}),Object(a.jsx)($.a,{path:"/weather",exact:!0,component:B}),Object(a.jsx)($.a,{path:"/board",exact:!0,component:q})]}),Object(a.jsx)("hr",{})]})}}]),n}(s.a.Component),lt=function(t){t&&t instanceof Function&&n.e(3).then(n.bind(null,63)).then((function(e){var n=e.getCLS,a=e.getFID,c=e.getFCP,s=e.getLCP,r=e.getTTFB;n(t),a(t),c(t),s(t),r(t)}))};o.a.render(Object(a.jsx)(s.a.StrictMode,{children:Object(a.jsx)(it,{})}),document.getElementById("root")),lt()}},[[61,1,2]]]);
//# sourceMappingURL=main.1fd41906.chunk.js.map