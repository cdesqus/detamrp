'use client';
import {useEffect,useState} from 'react';
import {formatQuantity} from '../lib/number-format';

type Supplier={id:string;name:string};
type Row={receivingNumber:string;receivingDate:string;deliveryNoteNumber:string;poNumber:string;supplierName:string;rawMaterialCode:string;rawMaterialName:string;baseUnitCode:string;kanbanReceived:number;receivedQuantity:string;outstandingQuantity:string;sageNumber:string;createdBy:string};
type Result={items:Row[];totals:{kanbanReceived:number;receivedQuantity:string}};
const empty:Result={items:[],totals:{kanbanReceived:0,receivedQuantity:'0'}};

export function ReportIndex(){
  const[fromDate,setFromDate]=useState(''),[toDate,setToDate]=useState(''),[supplierId,setSupplierId]=useState(''),[search,setSearch]=useState('');
  const[suppliers,setSuppliers]=useState<Supplier[]>([]),[data,setData]=useState<Result>(empty),[loading,setLoading]=useState(false),[error,setError]=useState(''),[appliedQuery,setAppliedQuery]=useState<string|null>(null);
  useEffect(()=>{fetch('/api/master-data/suppliers?active=true&limit=200',{credentials:'include'}).then(r=>r.json()).then(x=>setSuppliers(x.items??[])).catch(()=>{})},[]);
  async function apply(){
    if(!fromDate||!toDate)return;
    const params=new URLSearchParams({fromDate,toDate});if(supplierId)params.set('supplierId',supplierId);if(search.trim())params.set('search',search.trim());
    const query=params.toString();setLoading(true);setError('');
    try{const response=await fetch(`/api/reports/receiving?${query}`,{credentials:'include'});if(!response.ok)throw new Error();setData(await response.json());setAppliedQuery(query)}
    catch{setAppliedQuery(null);setError('Receiving report could not be loaded.')}finally{setLoading(false)}
  }
  function reset(){setFromDate('');setToDate('');setSupplierId('');setSearch('');setAppliedQuery(null);setData(empty);setError('')}
  return <section className="report-index">
    <div className="page-title-row"><div><h1>Receiving Report</h1><p className="muted">Choose a date range before loading the report.</p></div>{appliedQuery&&<a className="primary-button" aria-label="Export PDF" href={`/api/reports/receiving.pdf?${appliedQuery}`} target="_blank" rel="noopener noreferrer">Export PDF</a>}</div>
    <div className="report-filters">
      <label>From Date<input type="date" value={fromDate} onChange={e=>setFromDate(e.target.value)}/></label>
      <label>To Date<input type="date" value={toDate} onChange={e=>setToDate(e.target.value)}/></label>
      <label>Supplier<select value={supplierId} onChange={e=>setSupplierId(e.target.value)}><option value="">All suppliers</option>{suppliers.map(s=><option key={s.id} value={s.id}>{s.name}</option>)}</select></label>
      <label>Reference<input placeholder="Receiving, DN, or PO number" value={search} onChange={e=>setSearch(e.target.value)}/></label>
      <button className="primary-button" disabled={!fromDate||!toDate||loading} onClick={apply}>{loading?'Loading...':'Apply Filters'}</button><button onClick={reset}>Reset</button>
    </div>
    {error&&<p className="form-error" role="alert">{error}</p>}
    {appliedQuery&&<div className="table-frame"><table><thead><tr><th>Receiving</th><th>Date</th><th>DN</th><th>PO</th><th>Supplier</th><th>Raw Material</th><th>Kanban</th><th>Received Qty</th><th>Outstanding</th><th>Sage No.</th><th>Created By</th></tr></thead><tbody>
      {data.items.length===0?<tr><td colSpan={11}><div className="table-empty">No receiving transactions found.</div></td></tr>:data.items.map((r,i)=><tr key={`${r.receivingNumber}-${r.rawMaterialCode}-${i}`}><td>{r.receivingNumber}</td><td>{r.receivingDate.slice(0,10)}</td><td>{r.deliveryNoteNumber}</td><td>{r.poNumber}</td><td>{r.supplierName}</td><td>{r.rawMaterialCode} — {r.rawMaterialName}</td><td>{r.kanbanReceived}</td><td>{formatQuantity(r.receivedQuantity)} {r.baseUnitCode}</td><td>{formatQuantity(r.outstandingQuantity)} {r.baseUnitCode}</td><td>{r.sageNumber||'—'}</td><td>{r.createdBy}</td></tr>)}
    </tbody><tfoot><tr><th colSpan={6}>Total</th><th>{data.totals.kanbanReceived}</th><th>{formatQuantity(data.totals.receivedQuantity)}</th><th colSpan={3}/></tr></tfoot></table></div>}
  </section>
}
