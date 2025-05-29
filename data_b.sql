--
-- PostgreSQL database dump
--

-- Dumped from database version 15.12
-- Dumped by pg_dump version 15.12

-- Started on 2025-05-29 16:08:12

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
    id integer NOT NULL,
    "Port" integer DEFAULT 0 NOT NULL
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
-- TOC entry 3413 (class 0 OID 0)
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
-- TOC entry 3414 (class 0 OID 0)
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
-- TOC entry 3415 (class 0 OID 0)
-- Dependencies: 220
-- Name: agent-group_ID_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."agent-group_ID_seq" OWNED BY public."agent-group"."ID";


--
-- TOC entry 228 (class 1259 OID 41205)
-- Name: agent_health_checks; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.agent_health_checks (
    id integer NOT NULL,
    agent_id integer,
    "timestamp" timestamp without time zone NOT NULL,
    status character varying(10) NOT NULL
);


ALTER TABLE public.agent_health_checks OWNER TO postgres;

--
-- TOC entry 227 (class 1259 OID 41204)
-- Name: agent_health_checks_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.agent_health_checks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.agent_health_checks_id_seq OWNER TO postgres;

--
-- TOC entry 3416 (class 0 OID 0)
-- Dependencies: 227
-- Name: agent_health_checks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.agent_health_checks_id_seq OWNED BY public.agent_health_checks.id;


--
-- TOC entry 226 (class 1259 OID 41165)
-- Name: agent_link; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.agent_link (
    group_id integer NOT NULL,
    agent_id integer NOT NULL
);


ALTER TABLE public.agent_link OWNER TO postgres;

--
-- TOC entry 230 (class 1259 OID 41232)
-- Name: attempt_results; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.attempt_results (
    id integer NOT NULL,
    latency_ms double precision,
    jitter_ms double precision,
    throughput_kbps double precision,
    test_id integer DEFAULT 0 NOT NULL
);


ALTER TABLE public.attempt_results OWNER TO postgres;

--
-- TOC entry 229 (class 1259 OID 41231)
-- Name: attempt_results_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.attempt_results_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.attempt_results_id_seq OWNER TO postgres;

--
-- TOC entry 3417 (class 0 OID 0)
-- Dependencies: 229
-- Name: attempt_results_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.attempt_results_id_seq OWNED BY public.attempt_results.id;


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
    "In_progress" boolean DEFAULT false,
    failed boolean DEFAULT false,
    completed boolean DEFAULT false,
    "Error" boolean DEFAULT false
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
-- TOC entry 3418 (class 0 OID 0)
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
    packet_size integer,
    time_between_attempts integer NOT NULL
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
-- TOC entry 3419 (class 0 OID 0)
-- Dependencies: 222
-- Name: test_profile_ID_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public."test_profile_ID_seq" OWNED BY public.test_profile."ID";


--
-- TOC entry 3215 (class 2604 OID 16606)
-- Name: Agent_List id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Agent_List" ALTER COLUMN id SET DEFAULT nextval('public."Agent_List_id_seq"'::regclass);


--
-- TOC entry 3212 (class 2604 OID 16567)
-- Name: Test_Results id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Test_Results" ALTER COLUMN id SET DEFAULT nextval('public."Test_Results_id_seq"'::regclass);


--
-- TOC entry 3222 (class 2604 OID 16634)
-- Name: agent-group ID; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."agent-group" ALTER COLUMN "ID" SET DEFAULT nextval('public."agent-group_ID_seq"'::regclass);


--
-- TOC entry 3225 (class 2604 OID 41208)
-- Name: agent_health_checks id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.agent_health_checks ALTER COLUMN id SET DEFAULT nextval('public.agent_health_checks_id_seq'::regclass);


--
-- TOC entry 3226 (class 2604 OID 41235)
-- Name: attempt_results id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attempt_results ALTER COLUMN id SET DEFAULT nextval('public.attempt_results_id_seq'::regclass);


--
-- TOC entry 3217 (class 2604 OID 16625)
-- Name: test Id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test ALTER COLUMN "Id" SET DEFAULT nextval('public."planned_test_Id_seq"'::regclass);


