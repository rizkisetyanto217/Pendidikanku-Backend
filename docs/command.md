# Buat migrasi

muhammadrizkisetyanto@MacBook-Air-Muhammad arabiya-syari-fiber-1 % migrate create -ext sql -dir internals/databases/migrations create-table-user-profile
/Users/muhammadrizkisetyanto/Documents/arabiya-syari-fiber-1/internals/database/migrations/20250226092703_create-table-user-profile.up.sql
/Users/muhammadrizkisetyanto/Documents/arabiya-syari-fiber-1/internals/database/migrations/20250226092703_create-table-user-profile.down.sql

# Up-Down migrasi

Masukan number untuk mengarahkan mau berapa banyak file yang di up/down. Misal 1,2,3 dst
**UP**
migrate -path internals/databases/migrations \
 -database "postgresql://postgres.kkxflcqxkifqhysyijmx:Wedangjahe217@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require" up

**DOWN**
migrate -path internals/databases/migrations \
 -database "postgresql://postgres.kkxflcqxkifqhysyijmx:Wedangjahe217@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?sslmode=require" down

# Hapus semua

DROP SCHEMA public CASCADE;
CREATE SCHEMA public;


# Dirty migrasi

PGPASSWORD="Wedangjahe217" psql \
 -h kkxflcqxkifqhysyijmx.supabase.co \
 -p 5432 \
 -U postgres \
 -d postgres \
 -w

# Masuk database

muhammadrizkisetyanto@MacBook-Air-Muhammad arabiya-syari-fiber-1 % PGPASSWORD="qXdMRsMSGEgQvVrLuBjmUAGkytJwsaWk" psql -h trolley.proxy.rlwy.net -p 59123 -U postgres -d railway

# Refresh port

kill -9 $(lsof -t -i:3000)
kill -9 $(lsof -t -i:3000) && go run main.go

# Hapus Versi Migrasi yang Bermasalah dari Database

Jika ingin menghapus versi 20250306232632 dari database secara manual, jalankan perintah SQL berikut di PostgreSQL:

DELETE FROM schema_migrations WHERE version = 20250306232632;

Kemudian jalankan ulang migrasi:

# Akun Owner

SELECT fn_grant_role('2e6bf90c-2b2a-4ddb-a0e4-7593f7f1ec17'::uuid, 'owner', NULL, NULL);

# JWT

muhammadrizkisetyanto@MacBook-Air-Muhammad arabiya-syari-fiber-1 % export JWT_SECRET=rahasia_dong

muhammadrizkisetyanto@MacBook-Air-Muhammad arabiya-syari-fiber-1 % echo $JWT_SECRET

rahasia_dong

# Mencari kata

muhammadrizkisetyanto@MacBook-Air-Muhammad arabiya-syari-fiber-1 % grep -r "subategories_id" .

# Midtrans

https://simulator.sandbox.midtrans.com/

# Password

Wedangjahe217312!

# Seeding

muhammadrizkisetyanto@MacBook-Air-Muhammad quizku % go run internals/seeds/cmd/main.go all

# Bila postman macet

killall -9 Postman


# Menyamanakn denga



# Masuk ke railway PSQL
PGPASSWORD='nCzXbpAEDzqPjbwxGobwMvbyDHUpUsgP' psql \
  -h shortline.proxy.rlwy.net \
  -U postgres \
  -p 46351 \
  -d railway


# Command dengan railway
migrate -path internals/databases/migrations \
  -database "postgresql://postgres:nCzXbpAEDzqPjbwxGobwMvbyDHUpUsgP@shortline.proxy.rlwy.net:46351/railway?sslmode=disable" \
  down 2



{
  "success": true,
  "message": "ok",
  "data": [
    { ...term fields langsung... }
  ],
  "include": {
    "classes": [ ... ],
    "class_sections": [ ... ],
    "fee_rules": [ ... ]
  },
  "pagination": { ... }
}



1️⃣ Konfirmasi dulu: 5 “fundamental block” itu ini

Biar kita sinkron sekali lagi:

FK normal

Hanya simpan ID → join kalau butuh.

Snapshot (immutable)

Foto sekali → nggak diutak-atik lagi.

Cache / Denorm

Copy kecil buat baca cepat → boleh ikut berubah.

History / Audit

Riwayat perubahan master dari waktu ke waktu.

Summary / Aggregation / Projection

Data hasil olahan (rekap, view khusus, dll).

Semua pola lain (CQRS, event sourcing, MV, search index) kalau di-“zoom in” jatuhnya ke kombinasi ini juga.