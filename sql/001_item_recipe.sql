
create table item(
  id serial primary key,
  "name" varchar(120) not null,
  -- Distance from root
  depth int not null
);
-- Case-insensitive unique index
create unique index item_name_idx on item (lower("name"));

create table recipe(
  id    serial primary key,
  src_a int references item(id),
  src_b int references item(id),
  dst   int references item(id),
  valid boolean not null
);
-- Ensure src_a <= src_b, so we have a defined ordering for recipe sources
alter table recipe add constraint recipe_a_lt_b check (src_a <= src_b);

insert into item("name", depth) values
  ('Water', 0),
  ('Fire', 0),
  ('Wind', 0),
  ('Earth', 0);

---- create above / drop below ----

drop table recipe;
drop table item;
