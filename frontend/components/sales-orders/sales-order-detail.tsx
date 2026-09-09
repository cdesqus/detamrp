'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { MaterialRequirements, RequirementResult } from './material-requirements';
import { formatMoney, formatQuantity } from '../../lib/number-format';

type Line = { id:string; itemCode:string; name:string; unit:string; quantity:string; salesPrice:string; currency:string };
type Order = { id:string; number:string; customerName:string; status:string; orderDate:string; deliveryDate?:string; customerPoReference?:string; notes?:string; lines:Line[] };
type Calculation = { lineId:string; result:RequirementResult };

export function SalesOrderDetail({id}:{id:string}) {
 const router=useRouter(); const [order,setOrder]=useState<Order|null>(null); const [calculation,setCalculation]=useState<Calculation[]>([]); const [error,setError]=useState(''); const [submitting,setSubmitting]=useState(false);
 useEffect(()=>{fetch(`/api/sales-orders/${id}`,{credentials:'include'}).then(response=>response.ok?response.json():Promise.reject()).then((item:Order)=>{setOrder(item);if(item.status==='SUBMITTED')return fetch(`/api/sales-orders/${id}/requirements`,{credentials:'include'}).then(response=>response.ok?response.json():Promise.reject()).then(data=>setCalculation(data.lines??[]))}).catch(()=>setError('Sales order could not be loaded'));},[id]);
 async function submit(){setSubmitting(true);setError('');try{const response=await fetch(`/api/sales-orders/${id}/submit`,{method:'POST',credentials:'include'});if(!response.ok){const body=await response.json();setError(body.message??'Sales order could not be submitted');return}const item=await response.json();setOrder(item);const requirements=await fetch(`/api/sales-orders/${id}/requirements`,{credentials:'include'});if(requirements.ok){const data=await requirements.json();setCalculation(data.lines??[])}}catch{setError('Sales order could not be submitted')}finally{setSubmitting(false)}}
 if(error&&!order)return <section className="module-index"><div className="table-empty"><strong>Could not load sales order</strong><span>{error}</span></div></section>;
 if(!order)return <section className="module-index"><p className="muted">Loading sales order...</p></section>;
 return <section className="module-index"><div className="page-title-row"><div><h1>{order.number}</h1><p className="muted">{order.customerName} · {order.status}</p></div><div className="toolbar-actions"><button onClick={()=>router.push('/sales-orders')}>Back to orders</button>{order.status==='DRAFT'&&<button className="primary-button" disabled={submitting} onClick={submit}>{submitting?'Submitting...':'Submit sales order'}</button>}</div></div>{error&&<p className="form-error">{error}</p>}
 <div className="table-frame"><table><thead><tr><th>Finished Good</th><th>Quantity</th><th>Unit</th><th>Sales Price</th><th>Currency</th></tr></thead><tbody>{order.lines.map(line=><tr key={line.id}><td>{line.itemCode} — {line.name}</td><td>{formatQuantity(line.quantity)}</td><td>{line.unit}</td><td>{formatMoney(line.salesPrice,line.currency)}</td><td>{line.currency}</td></tr>)}</tbody></table></div>
 {order.status==='DRAFT'?<p className="muted">Submit this order to freeze its BOM calculation and show raw-material requirements.</p>:calculation.map(line=><div key={line.lineId}><h2 style={{marginTop:28}}>Calculation: {order.lines.find(item=>item.id===line.lineId)?.itemCode}</h2><MaterialRequirements result={line.result} showCosts/></div>)}</section>;
}
