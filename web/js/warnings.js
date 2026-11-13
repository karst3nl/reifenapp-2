'use strict';
(()=>{
 const $=id=>document.getElementById(id),dialog=$('warnings-dialog'),button=$('warnings-button');
 let scope='active',badgeRequest,listRequest,badgeGeneration=0,listGeneration=0;
 async function fetchWarnings(scope,signal){
  const response=await fetch('/api/warnings?scope='+scope,{cache:'no-store',signal});
  if(!response.ok)throw Error('Reifenwarnungen konnten nicht geladen werden. Bitte erneut versuchen.');
  return response.json();
 }
 function setBadge(data){$('warnings-count').textContent=data.count;button.classList.remove('is-error');button.setAttribute('aria-label',`Reifenwarnungen: ${data.count} aktuell eingelagerte Radsätze. Übersicht öffnen.`);button.title='Aktuell eingelagerte Radsätze mit niedriger Profiltiefe';}
 async function refreshBadge(){
  const generation=++badgeGeneration;badgeRequest?.abort();badgeRequest=new AbortController();
  try{const data=await fetchWarnings('active',badgeRequest.signal);if(generation===badgeGeneration)setBadge(data)}
  catch(error){if(error.name==='AbortError'||generation!==badgeGeneration)return;$('warnings-count').textContent='?';button.classList.add('is-error');button.setAttribute('aria-label','Reifenwarnungen derzeit nicht abrufbar. Übersicht öffnen und erneut versuchen.');button.title='Warnungszähler derzeit nicht verfügbar';}
 }
 async function loadWarnings(){
  const generation=++listGeneration;listRequest?.abort();listRequest=new AbortController();
  $('warnings-list').replaceChildren();$('warnings-list').setAttribute('aria-busy','true');$('warnings-status').classList.remove('is-error');$('warnings-status').textContent='Warnungen werden geladen …';
  $('warnings-active').setAttribute('aria-pressed',String(scope==='active'));$('warnings-all').setAttribute('aria-pressed',String(scope==='all'));
  try{
   const data=await fetchWarnings(scope,listRequest.signal);if(generation!==listGeneration||!dialog.open)return;
   if(scope==='active')setBadge(data);
   $('warnings-status').textContent=data.count?`${data.count} ${scope==='active'?'aktuell eingelagerte Radsätze':'Vorgänge inklusive Historie'} mit Verschleißwarnung · ${data.axle_count} betroffene Achsen`:(scope==='active'?'Keine Verschleißwarnungen für aktuell eingelagerte Radsätze.':'Keine gespeicherten Vorgänge mit Verschleißwarnung.');
   for(const item of data.items)$('warnings-list').append(searchCard(item));
  }catch(error){if(error.name==='AbortError'||generation!==listGeneration)return;$('warnings-status').textContent=error.message;$('warnings-status').classList.add('is-error');}
  finally{if(generation===listGeneration)$('warnings-list').setAttribute('aria-busy','false');}
 }
 button.addEventListener('click',()=>{scope='active';dialog.showModal();dialog.scrollTop=0;loadWarnings();refreshBadge();});
 $('warnings-close').addEventListener('click',()=>dialog.close());
 dialog.addEventListener('close',()=>{listGeneration++;listRequest?.abort();afterDialogFade(dialog,()=>{$('warnings-list').replaceChildren();$('warnings-list').setAttribute('aria-busy','false');});});
 for(const [id,value] of [['warnings-active','active'],['warnings-all','all']])$(id).addEventListener('click',()=>{scope=value;loadWarnings();});
 $('warnings-reload').addEventListener('click',()=>{loadWarnings();refreshBadge();});
 function refresh(){refreshBadge();if(dialog.open)loadWarnings();}
 window.addEventListener('workshop-data-changed',refresh);
 window.addEventListener('focus',()=>{if(!document.hidden)refresh();});
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh();});
 setInterval(()=>{if(!document.hidden)refreshBadge();},60000);
 refreshBadge();
})();
