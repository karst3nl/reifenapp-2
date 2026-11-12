'use strict';
(() => {
  const dialog=document.getElementById('release-dialog');
  const form=document.getElementById('release-form');
  const field=name=>form.elements.namedItem(name);
  const status=document.getElementById('release-status');
  const details=document.getElementById('release-details');
  const save=document.getElementById('release-save');
  const closeButtons=['release-x','release-back','release-close'].map(id=>document.getElementById(id));
  let storageAreas=[], target=null, timer, request, generation=0, saving=false, ready=false;
  function message(text,error=false){status.textContent=text;status.classList.toggle('is-error',error);}
  function clearTarget(){target=null;details.hidden=true;field('confirmed').checked=false;field('confirmed').disabled=true;save.disabled=true;}
  function range(){const area=storageAreas.find(a=>a.code===field('area').value);field('number').min=area?.first_number||1;field('number').max=area?.last_number||100000;}
  function reset(){
    clearTimeout(timer);request?.abort();generation++;form.reset();clearTarget();range();message('');
    form.querySelectorAll('[aria-invalid]').forEach(el=>el.removeAttribute('aria-invalid'));
    for(const id of ['release-name','release-vehicle','release-place'])document.getElementById(id).textContent='';
  }
  window.openRelease=async()=>{
    reset();dialog.showModal();ready=false;
    message('Lagerbereiche werden geladen …');
    try{
      const response=await fetch('/api/storage/options',{cache:'no-store'});if(!response.ok)throw Error();
      const options=await response.json();storageAreas=options.storage_areas;field('area').replaceChildren(new Option('Bitte wählen',''));
      for(const area of options.area)field('area').add(new Option(area,area));ready=true;range();message('');
    }catch{message('Lagerbereiche konnten nicht geladen werden. Bitte schließen und erneut öffnen.',true);}
  };
  function check(){
    if(saving||!ready)return;
    clearTimeout(timer);request?.abort();const id=++generation;clearTarget();message('');
    field('number').removeAttribute('aria-invalid');field('area').removeAttribute('aria-invalid');
    if(!dialog.open||!field('area').value||!field('number').value)return;
    if(!field('number').checkValidity()){field('number').setAttribute('aria-invalid','true');message('Bitte eine gültige Lagerplatznummer für diesen Bereich eingeben.',true);return;}
    const body={area:field('area').value,number:Number(field('number').value)};message('Belegung wird geprüft …');
    timer=setTimeout(async()=>{
      request=new AbortController();
      try{
        const response=await fetch('/api/releases/check',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body),signal:request.signal,cache:'no-store'});
        const data=await response.json();if(id!==generation||!dialog.open)return;
        if(!response.ok)throw Error(data.error||'Belegung nicht abrufbar.');
        message(data.message,data.status!=='occupied');
        if(data.status!=='occupied')return;
        target={...data,...body};details.hidden=false;
        document.getElementById('release-name').textContent=data.name;
        document.getElementById('release-vehicle').textContent=data.plate+' · '+data.car;
        document.getElementById('release-place').textContent=data.place;
        field('confirmed').disabled=false;
      }catch(error){if(error.name!=='AbortError'&&id===generation)message('Belegung nicht abrufbar. Bitte erneut prüfen.',true);}
    },180);
  }
  field('area').addEventListener('change',()=>{range();check();});field('number').addEventListener('input',check);
  document.getElementById('release-check').addEventListener('click',check);
  field('confirmed').addEventListener('change',()=>{save.disabled=saving||!target||!field('confirmed').checked;});
  closeButtons.forEach(button=>button.addEventListener('click',()=>{if(!saving)dialog.close();}));
  dialog.addEventListener('cancel',event=>{if(saving)event.preventDefault();});dialog.addEventListener('close',reset);
  form.addEventListener('submit',async event=>{
    event.preventDefault();if(saving||!target||!field('confirmed').checked)return;
    const receipt=window.ReceiptTabs.prepare();
    const snapshot=target;saving=true;save.disabled=true;closeButtons.forEach(button=>button.disabled=true);
    const enabled=[...form.elements].filter(el=>!el.disabled);enabled.forEach(el=>el.disabled=true);message('Auslagerung wird gespeichert …');
    try{
      const response=await fetch('/api/releases',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({area:snapshot.area,number:snapshot.number,claim_id:snapshot.claim_id,confirmed:true})});
      const data=await response.json();
      if(!response.ok)throw Error(data.error||'Auslagerung fehlgeschlagen.');
      const confirmation=document.getElementById('storage-confirmation');confirmation.textContent='Auslagerung auf '+data.place+' gespeichert. Der Lagerplatz ist wieder frei.';confirmation.hidden=false;
      receipt.complete(data.pdf_url,confirmation);
      dialog.close();
      const search=document.getElementById('search');if(search.value.trim())search.dispatchEvent(new Event('input'));
    }catch(error){receipt.cancel();clearTarget();message(error.message==='Failed to fetch'?'Keine Bestätigung erhalten. Bitte den Lagerplatz erneut prüfen.':error.message,true);}
    finally{
      enabled.forEach(el=>el.disabled=false);saving=false;closeButtons.forEach(button=>button.disabled=false);
      field('confirmed').disabled=!target;save.disabled=!target||!field('confirmed').checked;
      if(dialog.open)status.focus();
    }
  });
})();
