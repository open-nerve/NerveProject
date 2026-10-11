/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import type { WorkspaceMember } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";

/**
 * Who the creation's form offers as a new project's lead, of the workspace's members by id, each by its membership
 * (the members' store's): the admins and members whose membership is active, as nerve takes a lead (M3 design 3.19);
 * no guest, and no one whose membership ended.
 */
export function leadCandidates(
  memberIds: readonly string[] | null,
  membershipOf: (memberId: string) => WorkspaceMember | null
): string[] {
  return (memberIds ?? []).filter((id) => {
    const membership = membershipOf(id);
    return membership !== null && membership.is_active && membership.role >= EUserPermissions.MEMBER;
  });
}
