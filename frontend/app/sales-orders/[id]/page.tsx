'use client';
import { AppShell } from '../../../components/app-shell/app-shell';
import { SalesOrderDetail } from '../../../components/sales-orders/sales-order-detail';
export default async function Page({params}:{params:Promise<{id:string}>}){const {id}=await params;return <AppShell title="Sales Order"><SalesOrderDetail id={id}/></AppShell>}
