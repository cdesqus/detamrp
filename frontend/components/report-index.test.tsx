import {render,screen,waitFor} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {afterEach,describe,expect,it,vi} from 'vitest';
import {ReportIndex} from './report-index';

afterEach(()=>vi.unstubAllGlobals());

describe('ReportIndex',()=>{
  it('waits for a required date range before loading or exporting',async()=>{
    const fetchMock=vi.fn((input:string)=>{
      if(input.startsWith('/api/master-data/suppliers')) return Promise.resolve(new Response(JSON.stringify({items:[{id:'sup-1',name:'Supplier A'}]})));
      return Promise.resolve(new Response(JSON.stringify({items:[{receivingNumber:'RCV-1',receivingDate:'2026-07-23',deliveryNoteNumber:'DN-1',poNumber:'PO-1',supplierName:'Supplier A',rawMaterialCode:'RM-1',rawMaterialName:'Material A',baseUnitCode:'PC',kanbanReceived:2,receivedQuantity:'10',outstandingQuantity:'5',sageNumber:'',createdBy:'Administrator'}],totals:{kanbanReceived:2,receivedQuantity:'10'}})));
    });
    vi.stubGlobal('fetch',fetchMock);
    render(<ReportIndex/>);
    await waitFor(()=>expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(fetchMock.mock.calls[0][0]).toContain('/master-data/suppliers');
    expect(screen.queryByText('RCV-1')).not.toBeInTheDocument();
    expect(screen.queryByRole('link',{name:'Export PDF'})).not.toBeInTheDocument();
    expect(screen.getByRole('button',{name:'Apply Filters'})).toBeDisabled();

    await userEvent.type(screen.getByLabelText('From Date'),'2026-07-01');
    await userEvent.type(screen.getByLabelText('To Date'),'2026-07-31');
    await userEvent.click(screen.getByRole('button',{name:'Apply Filters'}));
    expect(await screen.findByText('RCV-1')).toBeInTheDocument();
    expect(screen.getByRole('link',{name:'Export PDF'})).toHaveAttribute('href','/api/reports/receiving.pdf?fromDate=2026-07-01&toDate=2026-07-31');

    await userEvent.click(screen.getByRole('button',{name:'Reset'}));
    expect(screen.queryByText('RCV-1')).not.toBeInTheDocument();
    expect(screen.queryByRole('link',{name:'Export PDF'})).not.toBeInTheDocument();
  });
});
