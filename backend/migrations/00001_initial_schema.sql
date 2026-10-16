-- +goose Up
CREATE TABLE family_members (
    telegram_user_id BIGINT PRIMARY KEY,
    username TEXT,
    display_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT NOT NULL REFERENCES family_members (telegram_user_id) ON DELETE RESTRICT,
    amount NUMERIC(14, 2) NOT NULL CHECK (amount > 0),
    category TEXT NOT NULL CHECK (
        category IN ('еда', 'транспорт', 'жильё', 'развлечения', 'здоровье', 'покупки', 'другое')
    ),
    spent_at DATE NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX expenses_user_spent_at_idx ON expenses (user_id, spent_at DESC);
CREATE INDEX expenses_category_spent_at_idx ON expenses (category, spent_at DESC);

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER family_members_set_updated_at
    BEFORE UPDATE ON family_members
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER expenses_set_updated_at
    BEFORE UPDATE ON expenses
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE expenses;
DROP TABLE family_members;
DROP FUNCTION set_updated_at();
