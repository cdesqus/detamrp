import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ReceivingIndex } from './receiving-index';
import { ReceivingSession } from './receiving-session';

const push=vi.fn();
vi.mock('next/navigation',()=>({useRouter:()=>({push})}));
vi.mock('../app-shell/app-shell',()=>({useCurrentUser:()=>({permissions:['receiving.view','receiving.create','receiving.submit']})}));

beforeEach(()=>{push.mockReset();global.fetch=vi.fn(async(input,init)=>{
 if(String(input).endsWith('/receiving-sessions')&&init?.method==='POST')return new Response(JSON.stringify({id:'session-1'}),{status:201});
 return new Response(JSON.stringify({items:[]}),{status:200});
}) as typeof fetch});

it('creates a focused receiving session by exact DN without suggestions',async()=>{
 render(<ReceivingIndex/>);
 expect(await screen.findByRole('columnheader',{name:'No.'})).toHaveClass('table-column-number');
 expect(screen.getByRole('columnheader',{name:'Document'})).toHaveClass('table-column-actions');
 fireEvent.click(screen.getByRole('button',{name:'Create receiving'}));
 const dn=await screen.findByLabelText('Scan or Type DN Number');
 expect(dn).not.toHaveAttribute('list');
 fireEvent.change(dn,{target:{value:' dn-1 '}});
 fireEvent.submit(dn.closest('form')!);
 await waitFor(()=>expect(push).toHaveBeenCalledWith('/receiving/session-1'));
 expect(global.fetch).toHaveBeenCalledWith('/api/receiving-sessions',expect.objectContaining({body:JSON.stringify({deliveryNoteNumber:'DN-1'})}));
});

it.each([
 ['DN_INVALID','Delivery Note is invalid.'],
 ['DN_FULLY_RECEIVED','Delivery Note has already been fully received.'],
 ['DN_IN_PROGRESS','Delivery Note is currently being received in another session.'],
])('shows English validation for %s',async(code,message)=>{
 global.fetch=vi.fn(async(input,init)=>String(input).endsWith('/receiving-sessions')&&init?.method==='POST'
  ?new Response(JSON.stringify({code}),{status:409,headers:{'Content-Type':'application/json'}})
  :new Response(JSON.stringify({items:[]}),{status:200})) as typeof fetch;
 render(<ReceivingIndex/>);fireEvent.click(await screen.findByRole('button',{name:'Create receiving'}));
 const dn=await screen.findByLabelText('Scan or Type DN Number');fireEvent.change(dn,{target:{value:'DN-X'}});fireEvent.submit(dn.closest('form')!);
 expect(await screen.findByRole('alert')).toHaveTextContent(message);
});

it('renders an empty receiving scan session when scans is null',async()=>{
 global.fetch=vi.fn(async()=>new Response(JSON.stringify({id:'session-1',receivingNumber:'RCV-1',deliveryNoteNumber:'DN-1',poNumber:'PO-1',supplierName:'PT A',status:'ACTIVE',planned:2,previouslyReceived:0,outstanding:2,scans:null}),{status:200})) as typeof fetch;
 render(<ReceivingSession id="session-1"/>);
 expect(await screen.findByText('Ready to scan.')).toBeInTheDocument();
 expect(screen.getByText('Ready to scan.').closest('td')).toHaveClass('table-row-empty');
 expect(screen.getByLabelText('Scan or Type Kanban ID')).toBeEnabled();
});
