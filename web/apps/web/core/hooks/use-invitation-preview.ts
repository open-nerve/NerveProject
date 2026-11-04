/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useLayoutEffect, useMemo } from "react";
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
  const life = useMemo(
    () => ({ invitationId, token, active: false, epoch: 0, requests: new Set<AbortController>() }),
    [invitationId, token]
  );
  const result = useSWR(
    life.invitationId && life.token ? ["INVITATION_PREVIEW", life.invitationId, life.token] : null,
    async ([, id, linkToken]: [string, string, string]) => {
      const epoch = life.epoch;
      const controller = new AbortController();
      const current = () => life.active && life.epoch === epoch && !controller.signal.aborted;
      if (!current()) throw new DOMException("Invitation view ended", "AbortError");
      life.requests.add(controller);
      try {
        const value = await previewInvitation(publicClient, id, linkToken, controller.signal);
        if (!current()) throw new DOMException("Invitation view ended", "AbortError");
        return value;
      } finally {
        life.requests.delete(controller);
      }
    },
    {
      revalidateOnMount: false,
      revalidateOnFocus: false,
      revalidateOnReconnect: false,
      shouldRetryOnError: false,
      dedupingInterval: 0,
    }
  );
  const { mutate } = result;
  useLayoutEffect(() => {
    life.active = true;
    life.epoch++;
    // A new committed owner must not reuse an aborted request from an earlier visit or StrictMode replay.
    if (life.invitationId && life.token)
      void mutate().catch(() => {
        /* SWR retains current link refusals. */
      });
    return () => {
      life.active = false;
      life.epoch++;
      for (const request of life.requests) request.abort();
      life.requests.clear();
    };
  }, [life, mutate]);
  return result;
}
