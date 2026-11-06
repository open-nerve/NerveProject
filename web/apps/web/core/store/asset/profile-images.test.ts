/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { answered, json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadProject, loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import { assetTab } from "./fake-assets";
import { accountOf, userOf } from "@/store/member/fake-users";
import { membershipOf } from "@/store/member/workspace/fake-members";
import { profileOf, seedAccount } from "@/store/user/fake-account";

vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { start: async () => {}, state: { status: "signed-in" } },
  publicClient: {},
}));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const close: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers({ now: Date.parse("2026-11-04T10:00:00Z") }));
afterEach(() => {
  close.splice(0).forEach((dispose) => dispose());
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const expiry = () => new Date(Date.now() + 60_000).toISOString();
async function fixture() {
  const tab = await assetTab();
  close.push(() => tab.app.dispose());
  const workspace = { ...tab.workspace, logo_url: "/static/old", logo_url_expires_at: expiry() };
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [workspace]);
  tab.nerve.calls.length = 0;
  return { ...tab, workspace, images: tab.app.asset.images, readWorkspace: tab.app.workspaceRoot.fetchWorkspaces };
}

it("merges active workspace images into one directory read and reuses proof while they remain mounted", async () => {
  const { app, nerve, workspace, images, readWorkspace } = await fixture();
  const other = workspaceOf("other", { logo_url: "/static/other-old", logo_url_expires_at: expiry() });
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace, other]);
  nerve.calls.length = 0;
  const a = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const a2 = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const b = images.acquire({ kind: "workspace", workspaceID: other.id });
  const result = track(Promise.all([a.read(readWorkspace), a2.read(readWorkspace), b.read(readWorkspace)]));
  await until(() => nerve.calls.length > 0 || result.settled, "coalesced workspace images");
  expect(nerve.calls).toHaveLength(1);
  expect(a.url).toBeUndefined();
  nerve.calls[0]!.answer(
    json(200, {
      data: [
        { ...workspace, logo_url: "/static/new" },
        { ...other, logo_url: "/static/other-new" },
      ],
    })
  );
  await until(() => result.settled, "directory proof");
  expect([a.url, a2.url, b.url]).toEqual(["/static/new", "/static/new", "/static/other-new"]);
  const a3 = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  await a3.read(readWorkspace);
  expect(a3.url).toBe("/static/new");
  expect(nerve.calls).toHaveLength(1);
});

it("unknown, null and preset images neither fetch nor fall back to a capability", async () => {
  const { app, nerve, workspace, project, images, readWorkspace } = await fixture();
  await loadWorkspaces(nerve, app.workspaceRoot, [{ ...workspace, logo_url: null, logo_url_expires_at: null }]);
  await loadProjects(nerve, app.projectRoot.project, workspace, [{ ...project, cover_builtin_id: "cover-01" }]);
  nerve.calls.length = 0;
  const leases = [
    images.acquire({ kind: "self", image: "avatar" }),
    images.acquire({ kind: "workspace", workspaceID: workspace.id }),
    images.acquire({ kind: "project", workspaceID: workspace.id, projectID: project.id }),
    images.acquire({ kind: "member", workspaceID: workspace.id, memberID: "unknown" }),
  ];
  await Promise.all(leases.map((lease) => lease.read(readWorkspace)));
  expect(leases.map((lease) => lease.url)).toEqual([undefined, undefined, undefined, undefined]);
  expect(nerve.calls).toHaveLength(0);
});

it("a later view awaits its shared in-flight owner read and sees its refusal", async () => {
  const { nerve, workspace, images, readWorkspace } = await fixture();
  const a = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const first = track(a.read(readWorkspace));
  await until(() => nerve.calls.length === 1, "shared read");
  const b = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const second = track(b.read(readWorkspace));
  await Promise.resolve();
  await Promise.resolve();
  expect(second.settled).toBe(false);
  nerve.calls[0]!.answer(problem(500, "unavailable"));
  await until(() => first.settled && second.settled, "shared refusal");
  expect(first.error).toMatchObject({ status: 500 });
  expect(second.error).toMatchObject({ status: 500 });
  expect(nerve.calls).toHaveLength(1);
});

