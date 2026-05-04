-- +goose Up

-- Tambah role 'auditor' (read-only pihak ketiga: inspektorat, BPK, dll).
-- Permission: surat:read + audit:read. NOT punya surat:read_secret —
-- auditor harus minta akses ke camat untuk surat rahasia (di luar scope MVP).
INSERT INTO roles (id, code, name, description) VALUES
    ('00000000-0000-0000-0001-000000000005',
     'auditor',
     'Auditor / Inspektorat',
     'Akses read-only untuk audit eksternal — tidak bisa create/update/delete')
ON CONFLICT (id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'auditor'
  AND p.code IN ('surat:read', 'audit:read')
ON CONFLICT DO NOTHING;

-- Pindahkan rekonsiliasi:resolve dari staf ke camat saja.
-- Justifikasi: keputusan supervisor harus konsisten — kalau staf bisa
-- pilih kanonik, hilang chain-of-command. Camat sebagai supervisor
-- adalah keputusan akhir.
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code = 'staf')
  AND permission_id = (SELECT id FROM permissions WHERE code = 'rekonsiliasi:resolve');

-- Demo user 'auditor' / demo123
INSERT INTO users (id, username, full_name, email, password_hash, is_active) VALUES
    ('00000000-0000-0000-0006-000000000006',
     'auditor',
     'Inspektorat Demo',
     'auditor@demo.local',
     -- bcrypt hash 'demo123' — sama dengan user demo lain (lihat 0004_seed_users.sql)
     '$2a$10$Cu76ynYLMRF4/4b.ZuPrd.gGO92oEIF4/RUBvUzLWDXIhiFwRo486',
     TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u, roles r
WHERE u.username = 'auditor' AND r.code = 'auditor'
ON CONFLICT DO NOTHING;


-- +goose Down

DELETE FROM user_roles WHERE user_id = '00000000-0000-0000-0006-000000000006';
DELETE FROM users WHERE id = '00000000-0000-0000-0006-000000000006';
DELETE FROM role_permissions
WHERE role_id = (SELECT id FROM roles WHERE code = 'auditor');
DELETE FROM roles WHERE code = 'auditor';

-- Restore rekonsiliasi:resolve untuk staf (revert)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.code = 'staf' AND p.code = 'rekonsiliasi:resolve'
ON CONFLICT DO NOTHING;
