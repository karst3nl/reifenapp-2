 'use strict';
const input=document.getElementById('search');
const results=document.getElementById('results');
const list=document.getElementById('result-list');
const count=document.getElementById('count');
let timer, controller, generation=0;
function message(text){
  list.replaceChildren();
  const p=document.createElement('p');p.className='empty';p.textContent=text;list.append(p);
}
async function search(id){
  const q=input.value.trim();
  if(!q)return;
  controller=new AbortController();
  results.classList.add('active');count.textContent='Suche läuft …';message('Daten werden geladen …');
  try {
    const response=await fetch('/api/search?q='+encodeURIComponent(q),{signal:controller.signal});
    if(!response.ok)throw new Error('HTTP '+response.status);
    const matches=await response.json();
    if(id!==generation)return;
    list.replaceChildren();count.textContent=matches.length===1?'1 Treffer':`${matches.length} Treffer`;
    if(!matches.length){message('Keine passenden Einträge gefunden.');return;}
    for(const item of matches)list.append(searchCard(item));
  }catch(error){
    if(error.name==='AbortError'||id!==generation)return;
    count.textContent='Suche fehlgeschlagen';message('Suche derzeit nicht verfügbar. Bitte erneut eingeben oder die Seite neu laden.');
  }
}
input.addEventListener('input',()=>{
  clearTimeout(timer);if(controller)controller.abort();const id=++generation;
  if(!input.value.trim()){list.replaceChildren();results.classList.remove('active');return;}
  timer=setTimeout(()=>search(id),180);
});
const routes={Administration:'/admin.html',Historie:'/history.html'};
document.querySelectorAll('[data-title]').forEach(button=>button.addEventListener('click',()=>{
 const title=button.dataset.title;if(routes[title]){location.assign(routes[title]);return;}
 const open={Auslagerung:window.openRelease,Radwechsel:window.openWheelChange,Reservierung:window.openReservation,Neueinlagerung:window.openStorage}[title];
 if(open)open();
}));
