'use strict';
window.LabelPrints={
 watch(id,confirmation){
  if(!id)return;const status=document.createElement('span');status.textContent=' · Etikett zum Druck vorgemerkt.';confirmation.append(status);
  let attempts=0;
  async function poll(){
   if(!status.isConnected)return;
   try{const response=await fetch('/api/label-jobs/'+id,{cache:'no-store'});if(!response.ok)throw Error();const job=await response.json();if(!status.isConnected)return;
    if(job.status==='submitted'){status.textContent=' · Etikett an Druckwarteschlange übergeben ('+job.cups_id+').';return}
    if(['failed','uncertain','cancelled'].includes(job.status)){status.className='label-print-error';status.textContent=' · Vorgang gespeichert. Etikett '+(job.status==='uncertain'?'mit unklarer Druckübergabe':job.status==='cancelled'?'abgebrochen':'nicht gedruckt')+' – Administration → Etikettendrucker prüfen.';return}
   }catch{status.textContent=' · Vorgang gespeichert. Etikettendruckstatus derzeit nicht verfügbar – Administration prüfen.';}
   if(++attempts<15)setTimeout(poll,2000);else status.textContent=' · Vorgang gespeichert. Etikettenauftrag #'+id+' – aktuellen Status in Administration → Etikettendrucker prüfen.';
  }
  poll();
 }
};
