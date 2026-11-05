/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { reaction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { heldChange } from "@/lib/auth/fake-tab";
import { json } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import { loadProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { contentAssetTab, imageFile } from "./fake-assets";

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
  const tab = await contentAssetTab();
  close.push(() => tab.app.dispose());
  return tab;
}

it.each(["issue", "comment", "new_comment", "draft"] as const)(
  "permanently cancels a %s transfer on permission loss even after restoration",
  async (kind) => {
    const tab = await fixture();
    const view = tab.app.asset.openContent(tab.hosts[kind]);
    const upload = tab.app.asset.createContentUpload(view, "description");
    upload.select(imageFile());
    const result = track(upload.start());
    const put = await tab.sent(result);
    if (kind === "draft") {
      await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: 5 }]);
      await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace]);
      await tab.readDrafts();
    } else {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
      await tab.readIssue();
      await tab.readComments();
    }
    expect(view.current).toBe(false);
    expect(view.signal.aborted).toBe(true);
    expect(put.signal?.aborted).toBe(true);
    expect(upload.readyAsset).toBeUndefined();
    put.answer(new Response(null, { status: 204 }));
    await until(() => result.settled, "retired transfer");
    expect(result.value).toBeUndefined();
    expect(result.error).toBeUndefined();
    expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(0);
    expect(tab.app.asset.openContent(tab.hosts[kind]).current).toBe(true);
    expect(await upload.start()).toBeUndefined();
  }
);

it.each(["prepare", "complete"] as const)(
  "ignores a successful %s response already delivered to a retired view",
  async (stage) => {
    const tab = await fixture();
    const gate = heldChange<Response>();
    let held: Response | undefined;
    tab.api.use({
      onResponse({ request, response }) {
        if (
          new URL(request.url).pathname === (stage === "prepare" ? "/api/v0/assets" : "/api/v0/assets/asset-1/complete")
        ) {
          held = response;
          return gate.sent;
        }
        return response;
      },
    });
    const view = tab.app.asset.openContent(tab.hosts.issue);
    const upload = tab.app.asset.createContentUpload(view, "attachment");
    upload.select(imageFile());
    const result = track(upload.start());
    if (stage === "complete") (await tab.sent(result)).answer(new Response(null, { status: 204 }));
    await until(() => held !== undefined || result.settled, "held JSON");
    expect(held).toBeDefined();
    view.dispose();
    gate.answer(held!);
    await until(() => result.settled, "late delivery");
    expect(result.value).toBeUndefined();
    expect(result.error).toBeUndefined();
    expect(upload.phase).toBe("disposed");
    expect(upload.readyAsset).toBeUndefined();
    expect(tab.puts).toHaveLength(stage === "prepare" ? 0 : 1);
  }
);

it.each(["preparing", "uploading", "confirming", "ready", "failed"])(
  "retirement during a synchronous %s notification cannot advance or publish",
  async (phase) => {
    const tab = await fixture();
    const view = tab.app.asset.openContent(tab.hosts.issue);
    const upload = tab.app.asset.createContentUpload(view, "attachment");
    upload.select(imageFile());
    close.push(
      reaction(
        () => upload.phase,
        (value) => {
          if (value === phase) view.dispose();
        }
      )
    );
    const result = track(upload.start());
    if (["confirming", "ready", "failed"].includes(phase)) {
      const put = await tab.sent(result);
      if (phase === "failed") put.fail();
      else put.answer(new Response(null, { status: 204 }));
    }
    await until(() => result.settled, "reentrant retirement");
    expect(result.error).toBeUndefined();
    expect(result.value).toBeUndefined();
    expect(upload.phase).toBe("disposed");
    if (phase !== "ready") expect(tab.nerve.calls.filter((call) => call.path.endsWith("/complete"))).toHaveLength(0);
  }
);

it("disposing one upload leaves another upload in the same view alive", async () => {
  const tab = await fixture();
  const view = tab.app.asset.openContent(tab.hosts.issue);
  const first = tab.app.asset.createContentUpload(view, "attachment");
  const second = tab.app.asset.createContentUpload(view, "description");
  first.select(imageFile());
  second.select(imageFile("second.png"));
  const old = track(first.start());
  const fresh = track(second.start());
  await until(() => tab.puts.length === 2 || (old.settled && fresh.settled), "two transfers");
  expect(tab.puts).toHaveLength(2);
  first.dispose();
  expect(view.current).toBe(true);
  expect(tab.puts[0]?.signal?.aborted).toBe(true);
  expect(tab.puts[1]?.signal?.aborted).toBe(false);
  tab.puts[0]!.answer(new Response(null, { status: 204 }));
  tab.puts[1]!.answer(new Response(null, { status: 204 }));
  await until(() => old.settled && fresh.settled, "independent result");
  expect(old.value).toBeUndefined();
  expect(fresh.value?.name).toBe("second.png");
});

it("a late old refusal cannot replace a new view's confirmed result", async () => {
  const tab = await fixture();
  const gate = heldChange<Response>();
  let held = false;
  tab.api.use({
    onResponse({ request, response }) {
      if (!held && new URL(request.url).pathname === "/api/v0/assets") {
        held = true;
        return gate.sent;
      }
      return response;
    },
  });
  const view = tab.app.asset.openContent(tab.hosts.issue);
  const oldUpload = tab.app.asset.createContentUpload(view, "attachment");
  oldUpload.select(imageFile());
  const old = track(oldUpload.start());
  await until(() => held || old.settled, "first prepare response");
  expect(held).toBe(true);
  view.dispose();
  const freshUpload = tab.app.asset.createContentUpload(tab.app.asset.openContent(tab.hosts.issue), "attachment");
  freshUpload.select(imageFile("fresh.png"));
  const fresh = track(freshUpload.start());
  (await tab.sent(fresh)).answer(new Response(null, { status: 204 }));
  await until(() => fresh.settled, "fresh upload");
  gate.answer(json(503, { status: 503, code: "server_busy", title: "Busy" }));
  await until(() => old.settled, "late refusal");
  expect(old.error).toBeUndefined();
  expect(freshUpload.phase).toBe("ready");
  expect(freshUpload.readyAsset?.name).toBe("fresh.png");
});
