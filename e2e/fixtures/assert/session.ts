import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import type { Database } from "../db";

// Database assertions of the identity stories' sessions: a refresh token's content, a session's row, and what a
// sign-in, a refresh and a revocation leave of it (M2 design 3.4, 3.5). The page version and the API version of a story
// call the same function (v0 design 8.2).

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

/** The generation a refresh token carries: 0 at the sign-in, one more at each refresh. */
export function generationOf(refreshToken: string): number {
  return parseRefreshToken(refreshToken).generation;
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

/** The columns of a SessionRow. */
const sessionColumns = `id, user_id, token_hash, generation, user_agent, ip, expires_at, created_at, last_refreshed_at,
       revoked_at, revoke_reason`;

/** The row of the session that refreshToken belongs to. */
export async function sessionOf(db: Database, refreshToken: string): Promise<SessionRow> {
  const rows = await db.query<SessionRow>(`SELECT ${sessionColumns} FROM auth_sessions WHERE id = $1`, [
    parseRefreshToken(refreshToken).sessionId,
  ]);
  expect(rows, "the session of the refresh token").toHaveLength(1);
  return rows[0] as SessionRow;
}

/** The rows of the sessions of the account userId, oldest first. */
export async function sessionsOf(db: Database, userId: string): Promise<SessionRow[]> {
  return db.query<SessionRow>(
    `SELECT ${sessionColumns} FROM auth_sessions WHERE user_id = $1 ORDER BY created_at, id`,
    [userId]
  );
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
export async function expectNewSession(db: Database, userId: string | undefined, s: SignIn): Promise<void> {
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

/**
 * A12, A13, A16: every session of the account userId, and it has one at
 * least, is revoked for reason.
 */
export async function expectAllSessionsRevoked(
  db: Database,
  userId: string,
  reason: "deactivated" | "password_reset" | "email_changed"
): Promise<void> {
  const sessions = await sessionsOf(db, userId);
  expect(sessions.length).toBeGreaterThan(0);
  for (const s of sessions) {
    expect({ revoked: s.revoked_at !== null, revoke_reason: s.revoke_reason }, `session ${s.id}`).toEqual({
      revoked: true,
      revoke_reason: reason,
    });
  }
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
