/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import useSWR from "swr";
// nerve imports
import { WORKSPACE_MEMBERS } from "@nerve/constants";
import type { IUserLite } from "@nerve/types";
// hooks
import { useMember } from "@/hooks/store/use-member";

type TProfileMember =
  | { status: "loading"; member: undefined }
  | { status: "load-failed"; member: undefined }
  | { status: "not-a-member"; member: undefined }
  | { status: "member"; member: IUserLite };

// Which workspace member a profile page is about (M1 design 3.2). The page header and the user card both
// read it here, so they cannot disagree about a user, for instance one who has left the workspace.
export const useProfileMember = (workspaceSlug: string, userId: string): TProfileMember => {
  const {
    workspace: { fetchWorkspaceMembers, getWorkspaceMemberDetails },
  } = useMember();
  // The workspace wrapper already fetches the members under this key, so SWR serves the same request;
  // subscribing here is what tells whether the members are still loading or failed to load. A failed load
  // shows as a settled request without a list, not as SWR's `error`: the service rethrows only the response
  // body, which is undefined when the request got no response at all.
  const { data: members, isLoading } = useSWR(
    workspaceSlug ? WORKSPACE_MEMBERS(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceMembers(workspaceSlug) : null,
    {
      revalidateIfStale: false,
      revalidateOnFocus: false,
    }
  );
  const memberDetails = userId ? getWorkspaceMemberDetails(userId) : null;
  // A member removed from the workspace stays in the store, marked inactive: it is no longer a member.
  const member = memberDetails?.is_active === false ? undefined : memberDetails?.member;

  if (member) return { status: "member", member };
  if (isLoading) return { status: "loading", member: undefined };
  if (!members) return { status: "load-failed", member: undefined };
  return { status: "not-a-member", member: undefined };
};
