create table outbox_events(
  id uuid primary key,
  topic text not null,
  message_key text not null ,
  payload jsonb not null,
  created_at timestamptz not null,
  published_at timestamptz
);