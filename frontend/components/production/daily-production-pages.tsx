"use client";
import { useCallback, useEffect, useState } from "react";
import { useCurrentUser } from "../app-shell/app-shell";
import { request, quantity, money } from "./execution-api";
import {
  Workspace,
  Panel,
  Metrics,
  Badge,
  Empty,
  ErrorMessage,
  ConfirmDialog,
  Tabs,
  date,
  dateTime,
} from "./execution-ui";
import { type ProductionEntry, errorText } from "./daily-production-api";

export function DailyProductionIndex() {
  const user = useCurrentUser(),
    [items, setItems] = useState<ProductionEntry[]>([]),
    [loading, setLoading] = useState(true),
    [error, setError] = useState(""),
    [search, setSearch] = useState(""),
    [status, setStatus] = useState(""),
    [from, setFrom] = useState(""),
    [to, setTo] = useState(""),
    [shift, setShift] = useState(""),
    [page, setPage] = useState(1),
    [close, setClose] = useState(false),
    [period, setPeriod] = useState(""),
    [reason, setReason] = useState(""),
    [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    setLoading(true);
    try {
      const r = await request<{ items: ProductionEntry[] }>(
        "/production-entries",
      );
      setItems(r.items ?? []);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setLoading(false);
    }
  }, []);
  useEffect(() => {
    void load();
  }, [load]);
  const filtered = items.filter(
    (e) =>
      (!status || e.status === status) &&
      (!from || e.productionDate >= from) &&
      (!to || e.productionDate <= to) &&
      (!shift || e.shift === shift) &&
      `${e.entryNumber} ${e.orderNumber} ${e.partNumber} ${e.operatorName} ${e.operationName}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  const posted = filtered.filter((e) => e.status === "POSTED"),
    processed = posted.reduce((n, e) => n + Number(e.processed), 0),
    good = posted.reduce((n, e) => n + Number(e.good), 0),
    rejected = posted.reduce((n, e) => n + Number(e.rejected), 0),
    pages = Math.max(1, Math.ceil(filtered.length / 20)),
    current = Math.min(page, pages);
  async function closePeriod() {
    if (!period || !reason.trim()) return;
    setBusy(true);
    setError("");
    try {
      await request(`/production-periods/${period}/close`, {
        method: "POST",
        body: JSON.stringify({ reason }),
      });
      setClose(false);
      setReason("");
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Workspace
      title="Daily Production"
      subtitle="Shift output, quality results and traceable production entries."
      crumbs={[{ label: "Daily production" }]}
      action={
        <>
          {user?.permissions.includes("production.period") && (
            <button className="ex-button" onClick={() => setClose(true)}>
              Close Period
            </button>
          )}
          {user?.permissions.includes("production.entry") && (
            <a className="ex-button primary" href="/daily-production/new">
              New Daily Entry
            </a>
          )}
        </>
      }
    >
      <ErrorMessage error={error} />
      <Metrics
        items={[
          {
            label: "Processed",
            value: quantity(processed),
            hint: "Posted entries in current filter",
            tone: "accent",
          },
          { label: "Good output", value: quantity(good), tone: "green" },
          { label: "Rejected", value: quantity(rejected), tone: "red" },
          {
            label: "Good yield",
            value: `${processed ? ((good / processed) * 100).toFixed(1) : "0.0"}%`,
            hint: "Good / processed across operations",
          },
        ]}
      />
      <Panel
        title="Production entries"
        note="Filter by order, process, date or shift."
      >
        <div className="execution-filters">
          <label>
            Search
            <input
              type="search"
              placeholder="Entry, order, part or operator"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(1);
              }}
            />
          </label>
          <label>
            From date
            <input
              type="date"
              value={from}
              onChange={(e) => {
                setFrom(e.target.value);
                setPage(1);
              }}
            />
          </label>
          <label>
            To date
            <input
              type="date"
              value={to}
              onChange={(e) => {
                setTo(e.target.value);
                setPage(1);
              }}
            />
          </label>
          <label>
            Shift
            <select
              value={shift}
              onChange={(e) => {
                setShift(e.target.value);
                setPage(1);
              }}
            >
              <option value="">All shifts</option>
              {[...new Set(items.map((e) => e.shift))].sort().map((s) => (
                <option key={s}>{s}</option>
              ))}
            </select>
          </label>
          <label>
            Status
            <select
              aria-label="Status filter"
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">All statuses</option>
              <option>POSTED</option>
              <option>VOIDED</option>
            </select>
          </label>
          <button
            className="ex-button"
            disabled={loading}
            onClick={() => void load()}
          >
            Refresh
          </button>
        </div>
        <div className="execution-table-wrap">
          <table className="execution-table">
            <thead>
              <tr>
                {[
                  "Entry / Date",
                  "Production Order / Part",
                  "Operation / Shift",
                  "Processed",
                  "Good",
                  "Reject",
                  "Operator",
                  "Status",
                  "Actions",
                ].map((h) => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td colSpan={9}>
                    <Empty title="Loading production entries…" />
                  </td>
                </tr>
              ) : filtered.length === 0 ? (
                <tr>
                  <td colSpan={9}>
                    <Empty
                      title="No entries found"
                      hint="Record production from a released order or adjust the filters."
                    />
                  </td>
                </tr>
              ) : (
                filtered.slice((current - 1) * 20, current * 20).map((e) => (
                  <tr key={e.id}>
                    <td>
                      <a
                        className="execution-link"
                        href={`/daily-production/${e.id}`}
                      >
                        {e.entryNumber}
                      </a>
                      <small>{date(e.productionDate)}</small>
                    </td>
                    <td>
                      <a href={`/production-orders/${e.orderId}`}>
                        {e.orderNumber}
                      </a>
                      <small>
                        {e.partNumber} · {e.unitCode}
                      </small>
                    </td>
                    <td>
                      <strong>{e.operationName}</strong>
                      <small>Shift {e.shift}</small>
                    </td>
                    <td>{quantity(e.processed)}</td>
                    <td>{quantity(e.good)}</td>
                    <td>{quantity(e.rejected)}</td>
                    <td>{e.operatorName}</td>
                    <td>
                      <Badge status={e.status} />
                      {e.periodClosed && <small>Period closed</small>}
                    </td>
                    <td>
                      <a
                        className="ex-button"
                        href={`/daily-production/${e.id}`}
                      >
                        View
                      </a>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
        <footer className="execution-pagination">
          <span>{filtered.length} entries</span>
          <button
            className="ex-button"
            disabled={current === 1}
            onClick={() => setPage(current - 1)}
          >
            Previous
          </button>
          <span>
            {current} / {pages}
          </span>
          <button
            className="ex-button"
            disabled={current === pages}
            onClick={() => setPage(current + 1)}
          >
            Next
          </button>
        </footer>
      </Panel>
      {close && (
        <ConfirmDialog
          title="Close Production Period"
          confirmLabel="Confirm Close"
          busy={busy}
          onConfirm={() => void closePeriod()}
          onClose={() => {
            if (!busy) setClose(false);
          }}
        >
          <p>
            Entries in this month become read-only. New entries and corrections
            will be blocked.
          </p>
          <label>
            Period
            <input
              required
              type="month"
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
            />
          </label>
          <label>
            Close reason
            <textarea
              required
              maxLength={2000}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </label>
          <ErrorMessage error={error} />
        </ConfirmDialog>
      )}
    </Workspace>
  );
}

export function DailyProductionDetail({ id }: { id: string }) {
  const user = useCurrentUser(),
    [entry, setEntry] = useState<ProductionEntry>(),
    [error, setError] = useState(""),
    [tab, setTab] = useState("Overview"),
    [dialog, setDialog] = useState(false),
    [reason, setReason] = useState(""),
    [busy, setBusy] = useState(false);
  const load = useCallback(async () => {
    try {
      setEntry(await request<ProductionEntry>(`/production-entries/${id}`));
    } catch (e) {
      setError(errorText(e));
    }
  }, [id]);
  useEffect(() => {
    void load();
  }, [load]);
  async function voidEntry() {
    if (!entry || !reason.trim()) return;
    setBusy(true);
    setError("");
    try {
      setEntry(
        await request<ProductionEntry>(`/production-entries/${id}/void`, {
          method: "POST",
          body: JSON.stringify({ version: entry.version, reason }),
        }),
      );
      setDialog(false);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Workspace
      title={entry?.entryNumber ?? "Daily Entry"}
      subtitle="Production result, material usage and correction history."
      crumbs={[
        { label: "Daily production", href: "/daily-production" },
        { label: entry?.entryNumber ?? "Detail" },
      ]}
      action={
        <>
          <a className="ex-button" href="/daily-production">
            Back to entries
          </a>
          {entry?.canCorrect &&
            user?.permissions.includes("production.entry") && (
              <a className="ex-button" href={`/daily-production/${id}/edit`}>
                Edit Entry
              </a>
            )}
          {entry?.canCorrect &&
            user?.permissions.includes("production.void") && (
              <button
                className="ex-button danger"
                onClick={() => setDialog(true)}
              >
                Void Entry
              </button>
            )}
        </>
      }
    >
      <ErrorMessage error={error} />
      {!entry ? (
        <Empty title={error ? "Entry unavailable" : "Loading daily entry…"} />
      ) : (
        <>
          <div className="execution-record-strip">
            <Badge status={entry.status} />
            <span>
              Recorded by {entry.createdBy} · Updated{" "}
              {dateTime(entry.updatedAt)}
            </span>
            {entry.periodClosed && <strong>Period closed · Read-only</strong>}
            {entry.status === "POSTED" &&
              !entry.canCorrect &&
              !entry.periodClosed && (
                <strong>
                  Corrections locked by downstream activity or order status
                </strong>
              )}
          </div>
          <Metrics
            items={[
              {
                label: "Processed",
                value: quantity(entry.processed),
                hint: entry.unitCode,
                tone: "accent",
              },
              {
                label: "Good output",
                value: quantity(entry.good),
                tone: "green",
              },
              {
                label: "Rejected",
                value: quantity(entry.rejected),
                tone: "red",
              },
              {
                label: "Good yield",
                value: `${Number(entry.processed) ? ((Number(entry.good) / Number(entry.processed)) * 100).toFixed(1) : "0.0"}%`,
              },
            ]}
          />
          {entry.status === "VOIDED" && (
            <Panel title="Voided entry">
              <div className="execution-panel-body">
                <p>{entry.voidReason}</p>
                <p>
                  This entry is retained for history and excluded from
                  production totals.
                </p>
              </div>
            </Panel>
          )}
          <Panel title="Entry detail">
            <Tabs
              tabs={["Overview", "Material usage", "Audit trail"]}
              active={tab}
              onSelect={setTab}
              label="Entry details"
            />
            {tab === "Overview" && (
              <div className="execution-panel-body execution-form-grid">
                <div className="execution-snapshot full">
                  <span>
                    <small>Production Order</small>
                    <a href={`/production-orders/${entry.orderId}`}>
                      <strong>{entry.orderNumber}</strong>
                    </a>
                  </span>
                  <span>
                    <small>Part</small>
                    <strong>{entry.partNumber}</strong>
                    {entry.partName}
                  </span>
                  <span>
                    <small>Plant</small>
                    <strong>{entry.plantName}</strong>
                  </span>
                  <span>
                    <small>Operation</small>
                    <strong>
                      {entry.sequence}. {entry.operationName}
                    </strong>
                  </span>
                  <span>
                    <small>Date / Shift</small>
                    <strong>
                      {date(entry.productionDate)} · {entry.shift}
                    </strong>
                  </span>
                  <span>
                    <small>Operator</small>
                    <strong>{entry.operatorName}</strong>
                  </span>
                </div>
                <div className="full daily-notes">
                  <small>Production notes</small>
                  <p>{entry.notes || "No notes recorded."}</p>
                </div>
              </div>
            )}
            {tab === "Material usage" && (
              <>
                {entry.materials.length === 0 ? (
                  <Empty
                    title="No direct material usage"
                    hint="Direct materials are recorded on the first operation."
                  />
                ) : (
                  <div className="execution-table-wrap">
                    <table className="execution-table">
                      <thead>
                        <tr>
                          <th>Material</th>
                          <th>Unit</th>
                          <th>Consumed quantity</th>
                          {user?.permissions.includes("production.report") && (
                            <>
                              <th>Price snapshot</th>
                              <th>Material cost</th>
                            </>
                          )}
                        </tr>
                      </thead>
                      <tbody>
                        {entry.materials.map((m) => (
                          <tr key={m.materialId}>
                            <td>
                              <strong>{m.partNumber}</strong>
                              <small>{m.partName}</small>
                            </td>
                            <td>{m.unitCode}</td>
                            <td>{quantity(m.quantity)}</td>
                            {user?.permissions.includes(
                              "production.report",
                            ) && (
                              <>
                                <td>{money(m.unitPrice, entry.currency)}</td>
                                <td>{money(m.cost, entry.currency)}</td>
                              </>
                            )}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
                {user?.permissions.includes("production.report") && (
                  <div className="execution-panel-body">
                    <Metrics
                      items={[
                        {
                          label: "Material snapshot",
                          value: money(entry.materialCost, entry.currency),
                        },
                        {
                          label: "Process snapshot",
                          value: money(entry.processCost, entry.currency),
                          hint: `${money(entry.processRate, entry.currency)} / processed unit`,
                        },
                      ]}
                    />
                  </div>
                )}
              </>
            )}
            {tab === "Audit trail" && (
              <div className="execution-panel-body">
                {entry.history.length === 0 ? (
                  <Empty title="No recorded changes" />
                ) : (
                  <ol className="daily-history">
                    {entry.history.map((h, n) => (
                      <li key={n}>
                        <strong>{h.action}</strong>
                        <span>
                          {h.actor} · {dateTime(h.occurredAt)}
                        </span>
                      </li>
                    ))}
                  </ol>
                )}
              </div>
            )}
          </Panel>
        </>
      )}
      {dialog && (
        <ConfirmDialog
          title="Void Daily Entry"
          confirmLabel="Confirm Void"
          tone="danger"
          busy={busy}
          onConfirm={() => void voidEntry()}
          onClose={() => {
            if (!busy) setDialog(false);
          }}
        >
          <p>
            Production totals will be recalculated. The original entry and its
            reason remain in history.
          </p>
          <label>
            Void reason
            <textarea
              required
              maxLength={2000}
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </label>
          <ErrorMessage error={error} />
        </ConfirmDialog>
      )}
    </Workspace>
  );
}
