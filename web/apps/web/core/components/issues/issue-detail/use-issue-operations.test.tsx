// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.2, R3: the detail's operations send a change of the work item's modules as additions and removals,
// applied in the work item's turn to the set it then holds, as the write unit's command does.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { IssueDetail } from "@nerve/api-client";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { track, until } from "@/lib/auth/fake-time";
import type { TIssueOperations } from "./root";
import { useIssueOperations } from "./use-issue-operations";
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "operations" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));

const MA = "m-alpha";
const MB = "m-beta";
/** An archived module the work item is in. */
const MG = "m-gamma";
const answerOf = (module_ids: string[], updated_at: string): IssueDetail => ({
  ...issueDetailOf("x"),
  module_ids,
  updated_at,
});
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let operations: TIssueOperations | undefined;
function Host() {
  operations = useIssueOperations("ws", "A", "x");
  return null;
}
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(answerOf([MA, MG], "2026-10-29T00:00:00Z"), tab.app.issue.captureRead());
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  operations = undefined;
  vi.useRealTimers();
});

it("adds and removes modules in the work item's turn, from the set it then holds", async () => {
  const { nerve } = tab;
  await act(async () => renderer.render(<Host />));
  const first = track(operations!.update("ws", "A", "x", { modules: { add: [MB] } }));
  await act(async () => until(() => nerve.calls.length === 1, "the first change"));
  expect(nerve.calls[0]).toMatchObject({ method: "PATCH", path: "/api/v0/issues/x" });
  expect(nerve.calls[0]?.body).toEqual({ module_ids: [MA, MG, MB] });
  const second = track(operations!.update("ws", "A", "x", { modules: { remove: [MA] } }));
  await act(async () => vi.advanceTimersByTimeAsync(1_000));
  expect(nerve.calls, "the second change waits for the first").toHaveLength(1);
  await act(async () => nerve.calls[0]?.answer(json(200, answerOf([MA, MG, MB], "2026-10-30T00:00:00Z"))));
  await act(async () => until(() => nerve.calls.length === 2, "the second change"));
  expect(nerve.calls[1]?.body, "worked out from the first one's answer").toEqual({ module_ids: [MG, MB] });
  await act(async () => nerve.calls[1]?.answer(json(200, answerOf([MG, MB], "2026-10-31T00:00:00Z"))));
  await act(async () => until(() => first.settled && second.settled, "both changes"));
  expect([first.error, second.error]).toEqual([undefined, undefined]);
  expect(tab.app.issue.issues.getIssueById("x")?.module_ids).toEqual([MG, MB]);
});
