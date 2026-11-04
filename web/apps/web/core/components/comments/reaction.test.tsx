// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { StoreContext } from "@/lib/store-context";
import type { RootStore } from "@/store/root.store";
import { seedAccount } from "@/store/user/fake-account";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter } from "react-router";
import { observer } from "mobx-react";
import type { CommentReaction } from "@nerve/api-client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { accountOf, userOf } from "@/store/member/fake-users";
import { issueCommentOf, issueDetailOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { useWorkItemCommentOperations } from "../issues/issue-detail/issue-activity/helper";
import { CommentCard } from "./card/root";
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({
  tokenManager: { state: { status: "signed-in", loginId: "one" }, start: async () => {}, subscribe: () => () => {} },
  publicClient: {},
}));
vi.mock("@/lib/store-context", async () => ({
  rootStore: undefined,
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => ({
  useTranslation: () => ({ t: (key: string, args?: { name: string }) => (args ? `${key}: ${args.name}` : key) }),
}));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => binding.tab!.app.workspaceRoot }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ workspace: binding.tab!.app.memberRoot.workspace, getUserDetails: () => userOf("Ann") }),
}));
vi.mock("@/hooks/store/user", () => ({
  useUser: () => binding.tab!.app.user,
  useUserPermissions: () => binding.tab!.app.user.permission,
}));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
// Keep real reaction buttons, comment operations and stores; rich text is unrelated to reactions.
vi.mock("@/components/editor/lite-text", () => ({ LiteTextEditor: () => <div>Comment body</div> }));
let root: Root, host: HTMLDivElement;
const comment = issueCommentOf("comment");
const code = "128105-127997-8205-128187";
const emoji = "👩🏽‍💻";
const reaction = (id: string, actorId: string): CommentReaction => ({
  id,
  comment_id: comment.id,
  actor_id: actorId,
  reaction: code,
  created_at: comment.created_at,
});
const mine = reaction("mine", "u-actor"),
  theirs = reaction("theirs", "u-other");
const route = "/api/v0/issue-comments/comment/reactions";
const Card = observer(function Card({ disabled = false }: { disabled?: boolean }) {
  const operations = useWorkItemCommentOperations("ws", "A", "x");
  const row = binding.tab!.app.issue.issueDetail.comment.getCommentById("x", comment.id)!;
  return (
    <CommentCard
      workspaceSlug="ws"
      projectId="A"
      comment={row}
      activityOperations={operations}
      ends={undefined}
      showCopyLinkOption
      disabled={disabled}
    />
  );
});
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  toasts.length = 0;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
  binding.tab = await issueStoreTab();
  vi.spyOn(binding.tab.app.issue.issueDetail.comment, "refreshAfterChange").mockResolvedValue();
  await seedAccount(binding.tab!.nerve, binding.tab!.app.user, accountOf("actor"));
  binding.tab.app.issue.issues.acceptRead(issueDetailOf("x"), binding.tab.app.issue.captureRead());
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  binding.tab!.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
async function loaded(reactions = [mine, theirs], disabled = false) {
  const tab = binding.tab!;
  const read = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.comment.fetchComments("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x/comments",
  ]);
  read.call.answer(json(200, { data: [{ ...comment, reactions }], next_cursor: null }));
  await read.promise;
  tab.nerve.calls.length = 0;
  await act(async () =>
    root.render(
      <StoreContext.Provider value={binding.tab!.app}>
        <MemoryRouter>
          <Card disabled={disabled} />
        </MemoryRouter>
      </StoreContext.Provider>
    )
  );
}
const button = () =>
  [...host.querySelectorAll<HTMLButtonElement>("button")].find(
    (b) => !b.querySelector("button") && b.textContent?.startsWith(emoji)
  )!;
const rows = () => binding.tab!.app.issue.issueDetail.comment.getCommentById("x", comment.id)!.reactions;
it("removesOnlyMineAfterConfirmation and locks both named triggers during one removal", async () => {
  await loaded();
  expect(host.querySelector("button button")).toBeNull();
  const triggers = [...host.querySelectorAll<HTMLButtonElement>("button")].filter((b) =>
    b.textContent?.includes("aria_labels.comment_add_reaction: Ann")
  );
  expect(triggers).toHaveLength(2);
  await act(async () => {
    button().click();
    button().click();
  });
  await act(async () => until(() => binding.tab!.nerve.to(route + "/" + code).length === 1, "one reaction removal"));
  expect(button().matches(":disabled")).toBe(true);
  expect(triggers.every((trigger) => trigger.matches(":disabled"))).toBe(true);
  expect(rows()).toEqual([mine, theirs]);
  await act(async () => binding.tab!.nerve.to(route + "/" + code)[0]!.answer(noContent()));
  expect(rows()).toEqual([theirs]);
  expect(binding.tab!.nerve.calls.map((call) => call.method)).toEqual(["DELETE"]);
});
it.each([403, 409])("keepsCountsOnConflictOrRefusal (%s) and sends combinedEmojiIsOneReaction", async (status) => {
  await loaded([theirs]);
  await act(async () => {
    button().click();
    button().click();
  });
  await act(async () => until(() => binding.tab!.nerve.to(route).length === 1, "one reaction addition"));
  const sent = binding.tab!.nerve.to(route)[0]!;
  expect(sent.method).toBe("POST");
  expect(sent.body).toEqual({ reaction: code });
  expect(rows()).toEqual([theirs]);
  await act(async () => sent.answer(problem(status, status === 409 ? "issue.reaction_exists" : "forbidden")));
  expect(rows()).toEqual([theirs]);
  expect(toasts).toHaveLength(1);
  expect(button().matches(":disabled")).toBe(false);
  await act(async () => button().click());
  await act(async () => until(() => binding.tab!.nerve.to(route).length === 2, "retry reaction addition"));
  await act(async () => binding.tab!.nerve.to(route)[1]!.answer(json(201, mine)));
  expect(rows()).toEqual([theirs, mine]);
  expect(binding.tab!.nerve.calls.map((call) => call.method)).toEqual(["POST", "POST"]);
});
it("archivedKeepsReadOnlyReactions with no write or add picker", async () => {
  await loaded([mine, theirs], true);
  expect(button()).toBeDefined();
  expect(button().matches(":disabled")).toBe(true);
  await act(async () => button().click());
  expect(rows()).toEqual([mine, theirs]);
  expect(host.querySelector("button[aria-haspopup]")).toBeNull();
  expect(binding.tab!.nerve.calls).toHaveLength(0);
});
