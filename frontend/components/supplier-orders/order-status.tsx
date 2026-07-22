const tones: Record<string, string> = {
  DRAFT: 'neutral', PENDING_APPROVAL: 'amber', APPROVED: 'green',
  PARTIALLY_RECEIVED: 'blue', FULLY_RECEIVED: 'dark-green',
  REJECTED: 'red', CANCELLED: 'red',
};

function label(status: string) {
  return status.toLowerCase().split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
}

export function OrderStatusBadge({ status }: { status: string }) {
  const tone = tones[status] ?? 'neutral';
  return <span className={`status-pill status-pill--${tone}`}>{label(status)}</span>;
}
