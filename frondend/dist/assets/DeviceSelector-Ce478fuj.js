import{u as V}from"./devices-TgUzsxsg.js";import{n as p,u as C,R as _,w as h,g,j as y,r as b,H as M,d as x,f as B,F as A,A as D,E as F,b as k}from"./index-dni9EJ_4.js";/**
 * @license @tabler/icons-vue v3.44.0 - MIT
 *
 * This source code is licensed under the MIT license.
 * See the LICENSE file in the root directory of this source tree.
 */var w={outline:{xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"none",stroke:"currentColor","stroke-width":2,"stroke-linecap":"round","stroke-linejoin":"round"},filled:{xmlns:"http://www.w3.org/2000/svg",width:24,height:24,viewBox:"0 0 24 24",fill:"currentColor",stroke:"none"}};/**
 * @license @tabler/icons-vue v3.44.0 - MIT
 *
 * This source code is licensed under the MIT license.
 * See the LICENSE file in the root directory of this source tree.
 */const N=(r,m,n,o)=>({color:u="currentColor",size:t=24,stroke:d=2,title:c,class:f,...e},{attrs:l,slots:a})=>{let s=[...o.map(v=>p(...v)),...a.default?[a.default()]:[]];return c&&(s=[p("title",c),...s]),p("svg",{...w[r],width:t,height:t,...l,class:["tabler-icon",`tabler-icon-${m}`],...r==="filled"?{fill:u}:{"stroke-width":d??w[r]["stroke-width"],stroke:u},...e},s)};/**
 * @license @tabler/icons-vue v3.44.0 - MIT
 *
 * This source code is licensed under the MIT license.
 * See the LICENSE file in the root directory of this source tree.
 */var P=N("outline","charging-pile","ChargingPile",[["path",{d:"M18 7l-1 1",key:"svg-0"}],["path",{d:"M14 11h1a2 2 0 0 1 2 2v3a1.5 1.5 0 0 0 3 0v-7l-3 -3",key:"svg-1"}],["path",{d:"M4 20v-14a2 2 0 0 1 2 -2h6a2 2 0 0 1 2 2v14",key:"svg-2"}],["path",{d:"M9 11.5l-1.5 2.5h3l-1.5 2.5",key:"svg-3"}],["path",{d:"M3 20l12 0",key:"svg-4"}],["path",{d:"M4 8l10 0",key:"svg-5"}]]);const S={class:"device-selector"},j=["value"],E={key:0,value:"",disabled:""},L=["value"],I={__name:"DeviceSelector",props:{modelValue:String,devices:Array},emits:["update:modelValue"],setup(r,{emit:m}){const n=r,o=m,u=V(),t=k(()=>n.devices||u.list),d=k(()=>{var l;const e=n.modelValue||((l=t.value[0])==null?void 0:l.id);return t.value.find(a=>a.id===e)||null});function c(e){o("update:modelValue",e)}C(()=>{!n.modelValue&&t.value.length&&o("update:modelValue",t.value[0].id)}),_(t,e=>{e.length&&!n.modelValue&&o("update:modelValue",e[0].id)});function f(e,l){return l?e==="Available"?"clr-green":e==="Charging"||e==="Preparing"||e==="Finishing"?"clr-amber":e==="Faulted"?"clr-red":"clr-gray":"clr-gray"}return(e,l)=>{var a,s,v;return h(),g("div",S,[y(M(P),{size:20,stroke:2,class:b(f((a=d.value)==null?void 0:a.status,(s=d.value)==null?void 0:s.online))},null,8,["class"]),x("select",{value:r.modelValue||((v=t.value[0])==null?void 0:v.id),onChange:l[0]||(l[0]=i=>c(i.target.value)),class:"ds-select"},[t.value.length?B("",!0):(h(),g("option",E,"— no devices —")),(h(!0),g(A,null,D(t.value,i=>(h(),g("option",{key:i.id,value:i.id},F(i.name),9,L))),128))],40,j)])}}};export{I as _};
