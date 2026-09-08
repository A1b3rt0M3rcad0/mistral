ALTER TABLE idempotency_commands
    ADD CONSTRAINT idempotency_commands_key_length_check
    CHECK (octet_length(idempotency_key) BETWEEN 1 AND 255);