--
-- TOC entry 3223 (class 2604 OID 16646)
-- Name: test_profile ID; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test_profile ALTER COLUMN "ID" SET DEFAULT nextval('public."test_profile_ID_seq"'::regclass);


--
-- TOC entry 3393 (class 0 OID 16590)
-- Dependencies: 216
-- Data for Name: Agent_List; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."Agent_List" ("Name", "Address", "Test_health", id, "Port") FROM stdin;
agentP	127.0.0.1	t	10	8081
agen2	127.0.0.1	t	9	8080
agent46	127.0.0.1	t	7	8081
agen1	127.0.0.1	t	8	8080
\.


--
-- TOC entry 3392 (class 0 OID 16564)
-- Dependencies: 215
-- Data for Name: Test_Results; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."Test_Results" (id, "PacketLossPercent", "AvgLatencyMs ", "AvgJitterMs ", "AvgThroughputKbps") FROM stdin;
\.


--
-- TOC entry 3401 (class 0 OID 16651)
-- Dependencies: 224
-- Data for Name: Threshold; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."Threshold" ("ID", "Name", creation_date, avg, min, max, avg_status, min_status, max_status, avg_opr, min_opr, max_opr, selected_metric) FROM stdin;
6	demp	2025-05-06 15:07:25.56+00	15	0	0	t	f	f	=			bandwidth
7	ttt	2025-05-07 07:37:26.085+00	0	77	0	f	t	f		<		jitter
8	lat	2025-05-21 17:37:56.434+00	0	0	3	f	f	t			=	latency
\.


--
-- TOC entry 3398 (class 0 OID 16631)
-- Dependencies: 221
-- Data for Name: agent-group; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public."agent-group" ("ID", group_name, number_of_agents, creation_date) FROM stdin;
31	grou7	2	2025-05-26 12:08:57.65+00
\.


--
-- TOC entry 3405 (class 0 OID 41205)
-- Dependencies: 228
-- Data for Name: agent_health_checks; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.agent_health_checks (id, agent_id, "timestamp", status) FROM stdin;
\.


--
-- TOC entry 3403 (class 0 OID 41165)
-- Dependencies: 226
-- Data for Name: agent_link; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.agent_link (group_id, agent_id) FROM stdin;
31	10
31	7
\.


