-- ** Database generated with pgModeler (PostgreSQL Database Modeler).
-- ** pgModeler version: 1.2.0-beta1
-- ** PostgreSQL version: 17.0
-- ** Project Site: pgmodeler.io
-- ** Model Author: ---

-- ** Database creation must be performed outside a multi lined SQL file. 
-- ** These commands were put in this file only as a convenience.

-- object: postgres | type: DATABASE --
-- DROP DATABASE IF EXISTS postgres;
CREATE DATABASE postgres;
-- ddl-end --


SET search_path TO pg_catalog,public;
-- ddl-end --

-- object: public.execution | type: TABLE --
-- DROP TABLE IF EXISTS public.execution CASCADE;
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
	trade_service_execution_id integer,
	version integer NOT NULL DEFAULT 1,
	CONSTRAINT execution_pk PRIMARY KEY (id)
);
-- ddl-end --
ALTER TABLE public.execution OWNER TO postgres;
-- ddl-end --

-- object: execution_order_id_ndx | type: INDEX --
-- DROP INDEX IF EXISTS public.execution_order_id_ndx CASCADE;
CREATE UNIQUE INDEX execution_order_id_ndx ON public.execution
USING btree
(
	execution_service_id
);
-- ddl-end --

-- object: execution_next_fill_ndx | type: INDEX --
-- DROP INDEX IF EXISTS public.execution_next_fill_ndx CASCADE;
CREATE INDEX execution_next_fill_ndx ON public.execution
USING btree
(
	next_fill_timestamp DESC NULLS LAST
);
-- ddl-end --


