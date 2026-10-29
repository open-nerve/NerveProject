// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { runInAction } from "mobx";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, noContent, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { accountOf, userOf } from "@/store/member/fake-users";
import { issueReactionOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { IssueReaction } from "./issue";
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-member", () => ({
  useMember: () => ({ getUserDetails: (id: string) => userOf(id === "u-actor" ? "Actor" : "Other") }),
}));
vi.mock("@nerve/propel/emoji-reaction", () => ({
  EmojiReactionPicker: ({
    label,
    onChange,
    disabled,
  }: {
    label: React.ReactNode;
    onChange: (value: string) => void;
    disabled?: boolean;
  }) => (
    <>
      {label}
      <button data-add disabled={disabled} onClick={() => onChange("128077")}>
        add
      </button>
    </>
  ),
  EmojiReactionGroup: ({
    reactions,
    onReactionClick,
  }: {
    reactions: { emoji: string; count: number; users: string[] }[];
    onReactionClick: (emoji: string) => void;
  }) => (
    <>
      {reactions.map((row) => (
        <button key={row.emoji} data-reaction onClick={() => onReactionClick(row.emoji)}>
          {row.count} {row.users.join(",")}
        </button>
      ))}
    </>
  ),
}));
let renderer: Root;
let container: HTMLElement;
const mine = issueReactionOf("mine");
const theirs = issueReactionOf("theirs", "x", "u-other");
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  toasts.length = 0;
  binding.tab = await issueStoreTab();
  runInAction(() => {
    binding.tab!.app.user.data = accountOf("actor");
  });
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
const render = (id = "x") =>
  act(async () =>
    renderer.render(
      <IssueReaction workspaceSlug="ws" projectId="A" issueId={id} currentUser={binding.tab!.app.user.data!} />
    )
  );
async function loaded() {
  const tab = binding.tab!;
  const sent = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.issueDetail.reaction.fetchReactions("ws", "A", "x"),
    ["GET", "/api/v0/issues/x/reactions"]
  );
  sent.call.answer(json(200, { data: [mine, theirs] }));
  await sent.promise;
}
it("uses native actors and keeps a refused reaction until the retry is confirmed", async () => {
  const tab = binding.tab!;
  await loaded();
  await render();
  expect(container.querySelector("[data-reaction]")?.textContent).toBe("2 Actor,Other");
  await act(async () => container.querySelector<HTMLButtonElement>("[data-reaction]")!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/reactions/128077").length === 1, "remove reaction"));
  expect(container.querySelector("[data-reaction]")?.textContent).toBe("2 Actor,Other");
  await act(async () => tab.nerve.to("/api/v0/issues/x/reactions/128077")[0]!.answer(problem(403, "forbidden")));
  expect(container.querySelector("[data-reaction]")?.textContent).toBe("2 Actor,Other");
  expect(toasts.at(-1)?.message).toBe("errors.forbidden");
  await act(async () => container.querySelector<HTMLButtonElement>("[data-reaction]")!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/reactions/128077").length === 2, "retry removal"));
  await act(async () => tab.nerve.to("/api/v0/issues/x/reactions/128077")[1]!.answer(noContent()));
  expect(container.querySelector("[data-reaction]")?.textContent).toBe("1 Other");
});
it("a reaction read failure has a read-only retry", async () => {
  const tab = binding.tab!;
  const sent = await requestedIssue(
    tab.nerve,
    () => tab.app.issue.issueDetail.reaction.fetchReactions("ws", "A", "x"),
    ["GET", "/api/v0/issues/x/reactions"]
  );
  sent.call.answer(problem(500, "unavailable"));
  await expect(sent.promise).rejects.toMatchObject({ status: 500 });
  await render();
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  await act(async () => container.querySelector<HTMLButtonElement>('[role="alert"] button')!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/reactions").length === 2, "read retry"));
  await act(async () => tab.nerve.to("/api/v0/issues/x/reactions")[1]!.answer(json(200, { data: [] })));
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(tab.nerve.calls.map((call) => call.method)).toEqual(["GET", "GET"]);
});
it("a retired host never displays success or failure in its replacement", async () => {
  const tab = binding.tab!;
  await render();
  await act(async () => container.querySelector<HTMLButtonElement>("[data-add]")!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/reactions").length === 1, "old add"));
  await render("other");
  await act(async () => tab.nerve.to("/api/v0/issues/x/reactions")[0]!.answer(problem(403, "forbidden")));
  expect(toasts).toHaveLength(0);
  expect(container.querySelector("[data-reaction]")).toBeNull();
});

it("a refused old command releases its local busy state after permission recovery", async () => {
  const tab = binding.tab!;
  await render();
  await act(async () => container.querySelector<HTMLButtonElement>("[data-add]")!.click());
  await act(async () => until(() => tab.nerve.to("/api/v0/issues/x/reactions").length === 1, "old add"));
  const { loadProjects } = await import("@/store/project/fake-projects");
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
  });
  await act(async () => tab.nerve.to("/api/v0/issues/x/reactions")[0]!.answer(problem(403, "forbidden")));
  expect(toasts).toHaveLength(0);
  expect(container.querySelector<HTMLButtonElement>("[data-add]")!.disabled).toBe(false);
});
