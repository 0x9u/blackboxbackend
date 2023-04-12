--
-- PostgreSQL database dump
--

-- Dumped from database version 14.4
-- Dumped by pg_dump version 14.4

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
-- Name: bannedips; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.bannedips (
    ip character varying(128)
);


ALTER TABLE public.bannedips OWNER TO postgres;

--
-- Name: blocked; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.blocked (
    user_id bigint,
    blocked_id bigint
);


ALTER TABLE public.blocked OWNER TO postgres;

--
-- Name: directmsgfiles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.directmsgfiles (
    directmsg_id bigint,
    file_id bigint
);


ALTER TABLE public.directmsgfiles OWNER TO postgres;

--
-- Name: directmsgmentions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.directmsgmentions (
    directmsg_id bigint,
    user_id bigint
);


ALTER TABLE public.directmsgmentions OWNER TO postgres;

--
-- Name: directmsgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.directmsgs (
    id bigint NOT NULL,
    content character varying(1024) NOT NULL,
    user_id bigint NOT NULL,
    dm_id bigint NOT NULL,
    created bigint NOT NULL,
    modified bigint,
    mentions_everyone boolean DEFAULT false NOT NULL,
    CONSTRAINT not_same CHECK ((user_id <> dm_id))
);


ALTER TABLE public.directmsgs OWNER TO postgres;

--
-- Name: directmsgsguild; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.directmsgsguild (
    id bigint NOT NULL
);


ALTER TABLE public.directmsgsguild OWNER TO postgres;

--
-- Name: TABLE directmsgsguild; Type: COMMENT; Schema: public; Owner: postgres
--

COMMENT ON TABLE public.directmsgsguild IS 'This is referenced by userdirectmsgsguild because if the user close a dm then it would be gone for the other user therefore this table keeps track of the dms yes I know this looks confusing as but it just works I think';


--
-- Name: files; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.files (
    id bigint NOT NULL,
    filename character varying(4096),
    created bigint,
    temp boolean DEFAULT false NOT NULL,
    filesize integer
);


ALTER TABLE public.files OWNER TO postgres;

--
-- Name: friends; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.friends (
    user_id bigint,
    friend_id bigint,
    friended boolean DEFAULT false NOT NULL,
    CONSTRAINT not_same CHECK ((user_id <> friend_id))
);


ALTER TABLE public.friends OWNER TO postgres;

--
-- Name: guilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guilds (
    id bigint NOT NULL,
    name character varying(16) NOT NULL,
    save_chat boolean DEFAULT true NOT NULL,
    image_id bigint
);


ALTER TABLE public.guilds OWNER TO postgres;

--
-- Name: invites; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.invites (
    invite character varying(10) NOT NULL,
    guild_id bigint NOT NULL
);


ALTER TABLE public.invites OWNER TO postgres;

--
-- Name: msgfiles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.msgfiles (
    msg_id bigint,
    file_id bigint
);


ALTER TABLE public.msgfiles OWNER TO postgres;

--
-- Name: msgmentions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.msgmentions (
    msg_id bigint,
    user_id bigint
);


ALTER TABLE public.msgmentions OWNER TO postgres;

--
-- Name: msgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.msgs (
    id bigint NOT NULL,
    content character varying(1024) NOT NULL,
    user_id bigint NOT NULL,
    guild_id bigint NOT NULL,
    created bigint NOT NULL,
    modified bigint DEFAULT 0,
    mentions_everyone boolean DEFAULT false NOT NULL
);


ALTER TABLE public.msgs OWNER TO postgres;

--
-- Name: permissions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.permissions (
    id integer NOT NULL,
    name character varying(64)
);


ALTER TABLE public.permissions OWNER TO postgres;

--
-- Name: permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.permissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.permissions_id_seq OWNER TO postgres;

--
-- Name: permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.permissions_id_seq OWNED BY public.permissions.id;


--
-- Name: rolepermissions; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.rolepermissions (
    role_id integer,
    permission_id integer
);


ALTER TABLE public.rolepermissions OWNER TO postgres;

--
-- Name: roles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.roles (
    id integer NOT NULL,
    name character varying(64)
);


ALTER TABLE public.roles OWNER TO postgres;

--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.roles_id_seq OWNER TO postgres;

--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;


--
-- Name: tokens; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.tokens (
    token character varying(64) NOT NULL,
    token_expires bigint NOT NULL,
    user_id bigint NOT NULL
);


ALTER TABLE public.tokens OWNER TO postgres;

--
-- Name: unreaddirectmsgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.unreaddirectmsgs (
    msg_id bigint DEFAULT 0,
    "time" bigint DEFAULT 0,
    dm_id bigint,
    user_id bigint
);


ALTER TABLE public.unreaddirectmsgs OWNER TO postgres;

--
-- Name: unreadmsgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.unreadmsgs (
    guild_id bigint,
    user_id bigint,
    msg_id bigint DEFAULT 0,
    "time" bigint DEFAULT 0
);


