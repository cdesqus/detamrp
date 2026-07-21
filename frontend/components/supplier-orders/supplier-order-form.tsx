'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { formatDecimal } from './supplier-order-index';

type Supplier = { id: string; code: string; name: string; currency: string; active: boolean };
type Material = { id: string; code: string; name: string; supplierId: string; baseUnitCode: string; qtyPerKanban: string; standardUnitPrice: string; active: boolean };
type OrderLine = { rawMaterialId: string; rawMaterialCode: string; rawMaterialName: string; baseUnitCode: string; qtyPerKanbanSnapshot: string; totalKanban: string; orderedBaseQty?: string; unitPriceSnapshot: string; lineTotal?: string };
type InitialOrder = { id: string; poNumber?: string; status: string; supplierId: string; currency: string; orderDate: string; expectedDeliveryDate: string; notes?: string; lines: OrderLine[] };
type Props = { orderId?: string; initialOrder?: InitialOrder };
type ApiError = { message?: string; fields?: Record<string, string> };

const today = () => new Date().toISOString().slice(0, 10);
const dateOnly = (value?: string) => (value || '').slice(0, 10);
const label = (value: { code: string; name: string }) => `${value.code} — ${value.name}`;
const integer = (value: string) => /^\d+$/.test(value) && BigInt(value) > 0n;

function fixedProduct(left: string, right: string) {
  const parse = (input: string) => { const [whole = '0', fraction = ''] = String(input).trim().split('.'); const decimals = fraction.replace(/\D/g, '').length; return { value: BigInt(`${whole.replace(/^\+/, '') || '0'}${fraction.replace(/\D/g, '')}`), decimals }; };
  try { const a = parse(left); const b = parse(right); const scale = a.decimals + b.decimals; const product = a.value * b.value; const divisor = 10n ** BigInt(scale); const whole = product / divisor; const fraction = (product % divisor).toString().padStart(scale, '0'); return formatDecimal(`${whole}.${fraction}`); } catch { return '0.000000'; }
}

function lineFromMaterial(material: Material): OrderLine { return { rawMaterialId: material.id, rawMaterialCode: material.code, rawMaterialName: material.name, baseUnitCode: material.baseUnitCode, qtyPerKanbanSnapshot: String(material.qtyPerKanban), totalKanban: '1', unitPriceSnapshot: String(material.standardUnitPrice) }; }

