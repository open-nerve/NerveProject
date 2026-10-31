// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueLink } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueDetailOf, issueLinkOf } from "@/store/issue/fake-issues";
import { issueSelectionTab, selectionButton } from "@/components/core/modals/fake-issue-selection";
import { IssueDetailHost } from "../../detail-host";
import { IssueLinksCollapsibleContent } from "./content";

vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "links" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
const pendingLink = issueLinkOf("preview");
const path = "/api/v0/issues/x/links";
beforeEach(async () => {
  tab = await issueSelectionTab();
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
});
afterEach(async () => tab.close());
const store = () => tab.app.issue.issueDetail.link;
const reads = () => tab.nerve.to(path).filter((call) => call.method === "GET");
async function loaded(links: IssueLink[] = [pendingLink]) {
  tab.nerve.respond = (call) => {
    if (call.path === path) call.answer(json(200, { data: links }));
  };
  await store().fetchLinks("ws", "A", "x");
  tab.nerve.respond = undefined;
  tab.nerve.calls.length = 0;
  await render();
}
const render = (active = true) =>
  tab.render(
    <IssueDetailHost workspaceSlug="ws" projectId="A" issueId="x">
      <IssueLinksCollapsibleContent workspaceSlug="ws" projectId="A" issueId="x" disabled={false} active={active} />
    </IssueDetailHost>
  );
const tick = (ms = 1000) => act(async () => vi.advanceTimersByTimeAsync(ms));
it("keeps the address while waiting, never overlaps its GET, and stops on terminal metadata", async () => {
  await loaded();
  expect(document.querySelector("a")?.getAttribute("href")).toBe(pendingLink.url);
  await tick();
  expect(reads()).toHaveLength(1);
  await tick(9000);
  expect(reads()).toHaveLength(1);
  await act(async () => reads()[0]!.answer(json(200, { data: [{ ...pendingLink, metadata: { title: "Arrived" } }] })));
  expect(document.body.textContent).toContain("Arrived");
  await tick(31000);
  expect(reads()).toHaveLength(1);
});
it.each([{ title: null }, { url: "" }, { error: "blocked" as const }, { error: "failed" as const }])(
  "does not poll an already terminal result %j",
  async (metadata) => {
    await loaded([{ ...pendingLink, metadata }]);
    await tick(31000);
    expect(reads()).toHaveLength(0);
    expect(document.querySelector("a")?.getAttribute("href")).toBe(pendingLink.url);
  }
);
it("does not poll an empty collection or a collapsed widget", async () => {
  await loaded([]);
  await tick(31000);
  expect(reads()).toHaveLength(0);
  await loaded();
  await render(false);
  await tick(31000);
  expect(reads()).toHaveLength(0);
});
it("ends the window within thirty seconds despite repeated empty responses; explicit Retry performs only GET and starts a new window", async () => {
  await loaded();
  tab.nerve.respond = (call) => call.answer(json(200, { data: [pendingLink] }));
  await tick(31000);
  expect(reads().length).toBeGreaterThan(0);
  expect(reads().length).toBeLessThanOrEqual(30);
  const count = reads().length;
  expect(document.body.textContent).toContain("links.metadata.expired");
  await tick(31000);
  expect(reads()).toHaveLength(count);
  await act(async () => selectionButton("common.retry").click());
  await tick();
  expect(reads().length).toBeGreaterThan(count);
  expect(tab.nerve.calls.every((call) => call.method === "GET")).toBe(true);
  expect(document.body.textContent).not.toContain("links.metadata.expired");
  const times = reads()
    .slice(0, count)
    .map((call) => call.at);
  expect(times.slice(1).every((at, i) => at - times[i]! >= 1000)).toBe(true);
});
it("stops automatic reads after a refusal, preserves the link and retries the read explicitly", async () => {
  await loaded();
  await tick();
  expect(reads()).toHaveLength(1);
  await act(async () => reads()[0]!.answer(problem(503, "unavailable")));
  expect(document.querySelector('[role="alert"]')).not.toBeNull();
  await tick(10000);
  expect(reads()).toHaveLength(1);
  expect(document.querySelector("a")?.getAttribute("href")).toBe(pendingLink.url);
  await act(async () => selectionButton("common.retry").click());
  expect(reads()).toHaveLength(2);
  await act(async () => reads()[1]!.answer(json(200, { data: [{ ...pendingLink, metadata: { error: "failed" } }] })));
  expect(document.querySelector('[role="alert"]')).toBeNull();
  await tick(31000);
  expect(reads()).toHaveLength(2);
});
it.each(["unmount", "permission"])("retires polling across %s, including the late in-flight result", async (change) => {
  await loaded();
  await tick();
  expect(reads()).toHaveLength(1);
  if (change === "unmount") await tab.render(null);
  else
    await act(async () => {
      tab.app.projectRoot.project.confirmProject(tab.project, (p) => ({ ...p, member_role: null, network: 0 }));
      tab.app.projectRoot.project.confirmProject(tab.project, () => tab.project);
    });
  await act(async () => reads()[0]!.answer(json(200, { data: [{ ...pendingLink, metadata: { title: "Retired" } }] })));
  await tick(31000);
  expect(reads()).toHaveLength(1);
  expect(document.body.textContent).not.toContain("Retired");
});

it.each(["replace", "delete"])(
  "retires the previous link identity on %s and an older GET cannot undo the confirmed command",
  async (change) => {
    await loaded();
    await tick();
    expect(reads()).toHaveLength(1);
    const url = "https://next.test/";
    tab.nerve.respond = (call) => {
      if (call.path === `/api/v0/issue-links/${pendingLink.id}`)
        call.answer(change === "delete" ? noContent() : json(200, { ...pendingLink, url }));
      else if (call.path === "/api/v0/issues/x") call.answer(json(200, issueDetailOf("x")));
    };
    await act(async () => {
      if (change === "delete") await store().removeLink("ws", "A", "x", pendingLink.id);
      else await store().updateLink("ws", "A", "x", pendingLink.id, { url });
    });
    await tick(5000);
    expect(reads()).toHaveLength(1);
    await act(async () =>
      reads()[0]!.answer(json(200, { data: [{ ...pendingLink, metadata: { title: "Retired title" } }] }))
    );
    expect(document.body.textContent).not.toContain("Retired title");
    if (change === "delete") {
      await tick(31000);
      expect(document.querySelector("a")).toBeNull();
      expect(reads()).toHaveLength(1);
    } else {
      expect(document.querySelector("a")?.getAttribute("href")).toBe(url);
      await tick();
      await until(() => reads().length === 2, "new URL metadata read");
      await act(async () =>
        reads()[1]!.answer(json(200, { data: [{ ...pendingLink, url, metadata: { title: "Current title" } }] }))
      );
      expect(document.body.textContent).toContain("Current title");
      await tick(31000);
      expect(reads()).toHaveLength(2);
    }
  }
);
