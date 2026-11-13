'use strict';
(()=>{
 const $=id=>document.getElementById(id),clone=x=>JSON.parse(JSON.stringify(x));
 let csrf='',state=null,defaults=null,values={},fields=[],selected='',dirty=false,undo=[],redo=[],drag=null,previewURL='',counter=0,busy=false,loadGeneration=0;
 const status=(message,error=false)=>{$('editor-status').textContent=message;$('editor-status').classList.toggle('error',error)};
 async function api(path,options={}){
  const headers={...(options.form?{}:{'Content-Type':'application/json'}),...(options.headers||{})};if(csrf)headers['X-CSRF-Token']=csrf;
  let response;try{response=await fetch('/api/admin/'+path,{...options,headers,cache:'no-store'})}catch{throw Error('Verbindung zum Server fehlgeschlagen. Bitte erneut versuchen.');}
  if(response.status===401&&path!=='login'){showLogin();$('login-status').textContent='Sitzung abgelaufen. Bitte neu anmelden.';throw Error('Sitzung abgelaufen. Bitte neu anmelden.');}
  if(!response.ok){let data;try{data=await response.json()}catch{}throw Error(data?.error||'Anfrage fehlgeschlagen.');}
  if(options.download){const match=/filename="([a-zA-Z0-9._-]+)"/.exec(response.headers.get('Content-Disposition')||'');return {blob:await response.blob(),filename:match?.[1]||'reifenapp-backup.zip'};}
  return options.pdf?response.blob():response.json();
 }
 function hideSections(){for(const id of ['admin-nav','areas-panel','backup-panel','import-panel','labels-panel','editor'])$(id).hidden=true;}
 function showLogin(){if(labelURL){URL.revokeObjectURL(labelURL);labelURL=''}$('label-preview-frame').removeAttribute('src');$('label-preview-frame').hidden=true;clearCSVImport();clearBackupLink();hideSections();loadGeneration++;csrf='';$('password-form').reset();$('password').value='';$('login-panel').hidden=false;$('editor').hidden=true;$('logout').hidden=true;$('change-password').hidden=true;$('password-panel').hidden=true;if(previewURL){URL.revokeObjectURL(previewURL);previewURL=''}$('pdf-preview').removeAttribute('src');}
 function selectedElement(){return state?.template.elements.find(e=>e.id===selected)}
 function remember(){undo.push(clone(state.template));if(undo.length>80)undo.shift();redo=[];dirty=true;}
 function round(n){return Math.round(n*100)/100}
 function gridStep(){return Number($('grid').value)}
 function snap(value,step){return step>0?round(Math.round(value/step)*step):round(value)}
 function gridNext(value,direction,multiple=1){const step=gridStep();if(!step)return value+direction*(multiple===2?2:0.5);const index=direction>0?Math.floor(value/step+1e-6)+multiple:Math.ceil(value/step-1e-6)-multiple;return round(index*step)}
 function updateGrid(){const step=gridStep();$('paper').classList.toggle('with-grid',step>0);$('paper').style.setProperty('--grid-step',step+'mm');$('paper').style.setProperty('--grid-line',(1/Number($('zoom').value))+'px');}
 function alignPosition(e){const step=gridStep();e.x=snap(e.x,step);e.y=snap(e.y,step);bounded(e);}
 function bounded(e){e.width=round(Math.min(210,Math.max(1,e.width)));e.height=round(Math.min(297,Math.max(1,e.height)));e.x=round(Math.max(0,Math.min(210-e.width,e.x)));e.y=round(Math.max(0,Math.min(297-e.height,e.y)));}
 function textFor(e){return (e.text+(e.type==='field'?(values[e.field]||''):'' )).replace(/\s+/g,' ').trim()}
 function paint(){
  if(!state)return;const paper=$('paper');paper.replaceChildren();
  for(const e of state.template.elements){
   const node=document.createElement('div');node.className='item'+(e.id===selected?' selected':'');node.dataset.id=e.id;node.tabIndex=0;node.setAttribute('role','button');node.setAttribute('aria-label',e.type==='field'?(fields.find(f=>f.key===e.field)?.label||e.field):e.text||e.type);
   Object.assign(node.style,{left:e.x+'mm',top:e.y+'mm',width:e.width+'mm',height:e.height+'mm',fontSize:e.font_size+'pt',fontWeight:e.bold?'700':'400',textAlign:{L:'left',C:'center',R:'right'}[e.align],color:e.color});
   if(e.type==='line'){node.classList.add('line-item');const line=document.createElement('div');line.className='line-mark';line.style.borderTop=`${e.stroke}mm solid ${e.color}`;node.append(line)}
   else if(e.type==='image'){const image=document.createElement('img');image.src=e.image;image.alt='Firmenlogo';image.draggable=false;image.className='logo-image';node.append(image)}
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
  $('prop-field').disabled=e.type!=='field';for(const key of ['text','font_size','align','bold'])$('prop-'+key).disabled=e.type==='line'||e.type==='box'||e.type==='image';$('prop-background').disabled=e.type!=='box';$('prop-color').disabled=e.type==='image';$('prop-stroke').disabled=e.type==='image';
 }
 function zoom(){const z=Number($('zoom').value);$('paper').style.transform=`scale(${z})`;$('paper-holder').style.width=210*96/25.4*z+'px';$('paper-holder').style.height=297*96/25.4*z+'px';updateGrid();}
 async function load(){
  const operation=$('operation').value,generation=++loadGeneration;const [data,sample]=await Promise.all([api('layout?operation='+operation),api('values?operation='+operation)]);
  if(generation!==loadGeneration||$('operation').value!==operation)return;state=data.layout;defaults=data.default;values=sample;selected='';dirty=false;undo=[];redo=[];paint();zoom();updateGrid();status('Elemente ziehen oder links hinzufügen. Vorschau enthält fiktive Daten.');
 }
 async function enter(){
  $('login-panel').hidden=true;$('password-panel').hidden=true;$('admin-nav').hidden=false;$('logout').hidden=false;$('change-password').hidden=false;
  fields=await api('fields');for(const id of ['palette-field','prop-field']){$(id).replaceChildren();for(const f of fields)$(id).add(new Option(f.label,f.key))}
  if(state&&dirty){$('operation').value=state.operation;const [data,sample]=await Promise.all([api('layout?operation='+state.operation),api('values?operation='+state.operation)]);defaults=data.default;values=sample;paint();zoom();status('Sitzung erneuert. Ungespeicherter Entwurf bleibt geöffnet.');}else{await load();}await document.fonts.load('12px Receipt');await document.fonts.load('bold 12px Receipt');paint();await selectSection(activeSection);
 }
 let activeSection='areas',areaRevision=0,areaSaving=false,areaDirty=false,areaGeneration=0,backupSaving=false,backupURL='',importBusy=false,importHash='',labelConfig=null,labelFields={},labelDefault=null,labelDirty=false,labelBusy=false,labelURL='';
 const areaMessage=(text,error=false)=>{$('area-status').textContent=text;$('area-status').classList.toggle('error',error)};
 async function selectSection(section){
  if(busy||areaSaving||backupSaving||importBusy||labelBusy)return;
  if(section!==activeSection&&labelDirty&&!confirm('Ungespeicherte Etikettenänderungen verwerfen?'))return;
  if(section!==activeSection&&areaDirty&&!confirm('Ungespeicherte Lagerplatzänderungen verwerfen?'))return;
  if(section!==activeSection){labelDirty=false;areaDirty=false;$('area-form').hidden=true;}
  activeSection=section;$('editor').hidden=section!=='pdf';$('areas-panel').hidden=section!=='areas';$('backup-panel').hidden=section!=='backup';$('import-panel').hidden=section!=='import';$('labels-panel').hidden=section!=='labels';
  for(const [id,name] of [['nav-areas','areas'],['nav-pdf','pdf'],['nav-backup','backup'],['nav-import','import'],['nav-labels','labels']]){$(id).setAttribute('aria-pressed',String(name===section));$(id).classList.toggle('secondary',name!==section);}
  if(section==='areas')await loadAreas();else if(section==='backup')await loadBackupInfo();else if(section==='import')await loadCSVImports();else if(section==='labels')await loadLabels();else{zoom();paint();}
 }
 async function loadAreas(){
  loadInactivitySettings();
  const generation=++areaGeneration;$('areas-status').textContent='Lagerorte werden geladen …';
  try{const areas=await api('areas');if(generation!==areaGeneration||!csrf)return;
   $('areas-list').replaceChildren();
   for(const a of areas){const row=document.createElement('tr'),count=a.last_number-a.first_number+1;
    for(const value of [a.code,a.first_number+'–'+a.last_number,count,a.occupied,a.reserved,count-a.occupied-a.reserved]){const cell=document.createElement('td');cell.textContent=value;row.append(cell)}
    const cell=document.createElement('td'),edit=document.createElement('button');edit.type='button';edit.className='secondary';edit.textContent='Bearbeiten';edit.setAttribute('aria-label','Lagerort '+a.code+' bearbeiten');edit.onclick=()=>editArea(a);cell.append(edit);row.append(cell);$('areas-list').append(row);
   }$('areas-status').textContent=areas.length+' Lagerorte';$('areas-status').classList.remove('error');
  }catch(error){$('areas-status').textContent=error.message;$('areas-status').classList.add('error');}
 }
 function editArea(a){
  if(areaSaving)return;if(areaDirty&&!confirm('Ungespeicherte Lagerplatzänderungen verwerfen?'))return;
  areaRevision=a?.revision||0;areaDirty=false;$('area-form').reset();$('area-form').hidden=false;
  $('area-title').textContent=a?'Lagerort bearbeiten':'Lagerort anlegen';$('area-code').value=a?.code||'';$('area-code').readOnly=!!a;$('area-first').value=a?.first_number||'';$('area-last').value=a?.last_number||'';areaMessage('');areaSummary();$('area-form').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'instant':'smooth'});$(a?'area-first':'area-code').focus();
 }
 function areaSummary(){const first=Number($('area-first').value),last=Number($('area-last').value);$('area-summary').textContent=first>0&&last>=first?`${last-first+1} Plätze · ${$('area-code').value}.${first} bis ${$('area-code').value}.${last}`:'';}
 async function loadInactivitySettings(){
  $('inactivity-months').disabled=true;$('inactivity-save').disabled=true;
  try{const settings=await api('inactivity');if(!csrf)return;$('inactivity-months').value=settings.months;$('inactivity-months').disabled=false;$('inactivity-save').disabled=false;$('inactivity-settings-status').textContent='';}
  catch(error){$('inactivity-settings-status').textContent=error.message;}
 }
 $('inactivity-settings').addEventListener('submit',async event=>{
  event.preventDefault();if(areaSaving||busy||backupSaving||importBusy||labelBusy)return;
  areaSaving=true;$('inactivity-save').disabled=true;$('inactivity-months').disabled=true;
  try{const settings=await api('inactivity',{method:'POST',body:JSON.stringify({months:Number($('inactivity-months').value)})});if(!csrf)return;$('inactivity-months').value=settings.months;$('inactivity-settings-status').textContent='Gespeichert. Die Anzeige auf der Startseite wird automatisch aktualisiert.';}
  catch(error){$('inactivity-settings-status').textContent=error.message;}
  finally{areaSaving=false;$('inactivity-save').disabled=false;$('inactivity-months').disabled=false;}
 });
 $('nav-labels').onclick=()=>selectSection('labels');$('nav-areas').onclick=()=>selectSection('areas');$('nav-pdf').onclick=()=>selectSection('pdf');$('nav-backup').onclick=()=>selectSection('backup');$('nav-import').onclick=()=>selectSection('import');
 $('area-new').onclick=()=>editArea(null);$('areas-reload').onclick=()=>{if(areaSaving)return;if(areaDirty&&!confirm('Ungespeicherte Lagerplatzänderungen verwerfen?'))return;areaDirty=false;$('area-form').hidden=true;loadAreas();};
 $('area-cancel').onclick=()=>{if(areaSaving)return;if(areaDirty&&!confirm('Ungespeicherte Lagerplatzänderungen verwerfen?'))return;areaDirty=false;$('area-form').hidden=true;};
 $('area-form').addEventListener('input',()=>{areaDirty=true;areaSummary()});
 $('area-form').addEventListener('submit',async event=>{
  event.preventDefault();if(areaSaving)return;
  const first=Number($('area-first').value),last=Number($('area-last').value);
  if(last<first||last-first>=10000){areaMessage('Bitte einen aufsteigenden Bereich mit höchstens 10000 Plätzen eingeben.',true);return}
  areaSaving=true;$('area-form').inert=true;$('change-password').disabled=true;$('logout').disabled=true;areaMessage('Lagerort wird gespeichert …');
  try{await api('areas',{method:'POST',body:JSON.stringify({code:$('area-code').value,first_number:first,last_number:last,revision:areaRevision})});areaDirty=false;$('area-form').hidden=true;await loadAreas();$('areas-status').textContent+=' · Änderung gespeichert.';}
  catch(error){areaMessage(error.message,true)}finally{areaSaving=false;$('area-form').inert=false;$('change-password').disabled=false;$('logout').disabled=false;}
 });
 function clearBackupLink(){if(backupURL)URL.revokeObjectURL(backupURL);backupURL='';$('backup-link').hidden=true;$('backup-link').removeAttribute('href');}
 async function loadBackupInfo(){
  $('backup-status').textContent='';$('backup-status').classList.remove('error');
  try{const info=await api('backup/info');if(!csrf)return;$('backup-database').textContent=info.database;$('backup-schema').textContent=info.schema_version;}
  catch(error){$('backup-status').textContent=error.message;$('backup-status').classList.add('error');}
 }
 $('backup-download').addEventListener('click',async()=>{
  if(backupSaving||busy||areaSaving)return;backupSaving=true;$('backup-download').disabled=true;$('logout').disabled=true;$('change-password').disabled=true;clearBackupLink();$('backup-status').classList.remove('error');$('backup-status').textContent='Konsistente Datenbanksicherung wird erstellt …';
  try{
   const result=await api('backup',{method:'POST',body:'{}',download:true});if(!csrf)return;
   backupURL=URL.createObjectURL(result.blob);const link=$('backup-link');link.href=backupURL;link.download=result.filename;link.hidden=false;link.click();
   $('backup-status').textContent='Backup erstellt. Der Download wurde gestartet. ZIP entpacken und uebersicht.html im Browser öffnen.';
  }catch(error){$('backup-status').textContent=error.message;$('backup-status').classList.add('error');}
  finally{backupSaving=false;$('backup-download').disabled=false;$('logout').disabled=false;$('change-password').disabled=false;}
 });
 function clearCSVImport(){importHash='';$('import-file').value='';$('import-confirm').checked=false;$('import-preview-panel').hidden=true;$('import-preview-rows').replaceChildren();$('import-commit').disabled=true;$('import-preview').disabled=true;}
 function csvMessage(message,error=false){$('import-status').textContent=message;$('import-status').classList.toggle('error',error);}
 function csvBusy(value){importBusy=value;for(const id of ['import-file','import-preview','import-confirm','logout','change-password'])$(id).disabled=value;$('import-commit').disabled=value||!importHash||!$('import-confirm').checked;}
 async function loadCSVImports(){
  try{const batches=await api('imports');if(!csrf)return;$('import-list').replaceChildren();
   for(const batch of batches){const row=document.createElement('tr');for(const value of [batch.filename,new Date(batch.imported_at).toLocaleString('de-DE'),batch.rows,batch.imported,batch.duplicates]){const cell=document.createElement('td');cell.textContent=value;row.append(cell)}const cell=document.createElement('td'),link=document.createElement('a');link.href='/api/admin/imports/'+batch.id+'/original';link.textContent='CSV herunterladen';link.download=batch.filename;cell.append(link);row.append(cell);$('import-list').append(row)}
   if(!batches.length){const row=document.createElement('tr'),cell=document.createElement('td');cell.colSpan=6;cell.textContent='Noch keine CSV importiert.';row.append(cell);$('import-list').append(row)}
  }catch(error){csvMessage(error.message,true)}
 }
 $('import-file').addEventListener('change',()=>{importHash='';$('import-confirm').checked=false;$('import-preview-panel').hidden=true;$('import-commit').disabled=true;const file=$('import-file').files[0];$('import-preview').disabled=!file;csvMessage(file?'Datei ausgewählt. Bitte zuerst die Vorschau prüfen.':'');});
 $('import-confirm').addEventListener('change',()=>{$('import-commit').disabled=!importHash||!$('import-confirm').checked||importBusy;});
 function csvForm(){const form=new FormData();form.append('file',$('import-file').files[0]);return form;}
 $('import-preview').addEventListener('click',async()=>{
  const file=$('import-file').files[0];if(!file||importBusy)return;if(file.size>10*1024*1024){csvMessage('Die CSV darf höchstens 10 MiB groß sein.',true);return}
  importHash='';$('import-confirm').checked=false;$('import-preview-panel').hidden=true;csvBusy(true);csvMessage('CSV wird geprüft …');
  try{
   const data=await api('imports/preview',{method:'POST',body:csvForm(),form:true});if(!csrf)return;
   importHash=data.already_imported?'':data.hash;
   $('import-summary').textContent=data.filename+' · '+data.encoding+' · '+data.rows+' Vorgänge · '+data.new+' neu · '+data.duplicates+' bereits vorhanden · '+data.warning_rows+' Zeilen mit Prüfhinweisen';
   $('import-preview-rows').replaceChildren();
   const labels={storage:'Neueinlagerung',wheel_change:'Radwechsel',release:'Auslagerung',reservation:'Reservierung',imported:'Sonstiger Altvorgang'};
   for(const item of data.preview){const row=document.createElement('tr');for(const value of [item.line,item.date?new Date(item.date).toLocaleString('de-DE'):'Unbekannt',labels[item.operation],item.name+' · '+item.plate,item.place,item.warnings.join(' ')||'OK']){const cell=document.createElement('td');cell.textContent=value;row.append(cell)}$('import-preview-rows').append(row)}
   $('import-preview-panel').hidden=false;csvMessage(data.already_imported?'Diese Datei wurde bereits importiert. Es werden keine weiteren Vorgänge angelegt.':'Vorschau bereit. Historische Vorgänge erzeugen keine aktuelle Lagerbelegung.');
  }catch(error){csvMessage(error.message,true)}finally{csvBusy(false);}
 });
 $('import-commit').addEventListener('click',async()=>{
  if(importBusy||!importHash||!$('import-confirm').checked||!$('import-file').files[0])return;
  const form=csvForm();form.append('confirmed','true');form.append('expected_sha256',importHash);csvBusy(true);csvMessage('Sicherung und Import werden erstellt …');
  try{
   const data=await api('imports/commit',{method:'POST',body:form,form:true});if(!csrf)return;
   importHash='';$('import-confirm').checked=false;await loadCSVImports();
   csvMessage(data.already_imported?'Datei war bereits importiert. Keine neuen Vorgänge angelegt.':'Import gespeichert: '+data.imported+' neue Vorgänge, '+data.duplicates+' bereits vorhanden. Sicherung: '+data.backup);
  }catch(error){csvMessage(error.message,true)}finally{csvBusy(false);}
 });

 function setLabelBusy(value){labelBusy=value;$('logout').disabled=value;$('change-password').disabled=value;}
 const labelMessage=(text,error=false)=>{$('label-status').textContent=text;$('label-status').classList.toggle('error',error)};
 function labelValues(){
  const s=labelConfig.settings;s.enabled=$('label-enabled').checked;s.queue=$('label-queue').value;s.copies=Number($('label-copies').value);s.storage=$('label-storage').checked;s.wheel_change=$('label-wheel-change').checked;
  s.elements=[...$('label-elements').rows].map(row=>{const value=key=>row.querySelector('[data-key="'+key+'"]');return {field:value('field').value,text:value('text').value,...Object.fromEntries(['x','y','width','height','size'].map(key=>[key,Number(value(key).value)])),bold:value('bold').checked}});return s;
 }
 function renderLabelElements(){
  $('label-elements').replaceChildren();labelConfig.settings.elements.forEach((element,index)=>{const row=document.createElement('tr');
   for(const key of ['field','text','x','y','width','height','size','bold']){const cell=document.createElement('td'),input=document.createElement(key==='field'?'select':'input');input.dataset.key=key;input.setAttribute('aria-label','Feld '+(index+1)+' '+key);
    if(key==='field'){for(const [value,label]of Object.entries(labelFields))input.add(new Option(label,value));input.value=element.field}
    else if(key==='bold'){input.type='checkbox';input.checked=element.bold}
    else{input.type=key==='text'?'text':'number';input.value=element[key];if(key==='text')input.maxLength=200;else{input.step='0.5';input.min=key==='size'?'6':(['width','height'].includes(key)?'2':'0');input.max=key==='size'?'32':(['y','height'].includes(key)?'51':'78');input.required=true}}
    cell.append(input);row.append(cell)}
   const cell=document.createElement('td');for(const [text,delta]of [['↑',-1],['↓',1],['Entfernen',0]]){const button=document.createElement('button');button.type='button';button.className='secondary';button.textContent=text;button.setAttribute('aria-label',text+' Feld '+(index+1));button.disabled=delta===-1&&index===0||delta===1&&index===labelConfig.settings.elements.length-1||delta===0&&labelConfig.settings.elements.length===1;button.onclick=()=>{const s=labelValues();if(delta){[s.elements[index],s.elements[index+delta]]=[s.elements[index+delta],s.elements[index]]}else s.elements.splice(index,1);labelDirty=true;renderLabelElements()};cell.append(button)}row.append(cell);$('label-elements').append(row);
  });
 }
 async function loadLabelPrinters(){
  const selected=labelConfig?$('label-queue').value:'';const data=await api('labels/printers');if(!csrf)return;
  $('label-queue').replaceChildren(new Option('Drucker auswählen',''));$('label-printer-list').replaceChildren();
  for(const printer of data.printers){$('label-queue').add(new Option(printer.name,printer.name));const item=document.createElement('li');item.textContent=printer.name+': '+printer.state;$('label-printer-list').append(item)}
  if(selected&&![...$('label-queue').options].some(o=>o.value===selected))$('label-queue').add(new Option(selected+' (nicht gefunden)',selected));$('label-queue').value=selected;
  $('label-printers-status').textContent=data.message+(!data.printers.length?' Keine USB-Druckerwarteschlange gefunden.':'');
 }
 async function loadLabelJobs(){const jobs=await api('labels/jobs');if(!csrf)return;$('label-jobs').replaceChildren();
  const states={pending:'Vorgemerkt',processing:'Übergabe läuft',submitted:'An CUPS übergeben',failed:'Fehlgeschlagen',uncertain:'Übergabe unklar',cancelled:'Abgebrochen'};
  for(const job of jobs){const row=document.createElement('tr');for(const value of [job.id,job.event_id?'Historie #'+job.event_id:'Testdruck',states[job.status]||job.status,job.cups_id||'—',job.message||'—']){const cell=document.createElement('td');cell.textContent=value;row.append(cell)}const cell=document.createElement('td');if(['failed','uncertain','pending'].includes(job.status)){const button=document.createElement('button');button.type='button';button.className='secondary';button.textContent=job.status==='pending'?'Abbrechen':'Erneut versuchen';button.onclick=async()=>{const retry=job.status!=='pending';if(!confirm(retry?'Vor erneutem Druck: CUPS und vorhandene Etiketten geprüft? Ein weiterer Ausdruck kann entstehen. Aktuelle gespeicherte Vorlage verwenden?':'Diesen noch nicht übergebenen Auftrag abbrechen?'))return;button.disabled=true;try{await api('labels/jobs/'+job.id,{method:'POST',body:JSON.stringify({action:retry?'retry':'cancel',confirmed:true})});await loadLabelJobs()}catch(error){$('label-jobs-status').textContent=error.message;button.disabled=false}};cell.append(button)}row.append(cell);$('label-jobs').append(row)}$('label-jobs-status').textContent=jobs.length?jobs.length+' letzte Aufträge.':'Noch keine Etikettenaufträge.';
 }
 async function loadLabels(){
  $('label-config-form').inert=true;try{const data=await api('labels/config');if(!csrf)return;labelConfig=data.config;labelFields=data.fields;labelDefault=data.default;labelDirty=false;const s=labelConfig.settings;$('label-enabled').checked=s.enabled;$('label-storage').checked=s.storage;$('label-wheel-change').checked=s.wheel_change;$('label-copies').value=s.copies;$('label-queue').replaceChildren(new Option(s.queue||'Drucker auswählen',s.queue));renderLabelElements();await loadLabelPrinters();await loadLabelJobs();labelMessage('Gespeicherte Vorlage. 78 × 51 mm.');}catch(error){labelMessage(error.message,true)}finally{$('label-config-form').inert=false;}
 }
 $('label-config-form').addEventListener('input',()=>{labelDirty=true;$('label-preview-frame').hidden=true;labelMessage('Ungespeicherte Änderungen. Vorschau prüfen und Einstellungen speichern.');});
 $('label-add').onclick=()=>{if(!labelConfig)return;const s=labelValues();if(s.elements.length>=20){labelMessage('Höchstens 20 Felder.',true);return}s.elements.push({field:'text',text:'Eigener Text',x:3,y:43,width:72,height:5,size:9,bold:false});labelDirty=true;renderLabelElements()};
 $('label-default').onclick=()=>{if(!labelConfig||!confirm('Etikettenlayout durch Standard ersetzen?'))return;labelValues().elements=clone(labelDefault.elements);labelDirty=true;renderLabelElements()};
 $('label-config-form').addEventListener('submit',async event=>{event.preventDefault();if(!labelConfig||labelBusy)return;labelValues();setLabelBusy(true);$('label-config-form').inert=true;try{const data=await api('labels/config',{method:'POST',body:JSON.stringify(labelConfig)});if(!csrf)return;labelConfig=data.config;labelDirty=false;labelMessage('Drucker und Etikettenvorlage gespeichert.');}catch(error){labelMessage(error.message,true)}finally{setLabelBusy(false);$('label-config-form').inert=false;}});
 $('label-preview').onclick=async()=>{if(!labelConfig||labelBusy)return;setLabelBusy(true);try{const blob=await api('labels/preview',{method:'POST',body:JSON.stringify(labelValues()),pdf:true});if(!csrf)return;if(labelURL)URL.revokeObjectURL(labelURL);labelURL=URL.createObjectURL(blob);$('label-preview-frame').src=labelURL;$('label-preview-frame').hidden=false;labelMessage('PDF-Vorschau mit fiktiven Daten. Noch nicht gespeicherte Änderungen sind enthalten.');}catch(error){labelMessage(error.message,true)}finally{setLabelBusy(false);}};
 $('label-printers-refresh').onclick=async()=>{try{await loadLabelPrinters()}catch(error){labelMessage(error.message,true)}};
 $('label-jobs-refresh').onclick=async()=>{try{await loadLabelJobs()}catch(error){$('label-jobs-status').textContent=error.message}};
 $('label-test').onclick=async()=>{if(labelDirty){labelMessage('Bitte zuerst Einstellungen speichern.',true);return}if(labelBusy||!confirm('Genau ein Testetikett mit fiktiven Daten auf dem gespeicherten USB-Drucker drucken?'))return;setLabelBusy(true);$('label-test').disabled=true;try{const result=await api('labels/test',{method:'POST',body:'{"confirmed":true}'});labelMessage('Testdruck #'+result.id+' vorgemerkt. Auftragsstatus unten aktualisieren.');await loadLabelJobs()}catch(error){labelMessage(error.message,true)}finally{setLabelBusy(false);$('label-test').disabled=false;}};
 setInterval(()=>{if(csrf&&activeSection==='labels'&&!document.hidden&&!labelBusy)loadLabelJobs().catch(()=>{$('label-jobs-status').textContent='Auftragsstatus derzeit nicht verfügbar.'})},5000);

 $('login-form').addEventListener('submit',async event=>{event.preventDefault();const input=$('password');const password=input.value;input.value='';$('login-status').textContent='Anmeldung …';try{const data=await api('login',{method:'POST',body:JSON.stringify({username:$('username').value,password})});csrf=data.csrf;if(data.must_change)showPasswordChange(true);else await enter();$('login-status').textContent=''}catch(error){$('login-status').textContent=error.message}});
 function showPasswordChange(required){hideSections();
  loadGeneration++;$('login-panel').hidden=true;$('editor').hidden=true;$('password-panel').hidden=false;$('logout').hidden=false;$('change-password').hidden=true;
  $('current-password-label').hidden=required;$('current-password').required=!required;$('current-password').disabled=required;$('cancel-password').hidden=required;
  $('password-title').textContent=required?'Neues Admin-Passwort festlegen':'Admin-Passwort ändern';
  $('password-help').textContent=required?'Bitte das Startpasswort jetzt durch ein eigenes Passwort ersetzen. Erst danach wird die Administration freigeschaltet.':'Aktuelles Passwort eingeben und ein neues Passwort wählen (mindestens 12 Zeichen).';
  $('password-form').reset();$('password-status').textContent='';$('password-panel').scrollIntoView();(required?$('new-password'):$('current-password')).focus();
 }
 $('change-password').addEventListener('click',()=>showPasswordChange(false));
 $('cancel-password').addEventListener('click',async()=>{try{$('password-form').reset();await enter()}catch(error){$('password-status').textContent=error.message}});
 $('password-form').addEventListener('submit',async event=>{
  event.preventDefault();if($('save-password').disabled)return;if($('new-password').value!==$('confirm-password').value){$('password-status').textContent='Die neuen Passwörter stimmen nicht überein.';return}
  const body={current_password:$('current-password').value,new_password:$('new-password').value,confirm_password:$('confirm-password').value};const enabled=[...$('password-form').elements].filter(el=>!el.disabled);enabled.forEach(el=>el.disabled=true);$('logout').disabled=true;$('password-status').textContent='Passwort wird gespeichert …';
  try{const data=await api('password',{method:'POST',body:JSON.stringify(body)});csrf=data.csrf;$('password-form').reset();await enter();status('Neues Admin-Passwort gespeichert. Bisherige Anmeldungen wurden beendet.');}
  catch(error){$('password-status').textContent=error.message}finally{enabled.forEach(el=>el.disabled=false);$('save-password').disabled=false;$('logout').disabled=false}
 });
 $('logout').addEventListener('click',async()=>{if((dirty||areaDirty||labelDirty)&&!confirm('Ungespeicherte Änderungen verwerfen?'))return;if(areaSaving||busy||backupSaving||importBusy||labelBusy)return;try{await api('logout',{method:'POST',body:'{}'});dirty=false;areaDirty=false;labelDirty=false;state=null;showLogin()}catch(error){status(error.message,true)}});
 $('operation').addEventListener('change',async()=>{if(dirty&&!confirm('Ungespeicherte Änderungen verwerfen?')){$('operation').value=state.operation;return}try{await load()}catch(error){status(error.message,true)}});
 $('reload').addEventListener('click',async()=>{if(dirty&&!confirm('Ungespeicherte Änderungen verwerfen?'))return;try{await load()}catch(error){status(error.message,true)}});
 function add(type){if(state.template.elements.length>=100){status('Maximal 100 Elemente erlaubt.',true);return}remember();let id;do{id='element-'+Date.now()+'-'+(++counter)}while(state.template.elements.some(e=>e.id===id));
  const e={id,type,text:type==='text'?'Neuer Text':'',field:type==='field'?$('palette-field').value:'',x:25,y:25,width:type==='line'?100:60,height:type==='line'?2:15,font_size:12,bold:false,align:'L',color:'#000000',background:'#ffffff',stroke:type==='line'||type==='box'?0.3:0};alignPosition(e);state.template.elements.push(e);selected=e.id;paint();}
 for(const type of ['text','field','line','box'])$('add-'+type).addEventListener('click',()=>add(type));
 $('import-logo').addEventListener('click',()=>{if(!busy)$('logo-file').click()});
 $('logo-file').addEventListener('change',async()=>{
  const file=$('logo-file').files[0];$('logo-file').value='';if(!file||busy||!state)return;
  if(state.template.elements.length>=100){status('Maximal 100 Elemente erlaubt.',true);return}
  busy=true;$('editor').inert=true;$('logout').disabled=true;$('change-password').disabled=true;status('Logo wird lokal aufbereitet …');
  try{
   const logo=await window.importLogo(file);
   const total=state.template.elements.reduce((bytes,e)=>bytes+(e.type==='image'?Math.floor((e.image.length-22)*3/4):0),0);
   if(total+Math.floor((logo.data.length-22)*3/4)>4*1024*1024)throw Error('Logos einer Vorlage dürfen zusammen höchstens 4 MiB belegen. Bitte vorhandene Logos entfernen oder kleinere Grafiken verwenden.');
   remember();const scale=Math.min(60/logo.width,80/logo.height),width=Math.max(1,logo.width*scale),height=Math.max(1,logo.height*scale);
   const e={id:'logo-'+Date.now()+'-'+(++counter),type:'image',image:logo.data,text:'',field:'',x:25,y:15,width,height,font_size:12,bold:false,align:'L',color:'#000000',background:'#ffffff',stroke:0};
   alignPosition(e);state.template.elements.push(e);selected=e.id;paint();status('Logo importiert. Ziehen oder skalieren; das Seitenverhältnis bleibt im Rahmen erhalten. Mit „Speichern & aktivieren“ übernehmen.');
  }catch(error){status(error.message,true)}finally{busy=false;$('editor').inert=false;$('logout').disabled=false;$('change-password').disabled=false;}
 });
 $('properties').addEventListener('submit',event=>event.preventDefault());
 for(const key of ['text','field','x','y','width','height','font_size','align','color','background','stroke','bold'])$('prop-'+key).addEventListener('change',()=>{
  const e=selectedElement();if(!e)return;const control=$('prop-'+key);if(control.type==='number'&&!control.checkValidity()){status('Bitte gültige Abmessungen eingeben.',true);properties();return}
  remember();e[key]=key==='bold'?control.checked:control.type==='number'?Number(control.value):control.value;bounded(e);paint();
 });
 function remove(){const e=selectedElement();if(!e)return;if(state.template.elements.length===1){status('Mindestens ein Element behalten.',true);return}remember();state.template.elements=state.template.elements.filter(item=>item.id!==selected);selected='';paint()}
 $('delete').addEventListener('click',remove);
 $('duplicate').addEventListener('click',()=>{const e=selectedElement();if(!e||state.template.elements.length>=100)return;remember();const item={...clone(e),id:'element-'+Date.now()+'-'+(++counter),x:gridStep()?gridNext(e.x,1):e.x+3,y:gridStep()?gridNext(e.y,1):e.y+3};bounded(item);state.template.elements.push(item);selected=item.id;paint()});
 for(const [id,delta]of [['backward',-1],['forward',1]])$(id).addEventListener('click',()=>{const i=state.template.elements.findIndex(e=>e.id===selected);if(i<0||(delta<0&&i===0)||(delta>0&&i===state.template.elements.length-1))return;remember();const [item]=state.template.elements.splice(i,1);if(delta<0)state.template.elements.unshift(item);else state.template.elements.push(item);paint()});
 function history(back){const from=back?undo:redo,to=back?redo:undo;if(!from.length)return;to.push(clone(state.template));state.template=from.pop();selected='';dirty=true;paint()}
 $('undo').addEventListener('click',()=>history(true));$('redo').addEventListener('click',()=>history(false));
 $('reset').addEventListener('click',()=>{if(!confirm('Entwurf durch die Ausgangsvorlage ersetzen? Gespeichert wird erst mit „Speichern & aktivieren“.'))return;remember();state.template=clone(defaults);selected='';paint()});
 $('zoom').addEventListener('change',zoom);
 $('grid').addEventListener('change',()=>{updateGrid();status(gridStep()?'Raster: '+gridStep()+' mm. Alt beim Ziehen erlaubt freie Bewegung.':'Raster ausgeschaltet. Freie Positionierung aktiv.');});
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
  const step=event.altKey?0:gridStep();
  if(drag.resize){const right=snap(drag.element.x+drag.element.width+dx,step),bottom=snap(drag.element.y+drag.element.height+dy,step);e.width=Math.min(210-e.x,Math.max(1,right-e.x));e.height=Math.min(297-e.y,Math.max(1,bottom-e.y))}else{e.x=snap(drag.element.x+dx,step);e.y=snap(drag.element.y+dy,step)}bounded(e);paint();
 });
 for(const type of ['pointerup','pointercancel','lostpointercapture'])$('paper').addEventListener(type,()=>drag=null);
 document.addEventListener('keydown',event=>{
  if(busy||$('editor').hidden||event.target.matches('input,textarea,select'))return;
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==='z'){event.preventDefault();history(!event.shiftKey);return}
  if((event.ctrlKey||event.metaKey)&&event.key.toLowerCase()==='y'){event.preventDefault();history(false);return}
  if(event.key==='Delete'){event.preventDefault();remove();return}
  const deltas={ArrowLeft:[-1,0],ArrowRight:[1,0],ArrowUp:[0,-1],ArrowDown:[0,1]},e=selectedElement();if(e&&deltas[event.key]){event.preventDefault();remember();const multiple=event.shiftKey?2:1;if(deltas[event.key][0])e.x=gridNext(e.x,deltas[event.key][0],multiple);if(deltas[event.key][1])e.y=gridNext(e.y,deltas[event.key][1],multiple);bounded(e);paint()}
 });
 async function persist(enabled){if(busy)return;busy=true;$('editor').inert=true;$('logout').disabled=true;$('change-password').disabled=true;$('save').disabled=true;$('standard').disabled=true;
  try{state=await api('layout',{method:'POST',body:JSON.stringify({...state,enabled})});dirty=false;paint();status(enabled?'Layout gespeichert und für neue PDF-Abrufe aktiviert.':'Standardbeleg aktiviert. Eigener Entwurf bleibt gespeichert.')}
  catch(error){status(error.message,true)}finally{busy=false;$('editor').inert=false;$('logout').disabled=false;$('change-password').disabled=false;$('save').disabled=false;$('standard').disabled=false}
 }
 $('save').addEventListener('click',()=>persist(true));$('standard').addEventListener('click',()=>persist(false));
 $('preview').addEventListener('click',async()=>{try{const blob=await api('preview',{method:'POST',body:JSON.stringify(state),pdf:true});if(previewURL)URL.revokeObjectURL(previewURL);previewURL=URL.createObjectURL(blob);$('pdf-preview').src=previewURL;$('preview-panel').hidden=false;$('preview-panel').scrollIntoView({behavior:matchMedia('(prefers-reduced-motion: reduce)').matches?'instant':'smooth'});status('Echte PDF-Vorschau erzeugt. Noch nicht gespeicherte Änderungen sind enthalten.')}catch(error){status(error.message,true)}});
 $('close-preview').addEventListener('click',()=>{$('preview-panel').hidden=true;$('pdf-preview').removeAttribute('src');if(previewURL){URL.revokeObjectURL(previewURL);previewURL=''}});
 window.addEventListener('beforeunload',event=>{if(dirty||areaDirty){event.preventDefault();event.returnValue=''}});
 (async()=>{try{const response=await fetch('/api/admin/session',{cache:'no-store'});const data=await response.json();if(response.ok){csrf=data.csrf;if(data.must_change)showPasswordChange(true);else await enter()}else if(response.status!==401){$('login-status').textContent=data.error||'Anmeldung derzeit nicht verfügbar.'}}catch{$('login-status').textContent='Verbindung zum Server nicht möglich.'}})();
})();