it("a new scope joins the directory refusal and only explicit retry sends another read", async () => {
  const { app, nerve, workspace, images, readWorkspace } = await fixture();
  const other = workspaceOf("other", { logo_url: "/static/other", logo_url_expires_at: expiry() });
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace, other]);
  nerve.calls.length = 0;
  const a = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const first = track(a.read(readWorkspace));
  await until(() => nerve.calls.length === 1, "shared scope read");
  const b = images.acquire({ kind: "workspace", workspaceID: other.id });
  const second = track(b.read(readWorkspace));
  nerve.calls[0]!.answer(problem(500, "unavailable"));
  await until(() => first.settled && second.settled, "shared refusal");
  expect(first.error).toMatchObject({ status: 500 });
  expect(second.error).toMatchObject({ status: 500 });
  expect(nerve.calls).toHaveLength(1);
  b.requestRetry();
  const retry = track(b.read(readWorkspace));
  await until(() => nerve.calls.length === 2, "explicit retry");
  nerve.calls[1]!.answer(json(200, { data: [workspace, other] }));
  await until(() => retry.settled, "retried proof");
  expect(b.url).toBe("/static/other");
  expect(retry.error).toBeUndefined();
});

it("a newer successful host read proves the active image; its old reply cannot strand it", async () => {
  const { app, nerve, workspace, images, readWorkspace } = await fixture();
  const lease = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const old = track(lease.read(readWorkspace));
  await until(() => nerve.calls.length === 1, "image read");
  await loadWorkspaces(nerve, app.workspaceRoot, [{ ...workspace, logo_url: "/static/other-reader" }]);
  expect(lease.url).toBe("/static/other-reader");
  nerve.calls[0]!.answer(json(200, { data: [workspace] }));
  await until(() => old.settled, "superseded read");
  expect(lease.url).toBe("/static/other-reader");
  expect(lease.error).toBeUndefined();
  expect(nerve.calls).toHaveLength(2);
});

it("expiry is checked at access even before a delayed timer, and invalid refreshed expiry never loops", async () => {
  const { nerve, workspace, images, readWorkspace } = await fixture();
  nerve.respond = (call) => call.answer(json(200, { data: [workspace] }));
  const lease = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  await lease.read(readWorkspace);
  expect(lease.url).toBe(workspace.logo_url);
  lease.arm();
  const stamp = lease.stamp;
  vi.setSystemTime(Date.now() + 60_001);
  expect(lease.url).toBeUndefined();
  expect(lease.expire(stamp)).toBe(true);
  expect(lease.expire(stamp)).toBe(false);
  await lease.read(readWorkspace);
  expect(lease.url).toBeUndefined();
  lease.arm();
  expect(lease.expire(lease.stamp)).toBe(false);
  await lease.read(readWorkspace);
  expect(nerve.calls).toHaveLength(2);
});

it("image error reauthorizes once, blocks another bad image, and ignores a late old-image error", async () => {
  const { nerve, workspace, images, readWorkspace } = await fixture();
  let image = 0;
  nerve.respond = (call) => call.answer(json(200, { data: [{ ...workspace, logo_url: `/static/image-${++image}` }] }));
  const lease = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  await lease.read(readWorkspace);
  const first = lease.stamp;
  expect(lease.fail(first)).toBe(true);
  expect(lease.url).toBeUndefined();
  await lease.read(readWorkspace);
  expect(lease.url).toBe("/static/image-2");
  expect(lease.fail(first)).toBe(false);
  expect(lease.url).toBe("/static/image-2");
  expect(lease.fail(lease.stamp)).toBe(false);
  expect(lease.url).toBeUndefined();
  await lease.read(readWorkspace);
  expect(nerve.calls).toHaveLength(2);
  lease.requestRetry();
  await lease.read(readWorkspace);
  expect(lease.url).toBe("/static/image-3");
});

