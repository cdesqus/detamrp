'use client';
import {useEffect,useState} from 'react';
import {formatQuantity} from '../lib/number-format';

type Supplier={id:string;name:string};
type Row={receivingNumber:string;receivingDate:string;deliveryNoteNumber:string;poNumber:string;supplierName:string;rawMaterialCode:string;rawMaterialName:string;baseUnitCode:string;kanbanReceived:number;receivedQuantity:string;outstandingQuantity:string;sageNumber:string;createdBy:string};
type Result={items:Row[];totals:{kanbanReceived:number;receivedQuantity:string}};
const empty:Result={items:[],totals:{kanbanReceived:0,receivedQuantity:'0'}};
const reportLabels:Record<string,string>={number:'Number',customer:'Customer',orderDate:'Order Date',status:'Status',itemCode:'Item Code',itemName:'Item Name',ordered:'Ordered',delivered:'Delivered',remaining:'Remaining',unit:'Unit',required:'Required Qty',qtyPerKanban:'Qty / Kanban',purchaseKanban:'Purchase Kanban',deliveryDate:'Delivery Date',salesOrderNumber:'Sales Order',quantity:'Quantity'};

export function ReportIndex(){
  const[report,setReport]=useState<'receiving'|'sales-orders'|'material-requirements'|'customer-deliveries'>('receiving');
  const[fromDate,setFromDate]=useState(''),[toDate,setToDate]=useState(''),[supplierId,setSupplierId]=useState(''),[search,setSearch]=useState('');
  const[suppliers,setSuppliers]=useState<Supplier[]>([]),[data,setData]=useState<Result>(empty),[loading,setLoading]=useState(false),[error,setError]=useState(''),[appliedQuery,setAppliedQuery]=useState<string|null>(null);
  useEffect(()=>{fetch('/api/master-data/suppliers?active=true&limit=200',{credentials:'include'}).then(r=>r.json()).then(x=>setSuppliers(x.items??[])).catch(()=>{})},[]);
  async function apply(){
    if(report!=='receiving'){setLoading(true);setError('');try{const response=await fetch(`/api/reports/${report}`,{credentials:'include'});if(!response.ok)throw new Error();const payload=await response.json() as Result;const items=payload.items.filter((item:unknown)=>{const row=item as Record<string,string>;const date=(row.orderDate??row.deliveryDate??'').slice(0,10);return (!fromDate||date>=fromDate)&&(!toDate||date<=toDate)&&(!search.trim()||Object.values(row).join(' ').toLowerCase().includes(search.trim().toLowerCase()))});setData({...payload,items});setAppliedQuery('loaded')}catch{setError('Report could not be loaded.')}finally{setLoading(false)};return}
    if(!fromDate||!toDate)return;
    const params=new URLSearchParams({fromDate,toDate});if(supplierId)params.set('supplierId',supplierId);if(search.trim())params.set('search',search.trim());
    const query=params.toString();setLoading(true);setError('');
    try{const response=await fetch(`/api/reports/receiving?${query}`,{credentials:'include'});if(!response.ok)throw new Error();setData(await response.json());setAppliedQuery(query)}
    catch{setAppliedQuery(null);setError('Receiving report could not be loaded.')}finally{setLoading(false)}
  }
  function reset(){setFromDate('');setToDate('');setSupplierId('');setSearch('');setAppliedQuery(null);setData(empty);setError('')}
  function exportCSV(){const rows=data.items as unknown as Record<string,unknown>[];if(!rows.length)return;const headers=Object.keys(rows[0]);const escape=(value:unknown)=>`"${String(value??'').replaceAll('"','""')}"`;const csv=[headers.join(','),...rows.map(row=>headers.map(header=>escape(row[header])).join(','))].join('\r\n');const url=URL.createObjectURL(new Blob([csv],{type:'text/csv;charset=utf-8'}));const link=document.createElement('a');link.href=url;link.download=`${report}-report.csv`;link.click();URL.revokeObjectURL(url)}
  return <section className="report-index">
    <div className="toolbar-actions"><button className={report==='receiving'?'primary-button':''} onClick={()=>setReport('receiving')}>Receiving</button><button className={report==='sales-orders'?'primary-button':''} onClick={()=>setReport('sales-orders')}>Sales Orders</button><button className={report==='material-requirements'?'primary-button':''} onClick={()=>setReport('material-requirements')}>Material Requirements</button><button className={report==='customer-deliveries'?'primary-button':''} onClick={()=>setReport('customer-deliveries')}>Customer Deliveries</button></div>
    <div className="page-title-row"><div><h1>{report==='receiving'?'Receiving Report':report==='sales-orders'?'Sales Order Report':report==='material-requirements'?'Material Requirement Report':'Customer Delivery Report'}</h1><p className="muted">{report==='receiving'?'Choose a date range before loading the report.':'Load the current report for manual SAGE input.'}</p></div>{report==='receiving'&&appliedQuery&&<a className="primary-button" aria-label="Export PDF" href={`/api/reports/receiving.pdf?${appliedQuery}`} target="_blank" rel="noopener noreferrer">Export PDF</a>}</div>
    {report==='receiving'&&<div className="report-filters">
      <label>From Date<input type="date" value={fromDate} onChange={e=>setFromDate(e.target.value)}/></label>
      <label>To Date<input type="date" value={toDate} onChange={e=>setToDate(e.target.value)}/></label>
      <label>Supplier<select value={supplierId} onChange={e=>setSupplierId(e.target.value)}><option value="">All suppliers</option>{suppliers.map(s=><option key={s.id} value={s.id}>{s.name}</option>)}</select></label>
      <label>Reference<input placeholder="Receiving, DN, or PO number" value={search} onChange={e=>setSearch(e.target.value)}/></label>
      <button className="primary-button" disabled={!fromDate||!toDate||loading} onClick={apply}>{loading?'Loading...':'Apply Filters'}</button><button onClick={reset}>Reset</button>
    </div>}
    {report!=='receiving'&&<div className="report-filters"><label>From Date<input type="date" value={fromDate} onChange={e=>setFromDate(e.target.value)}/></label><label>To Date<input type="date" value={toDate} onChange={e=>setToDate(e.target.value)}/></label><label>Search<input placeholder="Customer, SO, or item" value={search} onChange={e=>setSearch(e.target.value)}/></label><button className="primary-button" disabled={loading} onClick={apply}>{loading?'Loading...':'Load Report'}</button><button onClick={reset}>Reset</button>{appliedQuery&&<button onClick={exportCSV}>Export CSV</button>}</div>}
    {error&&<p className="form-error" role="alert">{error}</p>}
    {report==='receiving'&&appliedQuery&&<div className="table-frame"><table><thead><tr><th>Receiving</th><th>Date</th><th>DN</th><th>PO</th><th>Supplier</th><th>Raw Material</th><th>Kanban</th><th>Received Qty</th><th>Outstanding</th><th>Sage No.</th><th>Created By</th></tr></thead><tbody>
      {data.items.length===0?<tr><td colSpan={11}><div className="table-empty">No receiving transactions found.</div></td></tr>:data.items.map((r,i)=><tr key={`${r.receivingNumber}-${r.rawMaterialCode}-${i}`}><td>{r.receivingNumber}</td><td>{r.receivingDate.slice(0,10)}</td><td>{r.deliveryNoteNumber}</td><td>{r.poNumber}</td><td>{r.supplierName}</td><td>{r.rawMaterialCode} — {r.rawMaterialName}</td><td>{r.kanbanReceived}</td><td>{formatQuantity(r.receivedQuantity)} {r.baseUnitCode}</td><td>{formatQuantity(r.outstandingQuantity)} {r.baseUnitCode}</td><td>{r.sageNumber||'—'}</td><td>{r.createdBy}</td></tr>)}
    </tbody><tfoot><tr><th colSpan={6}>Total</th><th>{data.totals.kanbanReceived}</th><th>{formatQuantity(data.totals.receivedQuantity)}</th><th colSpan={3}/></tr></tfoot></table></div>}
    {report!=='receiving'&&appliedQuery&&<div className="table-frame"><table><thead><tr>{Object.keys(data.items[0]??{}).map(key=><th key={key}>{reportLabels[key]??key}</th>)}</tr></thead><tbody>{data.items.length===0?<tr><td><div className="table-empty">No report rows found.</div></td></tr>:data.items.map((row,i)=><tr key={i}>{Object.entries(row as Record<string,unknown>).map(([key,value])=><td key={key}>{['ordered','delivered','remaining','required','qtyPerKanban','purchaseKanban','quantity'].includes(key)?formatQuantity(String(value)):String(value)}</td>)}</tr>)}</tbody></table></div>}
  </section>
}
