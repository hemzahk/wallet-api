ALTER TABLE 
    users
ADD 
    COLUMN identity_id UUID NOT NULL REFERENCES identities(id);