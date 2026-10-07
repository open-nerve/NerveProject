/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import useSWR from "swr";
// lib
import { publicClient } from "@/lib/auth/api-client";
// services
import { previewInvitation } from "@/services/workspace/invitation-preview.service";

/**
 * The invitation a link names, as the link shows it, for the pages a link opens (sign-in, sign-up, the invitation's
 * own). It is public and the same for every session and for none, so it is the one M3 fetch keyed by the link
 * alone, not by a session (M3 design 7.1, 7.4); a link that names no invitation is SWR's error.
 */
export function useInvitationPreview(invitationId: string | null, token: string | null) {
  return useSWR(
    invitationId && token ? ["INVITATION_PREVIEW", invitationId, token] : null,
    ([, id, linkToken]: [string, string, string]) => previewInvitation(publicClient, id, linkToken),
    { revalidateOnFocus: false, shouldRetryOnError: false }
  );
}