it("last release retires the old view while a fresh view may join the current in-flight host read", async () => {
  const { app, nerve, workspace, images, readWorkspace } = await fixture();
  const first = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const old = track(first.read(readWorkspace));
  await until(() => nerve.calls.length === 1, "old image read");
  first.dispose();
  first.dispose();
  const fresh = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const next = track(fresh.read(readWorkspace));
  expect(nerve.calls).toHaveLength(1);
  nerve.calls[0]!.answer(json(200, { data: [{ ...workspace, logo_url: "/static/fresh" }] }));
  await until(() => next.settled && old.settled, "joined proof");
  expect(first.url).toBeUndefined();
  expect(fresh.url).toBe("/static/fresh");
  app.dispose();
  expect(fresh.url).toBeUndefined();
  expect(() => images.acquire({ kind: "workspace", workspaceID: workspace.id })).toThrow();
});

it("PC1 public nonmember cover cannot revive after directory absence; a fresh lifetime requires a fresh read", async () => {
  const { app, nerve, workspace, project, images } = await fixture();
  const publicProject = {
    ...project,
    member_role: null,
    cover_image_url: "/static/old-cover",
    cover_image_url_expires_at: expiry(),
  };
  await loadProjects(nerve, app.projectRoot.project, workspace, [publicProject]);
  await loadProject(nerve, app.projectRoot.project, publicProject);
  nerve.calls.length = 0;
  const source = { kind: "project", workspaceID: workspace.id, projectID: project.id } as const;
  const old = images.acquire(source);
  const read = track(old.read(() => app.projectRoot.project.fetchProject(project.id)));
  await until(() => nerve.calls.length > 0 || read.settled, "nonmember cover read");
  expect(nerve.calls).toHaveLength(1);
  await loadProjects(nerve, app.projectRoot.project, workspace, []);
  await loadProjects(nerve, app.projectRoot.project, workspace, [publicProject]);
  nerve.calls[0]!.answer(json(200, publicProject));
  await until(() => read.settled, "old nonmember cover");
  expect(old.url).toBeUndefined();
  expect(old.current).toBe(false);
  expect(app.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, project.id)).toBeUndefined();
  const fresh = images.acquire(source);
  const at = nerve.calls.length;
  const next = track(fresh.read(() => app.projectRoot.project.fetchProject(project.id)));
  await until(() => nerve.calls.length > at || next.settled, "fresh cover proof");
  expect(nerve.calls).toHaveLength(at + 1);
  nerve.calls[at]!.answer(json(200, { ...publicProject, cover_image_url: "/static/new-cover" }));
  await until(() => next.settled, "fresh cover");
  expect(fresh.url).toBe("/static/new-cover");
  expect(old.url).toBeUndefined();
});

it.each([403, 500])("failed owner response %i leaves placeholder until manual retry", async (status) => {
  const { nerve, workspace, images, readWorkspace } = await fixture();
  nerve.respond = (call) => call.answer(problem(status, "refused"));
  const lease = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  await expect(lease.read(readWorkspace)).rejects.toMatchObject({ status });
  expect(lease.url).toBeUndefined();
  expect(lease.error).toMatchObject({ status });
  await vi.advanceTimersByTimeAsync(180_000);
  await lease.read(readWorkspace);
  expect(nerve.calls).toHaveLength(1);
});