ALTER TABLE public.unreadmsgs OWNER TO postgres;

--
-- Name: userdirectmsgsguild; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.userdirectmsgsguild (
    dm_id bigint,
    user_id bigint,
    left_dm boolean DEFAULT false NOT NULL,
    receiver_id bigint
);


ALTER TABLE public.userdirectmsgsguild OWNER TO postgres;

--
-- Name: userguilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.userguilds (
    guild_id bigint NOT NULL,
    user_id bigint NOT NULL,
    banned boolean DEFAULT false NOT NULL,
    owner boolean DEFAULT false NOT NULL
);


ALTER TABLE public.userguilds OWNER TO postgres;

--
-- Name: userroles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.userroles (
    user_id bigint,
    role_id integer
);


ALTER TABLE public.userroles OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    email character varying(128) NOT NULL,
    password character varying(64) NOT NULL,
    username character varying(32),
    flags integer DEFAULT 0,
    image_id bigint,
    options integer DEFAULT 15
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: permissions id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permissions ALTER COLUMN id SET DEFAULT nextval('public.permissions_id_seq'::regclass);


--
-- Name: roles id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);


--
-- Data for Name: bannedips; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.bannedips (ip) FROM stdin;
\.


--
-- Data for Name: blocked; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.blocked (user_id, blocked_id) FROM stdin;
\.


--
-- Data for Name: directmsgfiles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.directmsgfiles (directmsg_id, file_id) FROM stdin;
\.


--
-- Data for Name: directmsgmentions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.directmsgmentions (directmsg_id, user_id) FROM stdin;
\.


--
-- Data for Name: directmsgs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.directmsgs (id, content, user_id, dm_id, created, modified, mentions_everyone) FROM stdin;
\.


--
-- Data for Name: directmsgsguild; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.directmsgsguild (id) FROM stdin;
\.


--
-- Data for Name: files; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.files (id, filename, created, temp, filesize) FROM stdin;
\.


--
-- Data for Name: friends; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.friends (user_id, friend_id, friended) FROM stdin;
\.


--
-- Data for Name: guilds; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.guilds (id, name, save_chat, image_id) FROM stdin;
\.


--
-- Data for Name: invites; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.invites (invite, guild_id) FROM stdin;
\.


--
-- Data for Name: msgfiles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.msgfiles (msg_id, file_id) FROM stdin;
\.


--
-- Data for Name: msgmentions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.msgmentions (msg_id, user_id) FROM stdin;
\.


--
-- Data for Name: msgs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.msgs (id, content, user_id, guild_id, created, modified, mentions_everyone) FROM stdin;
\.


--
-- Data for Name: permissions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.permissions (id, name) FROM stdin;
\.


--
-- Data for Name: rolepermissions; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.rolepermissions (role_id, permission_id) FROM stdin;
\.


--
-- Data for Name: roles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.roles (id, name) FROM stdin;
\.


--
-- Data for Name: tokens; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.tokens (token, token_expires, user_id) FROM stdin;
\.


--
-- Data for Name: unreaddirectmsgs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.unreaddirectmsgs (msg_id, "time", dm_id, user_id) FROM stdin;
\.


--
-- Data for Name: unreadmsgs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.unreadmsgs (guild_id, user_id, msg_id, "time") FROM stdin;
\.


--
-- Data for Name: userdirectmsgsguild; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.userdirectmsgsguild (dm_id, user_id, left_dm, receiver_id) FROM stdin;
\.


--
-- Data for Name: userguilds; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.userguilds (guild_id, user_id, banned, owner) FROM stdin;
\.


--
-- Data for Name: userroles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.userroles (user_id, role_id) FROM stdin;
\.


--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, email, password, username, flags, image_id, options) FROM stdin;
\.


--
-- Name: permissions_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.permissions_id_seq', 1, false);


--
-- Name: roles_id_seq; Type: SEQUENCE SET; Schema: public; Owner: postgres
--

SELECT pg_catalog.setval('public.roles_id_seq', 1, false);


--
-- Name: directmsgs directmessages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_pkey PRIMARY KEY (id);


--
-- Name: files files_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);


--
-- Name: guilds guilds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_pkey PRIMARY KEY (id);


--
-- Name: bannedips ip_unq; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.bannedips
    ADD CONSTRAINT ip_unq UNIQUE (ip);


--
-- Name: msgs msgs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs
    ADD CONSTRAINT msgs_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: tokens tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_pkey PRIMARY KEY (token);


--
-- Name: tokens tokens_user_id_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_user_id_key UNIQUE (user_id);


--
-- Name: directmsgsguild userdirectmsgs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgsguild
    ADD CONSTRAINT userdirectmsgs_pkey PRIMARY KEY (id);


