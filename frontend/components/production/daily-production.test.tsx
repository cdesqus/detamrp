import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { DailyProductionForm } from "./daily-production-form";
import {
  DailyProductionIndex,
  DailyProductionDetail,
} from "./daily-production-pages";
const { push, auth } = vi.hoisted(() => ({
  push: vi.fn(),
  auth: {
    permissions: ["production.view", "production.entry", "production.void"],
  },
}));
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));
vi.mock("../app-shell/app-shell", () => ({ useCurrentUser: () => auth }));
const reply = (v: unknown) => ({ ok: true, json: async () => v });
const order = {
  id: "o1",
  orderNumber: "PRO-0000001",
  partNumber: "FG-001",
  partName: "Panel",
  unitCode: "PCS",
  plantName: "Plant A",
  plannedQty: "100",
  periodStart: "2026-09-01",
  dueDate: "2026-09-30",
  status: "RELEASED",
  currency: "IDR",
  operations: [
    {
      id: "op1",
      code: "STAMPING",
      name: "Stamping",
      sequence: 1,
      plannedQty: "100",
      processedQty: "0",
      goodQty: "0",
      rejectQty: "0",
      rate: "10",
    },
    {
      id: "op2",
      code: "WELDING",
      name: "Welding",
      sequence: 2,
      plannedQty: "100",
      processedQty: "0",
      goodQty: "0",
      rejectQty: "0",
      rate: "20",
    },
  ],
  materials: [
    {
      id: "m1",
      partNumber: "RM-001",
      partName: "Steel",
      unitCode: "KG",
      usageQty: "2",
      unitPrice: "5",
    },
  ],
};
const options = {
  orders: [order],
  operators: [{ id: "u1", name: "Operator One" }],
  closedPeriods: [],
};
const entry = {
  id: "e1",
  entryNumber: "DP-0000001",
  orderId: "o1",
  orderNumber: "PRO-0000001",
  operationId: "op1",
  productionDate: "2026-09-14",
  shift: "1",
  operatorId: "u1",
  operatorName: "Operator One",
  operationCode: "STAMPING",
  operationName: "Stamping",
  sequence: 1,
  partNumber: "FG-001",
  partName: "Panel",
  unitCode: "PCS",
  plantName: "Plant A",
  processed: "10",
  good: "9",
  rejected: "1",
  status: "POSTED",
  notes: "First batch",
  version: 1,
  canCorrect: true,
  periodClosed: false,
  effectsLocked: false,
  createdBy: "Planner",
  updatedAt: "2026-09-14T01:00:00Z",
  currency: "IDR",
  materialCost: "100",
  processCost: "100",
  processRate: "10",
  materials: [
    {
      materialId: "m1",
      partNumber: "RM-001",
      partName: "Steel",
      unitCode: "KG",
      quantity: "20",
      unitPrice: "5",
      cost: "100",
    },
  ],
  history: [
    { action: "CREATED", actor: "Planner", occurredAt: "2026-09-14T01:00:00Z" },
  ],
};
beforeEach(() => {
  vi.clearAllMocks();
  auth.permissions = ["production.view", "production.entry", "production.void"];
});
it("creates a daily entry with operator, routing operation and material usage", async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce(reply(options))
    .mockResolvedValueOnce(reply({ id: "saved" }));
  vi.stubGlobal("fetch", fetcher);
  render(<DailyProductionForm orderId="o1" />);
  await screen.findByRole("option", { name: "Operator One" });
  fireEvent.change(screen.getByLabelText("Production date"), {
    target: { value: "2026-09-14" },
  });
  fireEvent.change(screen.getByLabelText("Operator"), {
    target: { value: "u1" },
  });
  fireEvent.change(screen.getByLabelText("Qty processed"), {
    target: { value: "10" },
  });
  fireEvent.change(screen.getByLabelText("Qty good"), {
    target: { value: "9" },
  });
  fireEvent.change(screen.getByLabelText("Qty rejected"), {
    target: { value: "1" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Record Production" }));
  await waitFor(() =>
    expect(push).toHaveBeenCalledWith("/daily-production/saved"),
  );
  const body = JSON.parse(fetcher.mock.calls[1][1].body);
  expect(body).toMatchObject({
    orderId: "o1",
    operationId: "op1",
    operatorId: "u1",
    processed: "10",
    good: "9",
    rejected: "1",
    materials: [{ materialId: "m1", quantity: "20" }],
  });
  expect(body).not.toHaveProperty("processRate");
});
it("prevents next operation entry without upstream output", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(reply(options)));
  render(<DailyProductionForm orderId="o1" />);
  await screen.findByRole("option", { name: "Operator One" });
  fireEvent.change(screen.getByLabelText("Operation"), {
    target: { value: "op2" },
  });
  expect(
    screen.getByText(/previous operation has not finished any output/),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Record Production" }),
  ).toBeDisabled();
});
it("rounds calculated material usage to the supported precision", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue(
        reply({
          ...options,
          orders: [
            {
              ...order,
              materials: [{ ...order.materials[0], usageQty: "0.1" }],
            },
          ],
        }),
      ),
  );
  render(<DailyProductionForm orderId="o1" />);
  await screen.findByRole("option", { name: "Operator One" });
  fireEvent.change(screen.getByLabelText("Qty processed"), {
    target: { value: "3" },
  });
  expect(screen.getByLabelText("Material quantity RM-001")).toHaveTextContent(/0[.,]3/);
  // Clearing and retyping the quantity recalculates instead of sticking.
  fireEvent.change(screen.getByLabelText("Qty processed"), { target: { value: "" } });
  fireEvent.change(screen.getByLabelText("Qty processed"), { target: { value: "5" } });
  expect(screen.getByLabelText("Material quantity RM-001")).toHaveTextContent(/0[.,]5/);
});
it("rejects good plus reject greater than processed", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(reply(options)));
  render(<DailyProductionForm orderId="o1" />);
  await screen.findByRole("option", { name: "Operator One" });
  fireEvent.change(screen.getByLabelText("Qty processed"), {
    target: { value: "10" },
  });
  fireEvent.change(screen.getByLabelText("Qty good"), {
    target: { value: "11" },
  });
  expect(
    screen.getByText(/Good plus rejected cannot exceed processed/),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Record Production" }),
  ).toBeDisabled();
});
it("lists production entries and hides writes for viewers", async () => {
  auth.permissions = ["production.view"];
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(reply({ items: [entry] })));
  render(<DailyProductionIndex />);
  await screen.findByText("DP-0000001");
  expect(screen.getByText("Operator One")).toBeInTheDocument();
  expect(screen.queryByText("New Daily Entry")).not.toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Status filter"), {
    target: { value: "VOIDED" },
  });
  expect(screen.queryByText("DP-0000001")).not.toBeInTheDocument();
});
it("keeps closed entries read-only", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue(
        reply({ ...entry, periodClosed: true, canCorrect: false }),
      ),
  );
  render(<DailyProductionDetail id="e1" />);
  await screen.findByRole("heading", { name: "DP-0000001" });
  expect(
    screen.queryByRole("button", { name: "Void Entry" }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("link", { name: "Edit Entry" }),
  ).not.toBeInTheDocument();
  expect(screen.getByText(/Period closed/)).toBeInTheDocument();
});
it("voids with a reason and version and preserves the detail record", async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce(reply(entry))
    .mockResolvedValueOnce(
      reply({
        ...entry,
        status: "VOIDED",
        canCorrect: false,
        voidReason: "Duplicate",
      }),
    )
    .mockResolvedValue(
      reply({
        ...entry,
        status: "VOIDED",
        canCorrect: false,
        voidReason: "Duplicate",
      }),
    );
  vi.stubGlobal("fetch", fetcher);
  render(<DailyProductionDetail id="e1" />);
  fireEvent.click(await screen.findByRole("button", { name: "Void Entry" }));
  fireEvent.change(screen.getByLabelText("Void reason"), {
    target: { value: "Duplicate" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Confirm Void" }));
  await screen.findByText("VOIDED");
  expect(fetcher.mock.calls[1][0]).toBe("/api/production-entries/e1/void");
  expect(JSON.parse(fetcher.mock.calls[1][1].body)).toEqual({
    version: 1,
    reason: "Duplicate",
  });
  expect(push).not.toHaveBeenCalled();
});

it("books no material at a later operation", async () => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(reply(options)));
  render(<DailyProductionForm orderId="o1" />);
  await screen.findByRole("option", { name: "Operator One" });
  expect(screen.getByLabelText("Material quantity RM-001")).toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Operation"), { target: { value: "op2" } });
  expect(screen.queryByLabelText("Material quantity RM-001")).not.toBeInTheDocument();
  expect(screen.getByText("No material is consumed at this operation.")).toBeInTheDocument();
});
