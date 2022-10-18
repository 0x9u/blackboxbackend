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
-- Name: guilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guilds (
    id integer NOT NULL,
    name character varying(16) NOT NULL,
    icon integer DEFAULT 0,
    owner_id integer NOT NULL,
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
-- Name: messages; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.messages (
    id integer NOT NULL,
    content character varying(1024) NOT NULL,
    user_id integer NOT NULL,
    guild_id integer NOT NULL,
    "time" bigint NOT NULL,
    edited boolean DEFAULT false NOT NULL
);


ALTER TABLE public.messages OWNER TO postgres;

--
-- Name: messages_id_seq; Type: SEQUENCE; Schema: public; Owner: postgres
--

CREATE SEQUENCE public.messages_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.messages_id_seq OWNER TO postgres;

--
-- Name: messages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: postgres
--

ALTER SEQUENCE public.messages_id_seq OWNED BY public.messages.id;


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
-- Name: unreadmessages; Type: TABLE; Schema: public; Owner: oliver
--

CREATE TABLE public.unreadmessages (
    guild_id integer,
    user_id integer,
    message_id integer DEFAULT 0,
    "time" bigint DEFAULT 0
);


ALTER TABLE public.unreadmessages OWNER TO oliver;

--
-- Name: userguilds; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.userguilds (
    guild_id integer NOT NULL,
    user_id integer NOT NULL,
    banned boolean DEFAULT false NOT NULL
);


ALTER TABLE public.userguilds OWNER TO postgres;

--
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id integer NOT NULL,
    email character varying(128) NOT NULL,
    password character varying(64) NOT NULL,
    username character varying(32)
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
-- Name: guilds id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds ALTER COLUMN id SET DEFAULT nextval('public.guilds_id_seq'::regclass);


--
-- Name: messages id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.messages ALTER COLUMN id SET DEFAULT nextval('public.messages_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: guilds guilds_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT guilds_pkey PRIMARY KEY (id);


--
-- Name: messages messages_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT messages_pkey PRIMARY KEY (id);


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
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: guilds fk_guild_owner; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guilds
    ADD CONSTRAINT fk_guild_owner FOREIGN KEY (owner_id) REFERENCES public.users(id);


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
-- Name: messages fk_message_guild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_message_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: messages fk_message_user; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.messages
    ADD CONSTRAINT fk_message_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: unreadmessages fk_unreadmessage_guild; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.unreadmessages
    ADD CONSTRAINT fk_unreadmessage_guild FOREIGN KEY (guild_id) REFERENCES public.guilds(id);


--
-- Name: unreadmessages fk_unreadmessage_user; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.unreadmessages
    ADD CONSTRAINT fk_unreadmessage_user FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: tokens fk_user_id; Type: FK CONSTRAINT; Schema: public; Owner: oliver
--

ALTER TABLE ONLY public.tokens
    ADD CONSTRAINT fk_user_id FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: userguilds fk_user_userguild; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.userguilds
    ADD CONSTRAINT fk_user_userguild FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- PostgreSQL database dump complete
--

