'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';

type Supplier = { id: string; code: string; name: string; currency: string; active: boolean };
type Material = { id: string; code: string; name: string; supplierId: string; baseUnitCode: string; qtyPerKanban: string; standardUnitPrice: string; active: boolean };
type OrderLine = { rawMaterialId: string; rawMaterialCode: string; rawMaterialName: string; baseUnitCode: string; qtyPerKanbanSnapshot: string; totalKanban: string; unitPriceSnapshot: string };
type InitialOrder = { id: string; poNumber?: string; status: string; supplierId: string; currency: string; orderDate: string; expectedDeliveryDate: string; notes?: string; lines: OrderLine[] };
type Props = { orderId?: string; initialOrder?: InitialOrder };
type ApiError = { message?: string; fields?: Record<string, string> };
type Decimal = { value: bigint; scale: number };

export function localDateISO(date: Date = new Date()) {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}
const dateOnly = (value?: string) => (value || '').slice(0, 10);
const optionLabel = (value: { code: string; name: string }) => `${value.code} — ${value.name}`;
const fieldKey = (index: number) => `lines[${index}].totalKanban`;

function parseDecimal(input: string | number | undefined): Decimal | null {
  const match = String(input ?? '0').trim().match(/^([+-]?)(\d*)(?:\.(\d*))?$/);
  if (!match) return null;
  const [, sign, whole = '', fraction = ''] = match;
  const digits = `${whole || '0'}${fraction}`.replace(/^0+(?=\d)/, '') || '0';
  return { value: (sign === '-' ? -1n : 1n) * BigInt(digits), scale: fraction.length };
}

function roundedMicro(decimal: Decimal): bigint {
  const micro = 6;
  if (decimal.scale <= micro) return decimal.value * 10n ** BigInt(micro - decimal.scale);
  const divisor = 10n ** BigInt(decimal.scale - micro);
  const negative = decimal.value < 0n;
  let quotient = (negative ? -decimal.value : decimal.value) / divisor;
  const remainder = (negative ? -decimal.value : decimal.value) % divisor;
  if (remainder * 2n >= divisor) quotient += 1n;
  return negative ? -quotient : quotient;
}

function formatMicro(value: bigint) {
  const negative = value < 0n;
  const digits = (negative ? -value : value).toString().padStart(7, '0');
  return `${negative ? '-' : ''}${digits.slice(0, -6)}.${digits.slice(-6)}`;
}

export function formatDecimal(input: string | number | undefined) {
  const decimal = parseDecimal(input);
  return decimal ? formatMicro(roundedMicro(decimal)) : '0.000000';
}

export function multiplyDecimals(left: string, right: string) {
  const a = parseDecimal(left); const b = parseDecimal(right);
  return a && b ? formatMicro(roundedMicro({ value: a.value * b.value, scale: a.scale + b.scale })) : '0.000000';
}

function addDecimals(left: string, right: string) {
  const a = parseDecimal(left); const b = parseDecimal(right);
  return formatMicro((a ? roundedMicro(a) : 0n) + (b ? roundedMicro(b) : 0n));
}

function isPositiveInteger(value: string) { return /^\d+$/.test(value) && BigInt(value) > 0n; }
function lineFromMaterial(material: Material): OrderLine { return { rawMaterialId: material.id, rawMaterialCode: material.code, rawMaterialName: material.name, baseUnitCode: material.baseUnitCode, qtyPerKanbanSnapshot: String(material.qtyPerKanban), totalKanban: '1', unitPriceSnapshot: String(material.standardUnitPrice) }; }
function errorFields(body: ApiError, fallback: string) { return { ...(body.fields ?? {}), _form: body.message ?? body.fields?._form ?? fallback }; }

