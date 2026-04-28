-- +goose Up

-- 5 thread referensi mendemonstrasikan tipe relasi:
--   1. Balasan: surat masuk → surat keluar yang membalas
--   2. Lanjutan: surat masuk → surat masuk berikutnya (di-input nanti, atau external_ref)
--   3. Disposisi hasil: surat masuk → surat keluar hasil tindak lanjut
--   4. Revisi: surat keluar lama → surat keluar revisi (supersedes)
--   5. External reference: surat masuk merujuk korespondensi lama yang tidak ter-input

INSERT INTO surat_references (id, from_surat_id, to_surat_id, relationship, external_ref, note, created_by) VALUES

-- 1. Balasan: surat keluar 0008-001 membalas surat masuk 0007-002
('00000000-0000-0000-000a-000000000001',
 '00000000-0000-0000-0008-000000000001', -- Tanggapan Edaran Pandemi
 '00000000-0000-0000-0007-000000000002', -- Edaran Pandemi
 'balasan',
 NULL,
 'Tanggapan resmi atas edaran Kemenkes',
 '00000000-0000-0000-0006-000000000001'),

-- 1b. Konfirmasi kehadiran membalas undangan rakor
('00000000-0000-0000-000a-000000000002',
 '00000000-0000-0000-0008-000000000002', -- Konfirmasi Kehadiran
 '00000000-0000-0000-0007-000000000001', -- Undangan Rakor
 'balasan',
 NULL,
 NULL,
 '00000000-0000-0000-0006-000000000002'),

-- 1c. Penyampaian data membalas permohonan BPS
('00000000-0000-0000-000a-000000000003',
 '00000000-0000-0000-0008-000000000003', -- Penyampaian data
 '00000000-0000-0000-0007-000000000003', -- Permohonan data BPS
 'balasan',
 NULL,
 'Lampiran data dalam format Excel',
 '00000000-0000-0000-0006-000000000005'),

-- 3. Disposisi hasil: Surket Tazkia (keluar) hasil dari Permohonan Surket (masuk)
('00000000-0000-0000-000a-000000000004',
 '00000000-0000-0000-0008-000000000004', -- Surket Tazkia
 '00000000-0000-0000-0007-000000000008', -- Permohonan Surket
 'disposisi_hasil',
 NULL,
 'Hasil tindak lanjut disposisi camat ke staf',
 '00000000-0000-0000-0006-000000000001'),

-- 3b. Undangan rapat pengamanan hari raya hasil disposisi koord polsek
('00000000-0000-0000-000a-000000000005',
 '00000000-0000-0000-0008-000000000006', -- Undangan rapat pengamanan
 '00000000-0000-0000-0007-00000000000b', -- Koordinasi Pengamanan
 'disposisi_hasil',
 NULL,
 NULL,
 '00000000-0000-0000-0006-000000000002'),

-- 3c. Undangan audiensi karang taruna
('00000000-0000-0000-000a-000000000006',
 '00000000-0000-0000-0008-000000000007', -- Undangan audiensi
 '00000000-0000-0000-0007-00000000000c', -- Permohonan audiensi
 'disposisi_hasil',
 NULL,
 NULL,
 '00000000-0000-0000-0006-000000000002'),

-- 4. Revisi: undangan audiensi versi revisi menggantikan versi awal
('00000000-0000-0000-000a-000000000007',
 '00000000-0000-0000-0008-000000000008', -- Revisi undangan
 '00000000-0000-0000-0008-000000000007', -- Undangan awal
 'revisi',
 NULL,
 'Perubahan jadwal karena bentrok agenda lain',
 '00000000-0000-0000-0006-000000000002'),

-- 5. External reference: laporan TA 2025 merujuk audit periode sebelumnya
('00000000-0000-0000-000a-000000000008',
 '00000000-0000-0000-0008-000000000005', -- Laporan tindak lanjut audit
 NULL,
 'lanjutan',
 'B-1500/IT/VI/2025 - Surat Audit Internal TA 2024 (arsip fisik)',
 'Audit sebelumnya belum di-digitalisasi, hanya dirujuk metadata',
 '00000000-0000-0000-0006-000000000002'),

-- 5b. Edaran pandemi merujuk edaran sebelumnya
('00000000-0000-0000-000a-000000000009',
 '00000000-0000-0000-0007-000000000002',
 NULL,
 'lanjutan',
 'KK.01/01/2024 tanggal 15 November 2024 (edaran pandemi awal)',
 NULL,
 '00000000-0000-0000-0006-000000000001')

ON CONFLICT (id) DO NOTHING;


-- +goose Down
DELETE FROM surat_references WHERE id LIKE '00000000-0000-0000-000a-%';
