CREATE TABLE IF NOT EXISTS identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email_address citext,
    phone_number TEXT,
    dob DATE,
    gender TEXT,
    state TEXT,
    city TEXT,
    street TEXT, 
    post_code TEXT,
    identity_type TEXT,
    organization_name TEXT,
    category TEXT
);