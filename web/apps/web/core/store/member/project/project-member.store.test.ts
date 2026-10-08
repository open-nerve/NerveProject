/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ProjectMember, ProjectMembersAdd, ProjectRole } from "@nerve/api-client";
import { ApiError } from "@/lib/api-error";
import { FakeNerve, answered, json, noContent, problem } from "@/lib/auth/fake-nerve";
import { settle, track, until } from "@/lib/auth/fake-time";
import { fetchedWhileChangeIsOut, inTurn, sent } from "@/store/fake-queue";
import { fakeRoot } from "@/store/fake-root";
import { ProjectMemberStore } from "@/store/member/project/project-member.store";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { loadArchivedProjects, projectOf, projectTab } from "@/store/project/fake-projects";
import { UserStore } from "@/store/user";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

// The members of a project (M3 design 7.3), against a fake nerve that answers each request when the test says. The
// caller is ann, the admin of acme and of web, whose address the tab has; ops is acme's other project, and his other
// workspace, beta, has lab.

// The account's store reads the tab's session as it fetches the account, which no test here does.
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));

const acme = workspaceOf("acme", { role: 20 });
const beta = workspaceOf("beta");
const web = projectOf("WEB", acme.id, { member_role: 20, member_ids: ["u-ann", "u-bob", "u-cat"] });
const ops = projectOf("OPS", acme.id);
const lab = projectOf("LAB", beta.id, { member_ids: ["u-ann"] });
const MEMBERS = `/api/v0/projects/${web.id}/members`;

/** A membership of a project (web unless it says) as nerve lists it: the name names the member and the membership. */
const memberOf = (name: string, role: ProjectRole = 15, projectId = web.id): ProjectMember => ({
  id: `pm-${projectId}-${name}`,
  project_id: projectId,
  member_id: `u-${name}`,
  role,
  created_at: "2026-10-01T09:00:00Z",
});
const ann = memberOf("ann", 20);
const bob = memberOf("bob");
const cat = memberOf("cat", 5);
const dee = memberOf("dee");
/** Bob's membership as nerve answers his change to a guest: its created_at, which only nerve gives, is not the list's. */
const demoted: ProjectMember = { ...memberOf("bob", 5), created_at: "2026-10-03T09:00:00Z" };
/** Dee, a member of the workspace, added to web. */
const adding: ProjectMembersAdd = { members: [{ member_id: "u-dee", role: 15 }] };
/** The profiles the workspace's members gave. */
const profile = (name: string) => membershipOf(name).member;
const membership = (name: string) => `/api/v0/project-members/pm-${web.id}-${name}`;

/** The store of a tab at web's address, whose caller is ann, and whose projects nerve listed: acme's and beta's. */
async function memberStore() {
  const { nerve, api, router, projectRoot } = await projectTab(
    { workspace: acme, projects: [web, ops] },
    { workspace: beta, projects: [lab] }
  );
  const user = new UserStore(fakeRoot({ router }), api);
  runInAction(() => {
    const account = { email: "ann@example.com", user_timezone: "UTC", cover_image_url: null };
    user.data = { ...profile("ann"), ...account, created_at: "2026-09-01T09:00:00Z" };
  });
  const users = Object.fromEntries(["ann", "bob", "cat", "dee"].map((name) => [`u-${name}`, profile(name)]));
  const store = new ProjectMemberStore({ memberMap: users }, fakeRoot({ router, user, projectRoot }), api);
  return { nerve, api, projects: projectRoot.project, store };
}

/** The store fetches the members of a project (web unless it says), and nerve lists these. */
function load(nerve: FakeNerve, store: ProjectMemberStore, memberships: ProjectMember[], projectId = web.id) {
  const fetch = () => store.fetchProjectMembers(projectId);
  return answered(nerve, fetch, ["GET", `/api/v0/projects/${projectId}/members`], { data: memberships }, "members");
}

/** A store whose members of web nerve gave as ann, bob and cat. */
async function loaded() {
  const tab = await memberStore();
  await load(tab.nerve, tab.store, [ann, bob, cat]);
  return tab;
}

beforeEach(() => {
  vi.useFakeTimers({ now: 1_000_000 });
});
afterEach(() => {
  vi.useRealTimers();
});

