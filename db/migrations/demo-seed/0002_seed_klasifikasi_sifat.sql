-- +goose Up

-- Klasifikasi (kode pola tata persuratan, contoh — sesuaikan dengan kantor)
INSERT INTO klasifikasi (id, kode, nama, deskripsi) VALUES
    ('00000000-0000-0000-0003-000000000001', '000', 'Umum',                    'Surat yang tidak masuk klasifikasi spesifik'),
    ('00000000-0000-0000-0003-000000000002', '100', 'Pemerintahan',            'Urusan pemerintahan umum'),
    ('00000000-0000-0000-0003-000000000003', '200', 'Politik',                 'Urusan politik dan kemasyarakatan'),
    ('00000000-0000-0000-0003-000000000004', '300', 'Keamanan',                'Keamanan dan ketertiban'),
    ('00000000-0000-0000-0003-000000000005', '400', 'Kesejahteraan Rakyat',    'Pendidikan, kesehatan, sosial'),
    ('00000000-0000-0000-0003-000000000006', '500', 'Perekonomian',            'Pembangunan ekonomi dan pertanian'),
    ('00000000-0000-0000-0003-000000000007', '600', 'Pekerjaan Umum',          'Infrastruktur, jalan, drainase'),
    ('00000000-0000-0000-0003-000000000008', '700', 'Pengawasan',              'Pengawasan internal'),
    ('00000000-0000-0000-0003-000000000009', '800', 'Kepegawaian',             'Urusan pegawai negeri'),
    ('00000000-0000-0000-0003-00000000000a', '900', 'Keuangan',                'Anggaran, belanja, pelaporan keuangan')
ON CONFLICT (id) DO NOTHING;

-- Sifat
INSERT INTO sifat (id, kode, nama, prioritas) VALUES
    ('00000000-0000-0000-0004-000000000001', 'biasa',    'Biasa',    1),
    ('00000000-0000-0000-0004-000000000002', 'segera',   'Segera',   3),
    ('00000000-0000-0000-0004-000000000003', 'penting',  'Penting',  2),
    ('00000000-0000-0000-0004-000000000004', 'rahasia',  'Rahasia',  4)
ON CONFLICT (id) DO NOTHING;


-- +goose Down
DELETE FROM sifat WHERE id LIKE '00000000-0000-0000-0004-%';
DELETE FROM klasifikasi WHERE id LIKE '00000000-0000-0000-0003-%';
