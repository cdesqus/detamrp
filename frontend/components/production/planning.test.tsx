import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { beforeEach, expect, it, vi } from "vitest";
import { PlanningForm } from "./planning-form";
import { PlanningActions } from "./planning-actions";
import { PlanningIndex } from "./planning-index";
import { PlanningDetail } from "./planning-detail";

const push = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));
vi.mock("../app-shell/app-shell", () => ({
  useCurrentUser: () => ({
    permissions: ["production.view", "production.plan", "production.order"],
  }),
}));
beforeEach(() => {
  vi.restoreAllMocks();
  push.mockClear();
});
const options = {
  plants: [{ id: "plant", name: "Main Plant" }],
  parts: [
    {
      id: "fg",
      kind: "FG",
      partNumber: "FG-001",
      partName: "Panel",
      unitCode: "PCS",
    },
    {
      id: "rm",
      kind: "RAW_MATERIAL",
      partNumber: "RM-001",
      partName: "Blank",
      unitCode: "PCS",
    },
  ],
};
it("saves a standalone draft with one exclusive output and server assigned number", async () => {
  const fetcher = vi
    .fn()
    .mockResolvedValueOnce({ ok: true, json: async () => options })
    .mockResolvedValueOnce({ ok: true, json: async () => ({ id: "saved" }) });
  vi.stubGlobal("fetch", fetcher);
  render(<PlanningForm />);
  await screen.findByRole("option", { name: "Main Plant" });
  fireEvent.change(screen.getByLabelText("Period start"), {
    target: { value: "2026-09-01" },
  });
  fireEvent.change(screen.getByLabelText("Period end"), {
    target: { value: "2026-09-30" },
  });
  fireEvent.change(screen.getByLabelText("Plant"), {
    target: { value: "plant" },
  });
  fireEvent.change(screen.getByLabelText("Part number 1"), {
    target: { value: "fg" },
  });
  fireEvent.change(screen.getByLabelText("Target quantity 1"), {
    target: { value: "25" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Save Draft" }));
  await waitFor(() =>
    expect(push).toHaveBeenCalledWith("/production-planning/saved"),
  );
  const payload = JSON.parse(fetcher.mock.calls[1][1].body);
  expect(payload.lines).toEqual([{ finishedGoodId: "fg", plannedQty: "25" }]);
  expect(payload).not.toHaveProperty("salesOrderId");
  expect(payload).not.toHaveProperty("planNumber");
});
it("clears the selected part when switching output type", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({ ok: true, json: async () => options }),
  );
  render(<PlanningForm />);
  await screen.findByRole("option", { name: "Main Plant" });
  fireEvent.change(screen.getByLabelText("Part number 1"), {
    target: { value: "fg" },
  });
  fireEvent.change(screen.getByLabelText("Output type 1"), {
    target: { value: "RAW_MATERIAL" },
  });
  expect(screen.getByLabelText("Part number 1")).toHaveValue("");
  expect(
    screen.queryByRole("option", { name: /FG-001/ }),
  ).not.toBeInTheDocument();
});
it("shows loading failures and does not present an editable form", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue({
        ok: false,
        json: async () => ({ message: "Access denied" }),
      }),
  );
  render(<PlanningForm />);
  expect(await screen.findByRole("alert")).toHaveTextContent("Access denied");
  expect(
    screen.queryByRole("button", { name: "Save Draft" }),
  ).not.toBeInTheDocument();
});
it("approved and closed plans expose only the permitted status actions", () => {
  const change = vi.fn();
  const { rerender } = render(
    <PlanningActions
      plan={{ id: "p", status: "APPROVED", linkedProductionOrders: 0 }}
      onChange={change}
    />,
  );
  expect(
    screen.getByRole("button", { name: "Create Production Order" }),
  ).toBeInTheDocument();
  expect(
    screen.getByRole("button", { name: "Close Planning" }),
  ).toBeInTheDocument();
  expect(screen.queryByText("Edit")).not.toBeInTheDocument();
  expect(screen.queryByText("Delete")).not.toBeInTheDocument();
  rerender(
    <PlanningActions
      plan={{ id: "p", status: "CLOSED", linkedProductionOrders: 1 }}
      onChange={change}
    />,
  );
  expect(screen.queryByRole("button")).not.toBeInTheDocument();
});
it("hides duplicate order creation after orders are linked", () => {
  render(
    <PlanningActions
      plan={{ id: "p", status: "APPROVED", linkedProductionOrders: 1 }}
      onChange={vi.fn()}
    />,
  );
  expect(
    screen.queryByRole("button", { name: "Create Production Order" }),
  ).not.toBeInTheDocument();
});
const plan = {
  id: "p",
  planNumber: "PP-202609-000001",
  status: "APPROVED",
  periodStart: "2026-09-01T00:00:00Z",
  periodEnd: "2026-09-30T00:00:00Z",
  plantName: "Main Plant",
  createdBy: "Planner",
  updatedAt: "2026-09-14T00:00:00Z",
  totalPart: 1,
  totalPlannedQty: "100",
  linkedProductionOrders: 1,
  notes: "Monthly target",
  lines: [
    {
      id: "line",
      partNumber: "FG-001",
      partName: "Panel",
      unitCode: "PCS",
      plannedQty: "100",
    },
  ],
  orders: [
    {
      id: "order",
      orderNumber: "PRO-001",
      plannedQty: "100",
      goodQty: "25",
      status: "PARTIAL",
    },
  ],
  history: [
    { action: "APPROVE", actor: "Planner", occurredAt: "2026-09-14T00:00:00Z" },
  ],
};
it("lists planning totals and filters by status", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValue({ ok: true, json: async () => ({ items: [plan] }) }),
  );
  render(<PlanningIndex />);
  await screen.findByText("PP-202609-000001");
  expect(
    screen.getByRole("columnheader", { name: "Linked Production Orders" }),
  ).toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Status filter"), {
    target: { value: "DRAFT" },
  });
  expect(screen.queryByText("PP-202609-000001")).not.toBeInTheDocument();
});
it("shows targets, linked order progress and change history", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue({ ok: true, json: async () => plan }),
  );
  render(<PlanningDetail id="p" />);
  await screen.findByText("PP-202609-000001");
  expect(screen.getByText("PRO-001")).toBeInTheDocument();
  expect(screen.getByRole("progressbar")).toHaveAttribute("value", "25");
  expect(screen.getByText("APPROVE")).toBeInTheDocument();
  expect(screen.getByText("FG-001")).toBeInTheDocument();
});
it("stays on detail after approval and makes order creation available", async () => {
  vi.stubGlobal(
    "fetch",
    vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          ...plan,
          status: "DRAFT",
          linkedProductionOrders: 0,
          orders: [],
        }),
      })
      .mockResolvedValue({
        ok: true,
        json: async () => ({ ...plan, linkedProductionOrders: 0, orders: [] }),
      }),
  );
  render(<PlanningDetail id="p" />);
  fireEvent.click(
    await screen.findByRole("button", { name: "Approve Planning" }),
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Confirm Approve Planning" }),
  );
  await screen.findByRole("button", { name: "Create Production Order" });
  expect(push).not.toHaveBeenCalled();
});
