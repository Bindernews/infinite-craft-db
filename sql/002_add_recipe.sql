-- Write your migrate up statements here

create or replace view recipe_ext as
    select
        r.id,
        r.dst,
        i_a.name as name_a,
        i_b.name as name_b,        
        i_a.depth + i_b.depth as depth
    from recipe r
    join item i_a on i_a.id = r.src_a
    join item i_b on i_b.id = r.src_b;

create or replace function v1_get_item(n text)
    returns item language sql as $$
select * from item i where lower(i.name) = lower(n);
$$;

create or replace function v1_add_recipe(name_a text, name_b text, name_dst text)
    returns int language plpgsql as
$$
declare
    -- Inputs
    inp_a      item := (select v1_get_item(name_a));
    inp_b      item;
    inp_c      item := (select v1_get_item(name_b));
    -- Recipe id
    recipe_id  recipe.id%type;
    -- Determine depth
    item_depth int;
    -- dest item id
    dst_id     int;
begin
    -- Get input rows and make sure a.id <= b.id
    if inp_a.id is null then raise exception 'item % not found', name_a; end if;
    if inp_c.id is null then raise exception 'item % not found', name_b; end if;
    if inp_a.id <= inp_c.id then
        inp_b = inp_c;
    else
        inp_b = inp_a;
        inp_a = inp_c;
    end if;
    -- Calculate new item depth for if we have to insert later
    item_depth := greatest(inp_a.depth, inp_b.depth) + 1;
    -- Search for existing recipe
    select id from recipe r where r.src_a = inp_a.id and r.src_b = inp_b.id into recipe_id;
    if found then
        return recipe_id;
    end if;
    -- Add new item, on conflict update to the lowest depth
    insert into item("name", "depth")
    values (name_dst, item_depth)
    on conflict(lower("name")) do update
        set depth = least(item_depth, item.depth)
    returning id into dst_id;
    -- Add new recipe
    insert into recipe(src_a, src_b, dst, valid)
    values (inp_a.id, inp_b.id, dst_id, false)
    returning id into recipe_id;
    return recipe_id;
end;
$$;

create or replace function v1_recipe_tree(item_name text)
    returns jsonb language plpgsql as
$$
declare
    root item := (select v1_get_item(item_name));
    best_recipe recipe%rowtype;
    name_a text;
    name_b text;
begin
    if root.depth = 0 then
        return to_jsonb(root.name);
    end if;
    select r.* from recipe r join recipe_ext rd on r.id = rd.id
               where r.dst = root.id
               order by rd.depth limit 1
               into best_recipe;
    name_a := (select i.name from item i where i.id = best_recipe.src_a);
    name_b := (select i.name from item i where i.id = best_recipe.src_b);
    return jsonb_build_object(
           'k', item_name,
           'in1', (select v1_find_recipe(name_a)),
           'in2', (select v1_find_recipe(name_b))
    );
end;
$$;

create or replace function v1_check_known_recipes(src_a text, src_b text[])
    returns text[] language plpgsql as 
$$
begin

end;
$$;

---- create above / drop below ----

drop function if exists v1_check_known_recipes;
drop function if exists v1_find_recipe;
drop function if exists v1_recipe_tree;
drop function v1_add_recipe;
drop function v1_get_item;
drop view recipe_ext;

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
