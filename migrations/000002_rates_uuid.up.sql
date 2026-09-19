begin;
alter table rates alter column id drop identity;
alter table rates alter column id type uuid using gen_random_uuid();
commit;
