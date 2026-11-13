'use strict';
(()=>{
 const $=id=>document.getElementById(id),dialog=$('inactivity-dialog'),button=$('inactivity-button');
 let request,generation=0;
 async function refresh(){
  request?.abort();request=new AbortController();const id=++generation;
  if(dialog.open){$('inactivity-list').setAttribute('aria-busy','true');$('inactivity-status').textContent='Lagerbewegungen werden geprüft …';}
  try{
   const response=await fetch('/api/inactivity',{cache:'no-store',signal:request.signal});if(!response.ok)throw Error('Lagerbewegungen konnten nicht geladen werden. Bitte erneut versuchen.');
   const data=await response.json();if(id!==generation)return;
   $('inactivity-count').textContent=data.count;button.classList.remove('is-error');button.classList.toggle('has-notices',data.count>0);
   button.title='Belegte Plätze seit mindestens '+data.months+' Monaten ohne erfasste Bewegung'+(data.unknown?' · '+data.unknown+' ohne verlässlichen Zeitpunkt':'');
   button.setAttribute('aria-label',data.count+' Lagerplätze seit mindestens '+data.months+' Monaten ohne Bewegung. '+data.unknown+' ohne verlässlichen Zeitpunkt. Übersicht öffnen.');
   if(!dialog.open)return;
   $('inactivity-status').classList.remove('is-error');
   $('inactivity-status').textContent=(data.count?data.count+' belegte Plätze seit mindestens '+data.months+' Monaten ohne Bewegung.':'Keine belegten Plätze seit mindestens '+data.months+' Monaten ohne Bewegung.')+(data.unknown?' Zusätzlich '+data.unknown+' Belegungen ohne verlässlichen Bewegungszeitpunkt; keine Frist ableitbar.':'');
   $('inactivity-list').replaceChildren();
   for(const item of data.items){const card=searchCard(item),notice=document.createElement('p');notice.className='result-warning';notice.textContent='Seit '+item.days+' Tagen ohne Bewegung · Letzte erfasste Bewegung: '+new Date(item.last_movement).toLocaleString('de-DE');card.querySelector('.result-summary').after(notice);$('inactivity-list').append(card);}
  }catch(error){if(error.name==='AbortError'||id!==generation)return;$('inactivity-count').textContent='?';button.classList.add('is-error');button.title='Lagerbewegungen derzeit nicht verfügbar';button.setAttribute('aria-label','Lagerbewegungen derzeit nicht verfügbar. Übersicht öffnen und erneut versuchen.');if(dialog.open){$('inactivity-list').replaceChildren();$('inactivity-status').textContent=error.message;$('inactivity-status').classList.add('is-error');}}
  finally{if(id===generation)$('inactivity-list').setAttribute('aria-busy','false');}
 }
 button.addEventListener('click',()=>{dialog.showModal();dialog.scrollTop=0;refresh();});
 $('inactivity-close').addEventListener('click',()=>dialog.close());
 dialog.addEventListener('close',()=>{$('inactivity-list').replaceChildren();});
 $('inactivity-reload').addEventListener('click',refresh);
 window.addEventListener('workshop-data-changed',refresh);
 window.addEventListener('focus',()=>{if(!document.hidden)refresh()});
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()});
 setInterval(()=>{if(!document.hidden)refresh()},60000);refresh();
})();
