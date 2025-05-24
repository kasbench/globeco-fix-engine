-- Create execution table
CREATE TABLE public.execution (
	id serial NOT NULL,
	execution_service_id integer NOT NULL,
	is_open boolean NOT NULL DEFAULT true,
	execution_status varchar(20) NOT NULL,
	trade_type varchar(10) NOT NULL,
	destination varchar(20) NOT NULL,
	security_id char(24) NOT NULL,
	ticker varchar(20) NOT NULL,
	quantity_ordered decimal(18,8) NOT NULL,
	limit_price decimal(18,8),
	received_timestamp timestamptz NOT NULL,
	sent_timestamp timestamptz NOT NULL,
	last_fill_timestamp timestamptz,
	quantity_filled decimal(18,8) NOT NULL DEFAULT 0,
	next_fill_timestamp timestamptz,
	number_of_fills smallint NOT NULL DEFAULT 0,
	total_amount decimal(18,8) NOT NULL DEFAULT 0,
	trade_service_execution_id integer NULL,
	version integer NOT NULL DEFAULT 1,
	CONSTRAINT execution_pk PRIMARY KEY (id)
);

-- Create unique index on order_id
CREATE UNIQUE INDEX execution_service_id_ndx ON public.execution
USING btree (execution_service_id);

-- Create index on next_fill_timestamp
CREATE INDEX execution_next_fill_ndx ON public.execution
USING btree (next_fill_timestamp DESC NULLS LAST); 