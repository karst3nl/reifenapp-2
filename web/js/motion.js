'use strict';
// Preserve the visible contents until the native dialog's fade has finished.
// Reopening or a newer close must never be reset by an older transition.
(()=>{
 const generations=new WeakMap();
 window.afterDialogFade=(dialog,reset)=>{
  const generation=(generations.get(dialog)||0)+1;generations.set(dialog,generation);
  if(dialog.open)return;
  const finish=()=>{if(!dialog.open&&generations.get(dialog)===generation)reset();};
  const fades=dialog.getAnimations?dialog.getAnimations().filter(animation=>animation.effect?.target===dialog&&animation.transitionProperty==='opacity'):[];
  if(!fades.length){finish();return;}
  Promise.allSettled(fades.map(animation=>animation.finished)).then(finish);
 };
})();
