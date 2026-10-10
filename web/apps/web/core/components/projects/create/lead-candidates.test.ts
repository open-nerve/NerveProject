/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { describe, expect, it, vi } from "vitest";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { leadCandidates } from "./lead-candidates";

// Who the creation's form offers as a new project's lead (M3 design 3.19): of the workspace's members, those nerve
// takes as one.

// The builders' module builds the account's store, which imports the tab's session; nothing here reads it.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const memberships = [
  membershipOf("ada", { role: 20 }),
  membershipOf("bob"),
  membershipOf("gus", { role: 5 }),
  membershipOf("eve", { is_active: false }),
];
/** The members' store's membership of a member, of those above. */
const membershipOfMember = (memberId: string) => memberships.find(({ member }) => member.id === memberId) ?? null;

describe("leadCandidates", () => {
  it.each([
    { who: "an admin", memberId: "u-ada", offers: ["u-ada"] },
    { who: "a member", memberId: "u-bob", offers: ["u-bob"] },
    { who: "no guest", memberId: "u-gus", offers: [] },
    { who: "no member whose membership ended", memberId: "u-eve", offers: [] },
  ])("offers $who", ({ memberId, offers }) => {
    expect(leadCandidates([memberId], membershipOfMember)).toEqual(offers);
  });
});
