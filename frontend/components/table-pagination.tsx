'use client';

export const pageSizes = [20, 50, 100] as const;

type Props = { page:number; pageSize:number; total:number; onPageChange:(page:number)=>void; onPageSizeChange:(size:number)=>void };

export function TablePagination({page,pageSize,total,onPageChange,onPageSizeChange}:Props){
  const pages=Math.max(1,Math.ceil(total/pageSize));
  const safePage=Math.min(page,pages);
  const first=total===0?0:(safePage-1)*pageSize+1;
  const last=Math.min(safePage*pageSize,total);
  return <div className="table-pagination" aria-label="Table pagination">
    <span>{first}–{last} of {total}</span>
    <button aria-label="Previous page" disabled={safePage<=1} onClick={()=>onPageChange(safePage-1)}>‹</button>
    <span>Page {safePage} of {pages}</span>
    <button aria-label="Next page" disabled={safePage>=pages} onClick={()=>onPageChange(safePage+1)}>›</button>
    <label><span>Rows</span><select aria-label="Rows per page" value={pageSize} onChange={e=>onPageSizeChange(Number(e.target.value))}>{pageSizes.map(size=><option key={size}>{size}</option>)}</select></label>
  </div>
}

export function pagedItems<T>(items:T[],page:number,pageSize:number){return items.slice((page-1)*pageSize,page*pageSize)}
