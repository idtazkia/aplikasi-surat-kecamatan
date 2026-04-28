-- +goose Up

-- 3 kasus dedup conflict — surat masuk yang di-input 2 kali oleh staf berbeda saat offline.
-- Saat sync online, server deteksi duplikat lewat composite key (instansi_id + nomor_surat + tanggal_terima).
-- Kedua record disimpan dan masuk reconciliation_queue dengan group_id sama.
--
-- ID konvensi:
--   surat duplikat: prefix 0007 dengan suffix -dup
--   reconciliation_queue: prefix 000d

-- Case 1: Edaran Kemendagri — staf1 dan staf2 masing-masing input
INSERT INTO surat (id, jenis, nomor_surat, perihal, tanggal_surat, tanggal_terima,
                   instansi_id, klasifikasi_id, sifat_id, access_level, created_by, created_at) VALUES
('00000000-0000-0000-0007-0000000000d1', 'masuk',
 'B-1234/SETJEN/IV/2026',
 'Edaran Penyederhanaan Pelayanan Publik',
 '2026-04-15', '2026-04-17',
 '00000000-0000-0000-0005-000000000001', -- Kemendagri
 '00000000-0000-0000-0003-000000000002',
 '00000000-0000-0000-0004-000000000003',
 'public',
 '00000000-0000-0000-0006-000000000001', -- staf1 yang input pertama
 '2026-04-17 09:30:00+07'::timestamptz),

('00000000-0000-0000-0007-0000000000d2', 'masuk',
 'B-1234/SETJEN/IV/2026',                                    -- nomor sama
 'Edaran Penyederhanaan Pelayanan Publik (versi staf2)',     -- perihal sedikit beda
 '2026-04-15', '2026-04-17',                                 -- tanggal sama
 '00000000-0000-0000-0005-000000000001',                     -- instansi sama
 '00000000-0000-0000-0003-000000000002',
 '00000000-0000-0000-0004-000000000003',
 'public',
 '00000000-0000-0000-0006-000000000005', -- staf2 input duplikat (offline, tidak tahu staf1 sudah input)
 '2026-04-17 14:15:00+07'::timestamptz),

-- Case 2: Undangan Pemprov Jabar
('00000000-0000-0000-0007-0000000000d3', 'masuk',
 '005/678/III/2026',
 'Undangan Rakor Implementasi Smart City',
 '2026-03-28', '2026-04-01',
 '00000000-0000-0000-0005-000000000002', -- Pemprov Jabar
 '00000000-0000-0000-0003-000000000002',
 '00000000-0000-0000-0004-000000000002',
 'public',
 '00000000-0000-0000-0006-000000000001',
 '2026-04-01 10:00:00+07'::timestamptz),

('00000000-0000-0000-0007-0000000000d4', 'masuk',
 '005/678/III/2026',
 'Undangan Rakor Smart City Jabar',                          -- perihal singkat
 '2026-03-28', '2026-04-01',
 '00000000-0000-0000-0005-000000000002',
 '00000000-0000-0000-0003-000000000002',
 '00000000-0000-0000-0004-000000000002',
 'public',
 '00000000-0000-0000-0006-000000000005',
 '2026-04-01 11:30:00+07'::timestamptz),

-- Case 3: Pemberitahuan PLN
('00000000-0000-0000-0007-0000000000d5', 'masuk',
 'PLN-456/CBN/IV/2026',
 'Pemberitahuan Pemadaman Listrik Berkala',
 '2026-04-08', '2026-04-10',
 '00000000-0000-0000-0005-000000000010', -- PLN Cibinong
 '00000000-0000-0000-0003-000000000007', -- PU
 '00000000-0000-0000-0004-000000000002',
 'public',
 '00000000-0000-0000-0006-000000000005',
 '2026-04-10 08:00:00+07'::timestamptz),

('00000000-0000-0000-0007-0000000000d6', 'masuk',
 'PLN-456/CBN/IV/2026',
 'Pemberitahuan Pemadaman PLN',
 '2026-04-08', '2026-04-10',
 '00000000-0000-0000-0005-000000000010',
 '00000000-0000-0000-0003-000000000007',
 '00000000-0000-0000-0004-000000000002',
 'public',
 '00000000-0000-0000-0006-000000000001',
 '2026-04-10 09:15:00+07'::timestamptz)

ON CONFLICT (id) DO NOTHING;

-- Reconciliation queue entries: link kedua surat duplikat per group
-- group_id = UUID arbitrary tapi konsisten antar kedua entry di group yang sama
INSERT INTO reconciliation_queue (id, group_id, surat_id, dedup_key, status) VALUES
    -- Group 1: Edaran Kemendagri
    ('00000000-0000-0000-000d-000000000001',
     '00000000-0000-0000-000e-000000000001',
     '00000000-0000-0000-0007-0000000000d1',
     '00000000-0000-0000-0005-000000000001|B-1234/SETJEN/IV/2026|2026-04-17',
     'pending'),
    ('00000000-0000-0000-000d-000000000002',
     '00000000-0000-0000-000e-000000000001',
     '00000000-0000-0000-0007-0000000000d2',
     '00000000-0000-0000-0005-000000000001|B-1234/SETJEN/IV/2026|2026-04-17',
     'pending'),

    -- Group 2: Undangan Pemprov Jabar
    ('00000000-0000-0000-000d-000000000003',
     '00000000-0000-0000-000e-000000000002',
     '00000000-0000-0000-0007-0000000000d3',
     '00000000-0000-0000-0005-000000000002|005/678/III/2026|2026-04-01',
     'pending'),
    ('00000000-0000-0000-000d-000000000004',
     '00000000-0000-0000-000e-000000000002',
     '00000000-0000-0000-0007-0000000000d4',
     '00000000-0000-0000-0005-000000000002|005/678/III/2026|2026-04-01',
     'pending'),

    -- Group 3: PLN
    ('00000000-0000-0000-000d-000000000005',
     '00000000-0000-0000-000e-000000000003',
     '00000000-0000-0000-0007-0000000000d5',
     '00000000-0000-0000-0005-000000000010|PLN-456/CBN/IV/2026|2026-04-10',
     'pending'),
    ('00000000-0000-0000-000d-000000000006',
     '00000000-0000-0000-000e-000000000003',
     '00000000-0000-0000-0007-0000000000d6',
     '00000000-0000-0000-0005-000000000010|PLN-456/CBN/IV/2026|2026-04-10',
     'pending')
ON CONFLICT (id) DO NOTHING;


-- +goose Down
DELETE FROM reconciliation_queue WHERE id LIKE '00000000-0000-0000-000d-%';
DELETE FROM surat WHERE id IN (
    '00000000-0000-0000-0007-0000000000d1',
    '00000000-0000-0000-0007-0000000000d2',
    '00000000-0000-0000-0007-0000000000d3',
    '00000000-0000-0000-0007-0000000000d4',
    '00000000-0000-0000-0007-0000000000d5',
    '00000000-0000-0000-0007-0000000000d6'
);
