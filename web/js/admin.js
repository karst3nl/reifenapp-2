'use strict';
(()=>{
 const $=id=>document.getElementById(id),clone=x=>JSON.parse(JSON.stringify(x));
 let csrf='',state=null,defaults=null,values={},fields=[],selected='',dirty=false,undo=[],redo=[],drag=null,previewURL='',counter=0,busy=false,loadGeneration=0;
 const status=(message,error=false)=>{$('editor-status').textContent=message;$('editor-status').classList.toggle('error',error)};
 async function api(path,options={}){
  const headers={'Content-Type':'application/json',...(options.headers||{})};if(csrf)headers['X-CSRF-Token']=csrf;
  let response;try{response=await fetch('/api/admin/'+path,{...options,headers,cache:'no-store'})}catch{throw Error('Verbindung zum Server fehlgeschlagen. Bitte erneut versuchen.');}
  if(response.status===401&&path!=='login'){showLogin();throw Error('Sitzung abgelaufen. Bitte neu anmelden.');}
  if(!response.ok){let data;try{data=await response.json()}catch{}throw Error(data?.error||'Anfrage fehlgeschlagen.');}
  return options.pdf?response.blob():response.json();
 }
 function showLogin(){loadGeneration++;csrf='';$('login-panel').hidden=false;$('editor').hidden=true;$('logout').hidden=true;if(previewURL){URL.revokeObjectURL(previewURL);previewURL=''}$('pdf-preview').removeAttribute('src');}
 function selectedElement(){return state?.template.elements.find(e=>e.id===selected)}
 function remember(){undo.push(clone(state.template));if(undo.length>80)undo.shift();redo=[];dirty=true;}
 function round(n){return Math.round(n*100)/100}
 function bounded(e){e.width=round(Math.min(210,Math.max(1,e.width)));e.height=round(Math.min(297,Math.max(1,e.height)));e.x=round(Math.max(0,Math.min(210-e.width,e.x)));e.y=round(Math.max(0,Math.min(297-e.height,e.y)));}
 function textFor(e){return (e.text+(e.type==='field'?(values[e.field]||''):'' )).replace(/\s+/g,' ').trim()}
 function paint(){
  if(!state)return;const paper=$('paper');paper.replaceChildren();
  for(const e of state.template.elements){
   const node=document.createElement('div');node.className='item'+(e.id===selected?' selected':'');node.dataset.id=e.id;node.tabIndex=0;node.setAttribute('role','button');node.setAttribute('aria-label',e.type==='field'?(fields.find(f=>f.key===e.field)?.label||e.field):e.text||e.type);
   Object.assign(node.style,{left:e.x+'mm',top:e.y+'mm',width:e.width+'mm',height:e.height+'mm',fontSize:e.font_size+'pt',fontWeight:e.bold?'700':'400',textAlign:{L:'left',C:'center',R:'right'}[e.align],color:e.color});
   if(e.type==='line'){node.classList.add('line-item');const line=document.createElement('div');line.className='line-mark';line.style.borderTop=`${e.stroke}mm solid ${e.color}`;node.append(line)}
   else if(e.type==='box'){node.style.background=e.background;node.style.border=`${e.stroke}mm solid ${e.color}`}
   else{node.textContent=textFor(e)}
   if(e.id===selected){const handle=document.createElement('span');handle.className='resize-handle';handle.setAttribute('aria-hidden','true');node.append(handle)}
   paper.append(node);
   if(e.type==='text'||e.type==='field'){
    let size=e.font_size;while(node.scrollHeight>node.clientHeight+1&&size>6){size-=0.5;node.style.fontSize=size+'pt'}
    node.title=node.scrollHeight>node.clientHeight+1?'Textfeld zu klein – in PDF-Vorschau prüfen.':'';
   }
  }
  properties();$('undo').disabled=!undo.length;$('redo').disabled=!redo.length;
  $('revision').textContent=(state.enabled?'Eigenes Layout aktiv':'Standardbeleg aktiv')+' · Revision '+state.revision+(dirty?' · Ungespeichert':'');
 }
 function properties(){const e=selectedElement();$('properties').hidden=!e;$('selection-hint').hidden=!!e;if(!e)return;
  for(const key of ['text','field','x','y','width','height','font_size','align','color','background','stroke'])$('prop-'+key).value=e[key];$('prop-bold').checked=e.bold;
  $('prop-field').disabled=e.type!=='field';for(const key of ['text','font_size','align','bold'])$('prop-'+key).disabled=e.type==='line'||e.type==='box';$('prop-background').disabled=e.type!=='box';
 }
 function zoom(){const z=Number($('zoom').value);$('paper').style.transform=`scale(${z})`;$('paper-holder').style.width=210*96/25.4*z+'px';$('paper-holder').style.height=297*96/25.4*z+'px';}
 async function load(){
  const operation=$('operation').value,generation=++loadGeneration;const [data,sample]=await Promise.all([api('layout?operation='+operation),api('values?operation='+operation)]);
  if(generation!==loadGeneration||$('operation').value!==operation)return;state=data.layout;defaults=data.default;values=sample;selected='';dirty=false;undo=[];redo=[];paint();zoom();status('Elemente ziehen oder links hinzufügen. Vorschau enthält fiktive Daten.');
 }
 async function enter(){
  $('login-panel').hidden=true;$('editor').hidden=false;$('logout').hidden=false;
  fields=await api('fields');for(const id of ['palette-field','prop-field']){$(id).replaceChildren();for(const f of fields)$(id).add(new Option(f.label,f.key))}
  if(state&&dirty){$('operation').value=state.operation;const [data,sample]=await Promise.all([api('layout?operation='+state.operation),api('values?operation='+state.operation)]);defaults=data.default;values=sample;paint();zoom();status('Sitzung erneuert. Ungespeicherter Entwurf bleibt geöffnet.');}else{await load();}await document.fonts.load('12px Receipt');await document.fonts.load('bold 12px Receipt');paint();
 }
 $('login-form').addEventListener('submit',async event=>{event.preventDefault();const input=$('password');const password=input.value;input.value='';$('login-status').textContent='Anmeldung …';try{const data=await api('login',{method:'POST',body:JSON.stringify({password})});csrf=data.csrf;await enter();$('login-status').textContent=''}catch(error){$('login-status').textContent=error.message}});
 $('logout').addEventListener('click',async()=>{if(dirty&&!confirm('Ungespeicherte Änderungen verwerfen?'))return;try{await api('logout',{method:'POST',body:'{}'});dirty=false;state=null;showLogin()}catch(error){status(error.message,true)}});
 $('operation').addEventListener('change',async()=>{if(dirty&&!confirm('Ungespeicherte Änderungen verwerfen?')){$('operation').value=state.operation;return}try{await load()}catch(error){status(error.message,true)}});
 $('reload').addEventListener('click',async()=>{if(dirty&&!confirm('Ungespeicherte Änderungen verwerfen?'))return;try{await load()}catch(error){status(error.message,true)}});
 function add(type){if(state.template.elements.length>=100){status('Maximal 100 Elemente erlaubt.',true);return}remember();let id;do{id='element-'+Date.now()+'-'+(++counter)}while(state.template.elements.some(e=>e.id===id));
  const e={id,type,text:type==='text'?'Neuer Text':'',field:type==='field'?$('palette-field').value:'',x:25,y:25,width:type==='line'?100:60,height:type==='line'?2:15,font_size:12,bold:false,align:'L',color:'#000000',background:'#ffffff',stroke:type==='line'||type==='box'?0.3:0};state.template.elements.push(e);selected=e.id;paint();}
 for(const type of ['text','field','line','box'])$('add-'+type).addEventListener('click',()=>add(type));
 $('properties').addEventListener('submit',event=>event.preventDefault());
 for(const key of ['text','field','x','y','width','height','font_size','align','color','background','stroke','bold'])$('prop-'+key).addEventListener('change',()=>{
  const e=selectedElement();if(!e)return;const control=$('prop-'+key);if(control.type==='number'&&!control.checkValidity()){status('Bitte gültige Abmessungen eingeben.',true);properties();return}
  remember();e[key]=key==='bold'?control.checked:control.type==='number'?Number(control.value):control.value;bounded(e);paint();
 });
 function remove(){const e=selectedElement();if(!e)return;if(state.template.elements.length===1){status('Mindestens ein Element behalten.',true);return}remember();state.template.elements=state.template.elements.filter(item=>item.id!==selected);selected='';paint()}
 $('delete').addEventListener('click',remove);
 $('duplicate').addEventListener('click',()=>{const e=selectedElement();if(!e||state.template.elements.length>=100)return;remember();const item={...clone(e),id:'element-'+Date.now()+'-'+(++counter),x:e.x+3,y:e.y+3};bounded(item);state.template.elements.push(item);selected=item.id;paint()});
 for(const [id,delta]of [['backward',-1],['forward',1]])$(id).addEventListener('click',()=>{const i=state.template.elements.findIndex(e=>e.id===selected);if(i<0||(delta<0&&i===0)||(delta>0&&i===state.template.elements.length-1))return;remember();const [item]=state.template.elements.splice(i,1);if(delta<0)state.template.elements.unshift(item);else state.template.elements.push(item);paint()});
 function history(back){const from=back?undo:redo,to=back?redo:undo;if(!from.length)return;to.push(clone(state.template));state.template=from.pop();selected='';dirty=true;paint()}
 $('undo').addEventListener('click',()=>history(true));$('redo').addEventListener('click',()=>history(false));
 $('reset').addEventListener('click',()=>{if(!confirm('Entwurf durch die Ausgangsvorlage ersetzen? Gespeichert wird erst mit „Speichern & aktivieren“.'))return;remember();state.template=clone(defaults);selected='';paint()});
 $('zoom').addEventListener('change',zoom);
 $('paper').addEventListener('focusin',event=>{
  const node=event.target.closest('.item');if(!node||node.dataset.id===selected)return;
  selected=node.dataset.id;for(const item of $('paper').children){item.classList.toggle('selected',item.dataset.id===selected);item.querySelector('.resize-handle')?.remove();}
  const handle=document.createElement('span');handle.className='resize-handle';handle.setAttribute('aria-hidden','true');node.append(handle);properties();
 });
 $('paper').addEventListener('pointerdown',event=>{
  if(busy||event.button!==0)return;const node=event.target.closest('.item');if(!node){selected='';paint();return}event.preventDefault();selected=node.dataset.id;const e=selectedElement();drag={moved:false,pointer:event.pointerId,startX:event.clientX,startY:event.clientY,element:clone(e),resize:event.target.classList.contains('resize-handle')};$('paper').setPointerCapture(event.pointerId);$('paper').focus();paint();
 });
 $('paper').addEventListener('pointermove',event=>{
  if(!drag||event.pointerId!==drag.pointer)return;const e=selectedElement();if(!e)return;const scale=96/25.4*Number($('zoom').value),dx=(event.clientX-drag.startX)/scale,dy=(event.clientY-drag.startY)/scale;
  if(!drag.moved){if(Math.abs(dx)+Math.abs(dy)<0.2)return;remember();drag.moved=true;}
  if(drag.resize){e.width=Math.min(210-e.x,Math.max(1,drag.element.width+dx));e.height=Math.min(297-e.y,Math.max(1,drag.element.height+dy))}else{e.x=drag.element.x+dx;e.y=drag.element.y+dy}bounded(e);paint();
 });
 for(const type of ['pointerup','pointercancel','lostpointercapture'])$('paper').addEventListener(type,()=>drag=null);
 document.addEventListener('keydown',event=>{
  if(busy||$('editor').hidden||event.target.matches('input,textarea,select'))return;
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==='z'){event.preventDefault();history(!event.shiftKey);return}
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==='y'){event.preventDefault();history(false);return}
  if(event.key==='Delete'){event.preventDefault();remove();return}
  const deltas={ArrowLeft:[-1,0],ArrowRight:[1,0],ArrowUp:[0,-1],ArrowDown:[0,1]},e=selectedElement();if(e&&deltas[event.key]){event.preventDefault();remember();const step=event.shiftKey?2:0.5;e.x+=deltas[event.key][0]*step;e.y+=deltas[event.key][1]*step;bounded(e);paint()}
 });
 async function persist(enabled){if(busy)return;busy=true;$('editor').inert=true;$('logout').disabled=true;$('save').disabled=true;$('standard').disabled=true;
  try{state=await api('layout',{method:'POST',body:JSON.stringify({...state,enabled})});dirty=false;paint();status(enabled?'Layout gespeichert und für neue PDF-Abrufe aktiviert.':'Standardbeleg aktiviert. Eigener Entwurf bleibt gespeichert.')}
  catch(error){status(error.message,true)}finally{busy=false;$('editor').inert=false;$('logout').disabled=false;$('save').disabled=false;$('standard').disabled=false}
 }
 $('save').addEventListener('click',()=>persist(true));$('standard').addEventListener('click',()=>persist(false));
 $('preview').addEventListener('click',async()=>{try{const blob=await api('preview',{method:'POST',body:JSON.stringify(state),pdf:true});if(previewURL)URL.revokeObjectURL(previewURL);previewURL=URL.createObjectURL(blob);$('pdf-preview').src=previewURL;$('preview-panel').hidden=false;$('preview-panel').scrollIntoView({behavior:'smooth'});status('Echte PDF-Vorschau erzeugt. Noch nicht gespeicherte Änderungen sind enthalten.')}catch(error){status(error.message,true)}});
 $('close-preview').addEventListener('click',()=>{$('preview-panel').hidden=true;$('pdf-preview').removeAttribute('src');if(previewURL){URL.revokeObjectURL(previewURL);previewURL=''}});
 window.addEventListener('beforeunload',event=>{if(dirty){event.preventDefault();event.returnValue=''}});
 (async()=>{try{const response=await fetch('/api/admin/session',{cache:'no-store'});const data=await response.json();if(response.ok){csrf=data.csrf;await enter()}else if(response.status===503){$('login-status').textContent='Administration ist noch nicht eingerichtet. Hinweise stehen in der Startanleitung.';$('login-form').hidden=true}}catch{$('login-status').textContent='Verbindung zum Server nicht möglich.'}})();
})();
