// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// The planning page fake's own rule (P7c spec 2.8, R11; re-71 M1): its helpers act on the page planningPageTab opened
// last, and on none once that page's tab has closed, so that a test that forgot to open its page is told so instead of
// acting on a tab an earlier test closed; closing a tab opened before it leaves that page to them.
import { afterEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { answeringPlanningLists, creates, planningPageTab } from "./fake-planning-page";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));

type Tab = Awaited<ReturnType<typeof planningPageTab>>;
const left: Tab[] = [];
/** Opens the store type's page; a tab the test leaves open is closed after it. */
async function open(type: Parameters<typeof planningPageTab>[0]) {
  const tab = await planningPageTab(type);
  bound.app = tab.app;
  left.push(tab);
  return tab;
}
const close = (tab: Tab) => {
  left.splice(left.indexOf(tab), 1);
  return tab.close();
};
afterEach(() => Promise.all(left.splice(0).map((tab) => tab.close())));

it("says no planning page is open once the page's tab has closed", async () => {
  const page = await open(EIssuesStoreType.CYCLE);
  expect(creates(), "the page's creations, while it is open").toEqual([]);
  await close(page);
  expect(() => creates()).toThrow("no planning page is open");
});

it("acts on the page opened last when a page opened before it closes first, and on none once that one closes", async () => {
  const first = await open(EIssuesStoreType.CYCLE);
  const second = await open(EIssuesStoreType.MODULE);
  await close(first);
  const answers = { first: first.nerve.respond, second: second.nerve.respond };
  answeringPlanningLists({ cycles: [], modules: [], archivedModules: [] });
  expect(first.nerve.respond, "the page closed").toBe(answers.first);
  expect(second.nerve.respond, "the page still open").not.toBe(answers.second);
  await close(second);
  expect(() => creates()).toThrow("no planning page is open");
});
