-- +goose Up

-- Roles
INSERT INTO roles (id, code, name, description) VALUES
    ('00000000-0000-0000-0001-000000000001', 'staf',    'Staf Kecamatan',  'Data entry surat masuk/keluar dan rekonsiliasi duplikat'),
    ('00000000-0000-0000-0001-000000000002', 'camat',   'Camat',           'Supervisi, review, disposisi, override'),
    ('00000000-0000-0000-0001-000000000003', 'admin',   'Administrator',   'Konfigurasi sistem, manajemen user, master data'),
    ('00000000-0000-0000-0001-000000000004', 'student', 'Mahasiswa Demo',  'Read-only ke instance demo untuk edukasi')
ON CONFLICT (id) DO NOTHING;

-- Permissions (Fase 0: minimal set, akan ditambah per fase fitur)
INSERT INTO permissions (id, code, description) VALUES
    ('00000000-0000-0000-0002-000000000001', 'surat:read',         'Lihat daftar dan detail surat'),
    ('00000000-0000-0000-0002-000000000002', 'surat:create',       'Input surat baru'),
    ('00000000-0000-0000-0002-000000000003', 'surat:update',       'Edit metadata surat'),
    ('00000000-0000-0000-0002-000000000004', 'surat:delete',       'Soft delete surat'),
    ('00000000-0000-0000-0002-000000000005', 'surat:restore',      'Restore surat yang ter-soft-delete'),
    ('00000000-0000-0000-0002-000000000006', 'surat:read_secret',  'Akses surat dengan access_level=secret'),
    ('00000000-0000-0000-0002-000000000007', 'disposisi:create',   'Buat disposisi'),
    ('00000000-0000-0000-0002-000000000008', 'disposisi:update',   'Update status disposisi'),
    ('00000000-0000-0000-0002-000000000009', 'komentar:append',    'Tambah komentar di surat'),
    ('00000000-0000-0000-0002-000000000010', 'rekonsiliasi:resolve', 'Resolve antrian rekonsiliasi duplikat'),
    ('00000000-0000-0000-0002-000000000011', 'instansi:manage',    'CRUD direktori instansi'),
    ('00000000-0000-0000-0002-000000000012', 'klasifikasi:manage', 'CRUD klasifikasi & sifat'),
    ('00000000-0000-0000-0002-000000000013', 'user:manage',        'CRUD users & role assignment'),
    ('00000000-0000-0000-0002-000000000014', 'audit:read',         'Akses audit log dan read audit log')
ON CONFLICT (id) DO NOTHING;

-- Role-permission mapping (Fase 0: staf, camat, admin sama-sama dapat permission inti)
-- Student: read-only saja. Akan diferensiasi di Fase 2.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code IN ('staf', 'camat', 'admin')
  AND p.code IN (
    'surat:read', 'surat:create', 'surat:update', 'surat:delete', 'surat:restore',
    'disposisi:create', 'disposisi:update',
    'komentar:append',
    'rekonsiliasi:resolve'
  )
ON CONFLICT DO NOTHING;

-- Permissi tambahan camat: read_secret, audit:read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'camat'
  AND p.code IN ('surat:read_secret', 'audit:read')
ON CONFLICT DO NOTHING;

-- Permissi admin only: instansi:manage, klasifikasi:manage, user:manage, audit:read
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'admin'
  AND p.code IN ('instansi:manage', 'klasifikasi:manage', 'user:manage', 'audit:read', 'surat:read_secret')
ON CONFLICT DO NOTHING;

-- Student: read-only
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'student'
  AND p.code = 'surat:read'
ON CONFLICT DO NOTHING;


-- +goose Down
DELETE FROM role_permissions WHERE role_id IN (
    SELECT id FROM roles WHERE code IN ('staf', 'camat', 'admin', 'student')
);
DELETE FROM permissions WHERE id LIKE '00000000-0000-0000-0002-%';
DELETE FROM roles WHERE id LIKE '00000000-0000-0000-0001-%';
