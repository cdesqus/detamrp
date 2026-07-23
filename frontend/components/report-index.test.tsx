import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {afterEach,describe,expect,it,vi} from 'vitest';
import {ReportIndex} from './report-index';

afterEach(()=>vi.unstubAllGlobals());

describe('ReportIndex',()=>{
  it('loads receiving rows and builds a filtered PDF URL',async()=>{
    const fetchMock=vi.fn((input:string)=>{
      if(input.startsWith('/api/master-data/suppliers')) return Promise.resolve(new Response(JSON.stringify({items:[{id:'sup-1',name:'Supplier A'}]})));
      return Promise.resolve(new Response(JSON.stringify({items:[{receivingNumber:'RCV-1',receivingDate:'2026-07-23',deliveryNoteNumber:'DN-1',poNumber:'PO-1',supplierName:'Supplier A',rawMaterialCode:'RM-1',rawMaterialName:'Material A',baseUnitCode:'PC',kanbanReceived:2,receivedQuantity:'10',outstandingQuantity:'5',sageNumber:'',createdBy:'Administrator'}],totals:{kanbanReceived:2,receivedQuantity:'10'}})));
    });
    vi.stubGlobal('fetch',fetchMock);
    render(<ReportIndex/>);
    expect(await screen.findByText('RCV-1')).toBeInTheDocument();
    await userEvent.selectOptions(screen.getByLabelText('Supplier'),'sup-1');
    await userEvent.click(screen.getByRole('button',{name:'Apply Filters'}));
    await waitFor(()=>expect(fetchMock).toHaveBeenLastCalledWith('/api/reports/receiving?supplierId=sup-1',{credentials:'include'}));
    expect(screen.getByRole('link',{name:'Export PDF'})).toHaveAttribute('href','/api/reports/receiving.pdf?supplierId=sup-1');
  });
});
