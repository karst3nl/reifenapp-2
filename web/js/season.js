'use strict';
(()=>{
 const $=id=>document.getElementById(id);let request,generation=0;
 async function refresh(){
  request?.abort();request=new AbortController();const id=++generation;
  try{
   const response=await fetch('/api/season',{cache:'no-store',signal:request.signal});if(!response.ok)throw Error();
   const data=await response.json();if(id!==generation)return;
   $('season-label').textContent=data.season.label;
   $('season-count').textContent=data.completed+' / '+data.total+' gewechselt';
   const details=[data.open+' offen'];if(data.unknown)details.push(data.unknown+' unklar');if(data.all_season)details.push(data.all_season+' Ganzjahr');
   $('season-details').textContent=details.join(' · ');$('season-status').classList.remove('is-error');
   const explanation='Aktueller Lagerbestand. Als gewechselt zählen Radwechsel dieser Saison mit eingelagerten '+data.season.stored_type+'reifen. '+data.open+' Radsätze mit '+data.season.mounted_type+'reifen sind noch eingelagert. '+data.unknown+' unklare Bestände und '+data.all_season+' Ganzjahressätze sind nicht im Nenner enthalten.';
   $('season-status').title=explanation;$('season-status').setAttribute('aria-label',data.season.label+': '+data.completed+' von '+data.total+' gewechselt. '+details.join(', ')+'. '+explanation);
  }catch(error){if(error.name==='AbortError'||id!==generation)return;$('season-label').textContent='Saisonstand';$('season-count').textContent='Nicht verfügbar';$('season-details').textContent='Wird erneut geladen';$('season-status').classList.add('is-error');$('season-status').title='Saisonstand konnte nicht geladen werden. Bitte Verbindung prüfen oder Seite neu laden.';$('season-status').setAttribute('aria-label','Saisonstand derzeit nicht verfügbar.');}
 }
 window.addEventListener('workshop-data-changed',refresh);
 window.addEventListener('focus',()=>{if(!document.hidden)refresh()});
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()});
 setInterval(()=>{if(!document.hidden)refresh()},60000);refresh();
})();
