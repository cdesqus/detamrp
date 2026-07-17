# NextGen Logistics & Production Control MVP — Design Specification

## 1. Tujuan

Membangun prototype yang dapat berkembang menjadi SaaS untuk procurement raw material, approval, auto-generation DN dan Kanban, receiving berbasis scan, inventory control, serta integrasi asynchronous dengan Sage X3 2022 (AR & Finance).

Prototype dipakai oleh satu perusahaan, tetapi database tetap tenant-aware. `tenant_id` diisi otomatis dan tidak ditampilkan kepada user operasional.

## 2. Prinsip Produk

- UX compact dan data-dense seperti Linear/Notion.
- Tinggi row tabel sekitar 36–40px dan font isi 13–14px.
- Tidak menggunakan heading, card, tombol, atau whitespace berukuran berlebihan.
- Semua data table mendukung search, filter, sorting, pagination, dan show/hide columns.
- Preferensi kolom disimpan per user.
- `Created By`, `Created At`, `Updated By`, dan `Updated At` tersedia pada semua module.
- Proses lokal tidak menunggu Sage atau SMTP.
- Inventory ledger bersifat append-only.
- Seluruh transaksi penting memiliki audit trail dan PDF.

## 3. Arsitektur

### 3.1 Komponen

- Modular monolith SaaS application.
- PostgreSQL dengan `tenant_id` dan Row-Level Security.
- Background worker untuk outbox, email, PDF, dan reconciliation.
- Object storage abstraction; MinIO atau managed directory dapat digunakan pada server kantor.
- On-premise Integration Agent yang hanya membuka koneksi outbound HTTPS.
- Sage X3 tetap berada di jaringan privat.

### 3.2 Module boundary

- Identity & Access
- Data Master
- Procurement
- Approval
- Delivery & Kanban
- Receiving
- Inventory
- Notification & Email
- Document Generation
- Sage Integration
- Reporting & Audit

### 3.3 Navigasi

```text
Dashboard

Procurement
├── Supplier Orders
└── Approval Inbox

Logistics
├── Delivery Notes
├── Receiving
├── Outgoing Material
└── Stock Movement

Inventory
├── Stock Raw Material
├── Stock Adjustment
└── Stock Taking

Data Master
├── Measurements
├── Suppliers
├── Raw Materials
├── Warehouses
└── Locations / Racks

Reports
├── Purchase Orders
├── Receiving
├── Inventory
├── Supplier Performance
└── Stock Audit

Settings
├── Users
├── Roles & Permissions
├── SMTP Settings
├── Email Log
├── Integration Monitor
└── General Settings
```

Menu disembunyikan jika user tidak memiliki permission.

## 4. Tenant dan Security Boundary

- Prototype memiliki satu tenant default.
- Tenant tidak dapat dipilih atau dilihat user operasional.
- Semua tabel milik tenant memiliki `tenant_id`.
- Composite foreign key selalu menyertakan `tenant_id`.
- Application role tidak boleh memiliki `BYPASSRLS` atau menjadi table owner.
- Gunakan `FORCE ROW LEVEL SECURITY`.
- Tenant context ditetapkan secara transaction-local.
- Background job memproses satu tenant per transaksi.
- Satu Integration Agent terikat pada satu tenant dan satu Sage environment.

Contoh policy:

```sql
USING (tenant_id = current_setting('app.tenant_id')::uuid)
WITH CHECK (tenant_id = current_setting('app.tenant_id')::uuid)
```

## 5. Authentication dan RBAC

### 5.1 Authentication MVP

- Username dan password lokal.
- Password disimpan menggunakan password hashing modern.
- Login rate limiting, session expiry, logout, dan password reset.
- Login page compact tanpa hero atau komponen oversized.
- Struktur authentication harus memungkinkan penambahan SSO kelak tanpa mengubah RBAC.

### 5.2 Role awal

