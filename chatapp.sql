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
-- Name: directmsgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.directmsgs (
    id bigint NOT NULL,
    content character varying(1024) NOT NULL,
    user_id bigint NOT NULL,
    dm_id bigint NOT NULL,
    created bigint NOT NULL,
    modified bigint,
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
    icon integer DEFAULT 0,
    save_chat boolean DEFAULT true NOT NULL
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
-- Name: msgs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.msgs (
    id bigint NOT NULL,
    content character varying(1024) NOT NULL,
    user_id bigint NOT NULL,
    guild_id bigint NOT NULL,
    created bigint NOT NULL,
    modified bigint DEFAULT 0
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
    left_dm boolean DEFAULT false NOT NULL
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
    flags integer DEFAULT 0
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
-- Name: directmsgs directmessages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_pkey PRIMARY KEY (id);


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
-- Name: directmsgs directmessages_dm_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_dm_id_fkey FOREIGN KEY (dm_id) REFERENCES public.directmsgsguild(id);


--
-- Name: directmsgs directmessages_sender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.directmsgs
    ADD CONSTRAINT directmessages_sender_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: blocked fk_blocked_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
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
-- Name: unreadmsgs fk_unreadmsg_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreadmsgs
    ADD CONSTRAINT fk_unreadmsg_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: unreadmsgs fk_unreadmsg_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreadmsgs
    ADD CONSTRAINT fk_unreadmsg_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: blocked fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.blocked
    ADD CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: tokens fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: userguilds fk_user_userguild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT fk_user_userguild FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: friends friends_friend_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_friend_id_fkey FOREIGN KEY (friend_id) REFERENCES public.users(id);


--
-- Name: friends friends_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.friends
    ADD CONSTRAINT friends_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: rolepermissions rolepermissions_permission_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.rolepermissions
    ADD CONSTRAINT rolepermissions_permission_id_fkey FOREIGN KEY (permission_id) REFERENCES public.permissions(id);


--
-- Name: rolepermissions rolepermissions_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.rolepermissions
    ADD CONSTRAINT rolepermissions_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: unreaddirectmsgs unreaddirectmsgs_dm_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreaddirectmsgs
    ADD CONSTRAINT unreaddirectmsgs_dm_id_fkey FOREIGN KEY (dm_id) REFERENCES public.directmsgsguild(id);


--
-- Name: unreaddirectmsgs unreaddirectmsgs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.unreaddirectmsgs
    ADD CONSTRAINT unreaddirectmsgs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: userdirectmsgsguild userdirectmsgsguild_dm_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userdirectmsgsguild
    ADD CONSTRAINT userdirectmsgsguild_dm_id_fkey FOREIGN KEY (dm_id) REFERENCES public.directmsgsguild(id);


--
-- Name: userdirectmsgsguild userdirectmsgsguild_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userdirectmsgsguild
    ADD CONSTRAINT userdirectmsgsguild_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: userroles userroles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT userroles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(id);


--
-- Name: userroles userroles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userroles
    ADD CONSTRAINT userroles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

