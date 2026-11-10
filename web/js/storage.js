'use strict';
(() => {
  const dialog=document.getElementById('storage-dialog');
  const form=document.getElementById('storage-form');
  const status=document.getElementById('storage-status');
  const save=document.getElementById('storage-save');
  const close=document.getElementById('storage-close');
  const nav=[document.getElementById('storage-back'),document.getElementById('storage-x')];
  nav.forEach(button=>button.addEventListener('click',()=>{if(!saving)dialog.close();}));
  document.getElementById('copy-front').addEventListener('click',()=>{
    for(const key of ['width','ratio','inches','load','speed']){const el=field('rear.'+key);el.value=field('front.'+key).value;if(el.hasAttribute('aria-invalid'))validateField(el);}
  });
  let ready=false, saving=false;
  let nameTimer, nameRequest, nameGeneration=0;
  let activeName=-1, conflictPlace='';
  const suggestions=document.getElementById('customer-names');
  const nameHint=document.getElementById('customer-hint');
  const field=name=>form.elements.namedItem(name);
  function numberRange(){
    const area=field('area').value;
    field('number').min=area==='C2'?201:1;
    field('number').max=area==='C1'?200:area==='C2'?400:60;
  }
  function clearFieldError(el){
    const id='error-'+el.name.replaceAll('.','-');
    document.getElementById(id)?.remove();
    el.removeAttribute('aria-invalid');
    if(el.name==='name')el.setAttribute('aria-describedby','customer-hint');
    else el.removeAttribute('aria-describedby');
  }
  function markError(el,message){
    clearFieldError(el);
    const error=document.createElement('span');error.className='field-error';error.id='error-'+el.name.replaceAll('.','-');error.textContent=message;
    el.setAttribute('aria-invalid','true');el.setAttribute('aria-describedby',(el.name==='name'?'customer-hint ':'')+error.id);el.parentElement.append(error);
  }
  function validateField(el){
    if(!el.matches('input,select,textarea'))return true;
    if(el.name==='number'&&conflictPlace===field('area').value+'.'+el.value){markError(el,'Dieser Lagerplatz ist belegt oder reserviert. Bitte einen anderen wählen.');return false;}
    clearFieldError(el);
    if(el.disabled)return true;
    if(!el.checkValidity()||(el.required&&el.type!=='checkbox'&&!el.value.trim())){
      markError(el,el.validity.valueMissing||!el.value.trim()?'Bitte dieses Pflichtfeld ausfüllen.':el.validationMessage);
      return false;
    }
    return true;
  }
  function clearErrors(){
    form.querySelectorAll('input,select,textarea').forEach(clearFieldError);
  }
  form.addEventListener('focusout',event=>validateField(event.target));
  for(const event of ['input','change'])form.addEventListener(event,event=>{
    if(event.target.hasAttribute('aria-invalid'))validateField(event.target);
  });
  function hideNames(){
    suggestions.hidden=true;activeName=-1;
    field('name').setAttribute('aria-expanded','false');
    field('name').removeAttribute('aria-activedescendant');
  }
  function activateName(index){
    const items=[...suggestions.children];if(!items.length)return;
    activeName=(index+items.length)%items.length;
    items.forEach((item,i)=>item.setAttribute('aria-selected',String(i===activeName)));
    field('name').setAttribute('aria-activedescendant',items[activeName].id);
    items[activeName].scrollIntoView({block:'nearest'});
  }
  function chooseName(){
    const item=suggestions.children[activeName];if(!item)return;
    field('name').value=item.textContent;clearTimeout(nameTimer);nameRequest?.abort();nameGeneration++;
    hideNames();validateField(field('name'));
    nameHint.textContent='Kundenname übernommen.';
  }
  suggestions.addEventListener('pointerdown',event=>event.preventDefault());
  suggestions.addEventListener('click',event=>{
    const item=event.target.closest('[role=option]');if(!item)return;
    activateName([...suggestions.children].indexOf(item));chooseName();field('name').focus();
  });
  field('name').addEventListener('keydown',event=>{
    if(suggestions.hidden)return;
    if(event.key==='ArrowDown'||event.key==='ArrowUp'){
      event.preventDefault();activateName(activeName+(event.key==='ArrowDown'?1:-1));
    }else if(event.key==='Enter'){
      event.preventDefault();chooseName();
    }else if(event.key==='Tab'&&!event.shiftKey){
      chooseName(); // Keep normal Tab focus movement after accepting the suggestion.
    }else if(event.key==='Escape'){
      event.preventDefault();event.stopPropagation();hideNames();
    }
  });
  field('name').addEventListener('blur',()=>{hideNames();clearTimeout(nameTimer);nameRequest?.abort();nameGeneration++;});
  field('name').addEventListener('input',()=>{
    clearTimeout(nameTimer);nameRequest?.abort();const generation=++nameGeneration;
    hideNames();suggestions.replaceChildren();
    const q=field('name').value.trim();
    nameHint.textContent='Vorhandenen Namen auswählen oder neuen eingeben.';
    if(!q)return;
    nameTimer=setTimeout(async()=>{
      nameRequest=new AbortController();
      try{
        const response=await fetch('/api/customers?q='+encodeURIComponent(q),{signal:nameRequest.signal,cache:'no-store'});
        if(!response.ok)throw Error();
        const names=await response.json();
        if(generation!==nameGeneration||!dialog.open||document.activeElement!==field('name'))return;
        names.forEach((name,index)=>{const item=document.createElement('div');item.id='customer-option-'+index;item.setAttribute('role','option');item.textContent=name;suggestions.append(item);});
        if(names.length){suggestions.hidden=false;field('name').setAttribute('aria-expanded','true');activateName(0);}
        nameHint.textContent=names.length?'↑/↓ wählen · Tab oder Enter übernehmen · Esc schließen.':'Kein vorhandener Name gefunden. Neuer Name möglich.';
      }catch(error){if(error.name!=='AbortError'&&generation===nameGeneration)nameHint.textContent='Vorschläge nicht verfügbar. Name kann frei eingegeben werden.';}
    },180);
  });
  function resetForm(){
    form.reset();conflictPlace='';clearErrors();status.textContent='';status.classList.remove('is-error');hideNames();numberRange();
    for(const key of ['hubcaps','bolts','lock']){field(key+'.location').required=false;field(key+'.location').disabled=true;}
    clearTimeout(nameTimer);nameRequest?.abort();nameGeneration++;suggestions.replaceChildren();
    nameHint.textContent='Vorhandenen Namen auswählen oder neuen eingeben.';
  }
  dialog.addEventListener('close',resetForm);
  window.openStorage=async()=>{
    resetForm();dialog.showModal();
    if(ready){status.textContent='';return;}
    save.disabled=true;status.textContent='Auswahllisten werden geladen …';
    try {
      const response=await fetch('/api/storage/options');
      if(!response.ok)throw Error();
      const options=await response.json();
      form.querySelectorAll('select').forEach(select=>{
        const key=select.dataset.options||select.name;
        select.replaceChildren(new Option(select.required?'Bitte wählen':'Keine Angabe',''));
        for(const value of options[key])select.add(new Option(value,value));
      });
      ready=true;numberRange();status.textContent='';save.disabled=false;
    }catch{status.textContent='Auswahllisten konnten nicht geladen werden. Bitte schließen und erneut öffnen.';}
  };
  function clearPlaceConflict(){
    if(conflictPlace){conflictPlace='';clearFieldError(field('number'));status.textContent='';status.classList.remove('is-error');}
  }
  field('area').addEventListener('change',()=>{numberRange();clearPlaceConflict();});
  field('number').addEventListener('input',clearPlaceConflict);
  for(const key of ['hubcaps','bolts','lock']){
    field(key+'.present').addEventListener('change',()=>{
      const present=field(key+'.present').checked;
      field(key+'.location').disabled=!present;
      field(key+'.location').required=present;
    });
  }
  close.addEventListener('click',()=>{if(!saving)dialog.close();});
  dialog.addEventListener('cancel',event=>{if(saving)event.preventDefault();});
  form.addEventListener('submit',async event=>{
    event.preventDefault();if(saving||!ready)return;clearErrors();
    let valid=true;
    form.querySelectorAll('input,select,textarea').forEach(el=>{if(!validateField(el))valid=false;});
    if(!valid){status.classList.add('is-error');status.textContent='Bitte die rot markierten Pflichtfelder prüfen.';form.querySelector('[aria-invalid=true]')?.focus();return;}
    const value=name=>field(name).value;
    const input={};
    for(const key of ['area','name','brand','model','tire_type','rim_type','front_depth','rear_depth','mechanic','notes'])input[key]=value(key);
    input.number=Number(value('number'));
    input.plate=value('plate_district')+'-'+value('plate_letters').trim().toUpperCase()+' '+value('plate_number').trim().toUpperCase();
    for(const axle of ['front','rear']){
      input[axle]={};for(const key of ['width','ratio','inches','load','speed'])input[axle][key]=value(axle+'.'+key);
    }
    for(const key of ['hubcaps','bolts','lock'])input[key]={present:field(key+'.present').checked,location:value(key+'.location')};
    hideNames();status.classList.remove('is-error');saving=true;save.disabled=true;close.disabled=true;nav.forEach(button=>button.disabled=true);status.textContent='Wird gespeichert …';
    // Freeze the submitted snapshot until the response arrives.
    const enabled=[...form.elements].filter(el=>!el.disabled);enabled.forEach(el=>el.disabled=true);
    try {
      const response=await fetch('/api/storage',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(input)});
      const data=await response.json();
      if(!response.ok){
        status.classList.add('is-error');
        if(response.status===409){conflictPlace=input.area+'.'+input.number;markError(field('number'),'Lagerplatz '+conflictPlace+' ist belegt oder reserviert. Bitte einen anderen wählen.');}
        status.textContent=data.error||'Speichern fehlgeschlagen.';
        for(const [name,message] of Object.entries(data.fields||{})){
          const el=field(name==='plate'?'plate_district':name);if(!el)continue;
          markError(el,message);
        }
        return;
      }
      form.reset();numberRange();
      for(const key of ['hubcaps','bolts','lock']){field(key+'.location').required=false;}
      status.textContent=`Neueinlagerung auf ${data.place} gespeichert. Über die Schnellsuche auffindbar.`;
      const confirmation=document.getElementById('storage-confirmation');
      confirmation.textContent=status.textContent;confirmation.hidden=false;
      dialog.close();
    }catch{status.classList.add('is-error');status.textContent='Keine Speicherbestätigung erhalten. Bitte zuerst in der Schnellsuche prüfen, ob die Einlagerung gespeichert wurde. Ihre Eingaben bleiben erhalten.';}
    finally{
      enabled.forEach(el=>el.disabled=false);saving=false;save.disabled=false;close.disabled=false;nav.forEach(button=>button.disabled=false);
      for(const key of ['hubcaps','bolts','lock'])field(key+'.location').disabled=!field(key+'.present').checked;
      if(dialog.open){if(conflictPlace)field('number').focus();else status.focus();}
    }
  });
})();