- `DIRECTOR`
- `PURCHASING`
- `LOGISTICS_PLANNER`
- `WAREHOUSE`
- `FINANCE`
- `ADMIN`
- `VIEWER`

Satu user dapat memiliki beberapa role. Authorization memakai permission granular, bukan hard-code role name.

Permission utama mencakup:

```text
po.view, po.create, po.edit_draft, po.submit, po.approve, po.reject
po.price.view, po.unit_price.edit
dn.view, dn.issue, dn.cancel
receiving.view, receiving.create, receiving.submit
inventory.view, inventory.consume, inventory.move
inventory.adjust_plus, inventory.adjust_minus, inventory.stock_take
integration.view, integration.retry
smtp_settings.view, smtp_settings.manage, smtp_settings.test
email_log.view, email_log.resend
user.manage, role.manage, configuration.manage
```

## 6. Data Master

### 6.1 Measurements

Field: unit code, unit name, decimal allowed, status, dan audit fields. Contoh: PCS, KG, BOX, ROLL, METER, LITER.

`KANBAN` bukan measurement. Kanban adalah physical lot grouping.

### 6.2 Suppliers

Field:

- Supplier Code
- Sage Supplier Code
- Supplier Name
- Email wajib
- Phone, address, contact person
- Currency
- Status
- Audit fields

Supplier yang sudah dipakai tidak dihapus; status diubah menjadi inactive.

### 6.3 Raw Materials

Field:

- Material Code
- Sage Item Code
- Material Name
- Supplier
- Base Unit
- Qty per Kanban
- Minimum Stock Quantity opsional
- Description
- Status
- Audit fields

Constraint: `qty_per_kanban > 0`. Harga tidak disimpan pada master. Unit price dimasukkan pada PO.

### 6.4 Warehouses dan Locations

Warehouse memiliki code, name, Sage warehouse code, address, type, dan status. Location memiliki warehouse, location code, zone, rack, bin, dan status.

## 7. Supplier Order

### 7.1 Create PO UX

Header:

- PO Number: auto-generated ketika disimpan
- Supplier: searchable dropdown
- Order Date
- Expected Delivery Date
- Currency
- Notes opsional

Supplier dipilih satu kali dan satu PO hanya memiliki satu supplier. Tombol `+ Raw Material` disabled sebelum supplier dipilih. Material picker hanya menampilkan active raw material milik supplier terpilih dan dapat dicari berdasarkan code atau name.

Raw material entry:

- Raw Material
- Qty per Kanban: otomatis dari master, read-only
- Total Kanban: integer positif
- Total Quantity: otomatis
- Unit Price: input user berizin
- Total Amount: otomatis

Tidak ada istilah `Add Line`, `Line Items`, atau `Total Lines` pada UI. Action diberi label `+ Raw Material`.

Formula:

```text
ordered_base_qty = qty_per_kanban × total_kanban
line_total       = ordered_base_qty × unit_price
```

Tidak ada Required Date per material. `Expected Delivery Date` pada header berlaku untuk seluruh PO.

Action:

```text
[ Save as Draft ] [ Save & Send for Approval ]
```

Jika supplier diganti setelah material ditambahkan, user harus menyetujui penghapusan seluruh raw material pada PO.

### 7.2 PO status

```text
DRAFT
PENDING_APPROVAL
APPROVED
PARTIALLY_RECEIVED
FULLY_RECEIVED
REJECTED
CANCELLED
```

Tidak ada status `PLANNED`. Informasi alokasi ditampilkan sebagai progress: Ordered, Allocated to DN, Not Yet Allocated, Received, dan Outstanding.

Status receiving berubah berdasarkan transaksi lokal, tidak menunggu Sage.

### 7.3 Database utama

`purchase_orders` menyimpan tenant, PO number, supplier, dates, currency, status, approval snapshot, version, Sage number, dan audit fields.

Constraint:

- `UNIQUE (tenant_id, id)`
- `UNIQUE (tenant_id, po_number)`
- Composite FK ke supplier
- Expected delivery tidak boleh sebelum order date
- Supplier, currency, quantities, dan prices immutable setelah submission

