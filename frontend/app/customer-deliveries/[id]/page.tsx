'use client';
import { AppShell } from '../../../components/app-shell/app-shell';
import { CustomerDeliveryDetail } from '../../../components/sales-orders/customer-delivery-detail';
export default function Page({params}:{params:{id:string}}){return <AppShell title="Customer Delivery"><CustomerDeliveryDetail id={params.id}/></AppShell>}
