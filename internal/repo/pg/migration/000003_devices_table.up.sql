CREATE TABLE IF NOT EXISTS devices (
		id int GENERATED BY DEFAULT AS IDENTITY NOT NULL,
		device_type varchar(320) NOT NULL,
		"name" varchar NOT NULL,
		address inet NOT NULL,
		responsible jsonb NOT NULL DEFAULT '{}',
		created_at timestamp without time zone NULL,
		updated_at timestamp without time zone NULL,
		deleted_at timestamp without time zone NULL,
		CONSTRAINT devices_pk PRIMARY KEY (id),
		CONSTRAINT devices_name_unique UNIQUE ("name")
	);

