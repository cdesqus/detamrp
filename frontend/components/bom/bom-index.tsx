'use client';

import { Fragment, useEffect, useState } from 'react';

type BOM = { id:string; output:{ itemCode:string; name:string; unit:string }; revision:number; status:string; components:{ itemCode:string; name:string; unit:string; usageQty:string }[] };

export function BOMIndex() {
  const [items, setItems] = useState<BOM[]>([]);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());
  const [error, setError] = useState('');
  useEffect(() => { fetch('/api/boms', { credentials:'include' }).then(response => response.ok ? response.json() : Promise.reject()).then(data => setItems(data.items ?? [])).catch(() => setError('BOMs could not be loaded')); }, []);
  function toggle(id:string) { setExpanded(current => { const next = new Set(current); if (next.has(id)) next.delete(id); else next.add(id); return next; }); }
  return <section className="module-index"><div className="page-title-row"><div><h1>Bill of Materials</h1><p className="muted">Component quantities required to make one unit of an item.</p></div><a className="primary-button" href="/boms/new">New BOM</a></div>{error ? <div className="table-empty"><strong>Could not load data</strong><span>{error}</span></div> : <div className="table-frame"><table><thead><tr><th>Output Item Code</th><th>Output Item</th><th>Revision</th><th>Status</th><th>Components</th></tr></thead><tbody>{items.length === 0 ? <tr><td colSpan={5}><div className="table-empty"><strong>No BOMs yet</strong><span>Create a BOM to define the required components.</span></div></td></tr> : items.map(item => <Fragment key={item.id}><tr><td>{item.components.length > 0 && <button className="table-action" aria-label={`${expanded.has(item.id) ? 'Collapse' : 'Expand'} ${item.output.itemCode}`} onClick={() => toggle(item.id)}>{expanded.has(item.id) ? '⌄' : '›'}</button>} {item.output.itemCode}</td><td>{item.output.name}</td><td>R{item.revision}</td><td>{item.status}</td><td>{item.components.length}</td></tr>{expanded.has(item.id) && item.components.map(component => <tr key={`${item.id}-${component.itemCode}`}><td style={{ paddingLeft:44 }}>{component.itemCode}</td><td>{component.name}</td><td>—</td><td>Component</td><td>{component.usageQty} {component.unit} / {item.output.unit}</td></tr>)}</Fragment>)}</tbody></table></div>}</section>;
}
