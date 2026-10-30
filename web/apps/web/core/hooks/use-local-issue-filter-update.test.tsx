// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot } from "react-dom/client";
import { StoreContext } from "@/lib/store-context";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { RootStore } from "@/store/root.store";
import { issuePageOf, requestedIssue, issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { heldChange } from "@/lib/auth/fake-tab";
import { json } from "@/lib/auth/fake-nerve";
import { useLocalIssueFilterUpdate } from "./use-local-issue-filter-update";
const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined, failures: [] as unknown[] }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "local-filter" } }, publicClient: {} }));
vi.mock("@/hooks/use-refusal-toast", () => ({ useRefusalToast: () => (e: unknown) => bound.failures.push(e) }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let renderer: ReturnType<typeof createRoot>;
const render = (node: React.ReactNode) =>
  act(async () => renderer.render(<StoreContext.Provider value={tab.app}>{node}</StoreContext.Provider>));
let update: ReturnType<typeof useLocalIssueFilterUpdate>;
function Probe({ target = "A" }: { target?: string }) {
  update = useLocalIssueFilterUpdate("ws", "A", target);
  return null;
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  bound.app = tab.app;
  bound.failures.length = 0;
  renderer = createRoot(document.createElement("div"));
  await render(<Probe />);
  const first = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.archivedIssues.fetchIssues("ws", "A", "init-loader", { limit: 10 }),
    ["GET", "/api/v0/projects/A/issues"]
  );
  first.call.answer(json(200, issuePageOf([])));
  await first.promise;
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  tab.app.dispose();
  vi.useRealTimers();
});
it.each([false, true])("consumes a current non-read refusal once (synchronous=%s)", async (synchronous) => {
  const error = new Error("local change failed");
  await expect(
    update(() => {
      if (synchronous) throw error;
      return Promise.reject(error);
    })
  ).resolves.toBeUndefined();
  expect(bound.failures).toEqual([error]);
});
it.each(["target ABA", "permission ABA", "unmount", "retire"] as const)(
  "does not report an old refusal after %s",
  async (ending) => {
    const held = heldChange<void>();
    const pending = update(() => held.sent);
    if (ending === "target ABA") {
      await render(<Probe target="B" />);
      await render(<Probe />);
    } else if (ending === "permission ABA") {
      await act(async () => {
        await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
        await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
      });
    } else if (ending === "retire") tab.app.dispose();
    else await render(null);
    held.refuse(new Error("old local refusal"));
    await pending;
    expect(bound.failures).toEqual([]);
  }
);
it("does not invoke an update captured by an ended host", async () => {
  const old = update;
  await render(null);
  const send = vi.fn(async () => undefined);
  await old(send);
  expect(send).not.toHaveBeenCalled();
});
