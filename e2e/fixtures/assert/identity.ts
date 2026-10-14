import { createHash } from "node:crypto";

import { expect } from "@playwright/test";

import { shownNameOf } from "../auth";
import type { Database } from "../db";
import {
  expectAllSessionsRevoked,
  expectNewSession,
  sessionOf,
  sessionsOf,
  type SessionRow,
  type SignIn,
} from "./session";

// Database assertions of the identity stories' accounts: their rows, profiles, passwords and personal access tokens,
// their sessions' being session.ts's. The page version and the API version of a story call the same function (v0
// design 8.2).

/**
 * A1, A17: one account has the address typed, lowercased, with an argon2id
 * hash, the display name from the address, active, and its default
 * profile. Returns its id.
 */
async function expectNewAccount(db: Database, typed: string): Promise<string> {
  const email = typed.toLowerCase();
  const user = await accountOf(db, email);
  expect(user.password).toMatch(/^\$argon2id\$/);
  expect(user.display_name).toBe(shownNameOf(email));
  expect(user.is_active).toBe(true);

  const profiles = await db.query(
    `SELECT theme, is_tour_completed, onboarding_step, is_onboarded, last_workspace_id, language, start_of_the_week
       FROM profiles WHERE user_id = $1`,
    [user.id]
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
  return user.id;
}

/** A1: registration added a new account and its one session, a new one. */
export async function expectRegistered(db: Database, r: SignIn): Promise<void> {
  const id = await expectNewAccount(db, r.email);
  expect(await sessionsOf(db, id)).toHaveLength(1);
  await expectNewSession(db, id, r);
}

/** A17: nerve users create added a new account of the address typed, without a session. */
export async function expectCreated(db: Database, typed: string): Promise<void> {
  const id = await expectNewAccount(db, typed);
  expect(await sessionsOf(db, id)).toEqual([]);
}

/** An account's row, as the account stories read it. */
export interface AccountRow {
  id: string;
  password: string;
  first_name: string;
  last_name: string;
  display_name: string;
  user_timezone: string;
  is_active: boolean;
  updated_at: Date;
}

/** The account of email, a lowercased address. */
export async function accountOf(db: Database, email: string): Promise<AccountRow> {
  const rows = await db.query<AccountRow>(
    `SELECT id, password, first_name, last_name, display_name, user_timezone, is_active, updated_at
       FROM users WHERE email = $1`,
    [email]
  );
  expect(rows, `the account of ${email}`).toHaveLength(1);
  return rows[0] as AccountRow;
}

/** The names and the time zone of an account, which A8 changes. */
type AccountChange = Partial<Pick<AccountRow, "first_name" | "last_name" | "display_name" | "user_timezone">>;

/**
 * A8: the account of email, a lowercased address, is the account `before` with change, and nothing else changed but
 * its updated_at, which moved on. Returns it now.
 */
export async function expectAccountChanged(
  db: Database,
  email: string,
  before: AccountRow,
  change: AccountChange
): Promise<AccountRow> {
  const after = await accountOf(db, email);
  expect(after).toEqual({ ...before, ...change, updated_at: after.updated_at });
  expect(after.updated_at.getTime()).toBeGreaterThan(before.updated_at.getTime());
  return after;
}

/** The preferences of a profile, which A9 changes. */
export interface Preferences {
  theme: string;
  language: string;
  start_of_the_week: number;
}

/** A9: the profile of the account of email, a lowercased address, holds preferences. */
export async function expectPreferences(db: Database, email: string, preferences: Preferences): Promise<void> {
  const { id } = await accountOf(db, email);
  expect(
    await db.query("SELECT theme, language, start_of_the_week FROM profiles WHERE user_id = $1", [id]),
    `the preferences of ${email}`
  ).toEqual([preferences]);
}

/** A profile's onboarding steps: done or not. */
export interface OnboardingSteps {
  profile_complete: boolean;
  workspace_create: boolean;
  workspace_invite: boolean;
  workspace_join: boolean;
}

/** The onboarding steps of the account of email, a lowercased address, as its profile holds them. */
export async function onboardingStepsOf(db: Database, email: string): Promise<OnboardingSteps> {
  const { id } = await accountOf(db, email);
  const rows = await db.query<{ onboarding_step: OnboardingSteps }>(
    "SELECT onboarding_step FROM profiles WHERE user_id = $1",
    [id]
  );
  expect(rows, `the profile of ${email}`).toHaveLength(1);
  return rows[0]?.onboarding_step as OnboardingSteps;
}

/**
 * A10: the account of email, a lowercased address, took the profile step: its first name is firstName, and its
 * onboarding steps are stepsBefore with the profile's done; the others keep their values (M2 design 3.14).
 */
export async function expectProfileStepTaken(
  db: Database,
  email: string,
  firstName: string,
  stepsBefore: OnboardingSteps
): Promise<void> {
  expect((await accountOf(db, email)).first_name).toBe(firstName);
  expect(await onboardingStepsOf(db, email)).toEqual({ ...stepsBefore, profile_complete: true });
}

/** The personal access tokens of the account userId, oldest first: id and deleted_at. */
export async function tokensOf(db: Database, userId: string): Promise<{ id: string; deleted_at: Date | null }[]> {
  return db.query("SELECT id, deleted_at FROM api_tokens WHERE user_id = $1 ORDER BY created_at, id", [userId]);
}

/** An account's row, its sessions and its personal access tokens, oldest first. */
interface AccountState {
  account: AccountRow;
  sessions: SessionRow[];
  tokens: { id: string; deleted_at: Date | null }[];
}

/**
 * A12, A13, A16: the state of the account of email, a lowercased address,
 * which a command given another account must leave as it was.
 */
export async function accountStateOf(db: Database, email: string): Promise<AccountState> {
  const account = await accountOf(db, email);
  return { account, sessions: await sessionsOf(db, account.id), tokens: await tokensOf(db, account.id) };
}

/** A7, A13: the password of the account `before` changed, to another argon2id hash. */
async function expectNewPassword(db: Database, before: AccountRow): Promise<void> {
  const [after] = await db.query<{ password: string }>("SELECT password FROM users WHERE id = $1", [before.id]);
  expect(after?.password).toMatch(/^\$argon2id\$/);
  expect(after?.password).not.toBe(before.password);
}

/**
 * A7: the password of the account `before` changed; every session of the
 * account is revoked for password_changed but survivingSession, the refresh
 * token of the page's own session, which stays live; the personal access
 * tokens are as they were (M2 design 3.5).
 */
export async function expectPasswordChanged(
  db: Database,
  before: AccountRow,
  tokensBefore: { id: string; deleted_at: Date | null }[],
  survivingSession?: string
): Promise<void> {
  await expectNewPassword(db, before);
  const surviving = survivingSession === undefined ? undefined : (await sessionOf(db, survivingSession)).id;
  const sessions = await sessionsOf(db, before.id);
  expect(sessions.length).toBeGreaterThan(0);
  for (const s of sessions) {
    expect(s.revoke_reason, `session ${s.id}`).toBe(s.id === surviving ? null : "password_changed");
  }
  expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
}

/**
 * A12: the account `before` is deactivated and its password unchanged;
 * every session is revoked for deactivated; its onboarding starts over;
 * its personal access tokens are as they were (M2 design 3.5, decision 3).
 */
export async function expectDeactivated(
  db: Database,
  before: AccountRow,
  tokensBefore: { id: string; deleted_at: Date | null }[]
): Promise<void> {
  expect(await db.query("SELECT is_active, password FROM users WHERE id = $1", [before.id])).toEqual([
    { is_active: false, password: before.password },
  ]);
  await expectAllSessionsRevoked(db, before.id, "deactivated");
  expect(
    await db.query(
      "SELECT onboarding_step, is_onboarded, is_tour_completed, last_workspace_id FROM profiles WHERE user_id = $1",
      [before.id]
    )
  ).toEqual([
    {
      onboarding_step: {
        profile_complete: false,
        workspace_create: false,
        workspace_invite: false,
        workspace_join: false,
      },
      is_onboarded: false,
      is_tour_completed: false,
      last_workspace_id: null,
    },
  ]);
  expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
}

/**
 * A13: the password of the account `before` changed; every session is
 * revoked for password_reset and every personal access token deleted (M2
 * design 3.5).
 */
export async function expectPasswordReset(db: Database, before: AccountRow): Promise<void> {
  await expectNewPassword(db, before);
  await expectAllSessionsRevoked(db, before.id, "password_reset");
  const tokens = await tokensOf(db, before.id);
  expect(tokens.length).toBeGreaterThan(0);
  for (const t of tokens) {
    expect(t.deleted_at, `token ${t.id}`).not.toBeNull();
  }
}

/**
 * A16: the account `before` has the address email, and its password; every
 * session is revoked for email_changed; its personal access tokens are as
 * they were (M2 decision 1).
 */
export async function expectEmailChanged(
  db: Database,
  before: AccountRow,
  email: string,
  tokensBefore: { id: string; deleted_at: Date | null }[]
): Promise<void> {
  expect(await db.query("SELECT email, password FROM users WHERE id = $1", [before.id])).toEqual([
    { email, password: before.password },
  ]);
  await expectAllSessionsRevoked(db, before.id, "email_changed");
  expect(await tokensOf(db, before.id)).toEqual(tokensBefore);
}

/**
 * A11: the new token's row holds the SHA-256 of the token, and no text
 * column holds the token's text; it expires at expiredAt and is live.
 */
export async function expectTokenStored(db: Database, id: string, token: string, expiredAt: string): Promise<void> {
  const rows = await db.query<{ token_hash: Buffer; expired_at: Date; deleted_at: Date | null; row: string }>(
    "SELECT token_hash, expired_at, deleted_at, row_to_json(t)::text AS row FROM api_tokens t WHERE id = $1",
    [id]
  );
  expect(rows).toHaveLength(1);
  const [row] = rows;
  expect(row?.token_hash.equals(createHash("sha256").update(token).digest())).toBe(true);
  // row_to_json writes the whole row as text, so this catches the token's
  // text, with or without its prefix, in a text column. A bytea column is
  // written as hex, which this does not search: that token_hash holds only
  // the hash is the Go store test's to check (TestCreateAPIToken).
  expect(row?.row).not.toContain(token.slice("nrv_pat_".length));
  expect(row?.expired_at.getTime()).toBe(new Date(expiredAt).getTime());
  expect(row?.deleted_at).toBeNull();
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
