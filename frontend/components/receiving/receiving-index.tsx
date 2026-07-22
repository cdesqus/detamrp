'use client';

import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';

type Receiving = { id:string; receivingNumber:string; deliveryNoteNumber:string; poNumber:string; supplierName:string; receivingDate:string; receivedNow:number; outstanding:number; status:string; sageReceiptNumber:string; createdBy:string };
type OpenSession = { id:string; receivingNumber:string; deliveryNoteNumber:string; status:string; scans?:unknown[] };

const validationMessages:Record<string,string> = {
  DN_INVALID: 'Delivery Note is invalid.',
  DN_FULLY_RECEIVED: 'Delivery Note has already been fully received.',
  DN_IN_PROGRESS: 'Delivery Note is currently being received in another session.',
};

export function ReceivingIndex() {
  const router = useRouter();
  const [items,setItems] = useState<Receiving[]>([]);
  const [openSessions,setOpenSessions] = useState<OpenSession[]>([]);
  const [open,setOpen] = useState(false);
  const [value,setValue] = useState('');
  const [busy,setBusy] = useState(false);
  const [error,setError] = useState('');

  useEffect(()=>{
    Promise.all([
      fetch('/api/receivings',{credentials:'include'}).then(r=>r.json()),
      fetch('/api/receiving-sessions',{credentials:'include'}).then(r=>r.json()),
    ]).then(([done,sessions])=>{
      setItems(done.items??[]);
      setOpenSessions(sessions.items??[]);
    }).catch(()=>setError('Receiving data could not be loaded.'));
  },[]);

  function closeModal(){ setOpen(false); setValue(''); setError(''); }

  async function create(event:FormEvent){
    event.preventDefault();
    const deliveryNoteNumber=value.trim().toUpperCase();
    if(!deliveryNoteNumber||busy)return;
    setBusy(true);setError('');
    try{
      const response=await fetch('/api/receiving-sessions',{
        method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},
        body:JSON.stringify({deliveryNoteNumber}),
      });
      const payload=await response.json().catch(()=>({}));
      if(!response.ok){
        throw new Error(validationMessages[payload.code]??'Receiving session could not be created.');
      }
      setOpen(false);
      router.push(`/receiving/${payload.id}`);
    }catch(cause){
      setError(cause instanceof Error?cause.message:'Receiving session could not be created.');
    }finally{setBusy(false)}
  }

  return <section className="module-index">
    <div className="page-title-row"><div><h1>Receiving</h1><p className="muted">Receive approved Kanban lots against the issued Delivery Note.</p></div><button className="primary-button" onClick={()=>{setOpen(true);setError('')}}>Create receiving</button></div>
    {!open&&error&&<p className="form-error" role="alert">{error}</p>}
    {openSessions.map(session=><div className="receiving-open-session" key={session.id}><span><b>{session.receivingNumber}</b> · {session.deliveryNoteNumber} · {session.status} · {session.scans?.length??0} scanned</span><button className="table-action" onClick={()=>router.push(`/receiving/${session.id}`)}>Resume session</button></div>)}
    <div className="table-frame"><table><thead><tr><th>Receiving Number</th><th>DN Number</th><th>PO Number</th><th>Supplier</th><th>Date</th><th>Kanban</th><th>Outstanding</th><th>Status</th><th>Sage Number</th><th>Created By</th><th>Document</th></tr></thead><tbody>{items.length===0?<tr><td colSpan={11}><div className="table-empty">No completed receiving yet.</div></td></tr>:items.map(item=><tr key={item.id}><td>{item.receivingNumber}</td><td>{item.deliveryNoteNumber}</td><td>{item.poNumber}</td><td>{item.supplierName}</td><td>{item.receivingDate?.slice(0,10)}</td><td>{item.receivedNow}</td><td>{item.outstanding}</td><td><span className="status-pill status-pill--green">{item.status}</span></td><td>{item.sageReceiptNumber||'—'}</td><td>{item.createdBy}</td><td><a className="supplier-order-document-link" target="_blank" rel="noopener noreferrer" href={`/api/receivings/${item.id}/document.pdf`}>PDF</a></td></tr>)}</tbody></table></div>
    {open&&<><button className="crud-scrim" aria-label="Close receiving form" onClick={closeModal}/><div className="crud-modal" role="dialog" aria-modal="true" aria-label="New receiving"><div className="crud-modal-heading"><div><strong>New receiving</strong><span>Scan the DN attached to the physical shipment.</span></div><button aria-label="Close receiving form" onClick={closeModal}>×</button></div><form onSubmit={create}><div className="crud-fields"><label>Scan or Type DN Number<input aria-label="Scan or Type DN Number" value={value} onChange={event=>setValue(event.target.value)} autoComplete="off" autoFocus/></label>{error&&<p className="form-error" role="alert">{error}</p>}</div><div className="crud-actions"><button type="button" onClick={closeModal}>Cancel</button><button className="primary-button" disabled={!value.trim()||busy}>{busy?'Validating...':'Continue'}</button></div></form></div></>}
  </section>;
}
