import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the identity stories. The page version and the API
// version of a story call the same function (v0 design 8.2).

const refreshTokenPrefix = "nrv_rt_";
const dayMs = 24 * 60 * 60 * 1000;

/** A refresh token's content (M2 design 3.4): nrv_rt_, then session id, generation, secret and tag in base64url. */
interface RefreshTokenParts {
  sessionId: string;
  generation: number;
  secret: Buffer;
}

function parseRefreshToken(token: string): RefreshTokenParts {
  expect(token.startsWith(refreshTokenPrefix), `${token} starts with ${refreshTokenPrefix}`).toBe(true);
  const raw = Buffer.from(token.slice(refreshTokenPrefix.length), "base64url");
  expect(raw.length, "a refresh token holds 68 bytes").toBe(68);
  const id = raw.subarray(0, 16).toString("hex");
  return {
    sessionId: `${id.slice(0, 8)}-${id.slice(8, 12)}-${id.slice(12, 16)}-${id.slice(16, 20)}-${id.slice(20)}`,
    generation: raw.readUInt32BE(16),
    secret: raw.subarray(20, 52),
  };
}

/** What a sign-up sent and got back. */
export interface Registration {
  /** The address as typed; the account holds it lowercased. */
  email: string;
  refreshToken: string;
  userAgent: string;
  ip: string;
}

/**
 * A1: registration added one account with the address lowercased, an
 * argon2id hash and the display name from the address; its default profile;
 * and a session of generation 0 whose hash is that of the refresh token's
 * secret, with the caller's User-Agent and IP, ending about 30 days later.
 */
export async function expectRegistered(db: Database, r: Registration): Promise<void> {
  const email = r.email.toLowerCase();
  const users = await db.query<{ id: string; password: string; display_name: string; is_active: boolean }>(
    "SELECT id, password, display_name, is_active FROM users WHERE email = $1",
    [email]
  );
  expect(users).toHaveLength(1);
  const [user] = users;
  expect(user?.password).toMatch(/^\$argon2id\$/);
  expect(user?.display_name).toBe(email.slice(0, email.indexOf("@")));
  expect(user?.is_active).toBe(true);

  const profiles = await db.query(
    `SELECT theme, is_tour_completed, onboarding_step, is_onboarded, last_workspace_id, language, start_of_the_week
       FROM profiles WHERE user_id = $1`,
    [user?.id]
  );
  expect(profiles).toEqual([
    {
      theme: "system",
      is_tour_completed: false,
      onboarding_step: {
        profile_complete: false,
        workspace_create: false,
        workspace_invite: false,
        workspace_join: false,
      },
      is_onboarded: false,
      last_workspace_id: null,
      language: "en",
      start_of_the_week: 0,
    },
  ]);

  const token = parseRefreshToken(r.refreshToken);
  const sessions = await db.query<{
    id: string;
    token_hash: Buffer;
    generation: number;
    user_agent: string;
    ip: string;
    expires_at: Date;
    created_at: Date;
    revoked_at: Date | null;
  }>(
    "SELECT id, token_hash, generation, user_agent, ip, expires_at, created_at, revoked_at FROM auth_sessions WHERE user_id = $1",
    [user?.id]
  );
  expect(sessions).toHaveLength(1);
  const [session] = sessions;
  expect(session?.id).toBe(token.sessionId);
  expect(token.generation).toBe(0);
  expect(session?.generation).toBe(0);
  expect(session?.token_hash.equals(createHash("sha256").update(token.secret).digest())).toBe(true);
  expect(session?.user_agent).toBe(r.userAgent);
  expect(session?.ip).toBe(r.ip);
  // auth.session_ttl is 720h: the session ends 30 days after it began.
  expect((session?.expires_at.getTime() ?? 0) - (session?.created_at.getTime() ?? 0)).toBe(30 * dayMs);
  expect(session?.revoked_at).toBeNull();
}

/** How many accounts, profiles and sessions there are. */
export interface IdentityCounts {
  users: number;
  profiles: number;
  sessions: number;
}

export async function countIdentity(db: Database): Promise<IdentityCounts> {
  const [counts] = await db.query<{ users: number; profiles: number; sessions: number }>(
    `SELECT (SELECT count(*)::int FROM users) AS users,
            (SELECT count(*)::int FROM profiles) AS profiles,
            (SELECT count(*)::int FROM auth_sessions) AS sessions`
  );
  if (!counts) {
    throw new Error("the counts query returned no row");
  }
  return counts;
}

/** A2: a refused sign-up added no account, profile or session. */
export async function expectNothingAdded(db: Database, before: IdentityCounts): Promise<void> {
  expect(await countIdentity(db)).toEqual(before);
}