`purchase_order_lines` adalah istilah database untuk raw material entries. Ia menyimpan material, base unit snapshot, qty per Kanban snapshot, total Kanban, ordered base quantity, unit price, dan line total.

Gunakan `numeric(20,6)` untuk quantity dan price; jangan gunakan floating point.

## 8. Approval

### 8.1 Policy

- Satu configurable approver per tenant.
- Approver identity dan email disalin sebagai snapshot saat submission.
- Director dapat approve melalui email tanpa login/VPN atau melalui Approval Inbox setelah login.
- Kedua jalur memakai approval request yang sama dan hanya satu yang dapat berhasil.

### 8.2 Secure email approval

- Random opaque token minimal 256-bit.
- Database hanya menyimpan token hash.
- Token single-use, expiring, revocable, dan terikat pada PO version.
- Resend mencabut token lama dan membuat token baru.
- Approval dilakukan dalam transaksi dengan row locking.
- Approval, PO state transition, DN/Kanban generation, Sage outbox, dan notification outbox dibuat atomically.

Email security scanner dapat membuka link otomatis. Karena itu HTTP `GET` tidak boleh mengubah status bisnis. Tombol email membuka halaman review mobile tanpa login; Director melakukan satu `POST` confirmation. Halaman tidak memakai third-party scripts dan menggunakan `Referrer-Policy: no-referrer`.

### 8.3 Approval audit

Event append-only: submitted, email queued/sent/failed, link opened, approved, rejected, expired, revoked, resent. Audit menyimpan timestamp, approver snapshot, PO version, masked IP, user agent, result, dan correlation ID.

## 9. Auto DN dan Kanban setelah Approval

Keputusan final prototype menggantikan rancangan manual multi-DN:

- Satu approved PO otomatis menghasilkan satu DN.
- DN memuat seluruh raw material dan seluruh ordered Kanban pada PO.
- Satu DN dapat dipenuhi melalui beberapa receiving parsial.
- User tidak perlu menjalankan Create DN.
- Split shipment dilakukan menggunakan DN yang sama dalam beberapa kedatangan.

Dalam transaksi approval:

1. Lock approval dan PO.
2. Ubah approval serta PO menjadi approved.
3. Buat `delivery_notes`.
4. Buat `delivery_note_lines` untuk seluruh PO material entries.
5. Buat seluruh `kanban_lots`.
6. Buat `SAGE_PO_CREATE` pada integration outbox.
7. Buat document/email job pada notification outbox.
8. Commit.

DN status:

```text
ISSUED → PARTIALLY_RECEIVED → RECEIVED
ISSUED → CANCELLED, hanya jika belum ada receiving
```

## 10. Kanban Model

- Satu Kanban lot merepresentasikan satu physical lot dengan quantity tetap.
- Contoh ID: `KBN-202607-000001`.
- Unique constraint: `(tenant_id, kanban_id)`.
- Barcode utama menggunakan Code 128; teks ID dicetak di bawahnya.
- QR kecil dapat disediakan sebagai fallback mobile.
- Barcode hanya membawa Kanban ID, bukan harga atau data sensitif.

Lifecycle:

```text
ISSUED → RECEIVED → CONSUMED
ISSUED → CANCELLED
```

Movement dicatat sebagai ledger event dan current location update; bukan status permanen.

Kanban label ukuran awal 100 × 150 mm dan memuat company, material, material code, quantity/unit, supplier, PO, DN, expected date, barcode, dan human-readable ID.

## 11. Supplier Email dan Documents

Setelah approval, supplier otomatis menerima tiga PDF terpisah:

```text
PO-00125.pdf
DN-00042.pdf
KANBAN-DN-00042.pdf
```

- PO PDF berisi nilai komersial.
- DN PDF berisi delivery reference dan daftar material/Kanban.
- Kanban PDF berisi satu printable label per Kanban.

