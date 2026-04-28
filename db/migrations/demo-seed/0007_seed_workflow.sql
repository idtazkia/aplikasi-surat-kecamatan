-- +goose Up

-- Disposisi: 5 chain dengan variasi status
INSERT INTO disposisi (id, surat_id, assigned_to, instruksi, deadline, status,
                       completed_at, created_by, created_at) VALUES

-- 1. Permohonan surket Tazkia → assign ke staf1, sudah selesai
('00000000-0000-0000-000b-000000000001',
 '00000000-0000-0000-0007-000000000008', -- Permohonan Surket
 '00000000-0000-0000-0006-000000000001', -- staf1
 'Tolong proses surat keterangan domisili sesuai format standar. Cek kelengkapan SK Yayasan terlampir.',
 '2026-03-15 23:59:59+07'::timestamptz,
 'done',
 '2026-03-18 14:30:00+07'::timestamptz,
 '00000000-0000-0000-0006-000000000002', -- camat
 '2026-03-13 09:15:00+07'::timestamptz),

-- 2. Koordinasi pengamanan hari raya → assign ke camat sendiri (review), in_progress
('00000000-0000-0000-000b-000000000002',
 '00000000-0000-0000-0007-00000000000b', -- Koordinasi pengamanan
 '00000000-0000-0000-0006-000000000002', -- camat (self-disposisi)
 'Saya akan handle koordinasi langsung. Siapkan ruang rapat dan undangan ke Koramil + Lurah.',
 '2026-04-20 23:59:59+07'::timestamptz,
 'in_progress',
 NULL,
 '00000000-0000-0000-0006-000000000002',
 '2026-04-19 08:00:00+07'::timestamptz),

-- 3. Permohonan audiensi karang taruna → assign ke staf2, in_progress
('00000000-0000-0000-000b-000000000003',
 '00000000-0000-0000-0007-00000000000c',
 '00000000-0000-0000-0006-000000000005', -- staf2
 'Susun jadwal audiensi minggu depan, koordinasi dengan ketua Karang Taruna.',
 '2026-04-30 23:59:59+07'::timestamptz,
 'in_progress',
 NULL,
 '00000000-0000-0000-0006-000000000002',
 '2026-04-23 10:00:00+07'::timestamptz),

-- 4. Permohonan data BPS → assign ke staf2, sudah selesai
('00000000-0000-0000-000b-000000000004',
 '00000000-0000-0000-0007-000000000003',
 '00000000-0000-0000-0006-000000000005',
 'Kompilasi data dari laporan kelurahan Q1, kirim format CSV.',
 '2026-04-10 23:59:59+07'::timestamptz,
 'done',
 '2026-04-10 16:45:00+07'::timestamptz,
 '00000000-0000-0000-0006-000000000002',
 '2026-04-04 11:30:00+07'::timestamptz),

-- 5. Edaran pandemi → assign ke staf1, pending
('00000000-0000-0000-000b-000000000005',
 '00000000-0000-0000-0007-000000000002',
 '00000000-0000-0000-0006-000000000001',
 'Distribusikan ke kelurahan via grup WA. Buat ringkasan poin penting untuk diumumkan.',
 NULL, -- no deadline
 'pending',
 NULL,
 '00000000-0000-0000-0006-000000000002',
 '2026-03-23 08:00:00+07'::timestamptz)

ON CONFLICT (id) DO NOTHING;

-- Komentar (append-only) di beberapa surat
INSERT INTO komentar (id, surat_id, user_id, body, created_at) VALUES

('00000000-0000-0000-000c-000000000001',
 '00000000-0000-0000-0007-000000000008',
 '00000000-0000-0000-0006-000000000001',
 'Sudah cek kelengkapan dokumen, lengkap. Siap proses surat keterangan.',
 '2026-03-14 10:00:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000002',
 '00000000-0000-0000-0007-000000000008',
 '00000000-0000-0000-0006-000000000002',
 'Bagus. Lanjutkan, pakai format SK domisili tahun lalu sebagai template.',
 '2026-03-14 11:30:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000003',
 '00000000-0000-0000-0007-000000000008',
 '00000000-0000-0000-0006-000000000001',
 'Surat sudah ditandatangan dan diserahkan ke kurir Tazkia hari ini.',
 '2026-03-18 14:35:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000004',
 '00000000-0000-0000-0007-00000000000c',
 '00000000-0000-0000-0006-000000000005',
 'Sudah hubungi ketua Karang Taruna, jadwal alternatif: Senin 28 atau Selasa 29 April.',
 '2026-04-23 14:00:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000005',
 '00000000-0000-0000-0007-00000000000c',
 '00000000-0000-0000-0006-000000000002',
 'Pilih Selasa 29 April, jam 10:00. Saya yang pimpin audiensi.',
 '2026-04-23 15:30:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000006',
 '00000000-0000-0000-0007-00000000000b',
 '00000000-0000-0000-0006-000000000002',
 'Konfirmasi: ruang rapat aula sudah dibooking 21 April. Polsek dan Koramil sudah dihubungi.',
 '2026-04-19 11:00:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000007',
 '00000000-0000-0000-0007-000000000005',
 '00000000-0000-0000-0006-000000000002',
 'Akses ke surat ini terbatas (restricted). Hanya saya dan inspektorat yang lihat.',
 '2026-01-19 09:00:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000008',
 '00000000-0000-0000-0007-000000000003',
 '00000000-0000-0000-0006-000000000005',
 'Data Q1 sudah dikompilasi dari 12 kelurahan. Total entries: 4,250.',
 '2026-04-09 14:00:00+07'::timestamptz),

('00000000-0000-0000-000c-000000000009',
 '00000000-0000-0000-0007-000000000003',
 '00000000-0000-0000-0006-000000000002',
 'Verified. Lanjut kirim ke BPS via email resmi + arsip surat keluar.',
 '2026-04-10 09:30:00+07'::timestamptz)

ON CONFLICT (id) DO NOTHING;


-- +goose Down
DELETE FROM komentar WHERE id LIKE '00000000-0000-0000-000c-%';
DELETE FROM disposisi WHERE id LIKE '00000000-0000-0000-000b-%';