describe("ProjectMemberStore, the members", () => {
  it("keeps a project's memberships as nerve lists them, each with its member's profile", async () => {
    const { nerve, store } = await memberStore();
    const fetched = await load(nerve, store, [bob, cat, ann]);
    expect(fetched.value).toEqual({ "u-ann": ann, "u-bob": bob, "u-cat": cat });
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...bob, member: profile("bob") });
    expect(store.getProjectMemberDetails("u-dee", web.id)).toBeNull();
    // the caller first, then by display name; the guest when asked for
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
    expect(store.getProjectMemberIds(web.id, false)).toEqual(["u-ann", "u-bob"]);
    expect(store.getProjectMemberIds(ops.id, true)).toBeNull();
    // the address's project's, by the members page's order and filters
    store.filters.updateFilters(web.id, { order_by: "-display_name" });
    expect(store.projectMemberIds).toEqual(["u-cat", "u-bob", "u-ann"]);
    store.filters.updateFilters(web.id, { roles: ["5"] });
    expect(store.getFilteredProjectMemberDetails("u-cat", web.id)).toEqual({ ...cat, member: profile("cat") });
    expect(store.getFilteredProjectMemberDetails("u-bob", web.id)).toBeNull();
  });

  it("shows none of the members of a project the caller left", async () => {
    const { nerve, projects, store } = await loaded();
    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    expect(store.getProjectMemberIds(web.id, true)).toBeNull();
    expect(store.getProjectMemberDetails("u-bob", web.id)).toBeNull();
    expect(store.projectMemberIds).toBeNull();
  });

  it("fails when nerve refuses them, keeping none, and again, keeping the members it had", async () => {
    const { nerve, store } = await memberStore();
    const refused = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 1, "the members");
    nerve.calls[0]?.answer(problem(403, "forbidden"));
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectMemberIds(web.id, true)).toBeNull();

    await load(nerve, store, [ann, bob]);
    const again = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 3, "the refetch");
    nerve.calls[2]?.answer(problem(503, "server_busy"));
    await until(() => again.settled, "the refusal");
    expect(again.error).toBeInstanceOf(ApiError);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob"]);
  });

  it("keeps the members it had, gives nothing and does not fail, when the session changes as it fetches again", async () => {
    const { nerve, api, store } = await loaded();
    FakeNerve.replaceSession(api);
    const fetched = await settle(store.fetchProjectMembers(web.id), "the refetch");
    expect(fetched).toEqual({ settled: true, value: undefined });
    expect(nerve.calls).toHaveLength(1);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
  });
});

