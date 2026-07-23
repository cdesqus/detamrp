'use client';
import { useEffect,useState } from 'react';
import { useRouter } from 'next/navigation';

type Doc={id:string;documentNumber:string;transactionDate:string;destination:string;kanbanCount:number;materialCount:number;status:string;createdBy:string};

export function OutgoingIndex(){
  const router=useRouter();
  const[items,setItems]=useState<Doc[]>([]);
  const[busy,setBusy]=useState(false);
  const[error,setError]=useState('');
  useEffect(()=>{fetch('/api/outgoing-material',{credentials:'include'}).then(r=>r.json()).then(d=>setItems(d.items??[])).catch(()=>setError('Outgoing data could not be loaded.'))},[]);
  async function create(){
    if(busy)return;
    setBusy(true);setError('');
    try{
      const response=await fetch('/api/outgoing-sessions',{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({destination:'',notes:''})});
      if(!response.ok)throw new Error('Outgoing session could not be created.');
      const session=await response.json();
      router.push(`/outgoing-material/${session.id}`);
    }catch(cause){setError(cause instanceof Error?cause.message:'Outgoing session could not be created.')}finally{setBusy(false)}
  }
  return <section className="module-index"><div className="page-title-row"><div><h1>Outgoing Material</h1><p className="muted">Consume complete Kanban lots from Raw Material stock.</p></div><button className="primary-button" disabled={busy} onClick={create}>{busy?'Creating...':'Create outgoing'}</button></div>{error&&<p className="form-error" role="alert">{error}</p>}<div className="table-frame"><table><thead><tr><th>Document Number</th><th>Date</th><th>Destination</th><th>Kanban</th><th>Materials</th><th>Status</th><th>Created By</th><th>Document</th></tr></thead><tbody>{items.length===0?<tr><td colSpan={8}><div className="table-empty">No outgoing material yet.</div></td></tr>:items.map(item=><tr key={item.id}><td>{item.documentNumber}</td><td>{item.transactionDate?.slice(0,10)}</td><td>{item.destination||'—'}</td><td>{item.kanbanCount}</td><td>{item.materialCount}</td><td><span className="status-pill status-pill--green">{item.status}</span></td><td>{item.createdBy}</td><td><a className="supplier-order-document-link" href={`/api/outgoing-material/${item.id}/document.pdf`} target="_blank" rel="noopener noreferrer">PDF</a></td></tr>)}</tbody></table></div></section>;
}
