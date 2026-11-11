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
    for(const item of matches){
      const card=document.createElement('div');card.className='result';
      const info=document.createElement('div');const title=document.createElement('strong');title.textContent=item.name||'Ohne Kundenzuordnung';
      const meta=document.createElement('small');meta.textContent=[item.plate,item.car].filter(Boolean).join(' · ')||'Keine Fahrzeugangaben im Altbestand';
      const event=document.createElement('small');
      const labels={storage:'Neueinlagerung',wheel_change:'Radwechsel',reservation:'Reservierung',release:'Auslagerung'};
      const state=item.operation==='release'?'Ausgelagert':item.active?(item.kind==='reservation'?'Reserviert':'Eingelagert'):item.status==='converted'?'In Einlagerung umgewandelt':'Abgeschlossen';
      const date=new Date(item.date);
      const stamp=Number.isNaN(date.getTime())?'Zeitpunkt unbekannt':date.toLocaleString('de-DE');
      event.textContent=`${labels[item.operation]||'Vorgang'} · ${state} · ${stamp}${item.source==='legacy'?' · Altbestand':''}`;
      info.append(title,meta,event);
      if(item.pdf_url){const link=document.createElement('a');link.href=item.pdf_url;link.target='_blank';link.rel='noopener noreferrer';link.textContent='Beleg als PDF';info.append(link);}
      const place=document.createElement('span');place.className='place';place.textContent=item.place;
      card.append(info,place);list.append(card);
    }
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
  const dialog=document.getElementById('placeholder');
  document.querySelectorAll('[data-title]').forEach(button=>button.addEventListener('click',()=>{if(button.dataset.title==='Auslagerung'){window.openRelease();return;}if(button.dataset.title==='Radwechsel'){window.openWheelChange();return;}if(button.dataset.title==='Reservierung'){window.openReservation();return;}if(button.dataset.title==='Neueinlagerung'){window.openStorage();return;}document.getElementById('modal-title').textContent=button.dataset.title;dialog.showModal();}));
  document.getElementById('close-modal').addEventListener('click',()=>dialog.close());
