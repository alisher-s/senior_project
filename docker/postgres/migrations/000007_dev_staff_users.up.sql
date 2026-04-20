-- Pre-created staff accounts for local Docker / CI (check-in and admin role management).
-- Password for both (bcrypt cost 12): DevStaffPass1!
-- Use only in non-production; rotate or omit this migration in real deployments.
INSERT INTO users (id, email, password_hash, role)
VALUES
  (
    '00000000-0000-4000-8000-000000000001',
    'staff.admin@nu.edu.kz',
    '$2a$12$TthCZdGe3Oq8/QOclNIzCei7z1U3ySZSAKVhvz1.ERiSb8Frkzoiq',
    'admin'
  ),
  (
    '00000000-0000-4000-8000-000000000002',
    'staff.organizer@nu.edu.kz',
    '$2a$12$TthCZdGe3Oq8/QOclNIzCei7z1U3ySZSAKVhvz1.ERiSb8Frkzoiq',
    'organizer'
  )
ON CONFLICT (email) DO NOTHING;

