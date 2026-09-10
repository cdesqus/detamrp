'use client';

import { useState } from 'react';
import { formatQuantity, formatMoney } from '../../lib/number-format';

export type RequirementNode = { path:string; parentPath:string; itemId:string; kind:string; itemCode:string; name:string; unit:string; usage:string; quantity:string; terminal:boolean; depth:number };
export type MaterialRequirement = { itemId:string; itemCode:string; name:string; unit:string; quantity:string; unitPrice?:string; currency?:string; value?:string; qtyPerKanban:string; kanbanEquivalent:string; purchaseKanban:string; paths:string[] };
export type RequirementResult = { nodes:RequirementNode[]; materials:MaterialRequirement[]; totals?:Record<string,string>; costComplete?:boolean };

export function MaterialRequirements({ result, showCosts = false, showBreakdown = true }: { result:RequirementResult; showCosts?:boolean; showBreakdown?:boolean }) {
 const [expanded,setExpanded]=useState<Set<string>>(new Set());
 const byPath=new Map(result.nodes.map(node=>[node.path,node]));
 const visible=result.nodes.filter(node=>{
  let parent=node.parentPath;
  const visited=new Set<string>();
  while(parent){if(visited.has(parent)||!expanded.has(parent))return false;visited.add(parent);parent=byPath.get(parent)?.parentPath??''}
  return true;
 });
 function toggle(path:string){setExpanded(previous=>{const next=new Set(previous);if(next.has(path))next.delete(path);else next.add(path);return next})}
 return <div className="module-index">
  {!showBreakdown ? null : <>
  <div className="page-title-row"><div><h2>BOM Breakdown</h2><p className="muted">Usage is per parent unit. Required quantities follow the selected order basis.</p></div><div className="toolbar-actions"><button className="table-action" onClick={()=>setExpanded(new Set(result.nodes.filter(node=>!node.terminal).map(node=>node.path)))}>Expand all</button><button className="table-action" onClick={()=>setExpanded(new Set())}>Collapse all</button></div></div>
  <div className="table-frame"><table><thead><tr><th>Item Code</th><th>Item Name</th><th>Usage / Parent Unit</th><th>Quantity Required</th><th>Unit</th></tr></thead><tbody>
   {visible.length===0?<tr><td colSpan={5}><div className="table-empty">No remaining requirements.</div></td></tr>:visible.map(node=><tr key={node.path}><td><span style={{paddingLeft:node.depth*20,display:'inline-flex',alignItems:'center',gap:8}}>{!node.terminal&&<button className="table-action" aria-expanded={expanded.has(node.path)} aria-label={`${expanded.has(node.path)?'Collapse':'Expand'} ${node.itemCode}`} onClick={()=>toggle(node.path)}>{expanded.has(node.path)?'⌄':'›'}</button>}{node.itemCode}</span></td><td>{node.name}</td><td>{node.depth===0?'—':formatQuantity(node.usage)}</td><td>{formatQuantity(node.quantity)}</td><td>{node.unit}</td></tr>)}
  </tbody></table></div>
  </>}
  <div className="page-title-row"><div><h2>Material Summary</h2><p className="muted">Combined requirements across all finished goods in this order.</p></div></div>
  {showCosts&&result.costComplete===false&&<p className="form-error">Some materials have no positive price. Material values are incomplete.</p>}
  <div className="table-frame"><table><thead><tr><th>Item Code</th><th>Raw Material</th><th>Required Qty</th><th>Unit</th><th>Kanban Equivalent</th><th>Purchase Kanban</th>{showCosts&&<><th>Unit Price</th><th>Material Value</th><th>Currency</th></>}</tr></thead><tbody>
   {result.materials.length===0?<tr><td colSpan={showCosts?9:6}><div className="table-empty">No material requirements.</div></td></tr>:result.materials.map((row,index)=><tr key={`${row.itemId}-${index}`}><td>{row.itemCode}</td><td>{row.name}</td><td>{formatQuantity(row.quantity)}</td><td>{row.unit}</td><td>{Number(row.qtyPerKanban)>0?formatQuantity(row.kanbanEquivalent):'—'}</td><td>{Number(row.qtyPerKanban)>0?formatQuantity(row.purchaseKanban):'—'}</td>{showCosts&&<><td>{row.unitPrice===undefined?'—':formatMoney(row.unitPrice,row.currency??'')}</td><td>{row.value===undefined?'—':formatMoney(row.value,row.currency??'')}</td><td>{row.currency??'—'}</td></>}</tr>)}
  </tbody></table></div>
  {showCosts&&result.totals&&<div className="toolbar-actions">{Object.entries(result.totals).map(([currency,total])=><strong key={currency}>Material Value ({currency}): {formatMoney(total,currency)}</strong>)}</div>}
 </div>;
}

