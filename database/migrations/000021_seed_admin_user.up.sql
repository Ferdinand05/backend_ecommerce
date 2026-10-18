-- Admin awal untuk dev/test: admin@ecommerce.test / admin123.
-- ponytail: password dev di repo — ganti (atau buat admin via bootstrap env var) sebelum production.
INSERT INTO users (
    id,
    role_id,
    email,
    password_hash,
    first_name,
    status,
    email_verified_at
)
VALUES (
    '00000000-0000-0000-0000-0000000000fa',
    '00000000-0000-0000-0000-000000000002', -- admin, lihat 000020_seed_roles
    'admin@ecommerce.test',
    '$2a$10$36470jMxOoD8BIn6x75qvuHiCBs.PPk4544goOdrS3XIqez1AQTq.', -- admin123
    'Admin',
    'active',
    NOW()
)
ON CONFLICT DO NOTHING;
