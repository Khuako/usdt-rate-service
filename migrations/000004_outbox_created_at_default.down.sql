alter table outbox_events
    alter column created_at drop default;
