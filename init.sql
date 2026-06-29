--
-- PostgreSQL database dump (Structure Only)
--

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';
SET default_table_access_method = heap;

-- 1. Create Tables and Sequences

CREATE TABLE public.attendance (
    user_id character varying(50) NOT NULL,
    date date NOT NULL,
    check_in time without time zone,
    check_out time without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE public.attendance_approvals (
    id integer NOT NULL,
    attendance_request_id integer,
    approver_id character varying(50) NOT NULL,
    approve_role character varying(100),
    status character varying(20) NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE public.attendance_approvals_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.attendance_approvals_id_seq OWNED BY public.attendance_approvals.id;


CREATE TABLE public.attendance_request_attachments (
    id integer NOT NULL,
    attendance_request_id integer,
    file_path character varying(255) NOT NULL,
    original_name character varying(255),
    file_type character varying(100),
    file_size integer,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE public.attendance_request_attachments_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.attendance_request_attachments_id_seq OWNED BY public.attendance_request_attachments.id;


CREATE TABLE public.attendance_requests (
    id integer NOT NULL,
    user_id character varying(50) NOT NULL,
    date_from timestamp with time zone NOT NULL,
    date_to timestamp with time zone NOT NULL,
    start_time character varying(10) NOT NULL,
    end_time character varying(10) NOT NULL,
    remark text,
    signature_path character varying(255),
    status character varying(20) DEFAULT 'pending'::character varying,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE public.attendance_requests_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.attendance_requests_id_seq OWNED BY public.attendance_requests.id;


CREATE TABLE public.company_holidays (
    id integer NOT NULL,
    holiday_date date NOT NULL,
    description character varying(255) NOT NULL,
    year integer NOT NULL
);

CREATE SEQUENCE public.company_holidays_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.company_holidays_id_seq OWNED BY public.company_holidays.id;


CREATE TABLE public.leave_approvals (
    id integer NOT NULL,
    leave_request_id integer NOT NULL,
    approver_name character varying(100) NOT NULL,
    approve_role character varying(100) NOT NULL,
    status character varying(20) NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE public.leave_approvals_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.leave_approvals_id_seq OWNED BY public.leave_approvals.id;


CREATE TABLE public.leave_attachments (
    id integer NOT NULL,
    leave_request_id integer,
    file_path character varying(255) NOT NULL,
    original_name character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    file_type character varying(50),
    file_size bigint
);

CREATE SEQUENCE public.leave_attachments_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.leave_attachments_id_seq OWNED BY public.leave_attachments.id;


CREATE TABLE public.leave_balances (
    id integer NOT NULL,
    user_id character varying(50) NOT NULL,
    leave_type_id integer NOT NULL,
    days_allowed double precision DEFAULT 0,
    days_used double precision DEFAULT 0,
    year integer
);

CREATE SEQUENCE public.leave_balances_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.leave_balances_id_seq OWNED BY public.leave_balances.id;


CREATE TABLE public.leave_requests (
    id integer NOT NULL,
    user_id character varying(50) NOT NULL,
    leave_type character varying(50) NOT NULL,
    date_from timestamp with time zone NOT NULL,
    date_to timestamp with time zone NOT NULL,
    from_date_morning boolean DEFAULT false,
    to_date_morning boolean DEFAULT false,
    remark text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    signature_path character varying(255),
    status character varying(20) DEFAULT 'pending'::character varying
);

CREATE SEQUENCE public.leave_requests_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.leave_requests_id_seq OWNED BY public.leave_requests.id;


CREATE TABLE public.leave_types (
    id integer NOT NULL,
    name_en character varying(50) NOT NULL,
    name_th character varying(100),
    default_days double precision
);

CREATE SEQUENCE public.leave_types_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.leave_types_id_seq OWNED BY public.leave_types.id;


CREATE TABLE public.notifications (
    id character varying(50) NOT NULL,
    user_id character varying(50) NOT NULL,
    title character varying(255) NOT NULL,
    message text NOT NULL,
    is_read boolean DEFAULT false,
    type character varying(50) NOT NULL,
    status character varying(50) NOT NULL,
    request_number character varying(50) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE public.role (
    role_id character varying(50) NOT NULL,
    role_name character varying(100),
    role_color character varying(50),
    role_type character varying(50)
);


CREATE TABLE public.subordinate_manager_roles (
    subordinate_id character varying(50) NOT NULL,
    manager_role_id character varying(50) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE public.system_configs (
    config_key character varying(50) NOT NULL,
    config_value jsonb NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE public.user_info (
    user_id character varying(50) NOT NULL,
    employee_id character varying(50),
    email character varying(100),
    fullname_eng character varying(150),
    fullname_thai character varying(150),
    gender character varying(20),
    nationality character varying(50),
    phone character varying(20),
    role_init character varying(30),
    picture text
);


CREATE TABLE public.user_roles (
    id integer NOT NULL,
    user_id character varying(50),
    role_id character varying(50),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE public.user_roles_id_seq
    AS integer START WITH 1 INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1;
ALTER SEQUENCE public.user_roles_id_seq OWNED BY public.user_roles.id;


CREATE TABLE public.users (
    user_id character varying(50) NOT NULL,
    employee_id character varying(50),
    signature_path character varying(255)
);

-- 2. Set Default ID values from Sequences

ALTER TABLE ONLY public.attendance_approvals ALTER COLUMN id SET DEFAULT nextval('public.attendance_approvals_id_seq'::regclass);
ALTER TABLE ONLY public.attendance_request_attachments ALTER COLUMN id SET DEFAULT nextval('public.attendance_request_attachments_id_seq'::regclass);
ALTER TABLE ONLY public.attendance_requests ALTER COLUMN id SET DEFAULT nextval('public.attendance_requests_id_seq'::regclass);
ALTER TABLE ONLY public.company_holidays ALTER COLUMN id SET DEFAULT nextval('public.company_holidays_id_seq'::regclass);
ALTER TABLE ONLY public.leave_approvals ALTER COLUMN id SET DEFAULT nextval('public.leave_approvals_id_seq'::regclass);
ALTER TABLE ONLY public.leave_attachments ALTER COLUMN id SET DEFAULT nextval('public.leave_attachments_id_seq'::regclass);
ALTER TABLE ONLY public.leave_balances ALTER COLUMN id SET DEFAULT nextval('public.leave_balances_id_seq'::regclass);
ALTER TABLE ONLY public.leave_requests ALTER COLUMN id SET DEFAULT nextval('public.leave_requests_id_seq'::regclass);
ALTER TABLE ONLY public.leave_types ALTER COLUMN id SET DEFAULT nextval('public.leave_types_id_seq'::regclass);
ALTER TABLE ONLY public.user_roles ALTER COLUMN id SET DEFAULT nextval('public.user_roles_id_seq'::regclass);

-- 3. Add Primary Keys

ALTER TABLE ONLY public.attendance_approvals ADD CONSTRAINT attendance_approvals_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.attendance ADD CONSTRAINT attendance_pkey PRIMARY KEY (user_id, date);
ALTER TABLE ONLY public.attendance_request_attachments ADD CONSTRAINT attendance_request_attachments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.attendance_requests ADD CONSTRAINT attendance_requests_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.company_holidays ADD CONSTRAINT company_holidays_holiday_date_key UNIQUE (holiday_date);
ALTER TABLE ONLY public.company_holidays ADD CONSTRAINT company_holidays_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.leave_approvals ADD CONSTRAINT leave_approvals_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.leave_attachments ADD CONSTRAINT leave_attachments_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.leave_balances ADD CONSTRAINT leave_balances_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.leave_requests ADD CONSTRAINT leave_requests_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.leave_types ADD CONSTRAINT leave_types_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.notifications ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.role ADD CONSTRAINT role_pkey PRIMARY KEY (role_id);
ALTER TABLE ONLY public.subordinate_manager_roles ADD CONSTRAINT subordinate_manager_roles_pkey PRIMARY KEY (subordinate_id, manager_role_id);
ALTER TABLE ONLY public.system_configs ADD CONSTRAINT system_configs_pkey PRIMARY KEY (config_key);
ALTER TABLE ONLY public.user_info ADD CONSTRAINT user_info_pkey PRIMARY KEY (user_id);
ALTER TABLE ONLY public.user_roles ADD CONSTRAINT user_roles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.users ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);

-- 4. Add Foreign Keys

ALTER TABLE ONLY public.attendance_approvals ADD CONSTRAINT attendance_approvals_attendance_request_id_fkey FOREIGN KEY (attendance_request_id) REFERENCES public.attendance_requests(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.attendance_request_attachments ADD CONSTRAINT attendance_request_attachments_attendance_request_id_fkey FOREIGN KEY (attendance_request_id) REFERENCES public.attendance_requests(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.attendance ADD CONSTRAINT attendance_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.subordinate_manager_roles ADD CONSTRAINT fk_manager_role FOREIGN KEY (manager_role_id) REFERENCES public.role(role_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_roles ADD CONSTRAINT fk_rel_role FOREIGN KEY (role_id) REFERENCES public.role(role_id);
ALTER TABLE ONLY public.user_roles ADD CONSTRAINT fk_rel_user FOREIGN KEY (user_id) REFERENCES public.users(user_id);
ALTER TABLE ONLY public.subordinate_manager_roles ADD CONSTRAINT fk_subordinate FOREIGN KEY (subordinate_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_info ADD CONSTRAINT fk_user_main FOREIGN KEY (user_id) REFERENCES public.users(user_id);
ALTER TABLE ONLY public.leave_approvals ADD CONSTRAINT leave_approvals_leave_request_id_fkey FOREIGN KEY (leave_request_id) REFERENCES public.leave_requests(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.leave_attachments ADD CONSTRAINT leave_attachments_leave_request_id_fkey FOREIGN KEY (leave_request_id) REFERENCES public.leave_requests(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.leave_balances ADD CONSTRAINT leave_balances_leave_type_id_fkey FOREIGN KEY (leave_type_id) REFERENCES public.leave_types(id);
ALTER TABLE ONLY public.leave_balances ADD CONSTRAINT leave_balances_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.user_info(user_id);