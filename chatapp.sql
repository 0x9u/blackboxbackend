--
-- PostgreSQL database dump
--

-- Dumped from database version 14.2
-- Dumped by pg_dump version 14.2

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
-- Name: bannedips; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.bannedips (
    ip character varying(128)
);


ALTER TABLE public.bannedips OWNER TO oliver;

--
-- Name: blocked; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.blocked (
    user_id integer,
    blocked_id integer
);


ALTER TABLE public.blocked OWNER TO oliver;

--
-- Name: directmsgs; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.directmsgs (
    id integer NOT NULL,
    content character varying(1024) NOT NULL,
    sender_id integer NOT NULL,
    receiver_id integer NOT NULL,
    created bigint NOT NULL,
    modified bigint,
    CONSTRAINT not_same CHECK ((sender_id <> receiver_id))
);


ALTER TABLE public.directmsgs OWNER TO oliver;

--
-- Name: directmsgs_id_seq; Type: SEQUENCE; Schema: public; Owner: oliver
--

CREATE SEQUENCE public.directmsgs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.directmsgs_id_seq OWNER TO oliver;

--
-- Name: directmsgs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: oliver
--

ALTER SEQUENCE public.directmsgs_id_seq OWNED BY public.directmsgs.id;


--
-- Name: friends; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.friends (
    user_id integer,
    friend_id integer,
    friended boolean DEFAULT false NOT NULL,
    CONSTRAINT not_same CHECK ((user_id <> friend_id))
);


ALTER TABLE public.friends OWNER TO oliver;

--
-- Name: guilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guilds (
    id integer NOT NULL,
    name character varying(16) NOT NULL,
    icon integer DEFAULT 0,
    save_chat boolean DEFAULT true NOT NULL
);


ALTER TABLE public.guilds OWNER TO postgres;

--
-- Name: guilds_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.guilds_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.guilds_id_seq OWNER TO postgres;

--
-- Name: guilds_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.guilds_id_seq OWNED BY public.guilds.id;


--
-- Name: invites; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.invites (
    invite character varying(10) NOT NULL,
    guild_id integer NOT NULL
);


ALTER TABLE public.invites OWNER TO postgres;

--
-- Name: msgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.msgs (
    id integer NOT NULL,
    content character varying(1024) NOT NULL,
    user_id integer NOT NULL,
    guild_id integer NOT NULL,
    created bigint NOT NULL,
    modified bigint DEFAULT 0
);


ALTER TABLE public.msgs OWNER TO postgres;

--
-- Name: msgs_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.msgs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.msgs_id_seq OWNER TO postgres;

--
-- Name: msgs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.msgs_id_seq OWNED BY public.msgs.id;


--
-- Name: permissions; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.permissions (
    id integer NOT NULL,
    name character varying(64)
);


ALTER TABLE public.permissions OWNER TO oliver;

--
-- Name: permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: oliver
--

CREATE SEQUENCE public.permissions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.permissions_id_seq OWNER TO oliver;

--
-- Name: permissions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: oliver
--

ALTER SEQUENCE public.permissions_id_seq OWNED BY public.permissions.id;


--
-- Name: rolepermissions; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.rolepermissions (
    role_id integer,
    permission_id integer
);


ALTER TABLE public.rolepermissions OWNER TO oliver;

--
-- Name: roles; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.roles (
    id integer NOT NULL,
    name character varying(64)
);


ALTER TABLE public.roles OWNER TO oliver;

--
-- Name: roles_id_seq; Type: SEQUENCE; Schema: public; Owner: oliver
--

CREATE SEQUENCE public.roles_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.roles_id_seq OWNER TO oliver;

--
-- Name: roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: oliver
--

ALTER SEQUENCE public.roles_id_seq OWNED BY public.roles.id;


--
-- Name: tokens; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.tokens (
    token character varying(32) NOT NULL,
    token_expires bigint NOT NULL,
    user_id integer NOT NULL
);


ALTER TABLE public.tokens OWNER TO oliver;

--
-- Name: unreaddirectmsgs; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.unreaddirectmsgs (
    receiver_id integer,
    sender_id integer,
    msg_id integer DEFAULT 0,
    "time" bigint DEFAULT 0,
    CONSTRAINT not_same CHECK ((sender_id <> receiver_id))
);


ALTER TABLE public.unreaddirectmsgs OWNER TO oliver;

--
-- Name: unreadmsgs; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.unreadmsgs (
    guild_id integer,
    user_id integer,
    msg_id integer DEFAULT 0,
    "time" bigint DEFAULT 0
);


ALTER TABLE public.unreadmsgs OWNER TO oliver;

--
-- Name: userdirectmsgs; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.userdirectmsgs (
    sender_id integer NOT NULL,
    receiver_id integer NOT NULL,
    CONSTRAINT not_same CHECK ((sender_id <> receiver_id))
);


