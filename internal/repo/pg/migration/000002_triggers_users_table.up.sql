CREATE OR REPLACE FUNCTION bd_tr_users()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
	BEGIN
		update users
		set deleted_at = current_timestamp
		where id = old.id;

		return NULL;
	END;
$function$
;

create or replace trigger tr_bd_users before
delete
	on
	users for each row execute function bd_tr_users();