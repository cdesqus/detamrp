'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';

type Line = { id:string; itemCode:string; name:string; unit:string; quantity:string; deliveredQuantity:string; remainingQuantity:string };
type Order = { number:string; customerName:string; lines:Line[] };

export function CustomerDeliveryForm({ orderId }:{ orderId:string }) {
  const router=useRouter(); const [order,setOrder]=useState<Order|null>(null); const [qty,setQty]=useState<Record<string,string>>({}); const [error,setError]=useState(''); const [saving,setSaving]=useState(false);
  useEffect(()=>{fetch(`/api/sales-orders/${orderId}`,{credentials:'include'}).then(r=>r.ok?r.json():Promise.reject()).then((item:Order)=>{setOrder(item);setQty(Object.fromEntries(item.lines.map(line=>[line.id,line.remainingQuantity])))}).catch(()=>setError('Sales order could not be loaded'));},[orderId]);
  async function save(){setSaving(true);setError('');try{const lines=Object.entries(qty).filter(([,quantity])=>Number(quantity)>0).map(([salesOrderLineId,quantity])=>({salesOrderLineId,quantity}));const response=await fetch(`/api/sales-orders/${orderId}/deliveries`,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({lines})});const data=await response.json();if(!response.ok){setError(data.message??'Delivery could not be created');return}router.push(`/sales-orders/${orderId}`)}catch{setError('Delivery could not be created')}finally{setSaving(false)}}
  if(!order)return <section className="module-index"><p className="muted">Loading sales order...</p></section>;
  return <section className="module-index"><div className="page-title-row"><div><h1>New Customer Delivery</h1><p className="muted">{order.number} · {order.customerName}. Delivery is final immediately.</p></div></div>{error&&<p className="form-error">{error}</p>}<div className="table-frame"><table><thead><tr><th>Finished Good</th><th>Ordered</th><th>Delivered</th><th>Remaining</th><th>Unit</th><th>Deliver Qty</th></tr></thead><tbody>{order.lines.map(line=><tr key={line.id}><td>{line.itemCode} — {line.name}</td><td>{line.quantity}</td><td>{line.deliveredQuantity}</td><td>{line.remainingQuantity}</td><td>{line.unit}</td><td><input type="number" min="0" max={line.remainingQuantity} step="0.000001" disabled={Number(line.remainingQuantity)<=0} value={qty[line.id]??''} onChange={event=>setQty(current=>({...current,[line.id]:event.target.value}))}/></td></tr>)}</tbody></table></div><div className="crud-actions"><button onClick={()=>router.push(`/sales-orders/${orderId}`)}>Cancel</button><button className="primary-button" disabled={saving} onClick={save}>{saving?'Saving...':'Create delivery'}</button></div></section>;
}
