/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { loadProject, loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { track, until } from "@/lib/auth/fake-time";
import { assetTab, avatarTarget, imageFile } from "./fake-assets";
import type { ProfileUploadTarget } from "./upload-session";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
const close: (() => void)[] = [];
beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  close.splice(0).forEach((dispose) => dispose());
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.useRealTimers();
});
async function fixture() {
  const tab = await assetTab();
  close.push(() => tab.app.dispose());
  return tab;
}

it.each(["workspace", "new_project", "project"] as const)(
  "%s loses its write lifetime permanently across permission ABA",
  async (kind) => {
    const tab = await fixture();
    const target: ProfileUploadTarget =
      kind === "workspace"
        ? { purpose: "WORKSPACE_LOGO", scope: { kind, id: tab.workspace.id } }
        : { purpose: "PROJECT_COVER", scope: { kind, id: kind === "project" ? tab.project.id : tab.workspace.id } };
    const upload = tab.app.asset.createUpload(target);
    upload.select(imageFile());
    const result = track(upload.start());
    const put = await tab.sent(result);
    if (kind === "project") {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    } else {
      await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
      await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
    }
    expect(upload.current).toBe(false);
    expect(upload.previewURL).toBeUndefined();
    expect(put.signal?.aborted).toBe(true);
    put.answer(new Response(null, { status: 204 }));
    await until(() => result.settled, "old upload after permission recovery");
    expect(result.value).toBeUndefined();
    expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(0);
    const fresh = tab.app.asset.createUpload(target);
    expect(fresh.current).toBe(true);
  }
);

it("user and workspace uploads ignore unrelated projects, and project uploads ignore ordinary issue deletion", async () => {
  const tab = await fixture();
  const peer = projectOf("peer", tab.workspace.id);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, peer]);
  const targets: ProfileUploadTarget[] = [
    avatarTarget,
    { purpose: "WORKSPACE_LOGO", scope: { kind: "workspace", id: tab.workspace.id } },
    { purpose: "PROJECT_COVER", scope: { kind: "project", id: tab.project.id } },
  ];
  const uploads = targets.map((target) => tab.app.asset.createUpload(target));
  const results = uploads.map((upload) => {
    upload.select(imageFile());
    return track(upload.start());
  });
  await until(() => tab.puts.length === 3 || results.every((result) => result.settled), "three independent uploads");
  expect(tab.puts).toHaveLength(3);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  tab.app.issue.invalidateProjectReads(tab.project.id, tab.workspace.id);
  expect(uploads.map((upload) => upload.current)).toEqual([true, true, true]);
  for (const put of tab.puts) put.answer(new Response(null, { status: 204 }));
  await until(() => results.every((result) => result.settled), "independent completions");
  expect(results.map((result) => result.value?.state)).toEqual(["uploaded", "uploaded", "uploaded"]);
});

it("user lifetime survives workspace loss while workspace lifetime follows UUID, never reused slug", async () => {
  const tab = await fixture();
  const user = tab.app.asset.createUpload(avatarTarget);
  const workspace = tab.app.asset.createUpload({
    purpose: "WORKSPACE_LOGO",
    scope: { kind: "workspace", id: tab.workspace.id },
  });
  user.select(imageFile());
  workspace.select(imageFile());
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, id: "new-workspace" }]);
  expect(user.current).toBe(true);
  expect(workspace.current).toBe(false);
  expect(workspace.previewURL).toBeUndefined();
});

it.each([
  { target: "workspace", wsRole: 15, projectRole: 20, archived: false },
  { target: "new_project", wsRole: 5, projectRole: 20, archived: false },
  { target: "project", wsRole: 15, projectRole: 15, archived: false },
  { target: "project", wsRole: 15, projectRole: null, archived: false },
  { target: "project", wsRole: 20, projectRole: 20, archived: true },
] as const)(
  "refuses unauthorized or archived upload: $target/$wsRole/$projectRole/$archived",
  async ({ target: kind, wsRole, projectRole, archived }) => {
    const tab = await fixture();
    await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: wsRole }]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      { ...tab.project, member_role: projectRole, archived_at: archived ? "2026-11-04T00:00:00Z" : null },
    ]);
    const target: ProfileUploadTarget =
      kind === "workspace"
        ? { purpose: "WORKSPACE_LOGO", scope: { kind, id: tab.workspace.id } }
        : { purpose: "PROJECT_COVER", scope: { kind, id: kind === "project" ? tab.project.id : tab.workspace.id } };
    const upload = tab.app.asset.createUpload(target);
    upload.select(imageFile());
    expect(upload.current).toBe(false);
    expect(await upload.start()).toBeUndefined();
    expect(tab.nerve.to("/api/v0/assets")).toHaveLength(0);
  }
);

it("member may create a project cover while only an admin may change the workspace logo", async () => {
  const tab = await fixture();
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 15 }]);
  const upload = tab.app.asset.createUpload({
    purpose: "PROJECT_COVER",
    scope: { kind: "new_project", id: tab.workspace.id },
  });
  upload.select(imageFile());
  const result = track(upload.start());
  (await tab.sent(result)).answer(new Response(null, { status: 204 }));
  await until(() => result.settled, "member new-project cover");
  expect(result.value?.state).toBe("uploaded");
});

it("an upload observing an archived project at its await boundary retires instead of remaining busy", async () => {
  const tab = await fixture();
  const upload = tab.app.asset.createUpload({
    purpose: "PROJECT_COVER",
    scope: { kind: "project", id: tab.project.id },
  });
  upload.select(imageFile());
  const result = track(upload.start());
  const put = await tab.sent(result);
  await loadProject(tab.nerve, tab.app.projectRoot.project, tab.project, {
    ...tab.project,
    archived_at: "2026-11-04T00:00:00Z",
  });
  put.answer(new Response(null, { status: 204 }));
  await until(() => result.settled, "archive observed after transfer");
  expect(result.value).toBeUndefined();
  expect(upload.busy).toBe(false);
  expect(upload.phase).toBe("disposed");
  await loadProject(tab.nerve, tab.app.projectRoot.project, tab.project);
  expect(upload.current).toBe(false);
  expect(upload.previewURL).toBeUndefined();
  expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(0);
});
