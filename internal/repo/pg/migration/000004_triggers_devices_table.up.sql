CREATE OR REPLACE FUNCTION bd_tr_devices()
 RETURNS trigger
 LANGUAGE plpgsql
AS $function$
	BEGIN
		update devices
		set deleted_at = current_timestamp
		where id = old.id;

		return NULL;
	END;
$function$
;

create or replace trigger tr_bd_devices before
delete
	on
	devices for each row execute function bd_tr_devices();