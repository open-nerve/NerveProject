// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act, StrictMode, useLayoutEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { useDetailScope } from "./use-detail-scope";

const state = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => state.tab!.app.issue.issueDetail }));
let renderer: Root;
let container: HTMLElement;
const capture: Record<string, () => () => boolean> = {};
function Host({ id }: { id: string }) {
  capture[id] = useDetailScope("ws", "A", id);
  return <span>{id}</span>;
}
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  state.tab = await issueStoreTab();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  state.tab!.app.dispose();
  vi.useRealTimers();
});
it("invalidates an old host's followup when the same session changes issue", async () => {
  await act(async () => renderer.render(<Host id="one" />));
  const old = capture.one!();
  expect(old()).toBe(true);
  await act(async () => renderer.render(<Host id="two" />));
  expect(old()).toBe(false);
  expect(capture.two!()()).toBe(true);
});
it("one host unmount does not retire another host's followup", async () => {
  await act(async () =>
    renderer.render(
      <>
        <Host key="one" id="one" />
        <Host key="two" id="two" />
      </>
    )
  );
  const one = capture.one!();
  const two = capture.two!();
  await act(async () => renderer.render(<Host key="two" id="two" />));
  expect(one()).toBe(false);
  expect(two()).toBe(true);
});
it("permission loss and recovery invalidates outstanding followups", async () => {
  await act(async () => renderer.render(<Host id="one" />));
  const old = capture.one!();
  const tab = state.tab!;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  expect(old()).toBe(false);
  expect(capture.one!()()).toBe(true);
});

it("StrictMode reacquisition does not revive a captured followup from its first mount", async () => {
  const snapshots: (() => boolean)[] = [];
  function EffectHost() {
    const take = useDetailScope("ws", "A", "one");
    useLayoutEffect(() => {
      snapshots.push(take());
    }, [take]);
    return null;
  }
  await act(async () =>
    renderer.render(
      <StrictMode>
        <EffectHost />
      </StrictMode>
    )
  );
  expect(snapshots).toHaveLength(2);
  expect(snapshots[0]!()).toBe(false);
  expect(snapshots[1]!()).toBe(true);
});