Email supplier tidak menunggu Sage. Jika email gagal, PO dan DN tetap valid; user dapat resend. Untuk file Kanban yang melampaui batas attachment provider, versi production dapat memakai secure download link.

## 12. Receiving

### 12.1 Index view

Compact table dengan default columns: Receiving Number, DN Number, PO Number, Supplier, Receiving Date, Kanban, Status, Sage Receipt Number.

Optional columns: warehouse, location, quantities, Sage sync status, Created/Updated/Completed By dan timestamps, notes. Filters mencakup date, status, supplier, PO, DN, warehouse, creator, completer, dan Sage number filled/empty.

### 12.2 Create Receiving form

Sebelum scanning, user mengisi form normal:

- Receiving Number auto-generated
- DN Number: scan atau searchable select
- PO dan Supplier auto-filled dari DN
- Receiving Date
- Warehouse Destination
- Location/Rack
- Notes

Preview menampilkan material, planned Kanban, received, dan outstanding. Klik `Create Receiving` membuat session dan membuka focused scanning screen.

### 12.3 Receiving session

- Input scanner selalu autofocus.
- Scan disimpan server-side satu per satu.
- Feedback valid, duplicate, already received, wrong DN, dan cancelled dibedakan dengan ikon, teks, warna, dan bunyi.
- Session dapat ACTIVE, PAUSED, SUBMITTED, CANCELLED, atau EXPIRED.
- Hasil scan tidak hilang saat refresh atau reconnect.
- Partial receiving diperbolehkan; excess dan duplicate dilarang.
- Satu DN hanya boleh memiliki satu ACTIVE session.
- Session ACTIVE terkunci untuk satu operator.
- Session PAUSED dapat di-resume dan diambil alih operator lain dengan audit event.
- Setelah partial receiving selesai, session baru dapat dibuat untuk outstanding Kanban pada DN yang sama.

Database partial unique constraint:

```sql
UNIQUE (tenant_id, delivery_note_id) WHERE status = 'ACTIVE'
```

Final submit mengunci seluruh Kanban, melakukan revalidation, membuat receiving dan ledger, mengubah Kanban menjadi received, membuat Sage outbox, menutup session, dan commit. Jika satu Kanban invalid, jangan melakukan partial silent submit.

### 12.4 Receiving PDF

PDF dibuat langsung setelah receiving lokal selesai, tanpa menunggu Sage. PDF berisi header transaksi, received Kanban, planned, previously received, received now, outstanding Kanban dan base quantity, DN status, creator/completer, dan Sage Receipt Number jika sudah ada.

Outstanding merupakan immutable snapshot pada waktu receiving selesai. Jika PDF generation gagal, receiving tidak di-rollback; user dapat regenerate.

## 13. Inventory

### 13.1 Stock Raw Material

Read-only projection per material, warehouse, dan location: available Kanban, total base quantity, unit, last receiving, movement, dan stock take.

### 13.2 Outgoing Material

Berada di Logistics. Menggunakan form dan focused scan session untuk Kanban berstatus received. Completion membuat `CONSUMPTION`, mengubah Kanban menjadi consumed, dan membuat PDF. Tidak ada Sage push pada Phase 1.

### 13.3 Stock Movement

Berada di Logistics. Scan Kanban dari source menuju destination. Source dan destination tidak boleh sama. Ledger membuat `MOVE_OUT` dan `MOVE_IN` berpasangan dan memperbarui current location projection.

### 13.4 Stock Adjustment

Mengubah stok melalui plus/minus dengan reason code dan detailed reason. Prototype hanya mendukung full-Kanban adjustment. Adjustment tidak boleh membuat stok negatif. Koreksi terhadap transaksi dilakukan melalui reversal, bukan edit/delete ledger.

### 13.5 Stock Taking

Stock Taking mengaudit system baseline terhadap scan fisik dan tidak langsung mengubah stok. Setelah reviewer menyetujui variance, sistem otomatis membuat adjustment atau movement terkait. Stock Taking dan Adjustment tetap merupakan transaksi terpisah serta saling mereferensikan.

