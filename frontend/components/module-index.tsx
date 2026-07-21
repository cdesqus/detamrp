type ModuleIndexProps = {
  title: string;
  description: string;
  actionLabel: string;
  columns: string[];
  searchPlaceholder: string;
  emptyMessage: string;
};

export function ModuleIndex({ title, description, actionLabel, columns, searchPlaceholder, emptyMessage }: ModuleIndexProps) {
  return (
    <section className="module-index">
      <div className="page-title-row">
        <div><h1>{title}</h1><p className="muted">{description}</p></div>
        <button className="primary-button" disabled title="Coming next">{actionLabel}</button>
      </div>
      <div className="table-toolbar">
        <input type="search" aria-label={`Search ${title}`} placeholder={searchPlaceholder} />
        <div className="toolbar-actions">
          <button disabled title="Coming next">Filter</button>
          <button disabled title="Coming next">Columns</button>
        </div>
      </div>
      <div className="table-frame">
        <table>
          <thead><tr>{columns.map(column => <th key={column}>{column}</th>)}</tr></thead>
          <tbody><tr><td colSpan={columns.length}><div className="table-empty"><strong>Belum ada data</strong><span>{emptyMessage}</span></div></td></tr></tbody>
        </table>
      </div>
    </section>
  );
}
