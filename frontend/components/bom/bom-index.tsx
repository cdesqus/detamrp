'use client';

import { Fragment, useCallback, useEffect, useState } from 'react';
import { Icon } from '../icons';

type Component = { itemCode:string; name:string; unit:string; usageQty:string };
type BOM = { id:string; output:{itemCode:string;name:string;unit:string}; revision:number; status:string; components:Component[] };

export function BOMIndex() {
  const [items,setItems]=useState<BOM[]>([]);
  const [expanded,setExpanded]=useState<Set<string>>(new Set());
  const [error,setError]=useState('');
  const [activating,setActivating]=useState<string|null>(null);
  const load=useCallback(()=>{setError('');return fetch('/api/boms',{credentials:'include'}).then(x=>x.ok?x.json():Promise.reject()).then(x=>setItems(x.items??[])).catch(()=>setError('BOMs could not be loaded'))},[]);
  useEffect(()=>{void load()},[load]);
  async function activate(id:string){setActivating(id);setError('');try{const x=await fetch(`/api/boms/${id}/activate`,{method:'POST',credentials:'include'});if(!x.ok)throw new Error('BOM could not be activated');await load()}catch(e){setError(e instanceof Error?e.message:'BOM could not be activated')}finally{setActivating(null)}}
  function toggle(id:string){setExpanded(value=>{const next=new Set(value);if(next.has(id))next.delete(id);else next.add(id);return next})}
  return <section className="module-index">
    <div className="page-title-row"><div><h1>Bill of Materials</h1><p className="muted">Manage draft, active, and revised recipes for each output item.</p></div><a className="primary-button" style={{textDecoration:'none'}} href="/boms/new">New BOM</a></div>
    {error&&<p className="form-error" role="alert">{error}</p>}
    <div className="table-frame"><table><thead><tr><th>Output</th><th>Item</th><th>Revision</th><th>Status</th><th>Components</th><th>Actions</th></tr></thead><tbody>
      {items.length===0?<tr><td colSpan={6}><div className="table-empty"><strong>No BOMs yet</strong><span>Create a BOM to define required components.</span></div></td></tr>:items.map(item=><Fragment key={item.id}>
        <tr className={expanded.has(item.id)?'bom-parent-row bom-parent-row--expanded':'bom-parent-row'}>
          <td><button className="bom-expand-button" aria-label={`${expanded.has(item.id)?'Hide':'Show'} components for ${item.output.itemCode}`} aria-expanded={expanded.has(item.id)} onClick={()=>toggle(item.id)}><Icon name={expanded.has(item.id)?'chevron-down':'chevron-right'} size={16}/></button><strong>{item.output.itemCode}</strong></td>
          <td>{item.output.name}</td><td>R{item.revision}</td><td><span className={`status-badge status-${item.status.toLowerCase()}`}>{item.status}</span></td><td>{item.components.length}</td>
          <td>{item.status==='DRAFT'&&<div className="toolbar-actions"><a className="table-action" href={`/boms/${item.id}/edit`}>Edit</a><button className="primary-button" disabled={activating===item.id} onClick={()=>activate(item.id)}>{activating===item.id?'Activating...':'Activate'}</button></div>}</td>
        </tr>
        {expanded.has(item.id)&&<><tr className="bom-detail-heading"><td colSpan={6}><strong>Components for {item.output.name}</strong><span>Revision R{item.revision} · {item.status} · usage basis: 1 {item.output.unit} output</span></td></tr>{item.components.map((component,index)=><tr className="bom-component-row" key={`${item.id}-${component.itemCode}-${index}`}><td><span className="bom-component-indent"><Icon name="chevron-right" size={14}/>{component.itemCode}</span></td><td>{component.name}</td><td>—</td><td><span className="status-badge status-component">COMPONENT</span></td><td>{component.usageQty} {component.unit} per 1 {item.output.unit} output</td><td/></tr>)}</>}
      </Fragment>)}
    </tbody></table></div>
  </section>;
}