### 13.6 Ledger

`inventory_ledger_entries` bersifat append-only dan mencatat event type, material, Kanban, warehouse/location, quantity delta, unit, business reference, timestamp, dan actor.

Event utama:

```text
RECEIPT
CONSUMPTION
MOVE_OUT
MOVE_IN
ADJUSTMENT_PLUS
ADJUSTMENT_MINUS
STOCK_TAKE_VARIANCE
```

## 14. Sage Integration

### 14.1 Transactional outbox

`integration_outbox` menyimpan tenant, aggregate, event, version, immutable JSON payload, status, availability, claim lease, attempt count, errors, correlation ID, dan timestamps.

Logical uniqueness:

```text
(tenant_id, aggregate_type, aggregate_id, event_type, aggregate_version)
```

Status internal:

```text
PENDING, CLAIMED, RETRYING, SUCCEEDED, DEAD_LETTER
```

PO approval membuat `SAGE_PO_CREATE`. Receiving completion membuat `SAGE_GOODS_RECEIPT_CREATE`. Goods receipt job memiliki explicit dependency pada successful PO-create job.

### 14.2 Agent delivery

- Agent claims jobs melalui outbound HTTPS.
- SaaS workers memakai `FOR UPDATE SKIP LOCKED` dan leased claims.
- Agent memiliki local inbox/idempotency store keyed by message ID.
- PO number atau receiving reference dikirim sebagai Sage external reference.
- Setelah timeout, agent melakukan lookup/reconciliation sebelum retry create.
- Duplicate acknowledgments idempotent; conflicting Sage IDs menghasilkan critical alert.
- Retry memakai exponential backoff dan jitter.
- Authentication dan business validation errors membutuhkan tindakan; transient errors retry otomatis.
- Reconciliation terjadwal membandingkan local document dan Sage external reference.

### 14.3 Operational UI

Order dan Receiving list default hanya menampilkan Sage document number. Nilai kosong berarti belum tersedia. Technical sync status tersedia melalui show/hide columns untuk semua role berizin dan label diterjemahkan menjadi bahasa manusia.

Kegagalan permanen menampilkan warning kecil pada Sage Number tanpa membuat tabel ramai. Detail lengkap tersedia pada Integration Monitor.

## 15. SMTP Settings dan Email Log

Semua email dikirim live ke alamat yang tersimpan. Tidak ada mode demo/live/disabled.

SMTP Settings berada di Settings dan memuat host, port, encryption, username, secret, from name/address, reply-to, public application URL, Save, Test Connection, dan Send Test Email.

Secret tidak pernah ditampilkan kembali atau dicatat pada log. Empty password update mempertahankan secret sebelumnya.

Email Log memakai compact table: time, type, recipient, reference, status, attempts; optional columns mencakup subject, last attempt, safe error, SMTP message ID, creator, dan timestamps. Resend membuat notification job baru dan mempertahankan history.

`notification_outbox` terpisah dari Sage integration outbox. Ia menangani approval email, supplier email, dan document generation workflow.

## 16. Dashboard dan Notifications

Global dashboard filters: period, warehouse, dan supplier.

Compact summaries:

- Pending Approval
- Expected Today
- Received Today
- Outstanding Kanban
- Low Stock Materials
- Interface Attention

Visual utama:

- Ordered vs Received Kanban
- Expected vs Actual Receiving
- PO Status horizontal distribution
- Supplier Delivery Performance
- Stock by Material
- Low Stock/Attention table
- Recent Activity

Jangan menjumlahkan unit berbeda dalam satu quantity chart. Dashboard mengikuti RBAC dan setiap visual dapat melakukan drill-down ke filtered module list.

Notification Center mencakup approval required, PO approved, supplier email failed, receiving completed, stock adjustment, dan interface attention. Polling ringan cukup untuk prototype.

