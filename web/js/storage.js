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
    for(const key of ['width','ratio','inches','load','speed'])field('rear.'+key).value=field('front.'+key).value;
  });
  let ready=false, saving=false;
  const field=name=>form.elements.namedItem(name);
  function numberRange(){
    const area=field('area').value;
    field('number').min=area==='C2'?201:1;
    field('number').max=area==='C1'?200:area==='C2'?400:60;
  }
  function clearErrors(){
    form.querySelectorAll('.field-error').forEach(el=>el.remove());
    form.querySelectorAll('[aria-invalid]').forEach(el=>{el.removeAttribute('aria-invalid');el.removeAttribute('aria-describedby');});
  }
  window.openStorage=async()=>{
    dialog.showModal();
    if(ready)return;
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
  field('area').addEventListener('change',numberRange);
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
    const value=name=>field(name).value;
    const input={};
    for(const key of ['area','name','brand','model','tire_type','rim_type','front_depth','rear_depth','mechanic','notes'])input[key]=value(key);
    input.number=Number(value('number'));
    input.plate=value('plate_district')+'-'+value('plate_letters').trim().toUpperCase()+' '+value('plate_number').trim().toUpperCase();
    for(const axle of ['front','rear']){
      input[axle]={};for(const key of ['width','ratio','inches','load','speed'])input[axle][key]=value(axle+'.'+key);
    }
    for(const key of ['hubcaps','bolts','lock'])input[key]={present:field(key+'.present').checked,location:value(key+'.location')};
    saving=true;save.disabled=true;close.disabled=true;nav.forEach(button=>button.disabled=true);status.textContent='Wird gespeichert …';
    // Freeze the submitted snapshot until the response arrives.
    const enabled=[...form.elements].filter(el=>!el.disabled);enabled.forEach(el=>el.disabled=true);
    try {
      const response=await fetch('/api/storage',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(input)});
      const data=await response.json();
      if(!response.ok){
        status.textContent=data.error||'Speichern fehlgeschlagen.';
        for(const [name,message] of Object.entries(data.fields||{})){
          const el=field(name==='plate'?'plate_district':name);if(!el)continue;
          const error=document.createElement('span');error.className='field-error';error.id='error-'+name.replaceAll('.','-');error.textContent=message;
          el.setAttribute('aria-invalid','true');el.setAttribute('aria-describedby',error.id);el.parentElement.append(error);
        }
        return;
      }
      form.reset();numberRange();
      for(const key of ['hubcaps','bolts','lock']){field(key+'.location').required=false;}
      status.textContent=`Neueinlagerung auf ${data.place} gespeichert. Über die Schnellsuche auffindbar.`;
      const search=document.getElementById('search');search.value=input.plate;search.dispatchEvent(new Event('input'));
    }catch{status.textContent='Keine Speicherbestätigung erhalten. Bitte zuerst in der Schnellsuche prüfen, ob die Einlagerung gespeichert wurde. Ihre Eingaben bleiben erhalten.';}
    finally{
      enabled.forEach(el=>el.disabled=false);saving=false;save.disabled=false;close.disabled=false;nav.forEach(button=>button.disabled=false);
      for(const key of ['hubcaps','bolts','lock'])field(key+'.location').disabled=!field(key+'.present').checked;
      status.focus();
    }
  });
})();
