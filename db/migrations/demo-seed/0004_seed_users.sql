-- +goose Up
--
-- DEV ONLY CREDENTIALS
-- ---------------------
-- Semua user demo punya password: "demo123"
-- Hash: bcrypt cost=10
-- JANGAN pakai password ini di production. Production punya schema saja
-- (folder demo-seed tidak di-apply di production deployment).

INSERT INTO users (id, username, full_name, email, password_hash, is_active) VALUES
    ('00000000-0000-0000-0006-000000000001',
     'staf1',
     'Siti Aminah',
     'staf1@demo.local',
     '$2a$10$Cu76ynYLMRF4/4b.ZuPrd.gGO92oEIF4/RUBvUzLWDXIhiFwRo486',
     TRUE),
    ('00000000-0000-0000-0006-000000000002',
     'camat',
     'Bu Camat Demo',
     'camat@demo.local',
     '$2a$10$Cu76ynYLMRF4/4b.ZuPrd.gGO92oEIF4/RUBvUzLWDXIhiFwRo486',
     TRUE),
    ('00000000-0000-0000-0006-000000000003',
     'admin',
     'Administrator Demo',
     'admin@demo.local',
     '$2a$10$Cu76ynYLMRF4/4b.ZuPrd.gGO92oEIF4/RUBvUzLWDXIhiFwRo486',
     TRUE),
    ('00000000-0000-0000-0006-000000000004',
     'student',
     'Mahasiswa Demo',
     'student@demo.local',
     '$2a$10$Cu76ynYLMRF4/4b.ZuPrd.gGO92oEIF4/RUBvUzLWDXIhiFwRo486',
     TRUE),
    ('00000000-0000-0000-0006-000000000005',
     'staf2',
     'Budi Santoso',
     'staf2@demo.local',
     '$2a$10$Cu76ynYLMRF4/4b.ZuPrd.gGO92oEIF4/RUBvUzLWDXIhiFwRo486',
     TRUE)
ON CONFLICT (id) DO NOTHING;

-- Assign roles
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id
FROM users u, roles r
WHERE
    (u.username = 'staf1'   AND r.code = 'staf')   OR
    (u.username = 'staf2'   AND r.code = 'staf')   OR
    (u.username = 'camat'   AND r.code = 'camat')  OR
    (u.username = 'admin'   AND r.code = 'admin')  OR
    (u.username = 'student' AND r.code = 'student')
ON CONFLICT DO NOTHING;


-- +goose Down
DELETE FROM user_roles WHERE user_id LIKE '00000000-0000-0000-0006-%';
DELETE FROM users WHERE id LIKE '00000000-0000-0000-0006-%';
