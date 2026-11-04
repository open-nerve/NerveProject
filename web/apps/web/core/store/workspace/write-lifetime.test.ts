/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { ApiError } from "@/lib/api-error";
import { json, problem } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { assetTab } from "@/store/asset/fake-assets";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "./fake-workspaces";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
let tab: Awaited<ReturnType<typeof assetTab>>;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await assetTab();
  tab.nerve.respond = undefined;
});
afterEach(() => {
  tab.app.dispose();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
const patches = () => tab.nerve.to(`/api/v0/workspaces/${tab.workspace.slug}`).filter((c) => c.method === "PATCH");
const firstPatch = () => {
  const call = patches()[0];
  if (!call) throw new Error("expected a workspace PATCH");
  return call;
};
const update = () => track(tab.app.workspaceRoot.updateWorkspace(tab.workspace.slug, { logo_asset_id: "old-upload" }));
const oldResponse = () => ({
  ...tab.workspace,
  logo_asset_id: "old-upload",
  logo_url: "/static/old-capability",
  logo_url_expires_at: "2030-01-01T00:00:00Z",
});

it.each(["downgraded", "guest", "removed", "same-slug-new-id", "role-aba", "membership-aba"])(
  "late binding cannot publish or complete after %s, while the original upload remains retired",
  async (transition) => {
    const { app, nerve, workspace } = tab;
    const target = { purpose: "WORKSPACE_LOGO", scope: { kind: "workspace", id: workspace.id } } as const;
    const oldUpload = app.asset.createUpload(target);
    const binding = update();
    await until(() => patches().length === 1, "old binding request");
    const member = { ...workspace, role: transition === "guest" ? (5 as const) : (15 as const) };
    const recovered = {
      ...workspace,
      logo_asset_id: "new-picture",
      ...(transition === "same-slug-new-id" ? { id: "new-workspace" } : {}),
    };
    const loss = ["downgraded", "guest", "role-aba"].includes(transition) ? [member] : [];
    await loadWorkspaces(nerve, app.workspaceRoot, loss);
    expect(oldUpload.current).toBe(false);
    const recover = ["same-slug-new-id", "role-aba", "membership-aba"].includes(transition);
    if (recover) await loadWorkspaces(nerve, app.workspaceRoot, [recovered]);
    const expected = app.workspaceRoot.workspaces;
    firstPatch().answer(json(200, oldResponse()));
    await until(() => binding.settled, "late binding response");
    expect.soft(app.workspaceRoot.workspaces).toEqual(expected);
    expect(oldUpload.current).toBe(false);
    expect(binding).toEqual({ settled: true, value: undefined });
    if (["downgraded", "guest"].includes(transition)) expect(app.asset.createUpload(target).current).toBe(false);
  }
);

it.each([false, true])("late refusal is silent only after its scope has ended: retired=%s", async (retired) => {
  const pending = update();
  await until(() => patches().length === 1, "write");
  if (retired) await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  firstPatch().answer(problem(403, "forbidden"));
  await until(() => pending.settled, "refusal");
  if (retired) expect(pending).toEqual({ settled: true, value: undefined });
  else expect(pending.error).toBeInstanceOf(ApiError);
});

it("permissions are captured before queuing: loss and recovery prevent the next PATCH from starting", async () => {
  const first = update();
  await until(() => patches().length === 1, "first write");
  const second = update();
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
  firstPatch().answer(json(200, oldResponse()));
  await until(() => (first.settled && second.settled) || patches().length === 2, "queue either stops or sends");
  expect(patches()).toHaveLength(1);
  expect([first, second]).toEqual([
    { settled: true, value: undefined },
    { settled: true, value: undefined },
  ]);
});

it.each([false, true])(
  "root retirement prevents completion, and prevents new requests: refused=%s",
  async (refused) => {
    const pending = update();
    await until(() => patches().length === 1, "write");
    tab.app.dispose();
    const after = update();
    firstPatch().answer(refused ? problem(500, "internal_error") : json(200, oldResponse()));
    await until(() => (pending.settled && after.settled) || patches().length > 1, "retired updates");
    expect(patches()).toHaveLength(1);
    expect([pending, after]).toEqual([
      { settled: true, value: undefined },
      { settled: true, value: undefined },
    ]);
    expect(tab.app.workspaceRoot.getWorkspaceBySlug(tab.workspace.slug)).toEqual(tab.workspace);
  }
);

it("an unknown workspace cannot bind to a later workspace that reused the slug", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, []);
  const pending = update();
  await until(() => pending.settled || patches().length > 0, "unknown scope");
  expect(patches()).toHaveLength(0);
  expect(pending).toEqual({ settled: true, value: undefined });
});

it("unchanged directory and unrelated workspace/project changes do not cancel a valid update", async () => {
  const elsewhere = workspaceOf("else", { role: 20 });
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, elsewhere]);
  const pending = update();
  await until(() => patches().length === 1, "write");
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, { ...elsewhere, role: 15 }]);
  firstPatch().answer(json(200, oldResponse()));
  await until(() => pending.settled, "current write");
  expect(pending.value).toEqual(oldResponse());
  expect(tab.app.workspaceRoot.getWorkspaceBySlug(tab.workspace.slug)?.logo_asset_id).toBe("old-upload");
});

it("a guest-to-admin change cannot turn an operation begun in the guest lifetime into an admin confirmation", async () => {
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
  const pending = update();
  await until(() => patches().length === 1, "guest operation");
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
  firstPatch().answer(json(200, oldResponse()));
  await until(() => pending.settled, "old guest confirmation");
  expect(tab.app.workspaceRoot.workspaces).toEqual([tab.workspace]);
  expect(pending).toEqual({ settled: true, value: undefined });
});

it.each([20, 15, 5] as const)("confirmation replay respects the untouched directory's role %s", async (role) => {
  const reading = track(tab.app.workspaceRoot.fetchWorkspaces());
  await until(() => tab.nerve.to("/api/v0/workspaces").length === 1, "held directory");
  const directory = tab.nerve.to("/api/v0/workspaces")[0];
  if (!directory) throw new Error("expected a directory request");
  const pending = update();
  await until(() => patches().length === 1, "write");
  firstPatch().answer(json(200, oldResponse()));
  await until(() => pending.settled, "confirmed update");
  expect(pending.value).toEqual(oldResponse());
  const row = { ...tab.workspace, role, logo_asset_id: "directory-picture" };
  directory.answer(json(200, { data: [row] }));
  await until(() => reading.settled, "directory replay");
  expect(tab.app.workspaceRoot.workspaces).toEqual([role === 20 ? oldResponse() : row]);
  expect(reading.value).toEqual(tab.app.workspaceRoot.workspaces);
});
