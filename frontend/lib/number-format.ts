function decimalParts(value: string | number | undefined, maximumFractionDigits: number) {
  const match=String(value??'0').trim().match(/^([+-]?)(\d*)(?:\.(\d*))?$/);
  if(!match)return {negative:false,whole:'0',fraction:''};
  const negative=match[1]==='-';
  const whole=(match[2]||'0').replace(/^0+(?=\d)/,'')||'0';
  const fraction=(match[3]||'').slice(0,maximumFractionDigits).replace(/0+$/,'');
  return {negative,whole,fraction};
}

function groupWhole(value:string){return value.replace(/\B(?=(\d{3})+(?!\d))/g,'.')}

export function formatQuantity(value:string|number|undefined,maximumFractionDigits=6){
  const {negative,whole,fraction}=decimalParts(value,maximumFractionDigits);
  return `${negative?'-':''}${groupWhole(whole)}${fraction?`,${fraction}`:''}`;
}

export function formatMoney(value:string|number|undefined,currency:string){
  const number=formatQuantity(value,currency.toUpperCase()==='IDR'?0:2);
  return `${currency.trim()} ${number}`.trim();
}
