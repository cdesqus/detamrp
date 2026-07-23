import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { OutgoingIndex } from './outgoing-index';
import { OutgoingSession } from './outgoing-session';

const push=vi.fn();
vi.mock('next/navigation',()=>({useRouter:()=>({push})}));

beforeEach(()=>{
  push.mockReset();
  global.fetch=vi.fn(async(_input,init)=>init?.method==='POST'
    ?new Response(JSON.stringify({id:'out-1'}),{status:201})
    :new Response(JSON.stringify({items:[]}),{status:200})) as typeof fetch;
});

it('shows an empty historical destination as a dash',async()=>{
  global.fetch=vi.fn(async()=>new Response(JSON.stringify({id:'out-1',documentNumber:'OUT-1',transactionDate:'2026-07-23',destination:'',notes:'',status:'ACTIVE',createdBy:'Admin',scans:null}),{status:200})) as typeof fetch;
  render(<OutgoingSession id="out-1"/>);
  expect(await screen.findByText('Destination: —')).toBeInTheDocument();
});

it('starts an outgoing scanner with one click and no destination form',async()=>{
  render(<OutgoingIndex/>);
  fireEvent.click(await screen.findByRole('button',{name:'Create outgoing'}));
  await waitFor(()=>expect(push).toHaveBeenCalledWith('/outgoing-material/out-1'));
  expect(screen.queryByLabelText('Destination')).not.toBeInTheDocument();
  expect(screen.queryByText('Notes')).not.toBeInTheDocument();
  expect(global.fetch).toHaveBeenCalledWith('/api/outgoing-sessions',expect.objectContaining({
    body:JSON.stringify({destination:'',notes:''}),
  }));
});
