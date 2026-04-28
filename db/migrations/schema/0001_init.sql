-- +goose Up

-- =====================================================================
-- Reference data: roles, permissions, klasifikasi, sifat, instansi
-- =====================================================================

CREATE TABLE roles (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE klasifikasi (
    id UUID PRIMARY KEY,
    kode TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    deskripsi TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE sifat (
    id UUID PRIMARY KEY,
    kode TEXT NOT NULL UNIQUE,
    nama TEXT NOT NULL,
    prioritas INT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE instansi (
    id UUID PRIMARY KEY,
    nama_kanonik TEXT NOT NULL UNIQUE,
    aliases TEXT[] NOT NULL DEFAULT '{}',
    alamat TEXT,
    kontak TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_instansi_nama_kanonik ON instansi (nama_kanonik) WHERE is_active;
CREATE INDEX idx_instansi_aliases ON instansi USING GIN (aliases);

-- =====================================================================
-- Users + role assignments
-- =====================================================================

CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

-- =====================================================================
-- Surat — entitas utama
-- =====================================================================

CREATE TABLE surat (
    id UUID PRIMARY KEY,
    jenis TEXT NOT NULL CHECK (jenis IN ('masuk', 'keluar')),
    nomor_surat TEXT NOT NULL,
    perihal TEXT NOT NULL,
    tanggal_surat DATE NOT NULL,
    tanggal_terima DATE,
    instansi_id UUID NOT NULL REFERENCES instansi(id),
    klasifikasi_id UUID REFERENCES klasifikasi(id),
    sifat_id UUID REFERENCES sifat(id),
    access_level TEXT NOT NULL DEFAULT 'public'
        CHECK (access_level IN ('public', 'restricted', 'secret')),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMPTZ,
    deleted_by UUID REFERENCES users(id),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (jenis = 'masuk' AND tanggal_terima IS NOT NULL) OR
        (jenis = 'keluar' AND tanggal_terima IS NULL)
    ),
    CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL AND deleted_by IS NULL) OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL AND deleted_by IS NOT NULL)
    )
);

-- concept:btree-partial-index-soft-delete:start
CREATE INDEX idx_surat_tanggal_terima ON surat (tanggal_terima)
    WHERE NOT is_deleted AND jenis = 'masuk';
CREATE INDEX idx_surat_tanggal_surat ON surat (tanggal_surat)
    WHERE NOT is_deleted;
CREATE INDEX idx_surat_instansi ON surat (instansi_id)
    WHERE NOT is_deleted;
CREATE INDEX idx_surat_jenis ON surat (jenis)
    WHERE NOT is_deleted;
-- concept:btree-partial-index-soft-delete:end

-- Unique constraint surat keluar (kunci dedup global)
CREATE UNIQUE INDEX idx_surat_keluar_nomor ON surat (nomor_surat)
    WHERE jenis = 'keluar' AND NOT is_deleted;

-- Surat masuk dedup key (normalized_sender + nomor + tanggal_terima)
-- Tidak unique enforce di DB — allow merge-on-sync, app layer handle deteksi
CREATE INDEX idx_surat_masuk_dedup ON surat (instansi_id, nomor_surat, tanggal_terima)
    WHERE jenis = 'masuk' AND NOT is_deleted;

-- =====================================================================
-- Surat: lampiran, tembusan, references, ACL
-- =====================================================================

-- concept:linked-list-version-chain:start
-- Linked list singly-linked: setiap row punya pointer `replaced_by` ke node
-- berikutnya (versi pengganti). Versi paling baru = node dengan replaced_by NULL
-- (tail). Traversal balik dari tail = walk linked list.
-- Trade-off vs separate versions table: kompak (1 table), tapi query "current
-- version of attachment X" butuh filter is_active=true (denormalisasi cache).
CREATE TABLE surat_attachments (
    id UUID PRIMARY KEY,
    surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('primary', 'lampiran')),
    file_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL CHECK (file_size > 0),
    mime_type TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    replaced_by UUID REFERENCES surat_attachments(id),  -- next pointer di linked list
    uploaded_by UUID NOT NULL REFERENCES users(id),
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- concept:linked-list-version-chain:end

CREATE INDEX idx_attachments_surat_active ON surat_attachments (surat_id) WHERE is_active;
CREATE INDEX idx_attachments_replaced ON surat_attachments (replaced_by) WHERE replaced_by IS NOT NULL;

CREATE TABLE surat_tembusan (
    id UUID PRIMARY KEY,
    surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    instansi_id UUID REFERENCES instansi(id),
    external_text TEXT,
    urutan INT NOT NULL,
    CHECK (instansi_id IS NOT NULL OR external_text IS NOT NULL)
);

CREATE INDEX idx_tembusan_surat ON surat_tembusan (surat_id, urutan);

CREATE TABLE surat_references (
    id UUID PRIMARY KEY,
    from_surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    to_surat_id UUID REFERENCES surat(id) ON DELETE SET NULL,
    relationship TEXT NOT NULL
        CHECK (relationship IN ('balasan', 'lanjutan', 'disposisi_hasil', 'revisi', 'terkait')),
    external_ref TEXT,
    note TEXT,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (to_surat_id IS NOT NULL OR external_ref IS NOT NULL)
);

