export function ReportIndex() {
  return (
    <section className="report-index">
      <div className="page-title-row"><div><h1>Reports</h1><p className="muted">Filter laporan procurement dan warehouse menggunakan data transaksi aktual.</p></div><button className="primary-button" disabled>Export</button></div>
      <div className="report-filters">
        <label>From Date<input type="date" /></label>
        <label>To Date<input type="date" /></label>
        <label>Supplier<select defaultValue=""><option value="">All suppliers</option></select></label>
        <label>Raw Material<select defaultValue=""><option value="">All raw materials</option></select></label>
        <label>PO Reference<input placeholder="PO number" /></label>
        <label>Receiving Reference<input placeholder="Receiving number" /></label>
        <label>Status<select defaultValue=""><option value="">All statuses</option></select></label>
        <button disabled>Apply filters</button>
      </div>
      <div className="report-empty"><span className="chart-empty-mark" aria-hidden="true" /><strong>Belum ada data report</strong><span>Hasil report akan tampil setelah transaksi PO dan receiving tersedia.</span></div>
    </section>
  );
}