export function SupplierOrderForm({ orderId, initialOrder }: Props) {
  const router = useRouter();
  const [savedId, setSavedId] = useState(initialOrder?.id ?? '');
  const [status, setStatus] = useState(initialOrder?.status ?? 'DRAFT');
  const [poNumber, setPONumber] = useState(initialOrder?.poNumber ?? '');
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [supplierId, setSupplierId] = useState(initialOrder?.supplierId ?? '');
  const [supplierQuery, setSupplierQuery] = useState('');
  const [materials, setMaterials] = useState<Material[]>([]);
  const [materialText, setMaterialText] = useState('');
  const [selectedMaterial, setSelectedMaterial] = useState<Material | null>(null);
  const [orderDate, setOrderDate] = useState(dateOnly(initialOrder?.orderDate) || localDateISO());
  const [expectedDeliveryDate, setExpectedDeliveryDate] = useState(dateOnly(initialOrder?.expectedDeliveryDate) || localDateISO());
  const [currency, setCurrency] = useState(initialOrder?.currency ?? '');
  const [notes, setNotes] = useState(initialOrder?.notes ?? '');
  const [lines, setLines] = useState<OrderLine[]>(initialOrder?.lines ?? []);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [supplierError, setSupplierError] = useState('');
  const [materialError, setMaterialError] = useState('');
  const [detailError, setDetailError] = useState('');
  const [detailAttempt, setDetailAttempt] = useState(0);
  const [detailLoading, setDetailLoading] = useState(Boolean(orderId && !initialOrder));
  const [saving, setSaving] = useState(false);
  const editable = !detailLoading && !detailError && status === 'DRAFT';

  const hydrate = useCallback((order: InitialOrder) => {
    setSavedId(order.id); setStatus(order.status); setPONumber(order.poNumber ?? ''); setSupplierId(order.supplierId);
    setCurrency(order.currency); setOrderDate(dateOnly(order.orderDate)); setExpectedDeliveryDate(dateOnly(order.expectedDeliveryDate));
    setNotes(order.notes ?? ''); setLines(order.lines ?? []);
  }, []);

  const loadSuppliers = useCallback(async () => {
    setSupplierError('');
    try {
      const response = await fetch('/api/master-data/suppliers?active=true&limit=200', { credentials: 'include' });
      if (!response.ok) throw new Error();
      const data = await response.json() as { items: Supplier[] };
      setSuppliers(data.items ?? []);
    } catch { setSupplierError('Suppliers could not be loaded.'); }
  }, []);

  useEffect(() => { void loadSuppliers(); }, [loadSuppliers]);
  useEffect(() => {
    if (!supplierId) return;
    const supplier = suppliers.find(item => item.id === supplierId);
    if (supplier) setSupplierQuery(optionLabel(supplier));
  }, [supplierId, suppliers]);
  useEffect(() => {
    if (!orderId || initialOrder) return;
    let current = true;
    setDetailLoading(true); setDetailError('');
    fetch(`/api/purchase-orders/${orderId}`, { credentials: 'include' })
      .then(response => response.ok ? response.json() as Promise<InitialOrder> : Promise.reject())
      .then(order => { if (current) hydrate(order); })
      .catch(() => { if (current) setDetailError('Supplier order could not be loaded.'); })
      .finally(() => { if (current) setDetailLoading(false); });
    return () => { current = false; };
  }, [detailAttempt, hydrate, initialOrder, orderId]);
  useEffect(() => {
    if (!supplierId) { setMaterials([]); setMaterialError(''); return; }
    const controller = new AbortController(); let current = true;
    setMaterialError('');
    fetch(`/api/master-data/raw-materials?active=true&limit=200&supplierId=${encodeURIComponent(supplierId)}`, { credentials: 'include', signal: controller.signal })
      .then(response => response.ok ? response.json() as Promise<{ items: Material[] }> : Promise.reject())
      .then(data => { if (current) setMaterials(data.items ?? []); })
      .catch(() => { if (current && !controller.signal.aborted) setMaterialError('Raw Materials could not be loaded.'); });
    return () => { current = false; controller.abort(); };
  }, [supplierId]);

  const selectSupplier = useCallback((candidate: Supplier) => {
    if (candidate.id === supplierId) { setSupplierQuery(optionLabel(candidate)); return; }
    if (lines.length > 0 && !window.confirm('Changing supplier clears all selected Raw Materials. Continue?')) {
      const existing = suppliers.find(item => item.id === supplierId);
      setSupplierQuery(existing ? optionLabel(existing) : '');
      return;
    }
    setSupplierId(candidate.id); setSupplierQuery(optionLabel(candidate)); setCurrency(candidate.currency); setLines([]); setMaterials([]); setMaterialText(''); setSelectedMaterial(null);
  }, [lines.length, supplierId, suppliers]);
  const changeSupplierText = (text: string) => {
    setSupplierQuery(text);
    const match = suppliers.find(item => optionLabel(item) === text);
    if (match) selectSupplier(match);
  };
  const restoreSupplierQuery = () => {
    const committed = suppliers.find(item => item.id === supplierId);
    if (supplierQuery !== (committed ? optionLabel(committed) : '')) setSupplierQuery(committed ? optionLabel(committed) : '');
  };
  const changeMaterialText = (text: string) => {
    setMaterialText(text);
    setSelectedMaterial(materials.find(item => optionLabel(item) === text) ?? null);
  };
  const addMaterial = () => {
    if (!selectedMaterial || lines.some(line => line.rawMaterialId === selectedMaterial.id)) return;
    setLines(current => [...current, lineFromMaterial(selectedMaterial)]); setMaterialText(''); setSelectedMaterial(null);
  };
  const changeKanban = (id: string, value: string) => {
    if (value !== '' && !/^\d+$/.test(value)) return;
    setLines(current => current.map(line => line.rawMaterialId === id ? { ...line, totalKanban: value } : line));
  };
  const validate = () => {
    const next: Record<string, string> = {};
    if (!supplierId) next.supplierId = 'Supplier is required';
    if (!orderDate) next.orderDate = 'Order Date is required';
    if (!expectedDeliveryDate) next.expectedDeliveryDate = 'Expected Delivery Date is required';
    else if (orderDate && expectedDeliveryDate < orderDate) next.expectedDeliveryDate = 'Expected Delivery Date cannot precede Order Date';
    lines.forEach((line, index) => { if (!isPositiveInteger(line.totalKanban)) next[fieldKey(index)] = 'Total Kanban must be a positive integer'; });
    return next;
  };
  const payload = () => ({ supplierId, orderDate, expectedDeliveryDate, currency, notes, lines: lines.map(line => ({ rawMaterialId: line.rawMaterialId, totalKanban: line.totalKanban })) });
  const total = useMemo(() => lines.reduce((result, line) => addDecimals(result, multiplyDecimals(multiplyDecimals(line.qtyPerKanbanSnapshot, line.totalKanban || '0'), line.unitPriceSnapshot)), '0.000000'), [lines]);

  async function save(submit: boolean) {
    if (!editable) return;
    const validation = validate();
    if (Object.keys(validation).length) { setErrors(validation); return; }
    setSaving(true); setErrors({});
    try {
      let id = savedId;
      const response = await fetch(id ? `/api/purchase-orders/${id}` : '/api/purchase-orders', { method: id ? 'PUT' : 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload()) });
      if (!response.ok) { setErrors(errorFields(await response.json() as ApiError, 'Supplier order could not be saved')); return; }
      const order = await response.json() as InitialOrder; id = order.id; hydrate(order);
      if (!submit) return;
      const submission = await fetch(`/api/purchase-orders/${id}/submit`, { method: 'POST', credentials: 'include' });
      if (!submission.ok) { setErrors(errorFields(await submission.json() as ApiError, 'Supplier order could not be submitted')); return; }
      hydrate(await submission.json() as InitialOrder);
    } catch { setErrors({ _form: 'Supplier order could not be saved' }); }
    finally { setSaving(false); }
  }
  async function cancel() {
    if (!savedId || !editable) return;
    setSaving(true); setErrors({});
    try {
      const response = await fetch(`/api/purchase-orders/${savedId}/cancel`, { method: 'POST', credentials: 'include' });
      if (!response.ok) { setErrors(errorFields(await response.json() as ApiError, 'Supplier order could not be cancelled')); return; }
      hydrate(await response.json() as InitialOrder);
    } catch { setErrors({ _form: 'Supplier order could not be cancelled' }); }
    finally { setSaving(false); }
  }

  if (detailLoading) return <section className="supplier-order-form"><div className="table-empty">Loading supplier order...</div></section>;
  if (detailError) return <section className="supplier-order-form"><div className="table-empty"><strong>Could not load supplier order</strong><span>{detailError}</span><button className="table-action" onClick={() => setDetailAttempt(value => value + 1)}>Retry</button></div></section>;
  const availableMaterials = materials.filter(item => !lines.some(line => line.rawMaterialId === item.id));
  return <section className="supplier-order-form">
    <div className="page-title-row"><div><h1>{poNumber || 'New Supplier Order'}</h1><p className="muted">{editable ? 'Complete the order and save it as a draft or send it for approval.' : `This supplier order is ${status.replaceAll('_', ' ').toLowerCase()} and read-only.`}</p></div>{poNumber && <span className="status-pill">{status.replaceAll('_', ' ')}</span>}</div>
    {errors._form && <p className="form-error" role="alert">{errors._form}</p>}
    {supplierError && <p className="form-error" role="alert">{supplierError} <button className="table-action" onClick={() => void loadSuppliers()}>Retry</button></p>}
    <div className="supplier-order-card"><div className="supplier-order-fields">
      <label>Supplier<input aria-label="Supplier" list="supplier-options" disabled={!editable} value={supplierQuery} onChange={event => changeSupplierText(event.target.value)} onBlur={restoreSupplierQuery} /><datalist id="supplier-options">{suppliers.map(item => <option key={item.id} value={optionLabel(item)} />)}</datalist>{errors.supplierId && <small role="alert">{errors.supplierId}</small>}</label>
      <label>Order Date<input type="date" value={orderDate} disabled={!editable} onChange={event => setOrderDate(event.target.value)} />{errors.orderDate && <small role="alert">{errors.orderDate}</small>}</label>
      <label>Expected Delivery Date<input type="date" min={orderDate} value={expectedDeliveryDate} disabled={!editable} onChange={event => setExpectedDeliveryDate(event.target.value)} />{errors.expectedDeliveryDate && <small role="alert">{errors.expectedDeliveryDate}</small>}</label>
      <label>Currency<input value={currency} aria-label="Currency" readOnly /></label>
      <label className="supplier-order-notes">Notes<textarea value={notes} disabled={!editable} onChange={event => setNotes(event.target.value)} /></label>
    </div></div>
    <div className="supplier-order-card supplier-order-materials"><div className="supplier-order-material-toolbar"><div><strong>Raw Materials</strong><span>Snapshots are read-only after selection.</span></div><div className="material-add"><input aria-label="Raw Material" list="material-options" disabled={!editable || !supplierId} value={materialText} onChange={event => changeMaterialText(event.target.value)} /><datalist id="material-options">{availableMaterials.map(item => <option key={item.id} value={optionLabel(item)} />)}</datalist><button type="button" className="primary-button" disabled={!editable || !supplierId || !selectedMaterial} onClick={addMaterial}>+ Raw Material</button></div></div>
      {materialError && <p className="form-error" role="alert">{materialError}</p>}
      <div className="table-frame"><table><thead><tr><th>Raw Material</th><th>Base Unit</th><th>Qty / Kanban</th><th>Total Kanban</th><th>Total Quantity</th><th>Unit Price</th><th>Amount</th><th></th></tr></thead><tbody>{lines.length === 0 ? <tr><td colSpan={8}><div className="table-empty">No Raw Materials selected.</div></td></tr> : lines.map((line, index) => { const quantity = multiplyDecimals(line.qtyPerKanbanSnapshot, line.totalKanban || '0'); const amount = multiplyDecimals(quantity, line.unitPriceSnapshot); const lineError = errors[fieldKey(index)]; return <tr key={line.rawMaterialId}><td>{line.rawMaterialCode} — {line.rawMaterialName}</td><td>{line.baseUnitCode}</td><td><output>{formatDecimal(line.qtyPerKanbanSnapshot)}</output></td><td><input aria-label={`Total Kanban for ${line.rawMaterialName}`} aria-describedby={lineError ? `kanban-error-${index}` : undefined} type="number" min="1" step="1" value={line.totalKanban} disabled={!editable} onChange={event => changeKanban(line.rawMaterialId, event.target.value)} />{lineError && <small id={`kanban-error-${index}`} role="alert">{lineError}</small>}</td><td><output>{quantity}</output></td><td><output>{formatDecimal(line.unitPriceSnapshot)}</output></td><td><output>{amount}</output></td><td>{editable && <button type="button" className="table-action" aria-label={`Remove ${line.rawMaterialName}`} onClick={() => setLines(current => current.filter(item => item.rawMaterialId !== line.rawMaterialId))}>Remove</button>}</td></tr>; })}</tbody><tfoot><tr><td colSpan={6}>Order Total</td><td><output>{total}</output></td><td></td></tr></tfoot></table></div></div>
    {editable && <div className="supplier-order-actions"><div>{errors.lines && <small role="alert">{errors.lines}</small>}</div><div>{savedId && <button type="button" onClick={cancel} disabled={saving}>Cancel draft</button>}<button type="button" onClick={() => router.push('/supplier-orders')} disabled={saving}>Back</button><button type="button" onClick={() => save(false)} disabled={saving}>Save as Draft</button><button type="button" className="primary-button" onClick={() => save(true)} disabled={saving}>Save &amp; Send for Approval</button></div></div>}
  </section>;
}