## 17. Reports dan Export

Report utama: Purchase Orders, Receiving, Inventory, Supplier Performance, dan Stock Audit.

Semua report yang dapat ditelusuri ke supplier mendukung searchable Supplier filter. Filter supplier ikut diterapkan pada chart, drill-down, PDF, Excel, dan CSV export.

Export dapat mengikuti current visible columns atau seluruh filtered results sesuai permission. User tanpa price permission tidak dapat mengekspor price. Audit log mencatat export. CSV/Excel output harus mencegah formula injection.

## 18. Documents dan Storage

Dokumen tersedia sebagai tab pada transaction detail. Metadata disimpan pada PostgreSQL; file disimpan melalui storage abstraction. Documents memiliki type, reference, version, size, generation timestamp/actor, dan checksum.

Dokumen transaksi tidak ditimpa. Regeneration membuat version baru, kecuali label Kanban identitas yang harus mempertahankan Kanban ID yang sama.

## 19. Global Audit dan Data Table Rules

Semua business records memiliki:

```text
created_by_user_id, created_at, updated_by_user_id, updated_at
```

Tambahkan actor khusus sesuai proses: submitted, approved, issued, completed, cancelled. Automated actions memakai actor `System`.

Semua index views menggunakan compact table dan `Columns` control. Preferences disimpan per user. Long text memakai ellipsis dan tooltip. Detail dapat dibuka melalui row click, compact drawer, atau focused detail page.

## 20. Edge Cases dan Invariants

- Kanban bukan base unit.
- PO quantity selalu integer Kanban dan base quantity merupakan hasil perkalian snapshot.
- Satu PO hanya memiliki satu supplier.
- PO material harus dimiliki supplier PO.
- Approved commercial data immutable.
- Satu approved PO menghasilkan tepat satu DN dalam prototype.
- Setiap Kanban ID hanya menjadi anggota satu DN entry dan hanya dapat diterima sekali.
- Satu DN boleh memiliki banyak partial receiving, tetapi hanya satu ACTIVE session.
- Receiving lokal dan PDF tidak menunggu Sage.
- Supplier email tidak menunggu Sage.
- Sage retry tidak boleh membuat duplicate financial documents.
- Email scanner tidak boleh dapat approve melalui HTTP GET.
- Inventory balance tidak boleh diedit langsung.
- Master record yang sudah direferensikan dinonaktifkan, bukan dihapus.
- Semua financial and quantity arithmetic memakai decimal.

## 21. Prototype Scope Exclusions

- Multiple tenant onboarding UI.
- Entra/SSO.
- Multi-stage atau threshold approval.
- Manual multi-DN planning atau satu PO line dibagi ke beberapa DN.
- Partial quantity inside one Kanban.
- Sage push untuk outgoing production consumption.
- Advanced report builder.
- Native mobile application.
- Automatic calendar/workday tolerance for supplier KPI.

## 22. Acceptance Outcomes

Prototype dianggap memenuhi desain ketika user dapat:

1. Login dengan username/password dan menerima menu sesuai permission.
2. Mengelola master supplier, material, unit, warehouse, dan location.
3. Membuat satu-supplier PO dengan searchable dependent material picker.
4. Menyimpan draft atau langsung mengirim approval.
5. Approve melalui email tanpa login atau melalui Approval Inbox.
6. Secara otomatis menghasilkan PO, satu DN, seluruh Kanban, dan tiga PDF.
7. Mengirim tiga PDF ke supplier dan melihat Email Log.
8. Membuat receiving form, menjalankan exclusive scan session, dan partial receive.
9. Langsung mendapatkan receiving PDF dengan outstanding snapshot.
10. Melihat PO berubah menjadi partially atau fully received tanpa menunggu Sage.
11. Menjalankan outgoing, movement, adjustment, dan stock taking dengan ledger audit.
12. Melihat Sage document numbers, optional integration detail, dashboard, reports, supplier filters, dan exports.
