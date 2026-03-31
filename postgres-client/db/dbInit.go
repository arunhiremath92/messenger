package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupDb(pDb *pgxpool.Pool) error {

	if pDb == nil || pDb.Ping(context.Background()) != nil {
		panic("invalid database connection request")
	}
	const userSchema = `
			DO $$
			BEGIN
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'conversation_type') THEN
					CREATE TYPE conversation_type AS ENUM ('direct', 'group');
				END IF;
				
				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'delivery_status') THEN
					CREATE TYPE delivery_status AS ENUM ('sent', 'delivered', 'read');
				END IF;

				IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'group_role') THEN
					CREATE TYPE group_role AS ENUM ('admin', 'member');
				END IF;
			END$$;

			CREATE TABLE IF NOT EXISTS groups (
				groupid uuid DEFAULT gen_random_uuid() PRIMARY KEY,
				groupname character varying(128) NOT NULL,
				description character varying(512),
				creator_userid uuid NOT NULL REFERENCES users(userid),
				created_at timestamp with time zone DEFAULT now() NOT NULL
			);

			CREATE TABLE IF NOT EXISTS conversations (
				conversation_id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
				type conversation_type NOT NULL,
				group_id uuid REFERENCES groups(groupid),
				created_at timestamp with time zone DEFAULT now() NOT NULL,
				CONSTRAINT chk_group_type CHECK (
					(type = 'group' AND group_id IS NOT NULL) OR 
					(type = 'direct' AND group_id IS NULL)
				)
			);

			CREATE TABLE IF NOT EXISTS conversation_members (
				conversation_id uuid REFERENCES conversations(conversation_id),
				userid uuid REFERENCES users(userid),
				joined_at timestamp with time zone DEFAULT now() NOT NULL,
				last_read_at timestamp with time zone,
				PRIMARY KEY (conversation_id, userid)
			);
		`
	indexSchema := `			
			-- Speed up message/attachment lookups by conversation
			CREATE INDEX IF NOT EXISTS idx_attachments_conv 
			ON attachments (conversation_id, uploaded_at DESC);

			-- Quick lookups for specific messages
			CREATE INDEX IF NOT EXISTS idx_attachments_message 
			ON attachments (messageid);

			-- Optimization for 'Seen' status and user message history
			CREATE INDEX IF NOT EXISTS idx_conv_members_last_read 
			ON conversation_members (userid, last_read_at);

			-- Unique constraint for usernames (essential for your GetUsersByName function)
			CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username 
			ON users (username);

			-- Partial index for phone numbers (ignores nulls to save space)
			CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone 
			ON users (phone) WHERE (phone IS NOT NULL);
`
	_, err := pDb.Exec(context.Background(), userSchema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	_, err = pDb.Exec(context.Background(), indexSchema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("Database schema initialized successfully.")
	return nil
}
