/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { answered } from "@/lib/auth/fake-nerve";
import { memberStore, membershipOf } from "@/store/member/workspace/fake-members";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The workspace members a project's modules list and archive filter by (P6a spec 2.4, R14), for their tests: the
// member store's own (fake-members.ts), with the members of acme, the workspace of the page's address, and of globex,
// which the router names (as it does in a tab whose router has not followed the address yet). A test file that uses
// it mocks @/hooks/store/use-member with the hook it gives, in jsdom (the account's store reads the tab's session from
// the browser's storage), under fake timers, as nerve's answers come.

/** acme's members as nerve lists them: the page's. */
const acmeMembers = [membershipOf("Ann"), membershipOf("Bob")];
/** globex's members as nerve lists them: the router's. */
const globexMembers = [membershipOf("Gil", {}, "globex")];

/**
 * The member store of a tab at a page of acme whose router names globex, nerve having listed both workspaces' members;
 * and the hook the pages read it by (useMember): the workspaces' members, and each member's details.
 */
export async function membersAtAnotherRouter() {
  const { nerve, store, users } = await memberStore("globex");
  const load = (slug: string, data: unknown) =>
    answered(
      nerve,
      () => store.fetchWorkspaceMembers(workspaceOf(slug)),
      ["GET", `/api/v0/workspaces/${slug}/members`],
      { data },
      `${slug}'s members`
    );
  await load("acme", acmeMembers);
  await load("globex", globexMembers);
  return { useMember: () => ({ workspace: store, getUserDetails: (id: string) => users[id] }) };
}
