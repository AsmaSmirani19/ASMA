--
-- PostgreSQL database dump
--

-- Dumped from database version 15.12
-- Dumped by pg_dump version 15.12

-- Started on 2025-05-15 17:40:04

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

--
-- TOC entry 216 (class 1259 OID 16590)
-- Name: Agent_List; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public."Agent_List" (
    "Name" text DEFAULT 'Inconnu'::text NOT NULL,
    "Address" text NOT NULL,
    "Test_health" boolean DEFAULT false NOT NULL,
    "Availability" numeric DEFAULT 0.00 NOT NULL,
    id integer NOT NULL
);


ALTER TABLE public."Agent_List" OWNER TO postgres;

--
-- TOC entry 217 (class 1259 OID 16605)
-- Name: Agent_List_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public."Agent_List_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public."Agent_List_id_seq" OWNER TO postgres;

--
-- TOC entry 3381 (class 0 OID 0)
-- Dependencies: 217
-- Name: Agent_List_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."Agent_List_id_seq" OWNED BY public."Agent_List".id;


--
-- TOC entry 215 (class 1259 OID 16564)
-- Name: Test_Results; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public."Test_Results" (
    id integer NOT NULL,
    "PacketLossPercent" real,
    "AvgLatencyMs " bigint,
    "AvgJitterMs " bigint,
    "AvgThroughputKbps" real
);


ALTER TABLE public."Test_Results" OWNER TO postgres;

--
-- TOC entry 214 (class 1259 OID 16563)
-- Name: Test_Results_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public."Test_Results_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public."Test_Results_id_seq" OWNER TO postgres;

--
-- TOC entry 3382 (class 0 OID 0)
-- Dependencies: 214
-- Name: Test_Results_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."Test_Results_id_seq" OWNED BY public."Test_Results".id;


--
-- TOC entry 225 (class 1259 OID 41158)
-- Name: threshold_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.threshold_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.threshold_id_seq OWNER TO postgres;

--
-- TOC entry 224 (class 1259 OID 16651)
-- Name: Threshold; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public."Threshold" (
    "ID" bigint DEFAULT nextval('public.threshold_id_seq'::regclass) NOT NULL,
    "Name" text,
    creation_date timestamp with time zone,
    avg integer,
    min integer,
    max integer,
    avg_status boolean,
    min_status boolean,
    max_status boolean,
    avg_opr character varying(1),
    min_opr character varying(1),
    max_opr character varying(1),
    selected_metric character varying(50) NOT NULL
);


ALTER TABLE public."Threshold" OWNER TO postgres;

--
-- TOC entry 221 (class 1259 OID 16631)
-- Name: agent-group; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public."agent-group" (
    "ID" integer NOT NULL,
    group_name text NOT NULL,
    number_of_agents bigint NOT NULL,
    creation_date timestamp with time zone NOT NULL
);


ALTER TABLE public."agent-group" OWNER TO postgres;

--
-- TOC entry 220 (class 1259 OID 16630)
-- Name: agent-group_ID_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public."agent-group_ID_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public."agent-group_ID_seq" OWNER TO postgres;

--
-- TOC entry 3383 (class 0 OID 0)
-- Dependencies: 220
-- Name: agent-group_ID_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."agent-group_ID_seq" OWNED BY public."agent-group"."ID";


--
-- TOC entry 219 (class 1259 OID 16622)
-- Name: test; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.test (
    "Id" integer NOT NULL,
    test_name text NOT NULL,
    test_duration interval NOT NULL,
    number_of_agents integer NOT NULL,
    creation_date timestamp with time zone NOT NULL,
    source_id integer,
    target_id integer,
    profile_id integer,
    threshold_id integer,
    test_type character varying(50),
    waiting boolean DEFAULT false,
    failed boolean DEFAULT false,
    completed boolean DEFAULT false
);


ALTER TABLE public.test OWNER TO postgres;

--
-- TOC entry 218 (class 1259 OID 16621)
-- Name: planned_test_Id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public."planned_test_Id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public."planned_test_Id_seq" OWNER TO postgres;

--
-- TOC entry 3384 (class 0 OID 0)
-- Dependencies: 218
-- Name: planned_test_Id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."planned_test_Id_seq" OWNED BY public.test."Id";


--
-- TOC entry 223 (class 1259 OID 16643)
-- Name: test_profile; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.test_profile (
    "ID" integer NOT NULL,
    profile_name text,
    creation_date timestamp with time zone,
    packet_size integer
);


ALTER TABLE public.test_profile OWNER TO postgres;

--
-- TOC entry 222 (class 1259 OID 16642)
-- Name: test_profile_ID_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public."test_profile_ID_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public."test_profile_ID_seq" OWNER TO postgres;

--
-- TOC entry 3385 (class 0 OID 0)
-- Dependencies: 222
-- Name: test_profile_ID_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."test_profile_ID_seq" OWNED BY public.test_profile."ID";


--
-- TOC entry 3202 (class 2604 OID 16606)
-- Name: Agent_List id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Agent_List" ALTER COLUMN id SET DEFAULT nextval('public."Agent_List_id_seq"'::regclass);


--
-- TOC entry 3198 (class 2604 OID 16567)
-- Name: Test_Results id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Test_Results" ALTER COLUMN id SET DEFAULT nextval('public."Test_Results_id_seq"'::regclass);


--
-- TOC entry 3207 (class 2604 OID 16634)
-- Name: agent-group ID; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."agent-group" ALTER COLUMN "ID" SET DEFAULT nextval('public."agent-group_ID_seq"'::regclass);


