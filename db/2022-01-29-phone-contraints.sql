alter table players add column phone not null default 0;
update players set phone = coalesce(cast(phone_number as int8), 0);
alter table players drop column phone_number;
