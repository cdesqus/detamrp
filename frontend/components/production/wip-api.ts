export type WIPMovement = {
  id: string;
  orderId: string;
  orderNumber: string;
  type: 'RECEIPT' | 'TRANSFER' | 'CONSUMPTION';
  sourceOperationId: string;
  sourceCode: string;
  destinationOperationId: string;
  destinationCode: string;
  quantity: string;
  unitCost?: string;
  totalCost?: string;
  currency: string;
  entryId: string;
  entryNumber: string;
  lotId: string;
  reversesId: string;
  reversed: boolean;
  movementDate: string;
  notes: string;
  createdBy: string;
  createdAt: string;
  canReverse: boolean;
};

export type WIPOperationBalance = {
  operationId: string;
  code: string;
  name: string;
  sequence: number;
  final: boolean;
  goodQty: string;
  processedQty: string;
  received: string;
  transferredOut: string;
  transferredIn: string;
  consumed: string;
  onHand: string;
  staged: string;
  balance: string;
  value?: string;
  reconciled: boolean;
  difference: string;
};

export type WIPOrderBalance = {
  orderId: string;
  orderNumber: string;
  planNumber: string;
  partNumber: string;
  partName: string;
  unitCode: string;
  plantName: string;
  status: string;
  currency: string;
  plannedQty: string;
  operations: WIPOperationBalance[];
  totalQty: string;
  totalValue?: string;
  reconciled: boolean;
  updatedAt: string;
  movements: WIPMovement[];
};

export const NIL_ID = '00000000-0000-0000-0000-000000000000';

/** A reversal is a signed twin of the row it annuls. */
export const movementLabel = (movement: WIPMovement) =>
  movement.reversesId && movement.reversesId !== NIL_ID ? 'REVERSAL' : movement.type;
