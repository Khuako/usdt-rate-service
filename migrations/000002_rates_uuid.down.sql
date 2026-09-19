begin;
alter table rates drop column id;
alter table rates add column id bigint generated always as identity primary key;
commit;
