/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { InvitationPreview } from "@nerve/api-client";
// lib
import { ApiError } from "@/lib/api-error";

/**
 * What the invitation page shows (M3 design 7.4, W5): a link without its id or its token, or one nerve finds no
 * invitation by (400 for an id that is none, 404), is not valid; nerve not answering, a retry; then, as the link shows
 * the invitation, that it was declined; to one signed out, the ways to sign in or up and accept; to one whose address
 * nerve found not the invitation's as he answered, that it was sent to another (decision 1); else, accepting or
 * ignoring it.
 */
export type InvitationView =
  | { kind: "invalid" }
  | { kind: "loading" }
  | { kind: "unavailable" }
  | { kind: "declined" | "sign-in" | "mismatch" | "answer"; invitation: InvitationPreview; token: string };

/** Whether nerve found no invitation by the link: an id that is none (400), or none of it and its token (404). */
const notFound = (error: unknown) => error instanceof ApiError && (error.status === 400 || error.status === 404);

/** The page's view of the link's invitation, from the link, the invitation's preview and the caller. */
export function invitationView(page: {
  link: { invitationId: string | null; token: string | null };
  preview: { data: InvitationPreview | undefined; error: unknown };
  signedIn: boolean;
  /** Whether nerve answered the caller's acceptance or decline that the invitation is another address's. */
  mismatched: boolean;
}): InvitationView {
  const { link, preview, signedIn, mismatched } = page;
  const { token } = link;
  if (!link.invitationId || !token) return { kind: "invalid" };
  if (preview.error) return notFound(preview.error) ? { kind: "invalid" } : { kind: "unavailable" };
  const invitation = preview.data;
  if (!invitation) return { kind: "loading" };
  if (invitation.declined) return { kind: "declined", invitation, token };
  if (!signedIn) return { kind: "sign-in", invitation, token };
  if (mismatched) return { kind: "mismatch", invitation, token };
  return { kind: "answer", invitation, token };
}
