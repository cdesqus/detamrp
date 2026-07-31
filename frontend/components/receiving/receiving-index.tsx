'use client';

import { FormEvent, useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { pagedItems, TablePagination } from '../table-pagination';
import { emptyTransactionFilters, TransactionFilters, TransactionListFilters } from '../transaction-list-filters';

type Receiving = { id:string; supplierId:string; receivingNumber:string; deliveryNoteNumber:string; poNumber:string; supplierName:string; receivingDate:string; receivedNow:number; outstanding:number; status:string; sageReceiptNumber:string; createdBy:string; createdAt:string };
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
  const [page,setPage]=useState(1);
  const [pageSize,setPageSize]=useState(20);
  const [loading,setLoading]=useState(true);
  const [filters,setFilters]=useState<TransactionFilters>(emptyTransactionFilters);
  const [appliedFilters,setAppliedFilters]=useState<TransactionFilters>(emptyTransactionFilters);
  const [suppliers,setSuppliers]=useState<{id:string;code?:string;name:string}[]>([]);

  useEffect(()=>{
    const params=new URLSearchParams();
    if(appliedFilters.supplierId)params.set('supplierId',appliedFilters.supplierId);
    if(appliedFilters.createdFrom)params.set('createdFrom',appliedFilters.createdFrom);
    if(appliedFilters.createdTo)params.set('createdTo',appliedFilters.createdTo);
    setLoading(true);
    Promise.all([
      fetch(`/api/receivings?${params}`,{credentials:'include'}).then(r=>{if(!r.ok)throw new Error();return r.json()}),
      fetch('/api/receiving-sessions',{credentials:'include'}).then(r=>r.json()),
    ]).then(([done,sessions])=>{
      setItems(done.items??[]);
      setOpenSessions(sessions.items??[]);
      setPage(1);
    }).catch(()=>setError('Receiving data could not be loaded.')).finally(()=>setLoading(false));
  },[appliedFilters]);

  useEffect(()=>{
    fetch('/api/master-data/suppliers?limit=200',{credentials:'include'}).then(r=>r.ok?r.json():Promise.reject()).then(data=>setSuppliers(data.items??[])).catch(()=>setSuppliers([]));
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

  return <section className="module-index receiving-index">
    <div className="page-title-row"><div><h1>Receiving</h1><p className="muted">Receive approved Kanban lots against the issued Delivery Note.</p></div><button className="primary-button" onClick={()=>{setOpen(true);setError('')}}>Create receiving</button></div>
    {!open&&error&&<p className="form-error" role="alert">{error}</p>}
    {openSessions.map(session=><div className="receiving-open-session" key={session.id}><span><b>{session.receivingNumber}</b> · {session.deliveryNoteNumber} · {session.status} · {session.scans?.length??0} scanned</span><button className="table-action" onClick={()=>router.push(`/receiving/${session.id}`)}>Resume session</button></div>)}
    <TransactionListFilters value={filters} suppliers={suppliers} recordCount={items.length} loading={loading} onChange={setFilters} onApply={()=>setAppliedFilters(filters)} onReset={()=>{setFilters(emptyTransactionFilters);setAppliedFilters(emptyTransactionFilters)}} />
    <div className="table-frame"><table><thead><tr><th className="transaction-number table-column-number">No.</th><th className="table-column-actions">Document</th><th>Status</th><th>Receiving Number</th><th>DN Number</th><th>PO Number</th><th>Supplier</th><th>Date</th><th>Kanban</th><th>Outstanding</th><th>Sage Number</th><th>Created By</th></tr></thead><tbody>{items.length===0?<tr><td className="table-row-empty" colSpan={12}><div className="table-empty">No completed receiving yet.</div></td></tr>:pagedItems(items,page,pageSize).map((item,index)=><tr key={item.id}><td className="transaction-number">{(page-1)*pageSize+index+1}</td><td><a className="compact-document-link" title={`Open Receiving PDF for ${item.receivingNumber}`} aria-label={`Open Receiving PDF for ${item.receivingNumber}`} target="_blank" rel="noopener noreferrer" href={`/api/receivings/${item.id}/document.pdf`}>RCV PDF</a></td><td><span className="status-pill status-pill--green">{item.status}</span></td><td>{item.receivingNumber}</td><td>{item.deliveryNoteNumber}</td><td>{item.poNumber}</td><td>{item.supplierName}</td><td>{item.receivingDate?.slice(0,10)}</td><td>{item.receivedNow}</td><td>{item.outstanding}</td><td>{item.sageReceiptNumber||'—'}</td><td>{item.createdBy}</td></tr>)}</tbody></table><TablePagination page={page} pageSize={pageSize} total={items.length} onPageChange={setPage} onPageSizeChange={size=>{setPageSize(size);setPage(1)}}/></div>
    {open&&<><button className="crud-scrim" aria-label="Close receiving form" onClick={closeModal}/><div className="crud-modal" role="dialog" aria-modal="true" aria-label="New receiving"><div className="crud-modal-heading"><div><strong>New receiving</strong><span>Scan the DN attached to the physical shipment.</span></div><button aria-label="Close receiving form" onClick={closeModal}>×</button></div><form onSubmit={create}><div className="crud-fields"><label>Scan or Type DN Number<input aria-label="Scan or Type DN Number" value={value} onChange={event=>setValue(event.target.value)} autoComplete="off" autoFocus/></label>{error&&<p className="form-error" role="alert">{error}</p>}</div><div className="crud-actions"><button type="button" onClick={closeModal}>Cancel</button><button className="primary-button" disabled={!value.trim()||busy}>{busy?'Validating...':'Continue'}</button></div></form></div></>}
  </section>;
}
