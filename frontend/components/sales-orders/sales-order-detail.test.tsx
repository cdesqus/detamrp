import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SalesOrderDetail } from './sales-order-detail';

const push = vi.fn();
vi.mock('next/navigation', () => ({ useRouter: () => ({ push }) }));

describe('SalesOrderDetail', () => {
  beforeEach(() => { vi.stubGlobal('fetch', vi.fn(async input => {
    const url=String(input);
    if(url === '/api/sales-orders/so-1') return response({id:'so-1',number:'SLO-202609-0001',customerName:'Customer One',status:'SUBMITTED',orderDate:'2026-09-09T00:00:00Z',lines:[{id:'line-1',itemCode:'FG-1',name:'Finished One',unit:'PCS',quantity:'2',salesPrice:'100',currency:'IDR'}]});
    if(url === '/api/sales-orders/so-1/requirements') return response({lines:[{lineId:'line-1',result:{nodes:[],materials:[{itemId:'rm-1',itemCode:'RM-1',name:'Steel',unit:'KG',quantity:'5',qtyPerKanban:'10',kanbanEquivalent:'0.5',purchaseKanban:'1',unitPrice:'10',currency:'IDR',value:'50',paths:[]}],totals:{IDR:'50'},costComplete:true}}]});
    if(url === '/api/sales-orders/so-1/deliveries') return response({items:[]});
    throw new Error(url);
  })); });
  it('shows submitted order calculation from its saved snapshot', async () => {
    render(<SalesOrderDetail id="so-1" />);
    expect(await screen.findByText('SLO-202609-0001')).toBeInTheDocument();
    expect(await screen.findByText('RM-1')).toBeInTheDocument();
    expect(screen.getByText(/Material Value \(IDR\): IDR 50/)).toBeInTheDocument();
  });
});
function response(body: unknown) { return Promise.resolve({ok:true,json:async()=>body} as Response); }
