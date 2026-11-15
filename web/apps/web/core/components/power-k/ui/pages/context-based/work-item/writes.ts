/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueUpdate } from "@nerve/api-client";
import type { IssueChange } from "@nerve/types";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { changeBody } from "@/store/issue/issue-write-fields";
import type { WorkItemCommandContext } from "./context";

/**
 * The writes of the palette's work item commands, through the work item's issue queue and the palette's guards: the
 * scope captured as the command runs, the session it ran in, and in the work item's turn the scope, the session and
 * whether it may still be changed (canEdit), its request dropped silently once any of them changed; nerve's refusal
 * said (useRefusalToast) while the scope and the session are the command's.
 */
export function useWorkItemWrites({ root, read, capture }: WorkItemCommandContext) {
  const refuse = useRefusalToast();
  /** An update, its body made in the work item's turn of the value the work item then holds. */
  const update = async (makeBody: (held: Issue) => IssueUpdate) => {
    const target = read();
    if (!target?.canEdit) return;
    const current = capture();
    if (!current()) return;
    const inSession = sessionGuard();
    try {
      await root.writes.updateIssue(
        target.issue.id,
        (held) => {
          // Read the authoritative value when this command reaches the existing issue queue.
          if (!current() || !inSession() || !read()?.canEdit) throw new SessionChangedError();
          return makeBody(held);
        },
        undefined,
        root.scope(target.workspaceSlug, target.project.id)
      );
    } catch (error) {
      if (current() && inSession() && !(error instanceof SessionChangedError)) refuse(error);
    }
  };
  /**
   * A change of the work item as its dropdowns send one (IssueChange, P7a spec 2.2, R3), through the guards above: its
   * request made in the work item's turn (changeBody), its modules' additions and removals applied to the set it then
   * holds; the type takes no whole set (P7c spec 2.5).
   */
  const change = (requested: IssueChange) => update((held) => changeBody(requested, held.module_ids));
  return { update, change };
}
