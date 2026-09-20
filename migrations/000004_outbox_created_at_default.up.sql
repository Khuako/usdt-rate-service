alter table outbox_events
    alter column created_at set default now();
