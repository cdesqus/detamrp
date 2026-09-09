'use client';
import { AppShell } from '../../../components/app-shell/app-shell';
import { SalesOrderDetail } from '../../../components/sales-orders/sales-order-detail';
export default function Page({params}:{params:{id:string}}){return <AppShell title="Sales Order"><SalesOrderDetail id={params.id}/></AppShell>}
