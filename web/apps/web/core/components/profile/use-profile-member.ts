/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// nerve imports
import type { MemberUser } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useWorkspace } from "@/hooks/store/use-workspace";
// lib
import { useSessionSWR } from "@/lib/use-session-swr";

type TProfileMember =
  | { status: "loading"; member: undefined }
  | { status: "load-failed"; member: undefined }
  | { status: "not-a-member"; member: undefined }
  | { status: "member"; member: MemberUser; joinedAt: string };

// Which workspace member a profile page is about (M1 design 3.2). The page header and the user card both
// read it here, so they cannot disagree about a user, for instance one who has left the workspace.
export const useProfileMember = (workspaceSlug: string, userId: string): TProfileMember => {
  const { getWorkspaceBySlug } = useWorkspace();
  const {
    workspace: { fetchWorkspaceMembers, getWorkspaceMemberDetails },
  } = useMember();
  // the address's workspace is the caller's once his list has it
  const workspace = workspaceSlug === "" ? null : getWorkspaceBySlug(workspaceSlug);
  // The workspace wrapper already fetches the members under this key, so SWR serves the same request;
  // subscribing here is what tells whether the members are still loading or failed to load: a failed load
  // leaves the request settled without a list.
  const { data: members, isLoading } = useSessionSWR(
    workspace && ["WORKSPACE_MEMBERS", workspace.id, workspace.slug],
    (id, slug) => fetchWorkspaceMembers({ id, slug }),
    {
      revalidateIfStale: false,
      revalidateOnFocus: false,
    }
  );
  const memberDetails = userId ? getWorkspaceMemberDetails(userId) : null;
  // A member removed from the workspace stays in the store, marked inactive: it is no longer a member.
  const membership = memberDetails?.is_active === false ? undefined : memberDetails;

  if (membership) return { status: "member", member: membership.member, joinedAt: membership.created_at };
  if (isLoading) return { status: "loading", member: undefined };
  if (!members) return { status: "load-failed", member: undefined };
  return { status: "not-a-member", member: undefined };
};