describe("ProjectMemberStore, the changes", () => {
  it("adds the members nerve answers, whom the project then has among its members", async () => {
    const { nerve, projects, store } = await loaded();
    const added = track(store.bulkAddMembersToProject(web.id, adding));
    await until(() => nerve.calls.length === 2, "the addition");
    expect(nerve.calls[1]).toMatchObject({ method: "POST", path: MEMBERS, body: adding });
    expect(store.getProjectMemberDetails("u-dee", web.id)).toBeNull();
    nerve.calls[1]?.answer(json(201, { data: [dee] }));
    await until(() => added.settled, "the answer");
    expect(added.value).toEqual([dee]);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat", "u-dee"]);
    expect(projects.getProjectById(web.id)?.member_ids).toEqual(["u-ann", "u-bob", "u-cat", "u-dee"]);
  });

  it("changes a role to nerve's answer, and the caller's own role in the project with his", async () => {
    const { nerve, projects, store } = await loaded();
    const changed = track(store.updateMemberRole(web.id, "u-bob", 5));
    await until(() => nerve.calls.length === 2, "the change");
    expect(nerve.calls[1]).toMatchObject({ method: "PATCH", path: membership("bob"), body: { role: 5 } });
    expect(store.getProjectMemberDetails("u-bob", web.id)?.role).toBe(15);
    nerve.calls[1]?.answer(json(200, demoted));
    await until(() => changed.settled, "the answer");
    expect(changed.value).toEqual(demoted);
    expect(store.getProjectMemberDetails("u-bob", web.id)).toEqual({ ...demoted, member: profile("bob") });
    expect(projects.getProjectById(web.id)?.member_role).toBe(20);

    // ann, an admin of the workspace, changes her own role: the project takes the role nerve answers, which the test
    // makes differ from the one she asked for
    const own: ProjectMember = { ...ann, role: 5 };
    await sent(nerve, () => store.updateMemberRole(web.id, "u-ann", 15), ["PATCH", membership("ann")], json(200, own));
    expect(store.getProjectMemberDetails("u-ann", web.id)?.role).toBe(5);
    expect(projects.getProjectById(web.id)?.member_role).toBe(5);
  });

  it("removes a member, whom the project then no longer has among its members", async () => {
    const { nerve, projects, store } = await loaded();
    const removed = await sent(
      nerve,
      () => store.removeMemberFromProject(web.id, "u-bob"),
      ["DELETE", membership("bob")],
      noContent()
    );
    expect(removed.error).toBeUndefined();
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-cat"]);
    expect(projects.getProjectById(web.id)?.member_ids).toEqual(["u-ann", "u-cat"]);
  });

  it("adds members to a project of his other workspace: it has them, and the address's project does not", async () => {
    const { nerve, projects, store } = await loaded();
    await load(nerve, store, [memberOf("ann", 15, lab.id)], lab.id);
    const labs = `/api/v0/projects/${lab.id}/members`;
    const added = json(201, { data: [memberOf("dee", 15, lab.id)] });
    await sent(nerve, () => store.bulkAddMembersToProject(lab.id, adding), ["POST", labs], added);
    expect(store.getProjectMemberIds(lab.id, true)).toEqual(["u-ann", "u-dee"]);
    expect(projects.getProjectById(lab.id)?.member_ids).toEqual(["u-ann", "u-dee"]);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
    expect(projects.getProjectById(web.id)).toEqual(web);
  });

  it("removes a member of an archived project, which the archived projects then show without him", async () => {
    const { nerve, projects, store } = await loaded();
    const old = projectOf("OLD", acme.id, { archived_at: "2026-10-02T09:00:00Z", member_ids: ["u-ann", "u-bob"] });
    await loadArchivedProjects(nerve, projects, acme, [old]);
    await load(nerve, store, [memberOf("ann", 20, old.id), memberOf("bob", 15, old.id)], old.id);
    const bobs = `/api/v0/project-members/pm-${old.id}-bob`;
    await sent(nerve, () => store.removeMemberFromProject(old.id, "u-bob"), ["DELETE", bobs], noContent());
    expect(projects.getProjectById(old.id)?.member_ids).toEqual(["u-ann"]);
  });

  const refusals: { change: string; send: (store: ProjectMemberStore) => Promise<unknown>; refusal: Response }[] = [
    {
      change: "an addition",
      send: (store) => store.bulkAddMembersToProject(web.id, adding),
      refusal: problem(422, "validation_failed"),
    },
    {
      change: "a role change",
      send: (store) => store.updateMemberRole(web.id, "u-bob", 20),
      refusal: problem(409, "project.role_too_high"),
    },
    {
      change: "a removal",
      send: (store) => store.removeMemberFromProject(web.id, "u-ann"),
      refusal: problem(409, "project.own_membership"),
    },
  ];
  it.each(refusals)("fails, changing nothing, when nerve refuses $change", async ({ send, refusal }) => {
    const { nerve, projects, store } = await loaded();
    const refused = track(send(store));
    await until(() => nerve.calls.length === 2, "the change");
    nerve.calls[1]?.answer(refusal);
    await until(() => refused.settled, "the refusal");
    expect(refused.error).toBeInstanceOf(ApiError);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-cat"]);
    expect(store.getProjectMemberDetails("u-bob", web.id)?.role).toBe(15);
    expect(projects.getProjectById(web.id)).toEqual(web);
  });

  it.each(refusals.slice(1))(
    "fails, asking nerve nothing, for $change of a member it has not listed",
    async ({ send }) => {
      const { nerve, store } = await memberStore();
      const refused = await settle(send(store), "the change");
      expect(refused).toMatchObject({ settled: true, error: new Error("Member not found") });
      expect(nerve.calls).toEqual([]);
    }
  );

  it("fails, asking nerve nothing, for a change of a member its list lacks, or of a project the caller left", async () => {
    const { nerve, projects, store } = await memberStore();
    await load(nerve, store, [ann, cat]);
    const unlisted = await settle(store.updateMemberRole(web.id, "u-bob", 5), "the change");
    expect(unlisted).toMatchObject({ settled: true, error: new Error("Member not found") });
    await sent(nerve, () => projects.leaveProject(web), ["POST", `/api/v0/projects/${web.id}/leave`], noContent());
    const left = await settle(store.removeMemberFromProject(web.id, "u-cat"), "the change");
    expect(left).toMatchObject({ settled: true, error: new Error("Member not found") });
    expect(nerve.calls).toHaveLength(2);
  });

  it("sends each change once nerve has answered the one before it, refused or not, finding the member in its turn", async () => {
    const { nerve, store } = await loaded();
    const promoted = track(store.updateMemberRole(web.id, "u-bob", 20));
    const changed = track(store.updateMemberRole(web.id, "u-bob", 5));
    const removed = track(store.removeMemberFromProject(web.id, "u-bob"));
    // made while bob is still listed; their turns come after his removal
    const changedLate = track(store.updateMemberRole(web.id, "u-bob", 15));
    const removedLate = track(store.removeMemberFromProject(web.id, "u-bob"));
    await inTurn(nerve, 1, ["PATCH", membership("bob")], problem(503, "server_busy"));
    await inTurn(nerve, 2, ["PATCH", membership("bob")], json(200, demoted));
    await inTurn(nerve, 3, ["DELETE", membership("bob")], noContent());
    await until(() => changedLate.settled && removedLate.settled, "the changes after the removal");
    expect(promoted.error).toBeInstanceOf(ApiError);
    expect(changed.value).toEqual(demoted);
    expect(removed).toEqual({ settled: true, value: undefined });
    expect(changedLate).toMatchObject({ settled: true, error: new Error("Member not found") });
    expect(removedLate).toMatchObject({ settled: true, error: new Error("Member not found") });
    expect(nerve.calls).toHaveLength(4);
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-cat"]);
  });

  it("fetches the members while a change is out: a fetch does not wait for it", async () => {
    const { nerve, store } = await loaded();
    await fetchedWhileChangeIsOut(
      nerve,
      { send: () => store.updateMemberRole(web.id, "u-bob", 5), request: ["PATCH", membership("bob")] },
      { send: () => store.fetchProjectMembers(web.id), request: ["GET", MEMBERS], body: { data: [ann] } }
    );
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann"]);
  });
});