export function SupplierOrderForm({ orderId, initialOrder }: Props) {
  const router = useRouter();
  const [savedId, setSavedId] = useState(initialOrder?.id ?? orderId ?? '');
  const [status, setStatus] = useState(initialOrder?.status ?? 'DRAFT');
  const [poNumber, setPONumber] = useState(initialOrder?.poNumber ?? '');
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [supplierId, setSupplierId] = useState(initialOrder?.supplierId ?? '');
  const [supplierText, setSupplierText] = useState('');
  const [materials, setMaterials] = useState<Material[]>([]);
  const [materialText, setMaterialText] = useState('');
  const [selectedMaterial, setSelectedMaterial] = useState<Material | null>(null);
  const [orderDate, setOrderDate] = useState(dateOnly(initialOrder?.orderDate) || today());
  const [expectedDeliveryDate, setExpectedDeliveryDate] = useState(dateOnly(initialOrder?.expectedDeliveryDate) || today());
  const [currency, setCurrency] = useState(initialOrder?.currency ?? '');
  const [notes, setNotes] = useState(initialOrder?.notes ?? '');
  const [lines, setLines] = useState<OrderLine[]>(initialOrder?.lines ?? []);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [saving, setSaving] = useState(false);
  const editable = status === 'DRAFT';
  const selectedSupplier = suppliers.find(item => item.id === supplierId);

  const chooseSupplier = useCallback((candidate: Supplier) => {
    if (candidate.id === supplierId) { setSupplierText(label(candidate)); return; }
    if (lines.length > 0 && !confirm('Changing supplier clears all selected Raw Materials. Continue?')) return;
    setSupplierId(candidate.id); setSupplierText(label(candidate)); setCurrency(candidate.currency); setLines([]); setMaterialText(''); setSelectedMaterial(null);
  }, [lines.length, supplierId]);
  const chooseMaterial = useCallback((candidate: Material) => { setSelectedMaterial(candidate); setMaterialText(label(candidate)); }, []);

  useEffect(() => { fetch('/api/master-data/suppliers?active=true&limit=200', { credentials: 'include' }).then(response => response.ok ? response.json() : Promise.reject()).then((data: { items: Supplier[] }) => { setSuppliers(data.items ?? []); const present = data.items?.find((item: Supplier) => item.id === initialOrder?.supplierId); if (present) setSupplierText(label(present)); }).catch(() => {}); }, [initialOrder?.supplierId]);
  useEffect(() => { if (!supplierId) { setMaterials([]); return; } const controller = new AbortController(); fetch(`/api/master-data/raw-materials?active=true&limit=200&supplierId=${encodeURIComponent(supplierId)}`, { credentials: 'include', signal: controller.signal }).then(response => response.ok ? response.json() : Promise.reject()).then((data: { items: Material[] }) => setMaterials(data.items ?? [])).catch(() => {}); return () => controller.abort(); }, [supplierId]);
  useEffect(() => { if (!orderId || initialOrder) return; fetch(`/api/purchase-orders/${orderId}`, { credentials: 'include' }).then(response => response.ok ? response.json() : Promise.reject()).then((order: InitialOrder) => { setSavedId(order.id); setStatus(order.status); setPONumber(order.poNumber ?? ''); setSupplierId(order.supplierId); setCurrency(order.currency); setOrderDate(dateOnly(order.orderDate)); setExpectedDeliveryDate(dateOnly(order.expectedDeliveryDate)); setNotes(order.notes ?? ''); setLines(order.lines ?? []); }).catch(() => setErrors({ _form: 'Supplier order could not be loaded' })); }, [initialOrder, orderId]);

  const total = useMemo(() => lines.reduce((result, line) => addFixed(result, fixedProduct(fixedProduct(line.qtyPerKanbanSnapshot, line.totalKanban), line.unitPriceSnapshot)), '0.000000'), [lines]);
  function addMaterial() { if (!selectedMaterial || lines.some(line => line.rawMaterialId === selectedMaterial.id)) return; setLines(current => [...current, lineFromMaterial(selectedMaterial)]); setSelectedMaterial(null); setMaterialText(''); }
  function changeKanban(id: string, value: string) { if (value !== '' && !/^\d+$/.test(value)) return; setLines(current => current.map(line => line.rawMaterialId === id ? { ...line, totalKanban: value } : line)); }
  function payload() { return { supplierId, orderDate, expectedDeliveryDate, currency, notes, lines: lines.map(line => ({ rawMaterialId: line.rawMaterialId, totalKanban: line.totalKanban })) }; }
  async function save(submit: boolean) {
    if (!editable) return; setSaving(true); setErrors({});
    try {
      let id = savedId;
      const response = await fetch(id ? `/api/purchase-orders/${id}` : '/api/purchase-orders', { method: id ? 'PUT' : 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload()) });
      if (!response.ok) { const body = await response.json() as ApiError; setErrors({ ...(body.fields ?? {}), _form: body.message ?? body.fields?._form ?? 'Supplier order could not be saved' }); return; }
      const order = await response.json() as InitialOrder; id = order.id; setSavedId(id); setPONumber(order.poNumber ?? '');
      if (submit) { const submission = await fetch(`/api/purchase-orders/${id}/submit`, { method: 'POST', credentials: 'include' }); if (!submission.ok) { const body = await submission.json() as ApiError; setErrors({ ...(body.fields ?? {}), _form: body.message ?? body.fields?._form ?? 'Supplier order could not be submitted' }); return; } const submitted = await submission.json() as InitialOrder; setStatus(submitted.status); }
    } catch { setErrors({ _form: 'Supplier order could not be saved' }); } finally { setSaving(false); }
  }
  async function cancel() { if (!savedId || !editable) return; setSaving(true); setErrors({}); try { const response = await fetch(`/api/purchase-orders/${savedId}/cancel`, { method: 'POST', credentials: 'include' }); if (!response.ok) { const body = await response.json() as ApiError; setErrors({ ...(body.fields ?? {}), _form: body.message ?? body.fields?._form ?? 'Supplier order could not be cancelled' }); return; } const order = await response.json() as InitialOrder; setStatus(order.status); } catch { setErrors({ _form: 'Supplier order could not be cancelled' }); } finally { setSaving(false); } }
  const duplicate = !!selectedMaterial && lines.some(line => line.rawMaterialId === selectedMaterial.id);
  return <section className="supplier-order-form"><div className="page-title-row"><div><h1>{poNumber || 'New Supplier Order'}</h1><p className="muted">{editable ? 'Complete the order and save it as a draft or send it for approval.' : `This supplier order is ${status.replaceAll('_', ' ').toLowerCase()} and read-only.`}</p></div>{poNumber && <span className="status-pill">{status.replaceAll('_', ' ')}</span>}</div>
    {errors._form && <p className="form-error">{errors._form}</p>}
    <div className="supplier-order-card"><div className="supplier-order-fields"><label>Supplier<input aria-label="Supplier" list="supplier-options" disabled={!editable} value={supplierText} onChange={event => { setSupplierText(event.target.value); const match = suppliers.find(item => label(item) === event.target.value); if (match) chooseSupplier(match); }} /><datalist id="supplier-options">{suppliers.map(item => <option key={item.id} value={label(item)} onClick={() => chooseSupplier(item)}>{label(item)}</option>)}</datalist>{errors.supplierId && <small>{errors.supplierId}</small>}</label><label>Order Date<input type="date" value={orderDate} disabled={!editable} onChange={event => setOrderDate(event.target.value)} />{errors.orderDate && <small>{errors.orderDate}</small>}</label><label>Expected Delivery Date<input type="date" min={orderDate} value={expectedDeliveryDate} disabled={!editable} onChange={event => setExpectedDeliveryDate(event.target.value)} />{errors.expectedDeliveryDate && <small>{errors.expectedDeliveryDate}</small>}</label><label>Currency<input value={currency} aria-label="Currency" readOnly /></label><label className="supplier-order-notes">Notes<textarea value={notes} disabled={!editable} onChange={event => setNotes(event.target.value)} /></label></div></div>
    <div className="supplier-order-card supplier-order-materials"><div className="supplier-order-material-toolbar"><div><strong>Raw Materials</strong><span>Snapshots are read-only after selection.</span></div><div className="material-add"><input aria-label="Raw Material" list="material-options" disabled={!editable || !supplierId} value={materialText} onChange={event => { setMaterialText(event.target.value); const match = materials.find(item => label(item) === event.target.value); if (match) chooseMaterial(match); }} /><datalist id="material-options">{materials.filter(item => !lines.some(line => line.rawMaterialId === item.id)).map(item => <option key={item.id} value={label(item)} onClick={() => chooseMaterial(item)}>{label(item)}</option>)}</datalist><button type="button" className="primary-button" disabled={!editable || !supplierId || !selectedMaterial || duplicate} onClick={addMaterial}>{duplicate && selectedMaterial ? `Add ${selectedMaterial.name}` : '+ Raw Material'}</button></div></div>
      <div className="table-frame"><table><thead><tr><th>Raw Material</th><th>Base Unit</th><th>Qty / Kanban</th><th>Total Kanban</th><th>Total Quantity</th><th>Unit Price</th><th>Amount</th><th></th></tr></thead><tbody>{lines.length === 0 ? <tr><td colSpan={8}><div className="table-empty">No Raw Materials selected.</div></td></tr> : lines.map(line => { const quantity = fixedProduct(line.qtyPerKanbanSnapshot, line.totalKanban || '0'); const amount = fixedProduct(quantity, line.unitPriceSnapshot); return <tr key={line.rawMaterialId}><td>{line.rawMaterialCode} — {line.rawMaterialName}</td><td>{line.baseUnitCode}</td><td><output>{formatDecimal(line.qtyPerKanbanSnapshot)}</output></td><td><input aria-label={`Total Kanban for ${line.rawMaterialName}`} type="number" min="1" step="1" value={line.totalKanban} disabled={!editable} onChange={event => changeKanban(line.rawMaterialId, event.target.value)} />{!integer(line.totalKanban) && <small>Total Kanban must be a positive integer</small>}</td><td><output>{quantity}</output></td><td><output>{formatDecimal(line.unitPriceSnapshot)}</output></td><td><output>{amount}</output></td><td>{editable && <button type="button" className="table-action" onClick={() => setLines(current => current.filter(item => item.rawMaterialId !== line.rawMaterialId))}>Remove</button>}</td></tr>; })}</tbody><tfoot><tr><td colSpan={6}>Order Total</td><td><output>{total}</output></td><td></td></tr></tfoot></table></div></div>
    {editable && <div className="supplier-order-actions"><div>{errors.lines && <small>{errors.lines}</small>}</div><div>{savedId && <button type="button" onClick={cancel} disabled={saving}>Cancel draft</button>}<button type="button" onClick={() => router.push('/supplier-orders')} disabled={saving}>Back</button><button type="button" onClick={() => save(false)} disabled={saving}>Save as Draft</button><button type="button" className="primary-button" onClick={() => save(true)} disabled={saving}>Save &amp; Send for Approval</button></div></div>}
  </section>;
}

function addFixed(left: string, right: string) {
  const toMicro = (value: string) => { const [whole = '0', fraction = ''] = value.split('.'); return BigInt(`${whole}${fraction.padEnd(6, '0').slice(0, 6)}`); };
  try { const value = toMicro(left) + toMicro(right); const sign = value < 0n ? '-' : ''; const digits = (value < 0n ? -value : value).toString().padStart(7, '0'); return `${sign}${digits.slice(0, -6)}.${digits.slice(-6)}`; } catch { return '0.000000'; }
}