--
-- TOC entry 3407 (class 0 OID 41232)
-- Dependencies: 230
-- Data for Name: attempt_results; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.attempt_results (id, latency_ms, jitter_ms, throughput_kbps, test_id) FROM stdin;
1	0	0	6206.896551724139	0
2	0	0	0.14305024446832654	0
3	0	0	0.10725317317237791	0
4	0	0	0.09537638784649156	0
5	0	0	0.08940971958585758	0
6	0.1126	0	646.734248684717	0
7	0	0.1126	0.14320549794320395	0
8	0	0	0.10728314551628293	0
9	0.514	0.514	0.09527603956898992	0
10	0	0.514	0.08932630452055366	0
11	0.5162	0	976.365749709415	0
12	0	0.5162	0.14314011845611627	0
13	0.1547	0.1547	0.1071927467375689	0
14	0	0.1547	0.09525597598358707	0
15	0.5149	0.5149	0.08928809865504381	0
16	0	0	2042.9671665991084	0
17	0	0	0.14313434393564067	0
18	0.5297	0	951.4819709269398	0
19	0.6012	0.0715	0.1431023922886891	0
20	0	0.33635	0.10729898583510748	0
21	0.5122	0.3949666666666667	0.09533454805751909	0
22	0	0.424275	0.08937539621319	0
23	0.5083	0	991.5404288805822	10
24	0	0.5083	0.1432525491279501	10
25	0	0.25415	0.10726984100970718	10
26	0	0.16943333333333335	0.09530412693295714	10
27	0.5114	0.254925	0.0893466494350544	10
28	0.0808	0	352.3982659767865	10
29	1.6146	1.5338	0.1429908522737106	10
30	0	1.5742	0.10725968826858581	10
31	0.5798	1.2427333333333332	0.09529020189039594	10
32	0.5742	0.93345	0.08929954014776564	10
33	0.6487	0	343.1839847473785	10
34	0	0.6487	0.14323701122732335	10
35	0.5117	0.5802	0.10739402308792506	10
36	0.5116	0.3868333333333333	0.09539884689199146	10
37	0	0.418025	0.08942792033146217	10
38	0	0	707.3684210526316	10
39	0	0	0.14315540153662837	10
40	0	0	0.10736843432792673	10
41	0.1157	0.038566666666666666	0.09545117845038137	10
42	0	0.05785	0.0894860328158137	10
43	0.5032	0	1001.5898251192369	10
44	0.5178	0.0146	0.14331615060838346	10
45	0	0.2662	0.10739921798026565	10
46	0	0.17746666666666666	0.0948897876767574	10
47	0.6724	0.3012	0.08908333597254499	10
48	0.505	0	998.0198019801979	10
49	0	0.505	0.14314399075562476	10
50	0.5118	0.5084	0.10735396759425346	10
51	0	0.5095333333333333	0.09544644742002216	10
52	0	0.38215	0.08943388288254008	10
53	0	0	Infinity	10
54	0.2316	0.2316	0.14319223216233906	10
55	0.2028	0.1302	0.10733934016207475	10
56	0.4062	0.1546	0.0954083961603459	10
57	0.2274	0.16065	0.0894390053383161	10
58	0	0	Infinity	10
59	0	0	0.14321820659177992	10
60	0	0	0.10739826287571634	10
61	0	0	0.09541271610418181	10
62	0.5137	0.128425	0.0894352648530255	10
63	0	0	Infinity	10
64	0	0	0.14321173397386108	10
65	0	0	0.10739751909599986	10
66	0	0	0.09545325240646242	10
67	0.084	0.021	0.0895020772490937	10
68	0	0	2865.2643547470157	10
69	0.5142	0	494.2145518729163	10
70	0.5126	0.0016	0.14314687123187697	10
71	0	0.2571	0.10728594003422677	10
72	0.5126	0.34226666666666666	0.09534366690948404	10
73	0	0	995.2606635071091	10
74	0	0	0.14301419701082452	10
75	0	0	0.10721704522970199	10
76	0	0	0.0951778766164146	10
77	0	0	991.1504424778763	1
78	0.8979	0.8979	0.1429845827299221	1
79	0	0	994.475138121547	1
80	0	0	863.1615002568932	10
81	0	0	485.0818094321463	10
82	0.6324	0.6324	0.1431046920706457	10
83	0	0.6324	0.10726597281851873	10
84	0	0.4216	0.09535655711063844	10
85	0	0.3162	0.08939999449125748	10
86	0	0	Infinity	10
87	0.5128	0.5128	0.1431052832809544	10
88	0	0.5128	0.107271857785044	10
89	0.5255	0.5170333333333333	0.09533138876530459	10
90	0	0	966.6283084004604	14
91	0.2682	0.2682	0.143943101348367	14
92	0.519	0.2595	0.10793858834016386	14
93	0	0.346	0.09595908624426167	14
94	0.5165	0.388625	0.08996580977921927	14
95	0.1004	0.39412	0.08636805566416618	14
96	0	0.3451666666666667	0.08396496401943024	14
97	0.5357	0.37238571428571426	0.08225263009567717	14
98	0	0	968.8581314878894	14
99	0.5112	0.5112	0.14395365720478415	14
100	0.1405	0.44095	0.107968938878698	14
101	0	0.3408	0.09597559294968626	14
102	0.1013	0.280925	0.08997941528062423	14
103	0.5188	0.30824	0.0863793104275664	14
104	0	0.3433333333333333	0.08397990480849225	14
105	0.1399	0.3142714285714286	0.0822653510929917	14
\.


--
-- TOC entry 3396 (class 0 OID 16622)
-- Dependencies: 219
-- Data for Name: test; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.test ("Id", test_name, test_duration, number_of_agents, creation_date, source_id, target_id, profile_id, threshold_id, test_type, "In_progress", failed, completed, "Error") FROM stdin;
14	test_k	00:00:50	1	2025-05-27 22:34:15.19+00	7	10	24	8	planned_test	f	f	t	f
\.


--
-- TOC entry 3400 (class 0 OID 16643)
-- Dependencies: 223
-- Data for Name: test_profile; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.test_profile ("ID", profile_name, creation_date, packet_size, time_between_attempts) FROM stdin;
24	tag	2025-05-21 17:00:22.434+00	36	7
\.


