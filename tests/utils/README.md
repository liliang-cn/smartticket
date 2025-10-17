# Test Utilities

This directory contains utility scripts and tools used for testing and development.

## Files

### fix_admin_password.sh
Fixes the admin user password in the database.

**Usage:**
```bash
./fix_admin_password.sh
```

**Purpose:**
- Updates the admin user password hash to `admin123`
- Ensures superadmin user exists with correct password
- Useful for resetting authentication during development/testing

### generate_hash.rs
Rust source code for generating bcrypt password hashes.

**Usage:**
```bash
# Compile and run
rustc generate_hash.rs -L target/debug/deps --extern bcrypt=target/debug/deps/libbcrypt-*.rlib
./generate_hash

# Or run directly if already compiled
./generate_hash
```

**Purpose:**
- Generates secure bcrypt password hashes
- Used to create password hashes for test users
- Default password: `admin123`

### generate_hash
Compiled binary for generating bcrypt password hashes.

**Usage:**
```bash
./generate_hash
```

**Output:**
```
$2b$12$r2IkLwr1orrSp4/kzpAfj.bu7bmqv3Y/KPWldwi8BeC4.KrHjPZfi
```

## Notes

These tools are specifically for development and testing environments.
- They help with database setup and user management
- Password hashes are for testing purposes only
- Do not use these tools in production environments

## Default Credentials

- **Email**: admin@test.smartticket.com
- **Password**: admin123
- **Email**: superadmin@smartticket.system
- **Password**: admin123