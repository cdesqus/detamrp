"use client";
import { FormEvent, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
type Scan = {
  kanbanLotId: string;
  kanbanId: string;
  materialCode: string;
  materialName: string;
  quantity: string;
  unit: string;
  warehouse: string;
  location: string;
};
type Session = {
  id: string;
  documentNumber: string;
  destination: string;
  status: string;
  scans: Scan[];
};
export function OutgoingSession({ id }: { id: string }) {
  const router = useRouter();
  const [data, setData] = useState<Session | null>(null);
  const [value, setValue] = useState("");
  const [error, setError] = useState("");
  const input = useRef<HTMLInputElement>(null);
  const load = () =>
    fetch(`/api/outgoing-sessions/${id}`, { credentials: "include" })
      .then((r) => r.json())
      .then((payload) =>
        setData({
          ...payload,
          scans: Array.isArray(payload.scans) ? payload.scans : [],
        }),
      )
      .catch(() => setError("Outgoing session could not be loaded."));
  useEffect(() => {
    void load();
  }, [id]);
  useEffect(() => input.current?.focus(), [data?.scans?.length]);
  async function scan(e: FormEvent) {
    e.preventDefault();
    setError("");
    const r = await fetch(`/api/outgoing-sessions/${id}/scans`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ kanbanId: value }),
    });
    if (!r.ok) {
      setError(
        "Kanban must be received, in stock, and not previously scanned.",
      );
      return;
    }
    setData(await r.json());
    setValue("");
  }
  async function complete() {
    const r = await fetch(`/api/outgoing-sessions/${id}/complete`, {
      method: "POST",
      credentials: "include",
    });
    if (!r.ok) {
      setError("Outgoing could not be completed.");
      return;
    }
    router.push("/outgoing-material");
  }
  if (!data)
    return (
      <div className="table-empty">
        {error || "Loading outgoing session..."}
      </div>
    );
  return (
    <section className="scan-session">
      <div className="scan-session-header">
        <button
          className="table-action"
          onClick={() => router.push("/outgoing-material")}
        >
          Back
        </button>
        <div>
          <h1>{data.documentNumber}</h1>
          <p>Destination: {data.destination || "—"}</p>
        </div>
        <span className="status-pill status-pill--blue">{data.status}</span>
      </div>
      <div className="scan-focus-zone">
        <span>Scan complete Kanban</span>
        <form onSubmit={scan}>
          <input
            ref={input}
            aria-label="Scan complete Kanban"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            autoFocus
          />
          <button className="primary-button">Add</button>
        </form>
        {error && (
          <p className="form-error" role="alert">
            {error}
          </p>
        )}
        <div className="scan-counters">
          <span>
            Full Kanban scanned <b>{data.scans?.length ?? 0}</b>
          </span>
        </div>
      </div>
      <div className="table-frame table-detail">
        <table>
          <thead>
            <tr>
              <th>Kanban ID</th>
              <th>Raw Material</th>
              <th>Full Quantity</th>
              <th>Warehouse / Location</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {(data.scans ?? []).length === 0 ? (
              <tr>
                <td colSpan={5}>
                  <div className="table-empty">
                    Ready to scan in-stock Kanban.
                  </div>
                </td>
              </tr>
            ) : (
              data.scans.map((x) => (
                <tr key={x.kanbanLotId}>
                  <td>{x.kanbanId}</td>
                  <td>
                    {x.materialCode} — {x.materialName}
                  </td>
                  <td>
                    {x.quantity} {x.unit}
                  </td>
                  <td>
                    {x.warehouse} / {x.location}
                  </td>
                  <td>
                    <button
                      className="table-action"
                      onClick={async () => {
                        await fetch(
                          `/api/outgoing-sessions/${id}/scans/${x.kanbanLotId}`,
                          { method: "DELETE", credentials: "include" },
                        );
                        void load();
                      }}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      <div className="supplier-order-actions">
        <span>Quantity is fixed by the Kanban master snapshot.</span>
        <button
          className="primary-button"
          disabled={(data.scans ?? []).length === 0}
          onClick={complete}
        >
          Complete outgoing
        </button>
      </div>
    </section>
  );
}