it("the same member in two workspaces uses each membership's URL, never the global profile cache", async () => {
  const { app, nerve, workspace, images } = await fixture();
  const other = workspaceOf("other");
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace, other]);
  const member = (url: string, workspaceID: string) =>
    membershipOf("bob", {
      workspace_id: workspaceID,
      member: { ...userOf("bob"), avatar_url: url, avatar_url_expires_at: expiry() },
    });
  const load = (scope: typeof other) =>
    answered(
      nerve,
      () => app.memberRoot.workspace.fetchWorkspaceMembers(scope),
      ["GET", `/api/v0/workspaces/${scope.slug}/members`],
      { data: [member(`/static/${scope.slug}-old`, scope.id)] },
      "members"
    );
  // The second list intentionally overwrites the global profile, which must not authorize the first scope.
  await load(workspace);
  await load(other);
  nerve.calls.length = 0;
  nerve.respond = (call) => {
    if (!call.path.endsWith("/members")) return;
    const scope = call.path.includes(`/${workspace.slug}/`) ? workspace : other;
    call.answer(json(200, { data: [member(`/static/${scope.slug}`, scope.id)] }));
  };
  const a = images.acquire({ kind: "member", workspaceID: workspace.id, memberID: "u-bob" });
  const a2 = images.acquire({ kind: "member", workspaceID: workspace.id, memberID: "u-bob" });
  const b = images.acquire({ kind: "member", workspaceID: other.id, memberID: "u-bob" });
  await Promise.all([
    a.read(() => app.memberRoot.workspace.fetchWorkspaceMembers(workspace)),
    a2.read(() => app.memberRoot.workspace.fetchWorkspaceMembers(workspace)),
    b.read(() => app.memberRoot.workspace.fetchWorkspaceMembers(other)),
  ]);
  expect(nerve.calls).toHaveLength(2);
  expect([a.url, a2.url, b.url]).toEqual([`/static/${workspace.slug}`, `/static/${workspace.slug}`, "/static/other"]);
  expect(app.memberRoot.memberMap["u-bob"]?.avatar_url).toBe("/static/other");
  await loadWorkspaces(nerve, app.workspaceRoot, [{ ...workspace, role: 5 }, other]);
  expect([a.url, a2.url, b.url]).toEqual([undefined, undefined, "/static/other"]);
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace, other]);
  expect(a.current).toBe(false);
  const fresh = images.acquire({ kind: "member", workspaceID: workspace.id, memberID: "u-bob" });
  expect(fresh.url).toBeUndefined();
  const count = nerve.calls.length;
  await fresh.read(() => app.memberRoot.workspace.fetchWorkspaceMembers(workspace));
  expect(nerve.calls).toHaveLength(count + 1);
  expect(fresh.url).toBe(`/static/${workspace.slug}`);
});

it("guest does not fetch other members; his own avatar and cover share the me read", async () => {
  const { app, nerve, workspace, images } = await fixture();
  const user = {
    ...accountOf("bob"),
    avatar_url: "/static/me",
    cover_image_url: "/static/cover",
    avatar_url_expires_at: expiry(),
    cover_image_url_expires_at: expiry(),
  };
  await seedAccount(nerve, app.user, user);
  await loadWorkspaces(nerve, app.workspaceRoot, [{ ...workspace, role: 5 }]);
  nerve.calls.length = 0;
  nerve.respond = (call) => call.answer(json(200, call.path === "/api/v0/me" ? user : profileOf()));
  const other = images.acquire({ kind: "member", workspaceID: workspace.id, memberID: "u-other" });
  const own = images.acquire({ kind: "member", workspaceID: workspace.id, memberID: user.id });
  const avatar = images.acquire({ kind: "self", image: "avatar" });
  const cover = images.acquire({ kind: "self", image: "cover" });
  await Promise.all([
    other.read(app.user.fetchCurrentUser),
    own.read(app.user.fetchCurrentUser),
    avatar.read(app.user.fetchCurrentUser),
    cover.read(app.user.fetchCurrentUser),
  ]);
  expect([own.url, avatar.url, cover.url, other.url]).toEqual(["/static/me", "/static/me", "/static/cover", undefined]);
  expect(nerve.calls.map((call) => call.path).toSorted()).toEqual(["/api/v0/me", "/api/v0/me/profile"]);
});

it("a new workspace view joins a shared read only within the same permission lifetime", async () => {
  const { app, nerve, workspace, images, readWorkspace } = await fixture();
  const other = workspaceOf("other", { logo_url: "/static/other", logo_url_expires_at: expiry() });
  await loadWorkspaces(nerve, app.workspaceRoot, [workspace, other]);
  nerve.calls.length = 0;
  const a = images.acquire({ kind: "workspace", workspaceID: workspace.id });
  const first = track(a.read(readWorkspace));
  await until(() => nerve.calls.length === 1, "first scope read");
  const b = images.acquire({ kind: "workspace", workspaceID: other.id });
  const second = track(b.read(readWorkspace));
  nerve.calls[0]!.answer(json(200, { data: [workspace, other] }));
  await until(() => first.settled && second.settled, "shared scope proof");
  expect(nerve.calls).toHaveLength(1);
  expect(b.url).toBe("/static/other");
});
