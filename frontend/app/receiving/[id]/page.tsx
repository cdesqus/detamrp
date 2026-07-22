import { AppShell } from '../../../components/app-shell/app-shell';
import { ReceivingSession } from '../../../components/receiving/receiving-session';
export default async function Page({params}:{params:Promise<{id:string}>}){const{id}=await params;return <AppShell title="Receiving Session"><ReceivingSession id={id}/></AppShell>}