ALTER TABLE public.userdirectmsgs OWNER TO oliver;

--
-- Name: userguilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.userguilds (
    guild_id integer NOT NULL,
    user_id integer NOT NULL,
    banned boolean DEFAULT false NOT NULL,
    owner boolean DEFAULT false NOT NULL
);


ALTER TABLE public.userguilds OWNER TO postgres;

--
-- Name: userroles; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.userroles (
    user_id integer,
    role_id integer
);


ALTER TABLE public.userroles OWNER TO oliver;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id integer NOT NULL,
    email character varying(128) NOT NULL,
    password character varying(64) NOT NULL,
    username character varying(32),
    flags integer DEFAULT 0
);


ALTER TABLE public.users OWNER TO postgres;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.users_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_id_seq OWNER TO postgres;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: directmsgs id; Type: DEFAULT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.directmsgs ALTER COLUMN id SET DEFAULT nextval('public.directmsgs_id_seq'::regclass);


--
-- Name: guilds id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds ALTER COLUMN id SET DEFAULT nextval('public.guilds_id_seq'::regclass);


--
-- Name: msgs id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs ALTER COLUMN id SET DEFAULT nextval('public.msgs_id_seq'::regclass);


--
-- Name: permissions id; Type: DEFAULT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.permissions ALTER COLUMN id SET DEFAULT nextval('public.permissions_id_seq'::regclass);


--
-- Name: roles id; Type: DEFAULT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.roles ALTER COLUMN id SET DEFAULT nextval('public.roles_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: directmsgs directmessages_pkey; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_pkey PRIMARY KEY (id);


--
-- Name: guilds guilds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_pkey PRIMARY KEY (id);


--
-- Name: bannedips ip_unq; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.bannedips
    ADD CONSTRAINT ip_unq UNIQUE (ip);


--
-- Name: msgs msgs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs
    ADD CONSTRAINT msgs_pkey PRIMARY KEY (id);


--
-- Name: permissions permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.permissions
    ADD CONSTRAINT permissions_pkey PRIMARY KEY (id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (id);


--
-- Name: tokens tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_pkey PRIMARY KEY (token);


--
-- Name: tokens tokens_user_id_key; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT tokens_user_id_key UNIQUE (user_id);


--
-- Name: userdirectmsgs userdirectmsgs_pkey; Type: CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.userdirectmsgs
    ADD CONSTRAINT userdirectmsgs_pkey PRIMARY KEY (sender_id, receiver_id);


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
-- Name: directmsgs directmessages_receiver_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public.users(id);


--
-- Name: directmsgs directmessages_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public.users(id);


--
-- Name: blocked fk_blocked_id; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.blocked
    ADD CONSTRAINT fk_blocked_id FOREIGN KEY (blocked_id) REFERENCES public.users(id);


--
-- Name: userguilds fk_guild_userguild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT fk_guild_userguild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: invites fk_invite_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.invites
    ADD CONSTRAINT fk_invite_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: msgs fk_msg_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs
    ADD CONSTRAINT fk_msg_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: msgs fk_msg_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.msgs
    ADD CONSTRAINT fk_msg_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: unreadmsgs fk_unreadmsg_guild; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.unreadmsgs
    ADD CONSTRAINT fk_unreadmsg_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: unreadmsgs fk_unreadmsg_user; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.unreadmsgs
    ADD CONSTRAINT fk_unreadmsg_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: tokens fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: blocked fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.blocked
    ADD CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: userguilds fk_user_userguild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT fk_user_userguild FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: friends friends_friend_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_friend_id_fkey FOREIGN KEY (friend_id) REFERENCES public.users(id);


--
-- Name: friends friends_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: rolepermissions rolepermissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.rolepermissions
    ADD CONSTRAINT rolepermissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id);


--
-- Name: rolepermissions rolepermissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.rolepermissions
    ADD CONSTRAINT rolepermissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: unreaddirectmsgs unreaddirectmsgs_receiver_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.unreaddirectmsgs
    ADD CONSTRAINT unreaddirectmsgs_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public.users(id);


--
-- Name: unreaddirectmsgs unreaddirectmsgs_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.unreaddirectmsgs
    ADD CONSTRAINT unreaddirectmsgs_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public.users(id);


--
-- Name: userdirectmsgs userdirectmessages_receiver_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.userdirectmsgs
    ADD CONSTRAINT userdirectmessages_receiver_id_fkey FOREIGN KEY (receiver_id) REFERENCES public.users(id);


--
-- Name: userdirectmsgs userdirectmessages_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.userdirectmsgs
    ADD CONSTRAINT userdirectmessages_sender_id_fkey FOREIGN KEY (sender_id) REFERENCES public.users(id);


--
-- Name: userroles userroles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT userroles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: userroles userroles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT userroles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

