type ReferenceItem = { id: string; code: string; name: string };
type ReferenceResponse = { items?: ReferenceItem[] };

async function referenceOptions(path: string, signal: AbortSignal) {
  const response = await fetch(path, { credentials: 'include', signal });
  if (!response.ok) throw new Error('Reference data could not be loaded');
  const payload = await response.json() as ReferenceResponse;
  return (payload.items ?? []).map(item => ({ value: item.id, label: `${item.code} — ${item.name}` }));
}

export async function loadRawMaterialOptions(signal: AbortSignal) {
  const [supplierId, baseUnitId, categoryId, packingId] = await Promise.all([
    referenceOptions('/api/master-data/suppliers?active=true&limit=200', signal),
    referenceOptions('/api/master-data/units?active=true&limit=200', signal),
    referenceOptions('/api/master-data/categories?active=true&limit=200', signal),
    referenceOptions('/api/master-data/packings?active=true&limit=200', signal)
  ]);
  return { supplierId, baseUnitId, categoryId, packingId };
}