--
-- TOC entry 3420 (class 0 OID 0)
-- Dependencies: 217
-- Name: Agent_List_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."Agent_List_id_seq"', 10, true);


--
-- TOC entry 3421 (class 0 OID 0)
-- Dependencies: 214
-- Name: Test_Results_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."Test_Results_id_seq"', 1, false);


--
-- TOC entry 3422 (class 0 OID 0)
-- Dependencies: 220
-- Name: agent-group_ID_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."agent-group_ID_seq"', 31, true);


--
-- TOC entry 3423 (class 0 OID 0)
-- Dependencies: 227
-- Name: agent_health_checks_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.agent_health_checks_id_seq', 14424, true);


--
-- TOC entry 3424 (class 0 OID 0)
-- Dependencies: 229
-- Name: attempt_results_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.attempt_results_id_seq', 105, true);


--
-- TOC entry 3425 (class 0 OID 0)
-- Dependencies: 218
-- Name: planned_test_Id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."planned_test_Id_seq"', 14, true);


--
-- TOC entry 3426 (class 0 OID 0)
-- Dependencies: 222
-- Name: test_profile_ID_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public."test_profile_ID_seq"', 24, true);


--
-- TOC entry 3427 (class 0 OID 0)
-- Dependencies: 225
-- Name: threshold_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.threshold_id_seq', 8, true);


--
-- TOC entry 3231 (class 2606 OID 16608)
-- Name: Agent_List Agent_List_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Agent_List"
    ADD CONSTRAINT "Agent_List_pkey" PRIMARY KEY (id);


--
-- TOC entry 3229 (class 2606 OID 16569)
-- Name: Test_Results Test_Results_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Test_Results"
    ADD CONSTRAINT "Test_Results_pkey" PRIMARY KEY (id);


--
-- TOC entry 3239 (class 2606 OID 16657)
-- Name: Threshold Threshold_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."Threshold"
    ADD CONSTRAINT "Threshold_pkey" PRIMARY KEY ("ID");


--
-- TOC entry 3235 (class 2606 OID 16638)
-- Name: agent-group agent-group_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public."agent-group"
    ADD CONSTRAINT "agent-group_pkey" PRIMARY KEY ("ID");


--
-- TOC entry 3243 (class 2606 OID 41210)
-- Name: agent_health_checks agent_health_checks_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.agent_health_checks
    ADD CONSTRAINT agent_health_checks_pkey PRIMARY KEY (id);


--
-- TOC entry 3241 (class 2606 OID 41169)
-- Name: agent_link agent_link_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.agent_link
    ADD CONSTRAINT agent_link_pkey PRIMARY KEY (group_id, agent_id);


--
-- TOC entry 3245 (class 2606 OID 41237)
-- Name: attempt_results attempt_results_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.attempt_results
    ADD CONSTRAINT attempt_results_pkey PRIMARY KEY (id);


--
-- TOC entry 3233 (class 2606 OID 16629)
-- Name: test planned_test_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test
    ADD CONSTRAINT planned_test_pkey PRIMARY KEY ("Id");


--
-- TOC entry 3237 (class 2606 OID 16650)
-- Name: test_profile test_profile_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.test_profile
    ADD CONSTRAINT test_profile_pkey PRIMARY KEY ("ID");


--
-- TOC entry 3248 (class 2606 OID 41211)
-- Name: agent_health_checks agent_health_checks_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.agent_health_checks
    ADD CONSTRAINT agent_health_checks_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public."Agent_List"(id) ON DELETE CASCADE;


--
-- TOC entry 3246 (class 2606 OID 41175)
-- Name: agent_link agent_link_agent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.agent_link
    ADD CONSTRAINT agent_link_agent_id_fkey FOREIGN KEY (agent_id) REFERENCES public."Agent_List"(id) ON DELETE CASCADE;


--
-- TOC entry 3247 (class 2606 OID 41170)
-- Name: agent_link agent_link_group_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.agent_link
    ADD CONSTRAINT agent_link_group_id_fkey FOREIGN KEY (group_id) REFERENCES public."agent-group"("ID") ON DELETE CASCADE;


-- Completed on 2025-05-29 16:08:12

--
-- PostgreSQL database dump complete
--

