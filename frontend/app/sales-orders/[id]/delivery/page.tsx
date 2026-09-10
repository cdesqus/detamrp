'use client';
import { AppShell } from '../../../../components/app-shell/app-shell';
import { CustomerDeliveryForm } from '../../../../components/sales-orders/customer-delivery-form';
export default function Page({params}:{params:{id:string}}){return <AppShell title="New Customer Delivery"><CustomerDeliveryForm orderId={params.id}/></AppShell>}
