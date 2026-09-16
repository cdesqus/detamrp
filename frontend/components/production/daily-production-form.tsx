"use client";
import { useEffect, useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { request, quantity } from "./execution-api";
import { Workspace, Panel, Metrics, ErrorMessage, Empty } from "./execution-ui";
import {
  type DailyOptions,
  type ProductionEntry,
  inputRemaining,
  errorText,
} from "./daily-production-api";

export function DailyProductionForm({
  id,
  orderId: initialOrderId,
}: {
  id?: string;
  orderId?: string;
}) {
  const router = useRouter();
  const [options, setOptions] = useState<DailyOptions>();
  const [original, setOriginal] = useState<ProductionEntry>();
  const [orderId, setOrderId] = useState(initialOrderId ?? ""),
    [operationId, setOperationId] = useState(""),
    [operatorId, setOperatorId] = useState(""),
    [productionDate, setDate] = useState(""),
    [shift, setShift] = useState("1"),
    [processed, setProcessed] = useState(""),
    [good, setGood] = useState("0"),
    [rejected, setRejected] = useState("0"),
    [notes, setNotes] = useState(""),
    [error, setError] = useState(""),
    [loaded, setLoaded] = useState(false),
    [busy, setBusy] = useState(false);
  useEffect(() => {
    let mounted = true;
    Promise.all([
      request<DailyOptions>("/production-entries/options"),
      id
        ? request<ProductionEntry>(`/production-entries/${id}`)
        : Promise.resolve(undefined),
    ])
      .then(([o, e]) => {
        if (!mounted) return;
        if (e) {
          if (!e.canCorrect)
            throw new Error(
              "This entry is read-only: period closed, voided, or used by downstream production.",
            );
          setOriginal(e);
          setOrderId(e.orderId);
          setOperationId(e.operationId);
          setOperatorId(e.operatorId);
          setDate(e.productionDate);
          setShift(e.shift);
          setProcessed(e.processed);
          setGood(e.good);
          setRejected(e.rejected);
          setNotes(e.notes);
        } else if (initialOrderId) {
          const chosen = o.orders.find((x) => x.id === initialOrderId);
          setOperationId(chosen?.operations[0]?.id ?? "");
        }
        setOptions(o);
      })
      .catch((e) => {
        if (mounted) setError(errorText(e));
      })
      .finally(() => {
        if (mounted) setLoaded(true);
      });
    return () => {
      mounted = false;
    };
  }, [id, initialOrderId]);
  const selected = options?.orders.find((o) => o.id === orderId),
    index = selected?.operations.findIndex((o) => o.id === operationId) ?? -1;
  const remaining =
    selected && index >= 0
      ? inputRemaining(selected, index) +
        (original ? Number(original.processed) : 0)
      : 0;
  const isClosed =
    options?.closedPeriods.includes(productionDate.slice(0, 7)) ?? false;
  const quantitiesValid =
    Number(processed) > 0 &&
    Number(processed) <= remaining &&
    Number(good) >= 0 &&
    Number(rejected) >= 0 &&
    Number(good) + Number(rejected) <= Number(processed);
  const validation =
    Number(good) + Number(rejected) > Number(processed) && processed
      ? "Good plus rejected cannot exceed processed quantity."
      : isClosed
        ? "Production period is closed."
        : index > 0 && remaining <= 0
          ? "The previous operation has not finished any output yet."
          : Number(processed) > remaining
            ? "Processed quantity exceeds what the previous operation has produced."
            : "";
  // Material is booked at the first operation only, and always at the BOM rate
  // for what is being processed.
  const materialRows =
    index === 0
      ? (selected?.materials ?? []).map((m) => ({
          materialId: m.id,
          quantity: String(
            Number((Number(m.usageQty) * (Number(processed) || 0)).toFixed(6)),
          ),
        }))
      : [];
  const eligible =
    options?.orders.filter(
      (o) =>
        ["RELEASED", "IN_PROGRESS"].includes(o.status) ||
        o.id === original?.orderId,
    ) ?? [];
  function chooseOrder(value: string) {
    setOrderId(value);
    setOperationId(
      options?.orders.find((o) => o.id === value)?.operations[0]?.id ?? "",
    );
    setProcessed("");
    setGood("0");
    setRejected("0");
  }
  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (!selected || index < 0 || !quantitiesValid || validation) {
      setError(
        validation || "Select an order, operation and valid quantities.",
      );
      return;
    }
    setBusy(true);
    try {
      const v = await request<ProductionEntry>(
        id ? `/production-entries/${id}` : "/production-entries",
        {
          method: id ? "PUT" : "POST",
          body: JSON.stringify({
            orderId,
            operationId,
            operatorId,
            productionDate,
            shift,
            processed,
            good,
            rejected,
            notes,
            version: original?.version ?? 0,
            materials: materialRows,
          }),
        },
      );
      router.push(`/daily-production/${v.id}`);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Workspace
      title={id ? "Edit Daily Entry" : "New Daily Entry"}
      subtitle="Record output against the order routing and track every production result."
      crumbs={[
        { label: "Daily production", href: "/daily-production" },
        { label: id ? "Edit entry" : "New entry" },
      ]}
      action={
        <a
          className="ex-button"
          href={id ? `/daily-production/${id}` : "/daily-production"}
        >
          Back
        </a>
      }
    >
      <ErrorMessage error={error} />
      {!loaded ? (
        <Empty title="Loading production setup…" />
      ) : (
        options && (
          <>
            {eligible.length === 0 ? (
              <Empty
                title="No orders available for entry"
                hint="Release a Production Order or resume a partial order before recording production."
              />
            ) : (
              <form onSubmit={submit}>
                <fieldset className="daily-fieldset" disabled={busy}>
                  <Panel
                    title="Production reference"
                    note="Operation sequence and output unit come from the released order."
                  >
                    <div className="execution-form-grid">
                      <label>
                        Production Order
                        <select
                          required
                          disabled={Boolean(id)}
                          value={orderId}
                          onChange={(e) => chooseOrder(e.target.value)}
                        >
                          <option value="">Select a Production Order</option>
                          {eligible.map((o) => (
                            <option key={o.id} value={o.id}>
                              {o.orderNumber} — {o.partNumber}
                            </option>
                          ))}
                        </select>
                      </label>
                      <label>
                        Operation
                        <select
                          required
                          disabled={Boolean(id)}
                          value={operationId}
                          onChange={(e) => {
                            setOperationId(e.target.value);
                          }}
                        >
                          <option value="">Select operation</option>
                          {selected?.operations.map((o) => (
                            <option key={o.id} value={o.id}>
                              {o.sequence}. {o.name}
                            </option>
                          ))}
                        </select>
                      </label>
                      {selected && (
                        <div className="execution-snapshot full">
                          <span>
                            <small>Part number</small>
                            <strong>{selected.partNumber}</strong>
                            {selected.partName}
                          </span>
                          <span>
                            <small>Plant</small>
                            <strong>{selected.plantName}</strong>
                          </span>
                          <span>
                            <small>Order period</small>
                            <strong>
                              {selected.periodStart} → {selected.dueDate}
                            </strong>
                          </span>
                          <span>
                            <small>Remaining input</small>
                            <strong>
                              {quantity(remaining)} {selected.unitCode}
                            </strong>
                          </span>
                        </div>
                      )}
                      <label>
                        Production date
                        <input
                          required
                          type="date"
                          min={selected?.periodStart}
                          max={selected?.dueDate}
                          value={productionDate}
                          onChange={(e) => setDate(e.target.value)}
                        />
                      </label>
                      <label>
                        Shift
                        <input
                          required
                          maxLength={30}
                          value={shift}
                          onChange={(e) => setShift(e.target.value)}
                          placeholder="e.g. 1, 2, Night"
                        />
                      </label>
                      <label>
                        Operator
                        <select
                          required
                          value={operatorId}
                          onChange={(e) => setOperatorId(e.target.value)}
                        >
                          <option value="">Select operator</option>
                          {options.operators.map((o) => (
                            <option key={o.id} value={o.id}>
                              {o.name}
                            </option>
                          ))}
                        </select>
                      </label>
                    </div>
                  </Panel>
                  <Panel
                    title="Production result"
                    note="Good and rejected quantities are part of the processed quantity."
                  >
                    <div className="execution-form-grid daily-result-grid">
                      <label>
                        Qty processed
                        <input
                          required
                          type="number"
                          min="0.000001"
                          step="0.000001"
                          max={remaining}
                          value={processed}
                          onChange={(e) => setProcessed(e.target.value)}
                        />
                      </label>
                      <label>
                        Qty good
                        <input
                          required
                          type="number"
                          min="0"
                          step="0.000001"
                          value={good}
                          onChange={(e) => setGood(e.target.value)}
                        />
                      </label>
                      <label>
                        Qty rejected
                        <input
                          required
                          type="number"
                          min="0"
                          step="0.000001"
                          value={rejected}
                          onChange={(e) => setRejected(e.target.value)}
                        />
                      </label>
                    </div>
                    <div className="execution-panel-body">
                      <ErrorMessage error={validation} />
                      {index > 0 && remaining <= 0 && selected ? (
                        <p className="execution-footer-note">
                          Record the previous operation first, or check the{' '}
                          <a href={`/production-wip/${selected.id}`}>WIP ledger</a> for this order.
                        </p>
                      ) : null}
                      <Metrics
                        items={[
                          {
                            label: "Good yield",
                            value: `${Number(processed) > 0 ? ((Number(good) / Number(processed)) * 100).toFixed(1) : "0.0"}%`,
                            tone: "green",
                          },
                          {
                            label: "Unclassified",
                            value: quantity(
                              Math.max(
                                0,
                                Number(processed) -
                                  Number(good) -
                                  Number(rejected),
                              ),
                            ),
                            hint: "Processed quantity not classified as good or reject",
                          },
                        ]}
                      />
                    </div>
                  </Panel>
                  {selected && index >= 0 && (
                    <Panel
                      title="Material usage"
                      note={
                        index === 0
                          ? "Taken from the bill of materials for the quantity processed. Prices are captured when this entry is recorded."
                          : "Material is booked at the first operation; this one works on output that already carries it."
                      }
                    >
                      {index === 0 ? (
                        <div className="execution-table-wrap">
                          <table className="execution-table">
                            <thead>
                              <tr>
                                <th>Material</th>
                                <th>Unit</th>
                                <th className="numeric">Per piece</th>
                                <th className="numeric">Quantity consumed</th>
                              </tr>
                            </thead>
                            <tbody>
                              {selected.materials.map((m, n) => (
                                <tr key={m.id}>
                                  <td>
                                    <strong>{m.partNumber}</strong>
                                    <small>{m.partName}</small>
                                  </td>
                                  <td>{m.unitCode}</td>
                                  <td className="numeric">{quantity(m.usageQty)}</td>
                                  <td className="numeric">
                                    <output aria-label={`Material quantity ${m.partNumber}`}>
                                      {quantity(materialRows[n]?.quantity ?? "0")} {m.unitCode}
                                    </output>
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </div>
                      ) : (
                        <p className="execution-panel-body">
                          No material is consumed at this operation.
                        </p>
                      )}
                    </Panel>
                  )}
                  <Panel title="Production notes">
                    <div className="execution-form-grid">
                      <label className="full">
                        Notes
                        <textarea
                          rows={3}
                          maxLength={4000}
                          value={notes}
                          onChange={(e) => setNotes(e.target.value)}
                          placeholder="Machine condition, rejection reason or handover notes"
                        />
                      </label>
                    </div>
                    <footer className="execution-form-footer">
                      <span className="execution-footer-note">
                        Entries remain traceable. Corrections are allowed only
                        while the period is open.
                      </span>
                      <a
                        className="ex-button"
                        href={
                          id ? `/daily-production/${id}` : "/daily-production"
                        }
                      >
                        Cancel
                      </a>
                      <button
                        className="ex-button primary"
                        type="submit"
                        disabled={
                          busy ||
                          !selected ||
                          index < 0 ||
                          !quantitiesValid ||
                          Boolean(validation)
                        }
                      >
                        {busy
                          ? "Saving…"
                          : id
                            ? "Save Changes"
                            : "Record Production"}
                      </button>
                    </footer>
                  </Panel>
                </fieldset>
              </form>
            )}
          </>
        )
      )}
    </Workspace>
  );
}
