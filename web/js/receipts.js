'use strict';
// Reserve a tab in the submit gesture; browsers may block window.open after await.
window.ReceiptTabs={
  prepare(){
    let tab=null;
    try{
      tab=window.open('about:blank','_blank');
      if(tab){tab.opener=null;tab.document.title='Vorgangsbeleg';tab.document.body.textContent='Vorgang wird gespeichert. Der PDF-Beleg wird anschließend geladen …';}
    }catch{if(tab&&!tab.closed)tab.close();tab=null;}
    return {
      cancel(){if(tab&&!tab.closed)tab.close();},
      complete(path,confirmation){
        try{
          const url=new URL(path,location.origin);
          if(url.origin!==location.origin||!url.pathname.startsWith('/api/receipts/')||!url.pathname.endsWith('.pdf'))throw Error();
          let opened=false;
          if(tab&&!tab.closed){try{tab.location.replace(url.href);opened=true;}catch{tab.close();}}
          const link=document.createElement('a');link.href=url.href;link.target='_blank';link.rel='noopener noreferrer';
          link.textContent=opened?'PDF erneut öffnen':'PDF öffnen';
          confirmation.append(document.createTextNode(opened?' · ':' · PDF-Tab nicht geöffnet: '),link);
        }catch{
          if(tab&&!tab.closed)tab.close();
          confirmation.append(document.createTextNode(' · PDF bitte über die Suche erneut öffnen.'));
        }
      }
    };
  }
};
