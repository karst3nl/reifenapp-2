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
function warningIcon(){
  const ns='http://www.w3.org/2000/svg',icon=document.createElementNS(ns,'svg');
  for(const [key,value] of Object.entries({viewBox:'0 0 24 24',width:'20',height:'20',fill:'none',stroke:'currentColor','stroke-width':'2','stroke-linecap':'round','stroke-linejoin':'round','aria-hidden':'true'}))icon.setAttribute(key,value);
  for(const d of ['M12 3 2 21h20L12 3Z','M12 9v5','M12 17h.01']){const path=document.createElementNS(ns,'path');path.setAttribute('d',d);icon.append(path)}return icon;
}
function searchCard(item){
  const card=document.createElement('article');card.className='result search-result';
  const header=document.createElement('div');header.className='result-summary';
  const info=document.createElement('div');info.className='result-info';
  const title=document.createElement('strong');
  const identity=[item.name||'Ohne Kundenzuordnung',item.plate,item.car].filter(Boolean);
  if(!item.plate&&!item.car)identity.push('Keine Fahrzeugangaben im Altbestand');
  title.textContent=identity.join(' · ');
  const event=document.createElement('small');
  const labels={storage:'Neueinlagerung',wheel_change:'Radwechsel',reservation:'Reservierung',release:'Auslagerung'};
  const state=item.operation==='release'?'Ausgelagert':item.active?(item.kind==='reservation'?'Reserviert':'Eingelagert'):item.status==='converted'?'In Einlagerung umgewandelt':'Abgeschlossen';
  const date=new Date(item.date),stamp=Number.isNaN(date.getTime())?'Zeitpunkt unbekannt':date.toLocaleString('de-DE');
  event.textContent=`${labels[item.operation]||'Vorgang'} · ${state} · ${stamp}${item.source==='legacy'?' · Altbestand':''}`;
  info.append(title,event);
  if(item.pdf_url){const link=document.createElement('a');link.href=item.pdf_url;link.target='_blank';link.rel='noopener noreferrer';link.textContent='Beleg als PDF';info.append(link);}
  const place=document.createElement('span');place.className='place';place.textContent=item.place;
  header.append(info,place);card.append(header);
  const data=item.details||{},warnings=document.createElement('div');warnings.className='result-warnings';
  const warn=(text,kind)=>{const row=document.createElement('p');row.className='result-warning '+kind;const label=document.createElement('span');label.textContent=text;row.append(warningIcon(),label);warnings.append(row)};
  if(data.notes?.trim())warn('Hinweis: '+data.notes,'notes-warning');
  for(const warning of item.tread_warnings||[]){
    warn(`Profiltiefe ${warning.axle}: ${warning.depth.toLocaleString('de-DE')} mm – niedrig (Warnschwelle ≤ ${warning.threshold} mm für ${data.tire_type}).`,'depth-warning');
  }
  if(warnings.childElementCount)card.append(warnings);
  const details=document.createElement('details');details.className='result-details';
  const summary=document.createElement('summary'),label=document.createElement('span');label.textContent='Alle Daten anzeigen';
  const arrow=document.createElement('span');arrow.className='result-chevron';arrow.textContent='⌄';arrow.setAttribute('aria-hidden','true');summary.append(label,arrow);
  details.addEventListener('toggle',()=>{label.textContent=details.open?'Details schließen':'Alle Daten anzeigen'});
  const grid=document.createElement('dl');grid.className='result-data';
  const add=(key,value)=>{const group=document.createElement('div'),term=document.createElement('dt'),description=document.createElement('dd');term.textContent=key;description.textContent=value===undefined||value===null||value===''?'Nicht erfasst':String(value);group.append(term,description);grid.append(group)};
  const size=axle=>{if(!axle||![axle.width,axle.ratio,axle.inches,axle.load,axle.speed].some(Boolean))return '';return `${axle.width||'?'}/${axle.ratio||'?'} R${axle.inches||'?'} ${axle.load||'?'}${axle.speed||'?'}${[axle.width,axle.ratio,axle.inches,axle.load,axle.speed].every(Boolean)?'':' (unvollständig)'}`};
  const depth=value=>typeof value==='string'&&value.trim()!==''?value.replace('.',',')+' mm':'';
  add('Kunde',item.name);add('Kennzeichen',item.plate);add('Fahrzeug',item.car);add('Marke',data.brand);add('Modell',data.model);add('Lagerplatz',item.place);add('Vorgang',labels[item.operation]);add('Status',state);add('Zeitpunkt',stamp);add('Reifenart',data.tire_type);add('Felgenart',data.rim_type);
  add('Reifendaten Vorderachse',size(data.front));add('Reifendaten Hinterachse',size(data.rear));add('Profiltiefe Vorderachse',depth(data.front_depth));add('Profiltiefe Hinterachse',depth(data.rear_depth));
  for(const [key,name] of [['hubcaps','Radkappen'],['bolts','Radschrauben'],['lock','Felgenschloss']]){const accessory=data.accessories?.[key];add(name,accessory?(accessory.present?'Vorhanden · Lagerort: '+(accessory.location||'Nicht erfasst'):'Nicht vorhanden'):'');}
  add('Monteur',data.mechanic);add('Hinweise',data.notes);add('Historieneintrag',item.event_id);
  if(item.source==='legacy')add('Datenstand','Altbestand: Nicht mehr vorhandene Angaben können nicht rekonstruiert werden.');
  else add('Datenstand','Gespeicherte Angaben zu diesem Vorgang.');
  details.append(summary,grid);card.append(details);return card;
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
  const dialog=document.getElementById('placeholder');
  document.querySelectorAll('[data-title]').forEach(button=>button.addEventListener('click',()=>{if(button.dataset.title==='Administration'){location.assign('/admin.html');return;}if(button.dataset.title==='Auslagerung'){window.openRelease();return;}if(button.dataset.title==='Radwechsel'){window.openWheelChange();return;}if(button.dataset.title==='Reservierung'){window.openReservation();return;}if(button.dataset.title==='Neueinlagerung'){window.openStorage();return;}document.getElementById('modal-title').textContent=button.dataset.title;dialog.showModal();}));
  document.getElementById('close-modal').addEventListener('click',()=>dialog.close());
