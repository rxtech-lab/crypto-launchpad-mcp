CREATE TABLE "next-auth-account" (
	"user_id" text NOT NULL,
	"type" text NOT NULL,
	"provider" text NOT NULL,
	"provider_account_id" text NOT NULL,
	"refresh_token" text,
	"access_token" text,
	"expires_at" integer,
	"token_type" text,
	"scope" text,
	"id_token" text,
	"session_state" text,
	CONSTRAINT "next-auth-account_provider_provider_account_id_pk" PRIMARY KEY("provider","provider_account_id")
);
--> statement-breakpoint
CREATE TABLE "next-auth-authenticators" (
	"credential_id" text NOT NULL,
	"user_id" text NOT NULL,
	"provider_account_id" text NOT NULL,
	"credential_public_key" text NOT NULL,
	"counter" integer NOT NULL,
	"credential_device_type" text NOT NULL,
	"credential_backed_up" boolean NOT NULL,
	"transports" text,
	CONSTRAINT "next-auth-authenticators_user_id_credential_id_pk" PRIMARY KEY("user_id","credential_id")
);
--> statement-breakpoint
CREATE TABLE "next-auth-jwt-tokens" (
	"id" text PRIMARY KEY NOT NULL,
	"user_id" text NOT NULL,
	"token_name" text NOT NULL,
	"jti" text NOT NULL,
	"aud" json,
	"client_id" text,
	"roles" json DEFAULT '[]'::json,
	"scopes" json DEFAULT '[]'::json,
	"created_at" timestamp DEFAULT now(),
	"expires_at" timestamp,
	"is_active" boolean DEFAULT true,
	CONSTRAINT "next-auth-jwt-tokens_jti_unique" UNIQUE("jti")
);
--> statement-breakpoint
CREATE TABLE "next-auth-session" (
	"session_token" text PRIMARY KEY NOT NULL,
	"user_id" text NOT NULL,
	"expires" timestamp NOT NULL
);
--> statement-breakpoint
CREATE TABLE "next-auth-users" (
	"id" text PRIMARY KEY NOT NULL,
	"email" text,
	"email_verified" timestamp,
	"name" text,
	"image" text,
	"created_at" timestamp DEFAULT now(),
	"updated_at" timestamp DEFAULT now(),
	CONSTRAINT "next-auth-users_email_unique" UNIQUE("email")
);
--> statement-breakpoint
CREATE TABLE "next-auth-verification_tokens" (
	"identifier" text NOT NULL,
	"token" text NOT NULL,
	"expires" timestamp NOT NULL,
	CONSTRAINT "next-auth-verification_tokens_identifier_token_pk" PRIMARY KEY("identifier","token")
);
--> statement-breakpoint
ALTER TABLE "next-auth-account" ADD CONSTRAINT "next-auth-account_user_id_next-auth-users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."next-auth-users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "next-auth-authenticators" ADD CONSTRAINT "next-auth-authenticators_user_id_next-auth-users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."next-auth-users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "next-auth-jwt-tokens" ADD CONSTRAINT "next-auth-jwt-tokens_user_id_next-auth-users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."next-auth-users"("id") ON DELETE cascade ON UPDATE no action;--> statement-breakpoint
ALTER TABLE "next-auth-session" ADD CONSTRAINT "next-auth-session_user_id_next-auth-users_id_fk" FOREIGN KEY ("user_id") REFERENCES "public"."next-auth-users"("id") ON DELETE cascade ON UPDATE no action;