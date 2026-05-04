-- +goose Up

-- Full-text search support untuk surat. Column `search_doc` adalah tsvector
-- yang di-update saat metadata surat berubah dan saat lampiran PDF di-upload.
-- Backfill awal hanya dari perihal + nomor_surat — extract teks PDF di-trigger
-- di handler upload (lihat internal/server/attachment.go).
--
-- Pakai text search config 'simple' (tokenize tanpa stemming): cocok untuk
-- Bahasa Indonesia karena PostgreSQL tidak punya 'indonesian' config built-in.
-- Stemming Bahasa Indonesia bisa di-tambahkan via extension snowball Fase
-- berikutnya kalau dibutuhkan.

ALTER TABLE surat ADD COLUMN IF NOT EXISTS search_doc tsvector;

-- GIN index untuk fast full-text lookup. Partial WHERE NOT is_deleted
-- supaya index size tetap kecil (soft-deleted rows tidak ikut).
CREATE INDEX IF NOT EXISTS idx_surat_search_doc ON surat USING GIN(search_doc)
    WHERE NOT is_deleted;

-- Backfill: perihal + nomor_surat sebagai search content awal.
UPDATE surat
SET search_doc = to_tsvector('simple',
    coalesce(perihal, '') || ' ' || coalesce(nomor_surat, ''))
WHERE search_doc IS NULL;

-- Trigger BEFORE INSERT untuk auto-set search_doc dari metadata.
-- Kenapa hanya INSERT bukan UPDATE: UPDATE bisa terjadi karena re-extract
-- attachment text yang sudah include perihal/nomor + text. Trigger UPDATE
-- akan overwrite extracted text dengan metadata-only — regression search.
-- Update path metadata-only di-handle handler dengan rebuildSearchDoc.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION init_surat_search_doc() RETURNS trigger AS $$
BEGIN
    IF NEW.search_doc IS NULL THEN
        NEW.search_doc := to_tsvector('simple',
            coalesce(NEW.perihal, '') || ' ' || coalesce(NEW.nomor_surat, ''));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_init_surat_search_doc ON surat;
CREATE TRIGGER trg_init_surat_search_doc
    BEFORE INSERT ON surat
    FOR EACH ROW
    EXECUTE FUNCTION init_surat_search_doc();


-- +goose Down

DROP TRIGGER IF EXISTS trg_init_surat_search_doc ON surat;
DROP FUNCTION IF EXISTS init_surat_search_doc();
DROP INDEX IF EXISTS idx_surat_search_doc;
ALTER TABLE surat DROP COLUMN IF EXISTS search_doc;
