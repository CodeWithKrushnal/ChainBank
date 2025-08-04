--
-- PostgreSQL database dump
--

-- Dumped from database version 14.15 (Ubuntu 14.15-0ubuntu0.22.04.1)
-- Dumped by pg_dump version 16.8 (Ubuntu 16.8-0ubuntu0.24.04.1)

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

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: api_requests_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_requests_log (
    request_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    endpoint character varying(255) NOT NULL,
    http_method character varying(10) NOT NULL,
    request_payload jsonb NOT NULL,
    response_status integer DEFAULT 0 NOT NULL,
    response_time_ms integer DEFAULT 0 NOT NULL,
    ip_address inet NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL
);


--
-- Name: dao_proposals; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dao_proposals (
    proposal_id uuid DEFAULT gen_random_uuid() NOT NULL,
    proposer_id uuid,
    title character varying(255) NOT NULL,
    description text NOT NULL,
    proposal_type character varying(50) NOT NULL,
    status character varying(50) NOT NULL,
    voting_start_time timestamp with time zone,
    voting_end_time timestamp with time zone,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: dao_votes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.dao_votes (
    vote_id uuid DEFAULT gen_random_uuid() NOT NULL,
    proposal_id uuid,
    voter_id uuid,
    vote_type character varying(20) NOT NULL,
    voting_power numeric(20,8) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: kyc_verifications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.kyc_verifications (
    kyc_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    document_type character varying(50) NOT NULL,
    document_number character varying(100) NOT NULL,
    verification_status character varying(50) NOT NULL,
    submitted_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    verified_at timestamp with time zone,
    verified_by uuid
);


--
-- Name: loan_applications; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.loan_applications (
    application_id uuid NOT NULL,
    borrower_id uuid NOT NULL,
    amount double precision NOT NULL,
    interest_rate double precision NOT NULL,
    term_months integer NOT NULL,
    status character varying(20) DEFAULT 'open'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: loan_offers; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.loan_offers (
    offer_id uuid DEFAULT gen_random_uuid() NOT NULL,
    lender_id uuid NOT NULL,
    amount numeric(50,20) NOT NULL,
    interest_rate numeric(5,2) NOT NULL,
    loan_term_months integer NOT NULL,
    status character varying(50) NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    application_id uuid NOT NULL
);


--
-- Name: loans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.loans (
    loan_id uuid NOT NULL,
    offer_id uuid NOT NULL,
    borrower_id uuid NOT NULL,
    lender_id uuid NOT NULL,
    total_principle double precision NOT NULL,
    remaining_principle double precision NOT NULL,
    status character varying(20) DEFAULT 'Active'::character varying NOT NULL,
    start_date timestamp without time zone DEFAULT now() NOT NULL,
    next_payment_date timestamp without time zone NOT NULL,
    application_id uuid NOT NULL,
    interest_rate double precision NOT NULL,
    settled_amount double precision DEFAULT 0.0 NOT NULL,
    settlement_date timestamp without time zone DEFAULT '9999-12-31 00:00:00'::timestamp without time zone NOT NULL,
    accrued_interest double precision DEFAULT 0.0 NOT NULL,
    disbursement_transaction_id uuid NOT NULL,
    settlement_transaction_id uuid
);


--
-- Name: password_reset_tokens; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.password_reset_tokens (
    token_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid,
    token character varying(255) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    used boolean DEFAULT false
);


--
-- Name: roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.roles (
    role_id integer NOT NULL,
    role_name character varying(50)
);


--
-- Name: transactions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.transactions (
    transaction_id uuid DEFAULT gen_random_uuid() NOT NULL,
    sender_wallet_id character varying(255) NOT NULL,
    receiver_wallet_id character varying(255) NOT NULL,
    amount numeric(25,5) NOT NULL,
    transaction_type character varying(50) NOT NULL,
    status character varying(50) NOT NULL,
    transaction_hash character varying(255),
    fee numeric(50,20) DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: user_roles_assignment; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_roles_assignment (
    role_assignment_id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    role_id integer NOT NULL
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(50) NOT NULL,
    email character varying(100) NOT NULL,
    password_hash character varying(255) NOT NULL,
    full_name character varying(100),
    date_of_birth date,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    last_login timestamp with time zone,
    is_active boolean DEFAULT true,
    is_blocked boolean DEFAULT false
);


--
-- Name: wallet_private_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wallet_private_keys (
    user_id uuid NOT NULL,
    wallet_id character varying(255) NOT NULL,
    private_key bytea NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: wallets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wallets (
    wallet_id character varying(255) NOT NULL,
    user_id uuid,
    balance numeric(50,20) DEFAULT 0,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    last_updated timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- Name: api_requests_log api_requests_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_requests_log
    ADD CONSTRAINT api_requests_log_pkey PRIMARY KEY (request_id);


--
-- Name: dao_proposals dao_proposals_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dao_proposals
    ADD CONSTRAINT dao_proposals_pkey PRIMARY KEY (proposal_id);


--
-- Name: dao_votes dao_votes_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dao_votes
    ADD CONSTRAINT dao_votes_pkey PRIMARY KEY (vote_id);


--
-- Name: kyc_verifications kyc_verifications_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kyc_verifications
    ADD CONSTRAINT kyc_verifications_pkey PRIMARY KEY (kyc_id);


--
-- Name: loan_offers lending_offers_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loan_offers
    ADD CONSTRAINT lending_offers_pkey PRIMARY KEY (offer_id);


--
-- Name: loan_applications loan_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loan_applications
    ADD CONSTRAINT loan_requests_pkey PRIMARY KEY (application_id);


--
-- Name: loans loans_application_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_application_id_key UNIQUE (application_id);


--
-- Name: loans loans_offer_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_offer_id_key UNIQUE (offer_id);


--
-- Name: loans loans_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_pkey PRIMARY KEY (loan_id);


--
-- Name: password_reset_tokens password_reset_tokens_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_pkey PRIMARY KEY (token_id);


--
-- Name: roles roles_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.roles
    ADD CONSTRAINT roles_pkey PRIMARY KEY (role_id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (transaction_id);


--
-- Name: user_roles_assignment user_roles_assignment_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles_assignment
    ADD CONSTRAINT user_roles_assignment_pkey PRIMARY KEY (role_assignment_id);


--
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);


--
-- Name: users users_username_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_username_key UNIQUE (username);


--
-- Name: wallet_private_keys wallet_private_keys_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallet_private_keys
    ADD CONSTRAINT wallet_private_keys_pkey PRIMARY KEY (user_id, wallet_id);


--
-- Name: wallets wallets_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_pkey PRIMARY KEY (wallet_id);


--
-- Name: idx_dao_proposals_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_dao_proposals_status ON public.dao_proposals USING btree (status);


--
-- Name: idx_kyc_user; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_kyc_user ON public.kyc_verifications USING btree (user_id);


--
-- Name: idx_transactions_receiver; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transactions_receiver ON public.transactions USING btree (receiver_wallet_id);


--
-- Name: idx_transactions_sender; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_transactions_sender ON public.transactions USING btree (sender_wallet_id);


--
-- Name: idx_users_email; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_email ON public.users USING btree (email);


--
-- Name: idx_wallet_private_keys_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_wallet_private_keys_user_id ON public.wallet_private_keys USING btree (user_id);


--
-- Name: idx_wallet_private_keys_wallet_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_wallet_private_keys_wallet_id ON public.wallet_private_keys USING btree (wallet_id);


--
-- Name: api_requests_log api_requests_log_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_requests_log
    ADD CONSTRAINT api_requests_log_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: dao_proposals dao_proposals_proposer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dao_proposals
    ADD CONSTRAINT dao_proposals_proposer_id_fkey FOREIGN KEY (proposer_id) REFERENCES public.users(user_id);


--
-- Name: dao_votes dao_votes_proposal_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dao_votes
    ADD CONSTRAINT dao_votes_proposal_id_fkey FOREIGN KEY (proposal_id) REFERENCES public.dao_proposals(proposal_id);


--
-- Name: dao_votes dao_votes_voter_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.dao_votes
    ADD CONSTRAINT dao_votes_voter_id_fkey FOREIGN KEY (voter_id) REFERENCES public.users(user_id);


--
-- Name: wallet_private_keys fk_wallet_user; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallet_private_keys
    ADD CONSTRAINT fk_wallet_user FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: wallet_private_keys fk_wallet_wallet; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallet_private_keys
    ADD CONSTRAINT fk_wallet_wallet FOREIGN KEY (wallet_id) REFERENCES public.wallets(wallet_id);


--
-- Name: kyc_verifications kyc_verifications_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kyc_verifications
    ADD CONSTRAINT kyc_verifications_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: kyc_verifications kyc_verifications_verified_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.kyc_verifications
    ADD CONSTRAINT kyc_verifications_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES public.users(user_id);


--
-- Name: loan_offers lending_offers_lender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loan_offers
    ADD CONSTRAINT lending_offers_lender_id_fkey FOREIGN KEY (lender_id) REFERENCES public.users(user_id);


--
-- Name: loan_offers lending_offers_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loan_offers
    ADD CONSTRAINT lending_offers_request_id_fkey FOREIGN KEY (application_id) REFERENCES public.loan_applications(application_id);


--
-- Name: loans loans_borrower_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_borrower_id_fkey FOREIGN KEY (borrower_id) REFERENCES public.users(user_id);


--
-- Name: loans loans_disbursement_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_disbursement_transaction_id_fkey FOREIGN KEY (disbursement_transaction_id) REFERENCES public.transactions(transaction_id);


--
-- Name: loans loans_lender_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_lender_id_fkey FOREIGN KEY (lender_id) REFERENCES public.users(user_id);


--
-- Name: loans loans_offer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_offer_id_fkey FOREIGN KEY (offer_id) REFERENCES public.loan_offers(offer_id);


--
-- Name: loans loans_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_request_id_fkey FOREIGN KEY (application_id) REFERENCES public.loan_applications(application_id);


--
-- Name: loans loans_settlement_transaction_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.loans
    ADD CONSTRAINT loans_settlement_transaction_id_fkey FOREIGN KEY (settlement_transaction_id) REFERENCES public.transactions(transaction_id);


--
-- Name: password_reset_tokens password_reset_tokens_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.password_reset_tokens
    ADD CONSTRAINT password_reset_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: transactions transactions_sender_wallet_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_sender_wallet_id_fkey FOREIGN KEY (sender_wallet_id) REFERENCES public.wallets(wallet_id);


--
-- Name: user_roles_assignment user_roles_role_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles_assignment
    ADD CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id);


--
-- Name: user_roles_assignment user_roles_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_roles_assignment
    ADD CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: wallets wallets_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wallets
    ADD CONSTRAINT wallets_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);


--
-- Name: SCHEMA public; Type: ACL; Schema: -; Owner: -
--

REVOKE USAGE ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO PUBLIC;


--
-- PostgreSQL database dump complete
--

