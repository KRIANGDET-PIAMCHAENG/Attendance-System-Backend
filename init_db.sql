--
-- PostgreSQL database dump
--

\restrict xXEK0zvxF4iU2jMqtFe0hICr3EF0ZVJdDKjMactXBtb1aFss4MNSdEqm7ytY53t

-- Dumped from database version 18.4
-- Dumped by pg_dump version 18.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: attendance; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.attendance (
    user_id character varying(50) NOT NULL,
    date date NOT NULL,
    check_in time without time zone,
    check_out time without time zone,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.attendance OWNER TO postgres;

--
-- Name: attendance_approvals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.attendance_approvals (
    id integer NOT NULL,
    attendance_request_id integer,
    approver_id character varying(50) NOT NULL,
    approve_role character varying(100),
    status character varying(20) NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.attendance_approvals OWNER TO postgres;

--
-- Name: attendance_approvals_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.attendance_approvals_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.attendance_approvals_id_seq OWNER TO postgres;

--
-- Name: attendance_approvals_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.attendance_approvals_id_seq OWNED BY public.attendance_approvals.id;


--
-- Name: attendance_request_attachments; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.attendance_request_attachments (
    id integer NOT NULL,
    attendance_request_id integer,
    file_path character varying(255) NOT NULL,
    original_name character varying(255),
    file_type character varying(100),
    file_size integer,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.attendance_request_attachments OWNER TO postgres;

--
-- Name: attendance_request_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.attendance_request_attachments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.attendance_request_attachments_id_seq OWNER TO postgres;

--
-- Name: attendance_request_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.attendance_request_attachments_id_seq OWNED BY public.attendance_request_attachments.id;


--
-- Name: attendance_requests; Type: TABLE; Schema: public; Owner: postgres
--

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


ALTER TABLE public.attendance_requests OWNER TO postgres;

--
-- Name: attendance_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.attendance_requests_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.attendance_requests_id_seq OWNER TO postgres;

--
-- Name: attendance_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.attendance_requests_id_seq OWNED BY public.attendance_requests.id;


--
-- Name: company_holidays; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.company_holidays (
    id integer NOT NULL,
    holiday_date date NOT NULL,
    description character varying(255) NOT NULL,
    year integer NOT NULL
);


ALTER TABLE public.company_holidays OWNER TO postgres;

--
-- Name: company_holidays_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.company_holidays_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.company_holidays_id_seq OWNER TO postgres;

--
-- Name: company_holidays_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.company_holidays_id_seq OWNED BY public.company_holidays.id;


--
-- Name: leave_approvals; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.leave_approvals (
    id integer NOT NULL,
    leave_request_id integer NOT NULL,
    approver_name character varying(100) NOT NULL,
    approve_role character varying(100) NOT NULL,
    status character varying(20) NOT NULL,
    reason text,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.leave_approvals OWNER TO postgres;

--
-- Name: leave_approvals_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.leave_approvals_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.leave_approvals_id_seq OWNER TO postgres;

--
-- Name: leave_approvals_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.leave_approvals_id_seq OWNED BY public.leave_approvals.id;


--
-- Name: leave_attachments; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.leave_attachments (
    id integer NOT NULL,
    leave_request_id integer,
    file_path character varying(255) NOT NULL,
    original_name character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    file_type character varying(50),
    file_size bigint
);


ALTER TABLE public.leave_attachments OWNER TO postgres;

--
-- Name: leave_attachments_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.leave_attachments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.leave_attachments_id_seq OWNER TO postgres;

--
-- Name: leave_attachments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.leave_attachments_id_seq OWNED BY public.leave_attachments.id;


--
-- Name: leave_balances; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.leave_balances (
    id integer NOT NULL,
    user_id character varying(50) NOT NULL,
    leave_type_id integer NOT NULL,
    days_allowed double precision DEFAULT 0,
    days_used double precision DEFAULT 0,
    year integer
);


ALTER TABLE public.leave_balances OWNER TO postgres;

--
-- Name: leave_balances_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.leave_balances_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.leave_balances_id_seq OWNER TO postgres;

--
-- Name: leave_balances_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.leave_balances_id_seq OWNED BY public.leave_balances.id;


--
-- Name: leave_requests; Type: TABLE; Schema: public; Owner: postgres
--

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


ALTER TABLE public.leave_requests OWNER TO postgres;

--
-- Name: leave_requests_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.leave_requests_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.leave_requests_id_seq OWNER TO postgres;

--
-- Name: leave_requests_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.leave_requests_id_seq OWNED BY public.leave_requests.id;


--
-- Name: leave_types; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.leave_types (
    id integer NOT NULL,
    name_en character varying(50) NOT NULL,
    name_th character varying(100),
    default_days double precision
);


ALTER TABLE public.leave_types OWNER TO postgres;

--
-- Name: leave_types_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.leave_types_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.leave_types_id_seq OWNER TO postgres;

--
-- Name: leave_types_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.leave_types_id_seq OWNED BY public.leave_types.id;


--
-- Name: notifications; Type: TABLE; Schema: public; Owner: postgres
--

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


ALTER TABLE public.notifications OWNER TO postgres;

--
-- Name: role; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.role (
    role_id character varying(50) NOT NULL,
    role_name character varying(100),
    role_color character varying(50),
    role_type character varying(50)
);


ALTER TABLE public.role OWNER TO postgres;

--
-- Name: subordinate_manager_roles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.subordinate_manager_roles (
    subordinate_id character varying(50) NOT NULL,
    manager_role_id character varying(50) NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.subordinate_manager_roles OWNER TO postgres;

--
-- Name: system_configs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.system_configs (
    config_key character varying(50) NOT NULL,
    config_value jsonb NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.system_configs OWNER TO postgres;

--
-- Name: user_info; Type: TABLE; Schema: public; Owner: postgres
--

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


ALTER TABLE public.user_info OWNER TO postgres;

--
-- Name: user_roles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.user_roles (
    id integer NOT NULL,
    user_id character varying(50),
    role_id character varying(50),
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.user_roles OWNER TO postgres;

--
-- Name: user_roles_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.user_roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.user_roles_id_seq OWNER TO postgres;

--
-- Name: user_roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.user_roles_id_seq OWNED BY public.user_roles.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    user_id character varying(50) NOT NULL,
    employee_id character varying(50),
    signature_path character varying(255)
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: attendance_approvals id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_approvals ALTER COLUMN id SET DEFAULT nextval('public.attendance_approvals_id_seq'::regclass);


--
-- Name: attendance_request_attachments id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_request_attachments ALTER COLUMN id SET DEFAULT nextval('public.attendance_request_attachments_id_seq'::regclass);


--
-- Name: attendance_requests id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_requests ALTER COLUMN id SET DEFAULT nextval('public.attendance_requests_id_seq'::regclass);


--
-- Name: company_holidays id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.company_holidays ALTER COLUMN id SET DEFAULT nextval('public.company_holidays_id_seq'::regclass);


--
-- Name: leave_approvals id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_approvals ALTER COLUMN id SET DEFAULT nextval('public.leave_approvals_id_seq'::regclass);


--
-- Name: leave_attachments id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_attachments ALTER COLUMN id SET DEFAULT nextval('public.leave_attachments_id_seq'::regclass);


--
-- Name: leave_balances id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_balances ALTER COLUMN id SET DEFAULT nextval('public.leave_balances_id_seq'::regclass);


--
-- Name: leave_requests id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_requests ALTER COLUMN id SET DEFAULT nextval('public.leave_requests_id_seq'::regclass);


--
-- Name: leave_types id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_types ALTER COLUMN id SET DEFAULT nextval('public.leave_types_id_seq'::regclass);


--
-- Name: user_roles id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_roles ALTER COLUMN id SET DEFAULT nextval('public.user_roles_id_seq'::regclass);


--
-- Data for Name: attendance; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.attendance (user_id, date, check_in, check_out, created_at, updated_at) FROM stdin;
\.


--
-- Data for Name: attendance_approvals; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.attendance_approvals (id, attendance_request_id, approver_id, approve_role, status, reason, created_at) FROM stdin;
\.


--
-- Data for Name: attendance_request_attachments; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.attendance_request_attachments (id, attendance_request_id, file_path, original_name, file_type, file_size, created_at) FROM stdin;
\.


--
-- Data for Name: attendance_requests; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.attendance_requests (id, user_id, date_from, date_to, start_time, end_time, remark, signature_path, status, created_at) FROM stdin;
\.


--
-- Data for Name: company_holidays; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.company_holidays (id, holiday_date, description, year) FROM stdin;
1	2026-01-01	วันขึ้นปีใหม่	2026
2	2026-01-02	วันหยุดทำการเพิ่มเป็นกรณีพิเศษ	2026
3	2026-03-03	วันมาฆบูชา	2026
4	2026-04-06	วันพระบาทสมเด็จพระพุทธยอดฟ้าจุฬาโลกมหาราช และวันที่ระลึกมหาจักรีบรมราชวงศ์	2026
5	2026-04-13	วันสงกรานต์	2026
6	2026-04-14	วันสงกรานต์	2026
7	2026-04-15	วันสงกรานต์	2026
8	2026-05-01	วันแรงงานแห่งชาติ	2026
9	2026-05-04	วันฉัตรมงคล	2026
10	2026-06-01	ชดเชยวันวิสาขบูชา (วันอาทิตย์ที่ 31 พฤษภาคม 2569)	2026
11	2026-06-03	วันเฉลิมพระชนมพรรษาสมเด็จพระนางเจ้าสุทิดา พัชรสุธาพิมลลักษณ พระบรมราชินี	2026
12	2026-07-28	วันเฉลิมพระชนมพรรษาพระบาทสมเด็จพระเจ้าอยู่หัว	2026
13	2026-07-29	วันอาสาฬหบูชา	2026
14	2026-08-12	วันเฉลิมพระชนมพรรษาสมเด็จพระนางเจ้าสิริกิติ์ พระบรมราชินีนาถ พระบรมราชชนนีพันปีหลวง และวันแม่แห่งชาติ	2026
15	2026-10-13	วันนวมินทรมหาราช	2026
16	2026-10-23	วันปิยมหาราช	2026
17	2026-12-07	ชดเชยวันคล้ายวันพระบรมราชสมภพ พระบาทสมเด็จพระบรมชนกาธิเบศร มหาภูมิพลอดุลยเดชมหาราช บรมนาถบพิตร วันชาติ และวันพ่อแห่งชาติ (วันเสาร์ที่ 5 ธันวาคม 2569)	2026
18	2026-12-10	วันรัฐธรรมนูญ	2026
19	2026-12-31	วันสิ้นปี	2026
\.


--
-- Data for Name: leave_approvals; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.leave_approvals (id, leave_request_id, approver_name, approve_role, status, reason, created_at) FROM stdin;
\.


--
-- Data for Name: leave_attachments; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.leave_attachments (id, leave_request_id, file_path, original_name, created_at, file_type, file_size) FROM stdin;
\.


--
-- Data for Name: leave_balances; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.leave_balances (id, user_id, leave_type_id, days_allowed, days_used, year) FROM stdin;
\.


--
-- Data for Name: leave_requests; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.leave_requests (id, user_id, leave_type, date_from, date_to, from_date_morning, to_date_morning, remark, created_at, signature_path, status) FROM stdin;
\.


--
-- Data for Name: leave_types; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.leave_types (id, name_en, name_th, default_days) FROM stdin;
1	sick	ลาป่วย	\N
2	personal	ลากิจ	\N
3	vacation	พักผ่อน	\N
4	maternity	ลาคลอดบุตร	\N
5	paternity	ลาดูแลภรรยาคลอด	\N
6	parental	ลาเลี้ยงดูบุตร	\N
\.


--
-- Data for Name: notifications; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.notifications (id, user_id, title, message, is_read, type, status, request_number, created_at) FROM stdin;
\.


--
-- Data for Name: role; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.role (role_id, role_name, role_color, role_type) FROM stdin;
2	ผู้ดูแลระบบ	a828ec	admin
\.


--
-- Data for Name: subordinate_manager_roles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.subordinate_manager_roles (subordinate_id, manager_role_id, created_at) FROM stdin;
\.


--
-- Data for Name: system_configs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.system_configs (config_key, config_value, updated_at) FROM stdin;
attendance_request	{"evidence-file": true, "specify-remark": true, "required-remark": false, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": false, "specify-approval-reason": true}	2026-03-07 13:11:10.557421
leave_config	{"sick": {"evidence-file": true, "specify-remark": true, "required-remark": true, "allow-retroactive": true, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": true}, "parental": {"evidence-file": true, "specify-remark": true, "required-remark": true, "allow-retroactive": false, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": true}, "personal": {"evidence-file": true, "specify-remark": true, "required-remark": true, "allow-retroactive": false, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": true}, "vacation": {"evidence-file": false, "specify-remark": false, "required-remark": false, "allow-retroactive": false, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": false}, "maternity": {"evidence-file": true, "specify-remark": true, "required-remark": true, "allow-retroactive": false, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": true}, "paternity": {"evidence-file": true, "specify-remark": true, "required-remark": true, "allow-retroactive": false, "approve-need-signature": true, "request-need-signature": true, "required-evidence-file": true}}	2026-02-28 13:54:05.214133
attendance_time	{"cutoff-time": {"hour": 0, "minute": 0}, "auto-checkout": false, "check-in-time": {"hour": 8, "minute": 30}, "check-out-time": {"hour": 16, "minute": 30}, "check-in-leave-time": {"hour": 13, "minute": 0}, "check-out-leave-time": {"hour": 12, "minute": 0}}	2026-02-24 06:53:20.583564
budget_year	{"day": 1, "month": 10}	2026-06-30 16:29:15.538114
\.


--
-- Data for Name: user_info; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.user_info (user_id, employee_id, email, fullname_eng, fullname_thai, gender, nationality, phone, role_init, picture) FROM stdin;
system-root	0000000000	attendance.system@eng.src.ku.ac.th	\N	\N	\N	\N	\N	admin	https://lh3.googleusercontent.com/a/ACg8ocJcAVntLicCasDaaxS7Wu1Poeud0TXzpqedSW9_JLKN7dOYzA=s96-c
\.


--
-- Data for Name: user_roles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.user_roles (id, user_id, role_id, created_at) FROM stdin;
0	system-root	2	2026-06-30 16:21:20.069993
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (user_id, employee_id, signature_path) FROM stdin;
system-root	0000000000	uploads/signatures/2026/06/670437aa-5bf1-4429-9eee-5d56343c27f2.png
\.


--
-- Name: attendance_approvals_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.attendance_approvals_id_seq', 1, false);


--
-- Name: attendance_request_attachments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.attendance_request_attachments_id_seq', 1, false);


--
-- Name: attendance_requests_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.attendance_requests_id_seq', 1, false);


--
-- Name: company_holidays_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.company_holidays_id_seq', 19, true);


--
-- Name: leave_approvals_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.leave_approvals_id_seq', 1, false);


--
-- Name: leave_attachments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.leave_attachments_id_seq', 1, false);


--
-- Name: leave_balances_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.leave_balances_id_seq', 1, false);


--
-- Name: leave_requests_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.leave_requests_id_seq', 1, false);


--
-- Name: leave_types_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.leave_types_id_seq', 6, true);


--
-- Name: user_roles_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.user_roles_id_seq', 1, false);


--
-- Name: attendance_approvals attendance_approvals_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_approvals
    ADD CONSTRAINT attendance_approvals_pkey PRIMARY KEY (id);


--
-- Name: attendance attendance_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance
    ADD CONSTRAINT attendance_pkey PRIMARY KEY (user_id, date);


--
-- Name: attendance_request_attachments attendance_request_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_request_attachments
    ADD CONSTRAINT attendance_request_attachments_pkey PRIMARY KEY (id);


--
-- Name: attendance_requests attendance_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_requests
    ADD CONSTRAINT attendance_requests_pkey PRIMARY KEY (id);


--
-- Name: company_holidays company_holidays_holiday_date_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.company_holidays
    ADD CONSTRAINT company_holidays_holiday_date_key UNIQUE (holiday_date);


--
-- Name: company_holidays company_holidays_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.company_holidays
    ADD CONSTRAINT company_holidays_pkey PRIMARY KEY (id);


--
-- Name: leave_approvals leave_approvals_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_approvals
    ADD CONSTRAINT leave_approvals_pkey PRIMARY KEY (id);


--
-- Name: leave_attachments leave_attachments_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_attachments
    ADD CONSTRAINT leave_attachments_pkey PRIMARY KEY (id);


--
-- Name: leave_balances leave_balances_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_balances
    ADD CONSTRAINT leave_balances_pkey PRIMARY KEY (id);


--
-- Name: leave_requests leave_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_requests
    ADD CONSTRAINT leave_requests_pkey PRIMARY KEY (id);


--
-- Name: leave_types leave_types_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_types
    ADD CONSTRAINT leave_types_pkey PRIMARY KEY (id);


--
-- Name: notifications notifications_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.notifications
    ADD CONSTRAINT notifications_pkey PRIMARY KEY (id);


--
-- Name: role role_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.role
    ADD CONSTRAINT role_pkey PRIMARY KEY (role_id);


--
-- Name: subordinate_manager_roles subordinate_manager_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subordinate_manager_roles
    ADD CONSTRAINT subordinate_manager_roles_pkey PRIMARY KEY (subordinate_id, manager_role_id);


--
-- Name: system_configs system_configs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.system_configs
    ADD CONSTRAINT system_configs_pkey PRIMARY KEY (config_key);


--
-- Name: user_info user_info_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_info
    ADD CONSTRAINT user_info_pkey PRIMARY KEY (user_id);


--
-- Name: user_roles user_roles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT user_roles_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);


--
-- Name: attendance_approvals attendance_approvals_attendance_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_approvals
    ADD CONSTRAINT attendance_approvals_attendance_request_id_fkey FOREIGN KEY (attendance_request_id) REFERENCES public.attendance_requests(id) ON DELETE CASCADE;


--
-- Name: attendance_request_attachments attendance_request_attachments_attendance_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance_request_attachments
    ADD CONSTRAINT attendance_request_attachments_attendance_request_id_fkey FOREIGN KEY (attendance_request_id) REFERENCES public.attendance_requests(id) ON DELETE CASCADE;


--
-- Name: attendance attendance_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attendance
    ADD CONSTRAINT attendance_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: subordinate_manager_roles fk_manager_role; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subordinate_manager_roles
    ADD CONSTRAINT fk_manager_role FOREIGN KEY (manager_role_id) REFERENCES public.role(role_id) ON DELETE CASCADE;


--
-- Name: user_roles fk_rel_role; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT fk_rel_role FOREIGN KEY (role_id) REFERENCES public.role(role_id);


--
-- Name: user_roles fk_rel_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_roles
    ADD CONSTRAINT fk_rel_user FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: subordinate_manager_roles fk_subordinate; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.subordinate_manager_roles
    ADD CONSTRAINT fk_subordinate FOREIGN KEY (subordinate_id) REFERENCES public.users(user_id) ON DELETE CASCADE;


--
-- Name: user_info fk_user_main; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.user_info
    ADD CONSTRAINT fk_user_main FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: leave_approvals leave_approvals_leave_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_approvals
    ADD CONSTRAINT leave_approvals_leave_request_id_fkey FOREIGN KEY (leave_request_id) REFERENCES public.leave_requests(id) ON DELETE CASCADE;


--
-- Name: leave_attachments leave_attachments_leave_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_attachments
    ADD CONSTRAINT leave_attachments_leave_request_id_fkey FOREIGN KEY (leave_request_id) REFERENCES public.leave_requests(id) ON DELETE CASCADE;


--
-- Name: leave_balances leave_balances_leave_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_balances
    ADD CONSTRAINT leave_balances_leave_type_id_fkey FOREIGN KEY (leave_type_id) REFERENCES public.leave_types(id);


--
-- Name: leave_balances leave_balances_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.leave_balances
    ADD CONSTRAINT leave_balances_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.user_info(user_id);


--
-- PostgreSQL database dump complete
--

\unrestrict xXEK0zvxF4iU2jMqtFe0hICr3EF0ZVJdDKjMactXBtb1aFss4MNSdEqm7ytY53t