--
-- TOC entry 3203 (class 2604 OID 16625)
-- Name: test Id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test ALTER COLUMN "Id" SET DEFAULT nextval('public."planned_test_Id_seq"'::regclass);


--
-- TOC entry 3208 (class 2604 OID 16646)
-- Name: test_profile ID; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test_profile ALTER COLUMN "ID" SET DEFAULT nextval('public."test_profile_ID_seq"'::regclass);


--
-- TOC entry 3366 (class 0 OID 16590)
-- Dependencies: 216
-- Data for Name: Agent_List; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."Agent_List" ("Name", "Address", "Test_health", "Availability", id) FROM stdin;
Agent	127.0.0.1	f	100	1
\.


--
-- TOC entry 3365 (class 0 OID 16564)
-- Dependencies: 215
-- Data for Name: Test_Results; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."Test_Results" (id, "PacketLossPercent", "AvgLatencyMs ", "AvgJitterMs ", "AvgThroughputKbps") FROM stdin;
\.


--
-- TOC entry 3374 (class 0 OID 16651)
-- Dependencies: 224
-- Data for Name: Threshold; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."Threshold" ("ID", "Name", creation_date, avg, min, max, avg_status, min_status, max_status, avg_opr, min_opr, max_opr, selected_metric) FROM stdin;
6	demp	2025-05-06 15:07:25.56+00	15	0	0	t	f	f	=			bandwidth
7	ttt	2025-05-07 07:37:26.085+00	0	77	0	f	t	f		<		jitter
\.


--
-- TOC entry 3371 (class 0 OID 16631)
-- Dependencies: 221
-- Data for Name: agent-group; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."agent-group" ("ID", group_name, number_of_agents, creation_date) FROM stdin;
12	f	5	2025-04-27 09:57:47.608+00
13	m	1	2025-04-27 09:58:45.035+00
15	jjj	5	2025-04-28 08:43:27.149+00
\.


--
-- TOC entry 3369 (class 0 OID 16622)
-- Dependencies: 219
-- Data for Name: test; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.test ("Id", test_name, test_duration, number_of_agents, creation_date, source_id, target_id, profile_id, threshold_id, test_type, waiting, failed, completed) FROM stdin;
17	lah	00:01:17	0	2025-05-08 18:05:03.951+00	0	0	3	0	planned_test	t	f	f
18	azss	00:00:01	0	2025-05-08 20:20:00.151+00	0	0	2	0	planned_test	t	f	f
19	ctoutta	00:01:18	0	2025-05-08 20:23:45.765+00	0	0	2	5	planned_test	t	f	f
20	q	00:00:04	0	2025-05-08 20:53:53.477+00	1	1	2	5	planned_test	t	f	f
22	zz1	00:01:17	0	2025-05-10 09:48:50.031+00	1	1	2	0	quick_test	t	f	f
24	qicl	00:00:12	0	2025-05-15 09:19:49.559+00	1	1	2	0	quick_test	t	f	f
\.


--
-- TOC entry 3373 (class 0 OID 16643)
-- Dependencies: 223
-- Data for Name: test_profile; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.test_profile ("ID", profile_name, creation_date, packet_size) FROM stdin;
1		0001-01-01 00:00:00+00	0
2	aaaa	2025-04-27 20:48:54.624+00	7
3	s55	2025-04-27 20:49:30.448+00	5
\.


--
-- TOC entry 3386 (class 0 OID 0)
-- Dependencies: 217
-- Name: Agent_List_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."Agent_List_id_seq"', 1, true);


--
-- TOC entry 3387 (class 0 OID 0)
-- Dependencies: 214
-- Name: Test_Results_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."Test_Results_id_seq"', 1, false);


--
-- TOC entry 3388 (class 0 OID 0)
-- Dependencies: 220
-- Name: agent-group_ID_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."agent-group_ID_seq"', 15, true);


--
-- TOC entry 3389 (class 0 OID 0)
-- Dependencies: 218
-- Name: planned_test_Id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."planned_test_Id_seq"', 24, true);


--
-- TOC entry 3390 (class 0 OID 0)
-- Dependencies: 222
-- Name: test_profile_ID_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."test_profile_ID_seq"', 3, true);


--
-- TOC entry 3391 (class 0 OID 0)
-- Dependencies: 225
-- Name: threshold_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.threshold_id_seq', 7, true);


--
-- TOC entry 3213 (class 2606 OID 16608)
-- Name: Agent_List Agent_List_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Agent_List"
    ADD CONSTRAINT "Agent_List_pkey" PRIMARY KEY (id);


--
-- TOC entry 3211 (class 2606 OID 16569)
-- Name: Test_Results Test_Results_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Test_Results"
    ADD CONSTRAINT "Test_Results_pkey" PRIMARY KEY (id);


--
-- TOC entry 3221 (class 2606 OID 16657)
-- Name: Threshold Threshold_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Threshold"
    ADD CONSTRAINT "Threshold_pkey" PRIMARY KEY ("ID");


--
-- TOC entry 3217 (class 2606 OID 16638)
-- Name: agent-group agent-group_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."agent-group"
    ADD CONSTRAINT "agent-group_pkey" PRIMARY KEY ("ID");


--
-- TOC entry 3215 (class 2606 OID 16629)
-- Name: test planned_test_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test
    ADD CONSTRAINT planned_test_pkey PRIMARY KEY ("Id");


--
-- TOC entry 3219 (class 2606 OID 16650)
-- Name: test_profile test_profile_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test_profile
    ADD CONSTRAINT test_profile_pkey PRIMARY KEY ("ID");


-- Completed on 2025-05-15 17:40:04

--
-- PostgreSQL database dump complete
--