--
-- Name: userguilds userguilds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT userguilds_pkey PRIMARY KEY (guild_id, user_id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: directmsgmentions directmsgmentions_directmsg_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgmentions
    ADD CONSTRAINT directmsgmentions_directmsg_id_fkey FOREIGN KEY (directmsg_id) REFERENCES public.directmsgs(id) ON DELETE CASCADE;


--
-- Name: directmsgmentions directmsgmentions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgmentions
    ADD CONSTRAINT directmsgmentions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: directmsgs directmsgs_dm_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmsgs_dm_id_fkey FOREIGN KEY (dm_id) REFERENCES public.directmsgsguild(id) ON DELETE CASCADE;


--
-- Name: directmsgs directmsgs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmsgs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: blocked fk_blocked_blocked_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocked
    ADD CONSTRAINT fk_blocked_blocked_id FOREIGN KEY (blocked_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: blocked fk_blocked_user_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocked
    ADD CONSTRAINT fk_blocked_user_id FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: directmsgfiles fk_directmsg_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgfiles
    ADD CONSTRAINT fk_directmsg_id FOREIGN KEY (directmsg_id) REFERENCES public.directmsgs(id) ON DELETE CASCADE;


--
-- Name: msgfiles fk_file_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgfiles
    ADD CONSTRAINT fk_file_id FOREIGN KEY (file_id) REFERENCES public.files(id) ON DELETE CASCADE;


--
-- Name: directmsgfiles fk_file_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgfiles
    ADD CONSTRAINT fk_file_id FOREIGN KEY (file_id) REFERENCES public.files(id) ON DELETE CASCADE;


--
-- Name: guilds fk_image_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT fk_image_id FOREIGN KEY (image_id) REFERENCES public.files(id) ON DELETE SET NULL;


--
-- Name: users fk_image_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT fk_image_id FOREIGN KEY (image_id) REFERENCES public.files(id) ON DELETE SET NULL;


--
-- Name: invites fk_invite_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.invites
    ADD CONSTRAINT fk_invite_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id) ON DELETE CASCADE;


--
-- Name: msgs fk_msg_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs
    ADD CONSTRAINT fk_msg_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id) ON DELETE CASCADE;


--
-- Name: msgfiles fk_msg_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgfiles
    ADD CONSTRAINT fk_msg_id FOREIGN KEY (file_id) REFERENCES public.files(id) ON DELETE CASCADE;


--
-- Name: msgs fk_msg_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs
    ADD CONSTRAINT fk_msg_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: tokens fk_token_user_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT fk_token_user_id FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: unreadmsgs fk_unreadmsg_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreadmsgs
    ADD CONSTRAINT fk_unreadmsg_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id) ON DELETE CASCADE;


--
-- Name: unreadmsgs fk_unreadmsg_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreadmsgs
    ADD CONSTRAINT fk_unreadmsg_user FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: userguilds fk_userguild_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT fk_userguild_guild FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: userguilds fk_userguild_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT fk_userguild_user FOREIGN KEY (guild_id) REFERENCES public.guilds(id) ON DELETE CASCADE;


--
-- Name: userroles fk_userrole_role_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT fk_userrole_role_id FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: userroles fk_userrole_user_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT fk_userrole_user_id FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: friends friend_friend_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friend_friend_id_fkey FOREIGN KEY (friend_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: friends friend_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friend_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: msgmentions msgmentions_msg_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgmentions
    ADD CONSTRAINT msgmentions_msg_id_fkey FOREIGN KEY (msg_id) REFERENCES public.msgs(id) ON DELETE CASCADE;


--
-- Name: msgmentions msgmentions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgmentions
    ADD CONSTRAINT msgmentions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: rolepermissions rolepermission_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.rolepermissions
    ADD CONSTRAINT rolepermission_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id) ON DELETE CASCADE;


--
-- Name: rolepermissions rolepermission_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.rolepermissions
    ADD CONSTRAINT rolepermission_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id) ON DELETE CASCADE;


--
-- Name: unreaddirectmsgs unreaddirectmsg_dm_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreaddirectmsgs
    ADD CONSTRAINT unreaddirectmsg_dm_id_fkey FOREIGN KEY (dm_id) REFERENCES public.directmsgsguild(id) ON DELETE CASCADE;


--
-- Name: unreaddirectmsgs unreaddirectmsg_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreaddirectmsgs
    ADD CONSTRAINT unreaddirectmsg_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: userdirectmsgsguild userdirectmsgsguild_dm_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userdirectmsgsguild
    ADD CONSTRAINT userdirectmsgsguild_dm_id_fkey FOREIGN KEY (dm_id) REFERENCES public.directmsgsguild(id) ON DELETE CASCADE;


--
-- Name: userdirectmsgsguild userdirectmsgsguild_receiver_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userdirectmsgsguild
    ADD CONSTRAINT userdirectmsgsguild_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: userdirectmsgsguild userdirectmsgsguild_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userdirectmsgsguild
    ADD CONSTRAINT userdirectmsgsguild_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--