CREATE INDEX idx_references_from ON surat_references (from_surat_id);
CREATE INDEX idx_references_to ON surat_references (to_surat_id) WHERE to_surat_id IS NOT NULL;
CREATE INDEX idx_references_external ON surat_references (external_ref) WHERE external_ref IS NOT NULL;

CREATE TABLE surat_acl (
    surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (surat_id, user_id)
);

-- =====================================================================
-- Workflow: disposisi, komentar
-- =====================================================================

CREATE TABLE disposisi (
    id UUID PRIMARY KEY,
    surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    assigned_to UUID NOT NULL REFERENCES users(id),
    nomor_disposisi TEXT,
    instruksi TEXT NOT NULL,
    deadline TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'in_progress', 'done', 'cancelled')),
    completed_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (status = 'done' AND completed_at IS NOT NULL) OR
        (status <> 'done')
    )
);

CREATE INDEX idx_disposisi_assigned ON disposisi (assigned_to, status);
CREATE INDEX idx_disposisi_surat ON disposisi (surat_id);
CREATE INDEX idx_disposisi_overdue ON disposisi (deadline)
    WHERE status IN ('pending', 'in_progress') AND deadline IS NOT NULL;

-- concept:append-only-immutability:start
-- Komentar append-only: tidak ada updated_at, tidak ada is_deleted.
-- Kalau salah ketik, append komentar koreksi baru. Audit by construction.
CREATE TABLE komentar (
    id UUID PRIMARY KEY,
    surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    body TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_komentar_surat ON komentar (surat_id, created_at);
-- concept:append-only-immutability:end

-- =====================================================================
-- Audit: write log + read log (terpisah untuk volume management)
-- =====================================================================

CREATE TABLE audit_log (
    id UUID PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete', 'restore')),
    actor_user_id UUID REFERENCES users(id),
    before_jsonb JSONB,
    after_jsonb JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_entity ON audit_log (entity_type, entity_id, created_at DESC);
CREATE INDEX idx_audit_actor ON audit_log (actor_user_id, created_at DESC);

CREATE TABLE read_audit_log (
    id UUID PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    actor_user_id UUID REFERENCES users(id),
    action TEXT NOT NULL CHECK (action IN ('view', 'download')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_read_audit_entity ON read_audit_log (entity_type, entity_id, created_at DESC);
CREATE INDEX idx_read_audit_actor ON read_audit_log (actor_user_id, created_at DESC);

-- =====================================================================
-- Sync: operation_log (idempotency by client_op_id) + reconciliation queue
-- =====================================================================

-- concept:operation-log-idempotency:start
-- client_op_id sebagai PK = idempotency by construction.
-- Kalau client retry op dengan ID yang sama, INSERT ... ON CONFLICT DO NOTHING.
-- Server pasti tidak apply ganda.
CREATE TABLE operation_log (
    client_op_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('create', 'update', 'delete', 'append')),
    field_changes JSONB,
    client_timestamp TIMESTAMPTZ NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oplog_user ON operation_log (user_id, applied_at DESC);
CREATE INDEX idx_oplog_entity ON operation_log (entity_type, entity_id);
-- concept:operation-log-idempotency:end

CREATE TABLE reconciliation_queue (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL,
    surat_id UUID NOT NULL REFERENCES surat(id) ON DELETE CASCADE,
    dedup_key TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'merged', 'kept_both')),
    resolved_by UUID REFERENCES users(id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recon_group ON reconciliation_queue (group_id);
CREATE INDEX idx_recon_pending ON reconciliation_queue (created_at)
    WHERE status = 'pending';

-- =====================================================================
-- Notifikasi (in-app, polling-based; push di Fase 7)
-- =====================================================================

-- concept:queue-fifo-natural-order:start
-- Queue FIFO per-user: notifikasi diterima dalam urutan masuk (chronological)
-- dan biasanya di-consume berurutan oleh client (mark-as-read sequential).
-- UUIDv7 sebagai PK = time-ordered → ORDER BY id ekuivalen ORDER BY created_at,
-- dan B-Tree index pada (user_id, created_at DESC) memberi enqueue O(log n)
-- + dequeue (peek + mark read) O(log n).
-- Bukan strict FIFO karena consumer bisa skip baca, tapi semantik queue cukup
-- untuk kebutuhan notifikasi (vs strict order untuk mis. message broker).
CREATE TABLE notifications (
    id UUID PRIMARY KEY,                                 -- UUIDv7 = time-ordered
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    payload_jsonb JSONB NOT NULL,
    read_at TIMESTAMPTZ,                                  -- NULL = unread (belum di-dequeue)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Partial index: query "unread queue" sangat cepat — hanya scan tail yang belum diconsume.
CREATE INDEX idx_notif_user_unread ON notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;
CREATE INDEX idx_notif_user_all ON notifications (user_id, created_at DESC);
-- concept:queue-fifo-natural-order:end


-- +goose Down

DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS reconciliation_queue;
DROP TABLE IF EXISTS operation_log;
DROP TABLE IF EXISTS read_audit_log;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS komentar;
DROP TABLE IF EXISTS disposisi;
DROP TABLE IF EXISTS surat_acl;
DROP TABLE IF EXISTS surat_references;
DROP TABLE IF EXISTS surat_tembusan;
DROP TABLE IF EXISTS surat_attachments;
DROP TABLE IF EXISTS surat;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS instansi;
DROP TABLE IF EXISTS sifat;
DROP TABLE IF EXISTS klasifikasi;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
