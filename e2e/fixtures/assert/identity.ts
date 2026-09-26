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

function secretHash(refreshToken: string): Buffer {
  return createHash("sha256").update(parseRefreshToken(refreshToken).secret).digest();
}

/** A session's row, as the session assertions read it. */
export interface SessionRow {
  id: string;
  user_id: string;
  token_hash: Buffer;
  generation: number;
  user_agent: string;
  ip: string;
  expires_at: Date;
  created_at: Date;
  last_refreshed_at: Date | null;
  revoked_at: Date | null;
  revoke_reason: string | null;
}

/** The row of the session that refreshToken belongs to. */
export async function sessionOf(db: Database, refreshToken: string): Promise<SessionRow> {
  const rows = await db.query<SessionRow>(
    `SELECT id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, last_refreshed_at,
            revoked_at, revoke_reason
       FROM auth_sessions WHERE id = $1`,
    [parseRefreshToken(refreshToken).sessionId]
  );
  expect(rows, "the session of the refresh token").toHaveLength(1);
  return rows[0] as SessionRow;
}

/** What a sign-up or a login sent and got back. */
export interface SignIn {
  /** The address as typed; the account holds it lowercased. */
  email: string;
  refreshToken: string;
  userAgent: string;
  ip: string;
}

/**
 * The session of s.refreshToken is new: generation 0 of the account userId,
 * holding the hash of the token's secret, with the caller's User-Agent and
 * IP, never refreshed, live, ending 30 days after it began.
 */
async function expectNewSession(db: Database, userId: string | undefined, s: SignIn): Promise<void> {
  const session = await sessionOf(db, s.refreshToken);
  expect(session.user_id).toBe(userId);
  expect(parseRefreshToken(s.refreshToken).generation).toBe(0);
  expect(session.generation).toBe(0);
  expect(session.token_hash.equals(secretHash(s.refreshToken))).toBe(true);
  expect(session.user_agent).toBe(s.userAgent);
  expect(session.ip).toBe(s.ip);
  // auth.session_ttl is 720h: the session ends 30 days after it began.
  expect(session.expires_at.getTime() - session.created_at.getTime()).toBe(30 * dayMs);
  expect(session.last_refreshed_at).toBeNull();
  expect(session.revoked_at).toBeNull();
}

/**
 * A1: registration added one account with the address lowercased, an
 * argon2id hash and the display name from the address; its default profile;
 * and its one session, a new one.
 */
export async function expectRegistered(db: Database, r: SignIn): Promise<void> {
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

  expect(await db.query("SELECT id FROM auth_sessions WHERE user_id = $1", [user?.id])).toHaveLength(1);
  await expectNewSession(db, user?.id, r);
}

/** A3: a login added a new session to the account of the address. */
export async function expectSignedIn(db: Database, s: SignIn): Promise<void> {
  const users = await db.query<{ id: string }>("SELECT id FROM users WHERE email = $1", [s.email.toLowerCase()]);
  expect(users).toHaveLength(1);
  await expectNewSession(db, users[0]?.id, s);
}

/**
 * A4: after `refreshes` refreshes, each with the token the last one
 * returned, the session is live at that generation, holds the hash of the
 * latest token's secret, was refreshed, and still ends at sessionEnd, the
 * refresh_token_expires_at of the login (M2 design 3.5).
 */
export async function expectRefreshed(
  db: Database,
  latest: string,
  refreshes: number,
  sessionEnd: string
): Promise<void> {
  const session = await sessionOf(db, latest);
  expect(parseRefreshToken(latest).generation).toBe(refreshes);
  expect(session.generation).toBe(refreshes);
  expect(session.token_hash.equals(secretHash(latest))).toBe(true);
  expect(session.last_refreshed_at).not.toBeNull();
  expect(session.expires_at.getTime()).toBe(new Date(sessionEnd).getTime());
  expect(session.revoked_at).toBeNull();
}

/** A5, A6: the session of refreshToken is revoked, for reason. */
export async function expectRevoked(
  db: Database,
  refreshToken: string,
  reason: "logout" | "reuse_detected"
): Promise<void> {
  const session = await sessionOf(db, refreshToken);
  expect(session.revoked_at).not.toBeNull();
  expect(session.revoke_reason).toBe(reason);
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

/** A2, A3, A15: a refused sign-up or login added no account, profile or session. */
export async function expectNothingAdded(db: Database, before: IdentityCounts): Promise<void> {
  expect(await countIdentity(db)).toEqual(before);
}