// A fetch's answer may be older than a change nerve confirmed while it was out: what the fetch shows carries the
// change (the order is forced: the fetch waits until the test answers it, after the change has finished). Of two
// fetches of one project's members, only the newer writes. An addition the refetch lists already cannot show twice:
// the memberships are kept by member, and the project takes only the added members its own refetch does not list.
describe("ProjectMemberStore, while a fetch is out", () => {
  it("keeps an addition, a role change and a removal nerve confirmed during a refetch", async () => {
    const { nerve, store } = await loaded();
    const refetched = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 2, "the refetch");
    await sent(
      nerve,
      () => store.bulkAddMembersToProject(web.id, adding),
      ["POST", MEMBERS],
      json(201, { data: [dee] })
    );
    await sent(
      nerve,
      () => store.updateMemberRole(web.id, "u-bob", 5),
      ["PATCH", membership("bob")],
      json(200, demoted)
    );
    await sent(nerve, () => store.removeMemberFromProject(web.id, "u-cat"), ["DELETE", membership("cat")], noContent());
    // the list was read before all three
    nerve.calls[1]?.answer(json(200, { data: [ann, bob, cat] }));
    await until(() => refetched.settled, "the refetch");

    expect(refetched.value).toEqual({ "u-ann": ann, "u-bob": demoted, "u-dee": dee });
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob", "u-dee"]);
  });

  it.each([
    { read: "before", listed: ["u-ann", "u-bob", "u-cat"] },
    { read: "after", listed: ["u-ann", "u-bob", "u-cat", "u-dee"] },
  ])("keeps an addition on the project once, its workspace's projects read $read it", async ({ listed }) => {
    const { nerve, projects, store } = await loaded();
    const refetched = track(projects.fetchProjects(acme));
    await until(() => nerve.calls.length === 2, "the projects");
    await sent(
      nerve,
      () => store.bulkAddMembersToProject(web.id, adding),
      ["POST", MEMBERS],
      json(201, { data: [dee] })
    );
    nerve.calls[1]?.answer(json(200, { data: [{ ...web, member_ids: listed }, ops] }));
    await until(() => refetched.settled, "the projects");
    expect(projects.getProjectById(web.id)?.member_ids).toEqual(["u-ann", "u-bob", "u-cat", "u-dee"]);
  });

  it("lets each project's newer fetch write: an older one answering last writes nothing", async () => {
    const { nerve, store } = await memberStore();
    const older = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 1, "the older members");
    const elsewhere = track(store.fetchProjectMembers(ops.id));
    await until(() => nerve.calls.length === 2, "ops's members");
    const newer = track(store.fetchProjectMembers(web.id));
    await until(() => nerve.calls.length === 3, "the newer members");
    nerve.calls[2]?.answer(json(200, { data: [ann, demoted] }));
    await until(() => newer.settled, "the newer members");
    // a newer fetch of web's members does not overtake one of ops's
    nerve.calls[1]?.answer(json(200, { data: [memberOf("dee", 15, ops.id)] }));
    await until(() => elsewhere.settled, "ops's members");
    // read before bob's change
    nerve.calls[0]?.answer(json(200, { data: [ann, bob] }));
    await until(() => older.settled, "the older members");

    expect(older).toEqual({ settled: true, value: undefined });
    expect(store.getProjectMemberIds(web.id, true)).toEqual(["u-ann", "u-bob"]);
    expect(store.getProjectMemberDetails("u-bob", web.id)?.role).toBe(5);
    expect(store.getProjectMemberIds(ops.id, true)).toEqual(["u-dee"]);
  });
});
