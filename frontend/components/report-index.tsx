'use client';
import {useCallback,useEffect,useMemo,useState} from 'react';
import {formatQuantity} from '../lib/number-format';

type Supplier={id:string;name:string};
type Row={receivingNumber:string;receivingDate:string;deliveryNoteNumber:string;poNumber:string;supplierName:string;rawMaterialCode:string;rawMaterialName:string;baseUnitCode:string;kanbanReceived:number;receivedQuantity:string;outstandingQuantity:string;sageNumber:string;createdBy:string};
type Result={items:Row[];totals:{kanbanReceived:number;receivedQuantity:string}};
const empty:Result={items:[],totals:{kanbanReceived:0,receivedQuantity:'0'}};

export function ReportIndex(){
  const[fromDate,setFromDate]=useState(''),[toDate,setToDate]=useState(''),[supplierId,setSupplierId]=useState(''),[search,setSearch]=useState('');
  const[applied,setApplied]=useState({fromDate:'',toDate:'',supplierId:'',search:''});
  const[suppliers,setSuppliers]=useState<Supplier[]>([]),[data,setData]=useState<Result>(empty),[loading,setLoading]=useState(true),[error,setError]=useState('');
  const query=useMemo(()=>{const p=new URLSearchParams();Object.entries(applied).forEach(([k,v])=>v&&p.set(k,v));return p.toString()},[applied]);
  const load=useCallback(async()=>{setLoading(true);setError('');try{const r=await fetch(`/api/reports/receiving${query?`?${query}`:''}`,{credentials:'include'});if(!r.ok)throw new Error();setData(await r.json())}catch{setError('Receiving report could not be loaded.')}finally{setLoading(false)}},[query]);
  useEffect(()=>{fetch('/api/master-data/suppliers?active=true&limit=200',{credentials:'include'}).then(r=>r.json()).then(x=>setSuppliers(x.items??[])).catch(()=>{});void load()},[load]);
  function apply(){setApplied({fromDate,toDate,supplierId,search:search.trim()})}
  function reset(){setFromDate('');setToDate('');setSupplierId('');setSearch('');setApplied({fromDate:'',toDate:'',supplierId:'',search:''})}
  const pdf=`/api/reports/receiving.pdf${query?`?${query}`:''}`;
  return <section className="report-index">
    <div className="page-title-row"><div><h1>Receiving Report</h1><p className="muted">Completed receiving transactions by material.</p></div><a className="primary-button" aria-label="Export PDF" href={pdf} target="_blank" rel="noopener noreferrer">Export PDF</a></div>
    <div className="report-filters">
      <label>From Date<input type="date" value={fromDate} onChange={e=>setFromDate(e.target.value)}/></label>
      <label>To Date<input type="date" value={toDate} onChange={e=>setToDate(e.target.value)}/></label>
      <label>Supplier<select value={supplierId} onChange={e=>setSupplierId(e.target.value)}><option value="">All suppliers</option>{suppliers.map(s=><option key={s.id} value={s.id}>{s.name}</option>)}</select></label>
      <label>Reference<input placeholder="Receiving, DN, or PO number" value={search} onChange={e=>setSearch(e.target.value)}/></label>
      <button onClick={apply}>Apply Filters</button><button onClick={reset}>Reset</button>
    </div>
    {error&&<p className="form-error" role="alert">{error}</p>}
    <div className="table-frame"><table><thead><tr><th>Receiving</th><th>Date</th><th>DN</th><th>PO</th><th>Supplier</th><th>Raw Material</th><th>Kanban</th><th>Received Qty</th><th>Outstanding</th><th>Sage No.</th><th>Created By</th></tr></thead><tbody>
      {loading?<tr><td colSpan={11}>Loading report...</td></tr>:data.items.length===0?<tr><td colSpan={11}><div className="table-empty">No receiving transactions found.</div></td></tr>:data.items.map((r,i)=><tr key={`${r.receivingNumber}-${r.rawMaterialCode}-${i}`}><td>{r.receivingNumber}</td><td>{r.receivingDate.slice(0,10)}</td><td>{r.deliveryNoteNumber}</td><td>{r.poNumber}</td><td>{r.supplierName}</td><td>{r.rawMaterialCode} — {r.rawMaterialName}</td><td>{r.kanbanReceived}</td><td>{formatQuantity(r.receivedQuantity)} {r.baseUnitCode}</td><td>{formatQuantity(r.outstandingQuantity)} {r.baseUnitCode}</td><td>{r.sageNumber||'—'}</td><td>{r.createdBy}</td></tr>)}
    </tbody><tfoot><tr><th colSpan={6}>Total</th><th>{data.totals.kanbanReceived}</th><th>{formatQuantity(data.totals.receivedQuantity)}</th><th colSpan={3}/></tr></tfoot></table></div>
  </section>
}
