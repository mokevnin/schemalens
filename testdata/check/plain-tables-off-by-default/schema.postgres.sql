CREATE TABLE account (id bigserial PRIMARY KEY, name text);
CREATE VIEW account_names AS SELECT name FROM account;
CREATE FUNCTION touch_account() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$;
CREATE TRIGGER account_audit AFTER INSERT ON account FOR EACH ROW EXECUTE FUNCTION touch_account();
