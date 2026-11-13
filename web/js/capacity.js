'use strict';
(()=>{
 const $=id=>document.getElementById(id),status=$('capacity-status');let request,generation=0;
 async function refresh(){
  request?.abort();request=new AbortController();const id=++generation;
  try{
   const response=await fetch('/api/capacity',{cache:'no-store',signal:request.signal});if(!response.ok)throw Error();
   const data=await response.json();if(id!==generation)return;
   $('capacity-reserved').textContent=data.reserved;$('capacity-free').textContent=data.free;status.classList.remove('is-error');
   status.title=data.total+' aktive Lagerplätze · '+data.occupied+' belegt · '+data.reserved+' reserviert · '+data.free+' frei';
   status.setAttribute('aria-label',data.reserved+' reservierte und '+data.free+' freie Lagerplätze.');
  }catch(error){if(error.name==='AbortError'||id!==generation)return;$('capacity-reserved').textContent='?';$('capacity-free').textContent='?';status.classList.add('is-error');status.title='Lagerplatzübersicht derzeit nicht verfügbar. Wird automatisch erneut geladen.';status.setAttribute('aria-label','Lagerplatzübersicht derzeit nicht verfügbar.');}
 }
 window.addEventListener('workshop-data-changed',refresh);
 window.addEventListener('focus',()=>{if(!document.hidden)refresh()});
 document.addEventListener('visibilitychange',()=>{if(!document.hidden)refresh()});
 setInterval(()=>{if(!document.hidden)refresh()},60000);refresh();
})();
