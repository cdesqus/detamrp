'use client';

import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';

type Customer = { id:string; code:string; name:string };
type FinishedGood = { id:string; itemCode:string; name:string; baseUnitCode:string; salesPrice:string; currency:string };
type OrderLine = { finishedGoodId:string; quantity:string };

export function SalesOrderForm() {
  const router = useRouter();
  const [customers,setCustomers] = useState<Customer[]>([]);
  const [goods,setGoods] = useState<FinishedGood[]>([]);
  const [customerId,setCustomerId] = useState('');
  const [customerPoReference,setCustomerPoReference] = useState('');
  const [notes,setNotes] = useState('');
  const [lines,setLines] = useState<OrderLine[]>([{finishedGoodId:'',quantity:'1'}]);
  const [error,setError] = useState('');
  const [saving,setSaving] = useState(false);
  useEffect(() => { Promise.all([fetch('/api/customers?active=true',{credentials:'include'}),fetch('/api/finished-goods?active=true',{credentials:'include'})]).then(async ([customersResponse,goodsResponse]) => {
    if (!customersResponse.ok || !goodsResponse.ok) throw new Error('Order choices could not be loaded');
    const [customerData,goodData] = await Promise.all([customersResponse.json(),goodsResponse.json()]);
    setCustomers(customerData.items ?? []); setGoods(goodData.items ?? []);
  }).catch((loadError:unknown) => setError(loadError instanceof Error ? loadError.message : 'Order choices could not be loaded')); }, []);
  function good(id:string) { return goods.find(item => item.id === id); }
  function update(index:number, patch:Partial<OrderLine>) { setLines(current => current.map((line,i) => i === index ? {...line,...patch} : line)); }
  async function submit(event:FormEvent) {
    event.preventDefault(); setSaving(true); setError('');
    try {
      const response = await fetch('/api/sales-orders',{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({customerId,customerPoReference,notes,lines})});
      const data = await response.json();
      if (!response.ok) { setError(data.message ?? 'Sales order could not be saved'); return; }
      router.push(`/sales-orders/${data.id}`);
    } catch { setError('Sales order could not be saved'); } finally { setSaving(false); }
  }
  return <section className="module-index"><div className="page-title-row"><div><h1>New Sales Order</h1><p className="muted">Select a customer and the finished goods they ordered. Prices follow the FG master and cannot be edited here.</p></div></div>
    <form onSubmit={submit} className="crud-modal crud-modal--wide" style={{position:'static',transform:'none',margin:0,width:'100%',maxWidth:980}}>{error&&<p className="form-error">{error}</p>}<div className="crud-fields"><label><span>Customer *</span><select aria-label="Customer *" required value={customerId} onChange={event=>setCustomerId(event.target.value)}><option value="">Select...</option>{customers.map(customer=><option key={customer.id} value={customer.id}>{customer.code} — {customer.name}</option>)}</select></label><label><span>Customer PO Reference</span><input value={customerPoReference} onChange={event=>setCustomerPoReference(event.target.value)}/></label><label><span>Notes</span><textarea value={notes} onChange={event=>setNotes(event.target.value)}/></label></div>
    <div className="table-frame"><table><thead><tr><th>Finished Good</th><th>Quantity</th><th>Unit</th><th>Master Price</th><th>Currency</th><th></th></tr></thead><tbody>{lines.map((line,index)=>{const selected=good(line.finishedGoodId);return <tr key={index}><td><select aria-label="Finished Good *" required value={line.finishedGoodId} onChange={event=>update(index,{finishedGoodId:event.target.value})}><option value="">Select...</option>{goods.map(item=><option key={item.id} value={item.id} disabled={lines.some((other,i)=>i!==index&&other.finishedGoodId===item.id)}>{item.itemCode} — {item.name}</option>)}</select></td><td><input aria-label="Quantity *" required type="number" min="0.000001" step="0.000001" value={line.quantity} onChange={event=>update(index,{quantity:event.target.value})}/></td><td>{selected?.baseUnitCode ?? '—'}</td><td><input aria-label="Master Price" value={selected?.salesPrice ?? ''} disabled readOnly/></td><td>{selected?.currency ?? '—'}</td><td>{lines.length>1&&<button type="button" className="table-action" onClick={()=>setLines(current=>current.filter((_,i)=>i!==index))}>Remove</button>}</td></tr>})}</tbody></table></div>
    <div className="crud-actions"><button type="button" onClick={()=>setLines(current=>[...current,{finishedGoodId:'',quantity:'1'}])}>+ Add finished good</button><span style={{flex:1}}/><button type="button" onClick={()=>router.push('/sales-orders')}>Cancel</button><button className="primary-button" disabled={saving}>{saving?'Saving...':'Save draft'}</button></div></form>
  </section>;
}
