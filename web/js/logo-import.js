'use strict';
// Decode locally and store only normalized pixels, never executable SVG markup.
window.importLogo=async function(file){
 if(!file||file.size>5*1024*1024)throw Error('Bitte eine PNG-, JPEG- oder SVG-Datei mit höchstens 5 MiB auswählen.');
 const extension=file.name.split('.').pop().toLowerCase();
 if(!['png','jpg','jpeg','svg'].includes(extension))throw Error('Unterstützte Logoformate: PNG, JPEG und SVG.');
 let blob=file;
 if(extension==='svg'){
  const source=await file.text();
  if(/<!DOCTYPE|<!ENTITY|<\?xml-stylesheet/i.test(source))throw Error('SVG mit Dokumenttyp oder Entitäten wird nicht unterstützt.');
  const doc=new DOMParser().parseFromString(source,'image/svg+xml'),root=doc.documentElement;
  if(doc.querySelector('parsererror')||root.localName!=='svg'||root.namespaceURI!=='http://www.w3.org/2000/svg')throw Error('Die SVG-Datei ist ungültig.');
  for(const node of [root,...root.querySelectorAll('*')]){
   if(node.namespaceURI!==root.namespaceURI||['script','foreignObject','image','animate','animateMotion','animateTransform','set','a'].includes(node.localName))throw Error('SVG bitte als statische Grafik ohne eingebettete Bilder, Skripte oder Verknüpfungen exportieren.');
   for(const attr of node.attributes){
    if(/^on/i.test(attr.localName)||(/href$/i.test(attr.localName)&&!attr.value.startsWith('#')))throw Error('SVG mit aktiven Inhalten oder externen Verweisen wird nicht unterstützt.');
   }
   const css=node.localName==='style'?node.textContent:'';
   const attrs=[...node.attributes].map(a=>a.value).join(' '),content=css+' '+attrs;
   if(/@import|@font-face|javascript:|https?:|\/\//i.test(css)||/url\s*\(\s*(?!["']?#)[^)]*\)/i.test(content))throw Error('SVG mit externen Ressourcen wird nicht unterstützt.');
  }
  const view=(root.getAttribute('viewBox')||'').trim().split(/[\s,]+/).map(Number);
  const dimension=value=>{const match=/^([\d.]+)(px|mm|cm|in|pt|pc)?$/.exec(value||'');return match?Number(match[1])*({px:1,mm:96/25.4,cm:96/2.54,in:96,pt:96/72,pc:16}[match[2]||'px']):0;};
  const width=dimension(root.getAttribute('width'))||(view.length===4?view[2]:0),height=dimension(root.getAttribute('height'))||(view.length===4?view[3]:0);
  if(!Number.isFinite(width)||!Number.isFinite(height)||width<=0||height<=0)throw Error('SVG benötigt gültige Breite/Höhe oder eine viewBox.');
  if(!root.hasAttribute('viewBox'))root.setAttribute('viewBox',`0 0 ${width} ${height}`);
  const scale=2048/Math.max(width,height);root.setAttribute('width',String(Math.max(1,Math.round(width*scale))));root.setAttribute('height',String(Math.max(1,Math.round(height*scale))));
  blob=new Blob([new XMLSerializer().serializeToString(root)],{type:'image/svg+xml'});
 }else{
  // Use the selected format explicitly; the decoder still verifies actual bytes.
  blob=new Blob([await file.arrayBuffer()],{type:extension==='png'?'image/png':'image/jpeg'});
 }
 const url=URL.createObjectURL(blob),image=new Image();
 try{
  image.src=url;await image.decode();
  if(!image.naturalWidth||!image.naturalHeight)throw Error();
  const scale=Math.min(1,2048/Math.max(image.naturalWidth,image.naturalHeight));
  const canvas=document.createElement('canvas');canvas.width=Math.max(1,Math.round(image.naturalWidth*scale));canvas.height=Math.max(1,Math.round(image.naturalHeight*scale));
  const context=canvas.getContext('2d');context.drawImage(image,0,0,canvas.width,canvas.height);
  const data=canvas.toDataURL('image/png');
  if((data.length-'data:image/png;base64,'.length)*3/4>2*1024*1024)throw Error('Das aufbereitete Logo ist zu groß (maximal 2 MiB). Bitte eine kleinere Grafik verwenden.');
  return {data,width:canvas.width,height:canvas.height};
 }catch(error){throw Error(error.message.startsWith('Das aufbereitete')?error.message:'Logo konnte nicht gelesen werden. Bitte als gültiges PNG, JPEG oder statisches SVG exportieren.');}
 finally{URL.revokeObjectURL(url);}
};
