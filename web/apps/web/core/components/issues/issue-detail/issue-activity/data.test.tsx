// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import type { IssueActivity } from "@nerve/api-client";
import { E_SORT_ORDER, defaultActivityFilters } from "@nerve/constants";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueActivityOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { userOf } from "@/store/member/fake-users";
import { IssueActivityItem } from "./activity/activity-list";
import { IssueActivityCommentRoot } from "./activity-comment-root";
import { useWorkItemCommentOperations } from "./helper";
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-label", () => ({ useLabel: () => ({ getLabelById: () => undefined }) }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => (id === "u-actor" ? userOf("Actor") : undefined) }),
}));
vi.mock("@/hooks/store/user", () => ({ useUser: () => binding.tab!.app.user }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/components/comments/card/root", () => ({ CommentCard: () => <div data-comment /> }));
let renderer: Root;
let container: HTMLElement;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  binding.tab = await issueStoreTab();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  binding.tab!.app.dispose();
  vi.useRealTimers();
});
async function loaded(row: IssueActivity) {
  const sent = await requestedIssue(
    binding.tab!.nerve,
    () => binding.tab!.app.issue.issueDetail.activity.fetchActivities("ws", "A", "x"),
    ["GET", "/api/v0/issues/x/activities"]
  );
  sent.call.answer(json(200, { data: [row], next_cursor: null }));
  await sent.promise;
}
const render = (item: React.ReactNode) => act(async () => renderer.render(<MemoryRouter>{item}</MemoryRouter>));
it.each(["u-actor", "ended-user", null])("native actor %s uses the directory or the system fallback", async (actor) => {
  await loaded({ ...issueActivityOf("a"), actor_id: actor });
  await render(<IssueActivityItem workspaceSlug="ws" issueId="x" activityId="a" ends={undefined} />);
  expect(container.textContent).toContain(
    actor === null ? "Nerve" : actor === "u-actor" ? "Actor" : "common.deactivated_user"
  );
  if (actor === "u-actor") expect(container.querySelector("a")?.getAttribute("href")).toBe("/ws/profile/u-actor");
  else expect(container.querySelector("a")).toBeNull();
});
it("native creation with nullable fields renders without invented source details", async () => {
  await loaded({ ...issueActivityOf("a"), field: null, verb: "created", old_value: null, new_value: null });
  await render(<IssueActivityItem workspaceSlug="ws" issueId="x" activityId="a" ends={undefined} />);
  expect(container.textContent).toContain("Actor");
  expect(container.textContent).toContain("created the work item");
  expect(container.textContent).not.toContain("undefined");
});
function Activity() {
  const operations = useWorkItemCommentOperations("ws", "A", "x");
  return (
    <IssueActivityCommentRoot
      workspaceSlug="ws"
      projectId="A"
      issueId="x"
      isIntakeIssue={false}
      selectedFilters={defaultActivityFilters}
      sortOrder={E_SORT_ORDER.ASC}
      activityOperations={operations}
    />
  );
}
it("a refused activity read remains visible and its retry sends only GET", async () => {
  const tab = binding.tab!;
  const sent = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.issueDetail.activity.fetchActivities("ws", "A", "x"),
    ["GET", "/api/v0/issues/x/activities"]
  );
  sent.call.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  await render(<Activity />);
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  await act(async () => container.querySelector<HTMLButtonElement>('[role="alert"] button')!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/activities").length === 2, "retry activity GET"));
  await act(async () =>
    tab.nerve.to("/api/v0/issues/x/activities")[1]!.answer(json(200, { data: [], next_cursor: null }))
  );
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(tab.nerve.calls.map((call) => call.method)).toEqual(["GET", "GET"]);
});

it.each(["link", "assignees", "cycles", "modules"])(
  "nullable %s identifiers and values never create a fabricated navigation target",
  async (field) => {
    await loaded({
      ...issueActivityOf("a"),
      field,
      verb: "created",
      new_value: null,
      old_value: null,
      new_identifier: null,
      old_identifier: null,
    });
    await render(<IssueActivityItem workspaceSlug="ws" issueId="x" activityId="a" ends={undefined} />);
    const targets = [...container.querySelectorAll("a[href]")].map((link) => link.getAttribute("href"));
    expect(targets).toEqual(["/ws/profile/u-actor"]);
  }
);
