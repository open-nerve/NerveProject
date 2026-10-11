/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// hooks
import { useProject } from "@/hooks/store/use-project";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * Joins a project the caller sees and is no member of (M3 design 3.5, 7.6): the store then shows it as he now sees
 * it, a member. The page follows nerve's answer only in the session it was sent in (M3 design 7.1): joined, what the
 * page does next (done: the card's modal opens the project; the project's own pages show once the store has him a
 * member); refused, nerve's reason in a toast. Settles once the follow-up has; never rejects.
 */
export function useJoinProject(): (projectId: string, done?: () => void | Promise<void>) => Promise<void> {
  const { joinProject } = useProject();
  const toastRefusal = useRefusalToast();
  return (projectId, done) => followInSession(() => joinProject(projectId), { done, failed: toastRefusal });
}
