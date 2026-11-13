'use strict';
(()=>{
 const $=id=>document.getElementById(id),form=$('history-filters');
 let page=1,snapshot=0,request,generation=0,timer,totalPages=0,loaded=false;
 const labels={storage:'Neueinlagerungen',wheel_change:'Radwechsel',reservation:'Reservierungen',release:'Auslagerungen',imported:'Sonstige Altvorgänge'};
 function dateBoundary(value,nextDay=false){if(!value)return '';const [y,m,d]=value.split('-').map(Number),date=new Date(y,m-1,d);if(nextDay)date.setDate(date.getDate()+1);return date.toISOString();}
 function params(){const q=new URLSearchParams({page:String(page),page_size:$('history-size').value});for(const [key,id]of [['q','history-query'],['operation','history-operation'],['status','history-state'],['source','history-source']]){const v=$(id).value.trim();if(v)q.set(key,v)}if($('history-from').value)q.set('from',dateBoundary($('history-from').value));if($('history-until').value)q.set('until',dateBoundary($('history-until').value,true));if($('history-warnings').checked)q.set('warnings','true');if(snapshot)q.set('snapshot',String(snapshot));return q;}
 function controls(){ $('history-prev').disabled=!loaded||page<=1;$('history-next').disabled=!loaded||page>=totalPages;}
 function filterBy(value){form.reset();$('history-query').value=value;page=1;snapshot=0;load();form.scrollIntoView({behavior:'smooth',block:'start'});}
 function card(item){const node=searchCard(item);const status=node.querySelector('.result-info small');if(item.source!=='csv')status.textContent=status.textContent.replace(' · ',' · Lagerstatus heute: ');const actions=document.createElement('div');actions.className='history-card-actions';for(const [text,value]of [['Kundenverlauf',item.name],['Fahrzeugverlauf',item.plate],['Platzverlauf',item.place]]){if(!value)continue;const button=document.createElement('button');button.type='button';button.textContent=text;button.addEventListener('click',()=>filterBy(value));actions.append(button)}node.append(actions);return node;}
 async function load(){
  clearTimeout(timer);request?.abort();const id=++generation;request=new AbortController();loaded=false;controls();$('history-list').replaceChildren();$('history-list').setAttribute('aria-busy','true');$('history-stats').replaceChildren();$('history-updated').textContent='';$('history-status').classList.remove('is-error');$('history-status').textContent='Historie wird geladen …';
  try{
   if(!form.checkValidity())throw Error('Bitte gültige Filter eingeben.');
   if($('history-from').value&&$('history-until').value&&$('history-from').value>$('history-until').value)throw Error('Das Anfangsdatum muss vor oder am Enddatum liegen.');
   const response=await fetch('/api/history?'+params(),{signal:request.signal,cache:'no-store'});
   if(!response.ok){let data;try{data=await response.json()}catch{}throw Error(data?.error||'Historie konnte nicht geladen werden. Bitte erneut versuchen.');}
   const data=await response.json();if(id!==generation)return;snapshot=data.snapshot;totalPages=data.pages;
   if(data.total>0&&page>totalPages){page=totalPages;load();return;}
   for(const [key,label]of Object.entries(labels)){if(key==='imported'&&!data.operations[key])continue;const stat=document.createElement('div');stat.className='history-stat';const number=document.createElement('strong');number.textContent=data.operations[key]||0;stat.append(number,document.createTextNode(label));$('history-stats').append(stat);}
   for(const item of data.items)$('history-list').append(card(item));
   $('history-status').textContent=data.total?data.total+(data.total===1?' Vorgang · Einträge ':' Vorgänge · Einträge ')+((page-1)*data.page_size+1)+'–'+((page-1)*data.page_size+data.items.length):'Keine Vorgänge für diese Auswahl gefunden.';
   $('history-page').textContent='Seite '+page+' von '+Math.max(1,totalPages);$('history-updated').textContent='Geladen: '+new Date().toLocaleTimeString('de-DE');loaded=true;controls();
  }catch(error){if(error.name==='AbortError'||id!==generation)return;$('history-status').textContent=error.message;$('history-status').classList.add('is-error');$('history-page').textContent='Seite nicht geladen';}
  finally{if(id===generation)$('history-list').setAttribute('aria-busy','false');}
 }
 function changed(){page=1;snapshot=0;clearTimeout(timer);request?.abort();generation++;loaded=false;controls();$('history-list').replaceChildren();$('history-stats').replaceChildren();$('history-list').setAttribute('aria-busy','true');$('history-status').classList.remove('is-error');$('history-status').textContent='Filter werden angewendet …';timer=setTimeout(load,200);}
 form.addEventListener('submit',e=>{e.preventDefault();page=1;snapshot=0;load()});
 form.addEventListener('input',changed);
 $('history-reset').addEventListener('click',()=>{form.reset();page=1;snapshot=0;load()});
 $('history-reload').addEventListener('click',()=>{page=1;snapshot=0;load()});
 for(const [id,delta]of [['history-prev',-1],['history-next',1]])$(id).addEventListener('click',()=>{if(!loaded)return;page+=delta;load();$('history-status').scrollIntoView({behavior:'smooth',block:'start'});});
 load();
})();
