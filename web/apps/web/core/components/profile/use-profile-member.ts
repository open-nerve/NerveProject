/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { MemberUser, Workspace } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspaceMembersFetch } from "@/hooks/use-workspace-members-fetch";

/** What a profile page shows of its user: a wait for the members, that they failed to load, or whether he is one. */
type ProfileMember =
  | { kind: "loading"; member: undefined }
  | { kind: "load-failed"; member: undefined }
  | { kind: "not-a-member"; member: undefined }
  | { kind: "member"; member: MemberUser; joinedAt: string };

/**
 * Which member of the workspace a profile page is about (M1 design 3.2). The page header and the user card both
 * read it here, so they cannot disagree about a user, for instance one who has left the workspace. The workspace is
 * the page's as WorkspaceAuthWrapper gives it, which shows a page only once the caller's list has it; the user's
 * membership is read in that workspace, and the members' fetch alone decides the rest (useWorkspaceMembersFetch,
 * which the wrapper's fetch shares).
 */
export const useProfileMember = (workspace: Workspace | null, userId: string): ProfileMember => {
  const {
    workspace: { getMemberships },
  } = useMember();
  const members = useWorkspaceMembersFetch(workspace);
  const memberDetails = workspace && userId ? getMemberships(workspace.slug)?.[userId] : undefined;
  // A member removed from the workspace stays in the store, marked inactive: it is no longer a member.
  const membership = memberDetails?.is_active === false ? undefined : memberDetails;

  if (membership) return { kind: "member", member: membership.member, joinedAt: membership.created_at };
  if (members.error) return { kind: "load-failed", member: undefined };
  if (members.data === undefined) return { kind: "loading", member: undefined };
  return { kind: "not-a-member", member: undefined };
};
