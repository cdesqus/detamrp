import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { ReceivingIndex } from './receiving-index';

const push=vi.fn();
vi.mock('next/navigation',()=>({useRouter:()=>({push})}));
vi.mock('../app-shell/app-shell',()=>({useCurrentUser:()=>({permissions:['receiving.view','receiving.create','receiving.submit']})}));

beforeEach(()=>{push.mockReset();global.fetch=vi.fn(async(input,init)=>{
 const url=String(input); if(url.includes('receiving-options')) return new Response(JSON.stringify({items:[{deliveryNoteId:'11111111-1111-1111-1111-111111111111',deliveryNoteNumber:'DN-1',poNumber:'PO-1',supplierName:'PT A',planned:10,received:2,outstanding:8}]}),{status:200});
 if(url.endsWith('/receiving-sessions')&&init?.method==='POST')return new Response(JSON.stringify({id:'session-1'}),{status:201});
 return new Response(JSON.stringify({items:[]}),{status:200});
}) as typeof fetch});

it('creates a focused receiving session from an outstanding DN',async()=>{
 render(<ReceivingIndex/>); fireEvent.click(await screen.findByRole('button',{name:'Create receiving'}));
 const dn=await screen.findByLabelText('Delivery Note'); fireEvent.change(dn,{target:{value:'DN-1 — PO-1 — PT A'}});
 expect(screen.getByText(/Outstanding: 8 Kanban/)).toBeInTheDocument();
 fireEvent.click(screen.getByRole('button',{name:'Start scanning'}));
 await waitFor(()=>expect(push).toHaveBeenCalledWith('/receiving/session-1'));
});
