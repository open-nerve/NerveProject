/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { runInAction } from "mobx";
import { json, problem } from "@/lib/auth/fake-nerve";
import { heldChange } from "@/lib/auth/fake-tab";
import { track, until } from "@/lib/auth/fake-time";
import { workspaceOf, loadWorkspaces } from "@/store/workspace/fake-workspaces";
import { projectOf, loadProjects } from "@/store/project/fake-projects";
import { accountOf } from "@/store/member/fake-users";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { issueCommentOf, issueDetailOf, issueLinkOf, issueReactionOf, issueStoreTab } from "../fake-issues";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: {}, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
const detail = () => tab.app.issue.issueDetail;
const commands = {
  description: () => tab.app.issue.updateIssue("x", () => ({ description_html: "<p>changed</p>" })),
  parent: () => tab.app.issue.updateIssue("child", () => ({ parent_id: null })),
  restore: () => tab.app.issue.unarchiveIssue("x", tab.app.issue.scope("ws", "A")),
  subAdd: () => detail().createSubIssues("ws", "A", "x", ["child"]),
  subRemove: () => detail().removeSubIssue("ws", "A", "x", "child"),
  subDelete: () => detail().deleteSubIssue("ws", "A", "x", "child"),
  relationAdd: () => detail().createRelation("ws", "A", "x", "blocking", ["other"]),
  relationRemove: () => detail().removeRelation("ws", "A", "x", "blocking", "other"),
  linkAdd: () => detail().createLink("ws", "A", "x", { url: "https://new.example.test" }),
  linkEdit: () => detail().updateLink("ws", "A", "x", "link", { title: "changed" }),
  linkRemove: () => detail().removeLink("ws", "A", "x", "link"),
  commentAdd: () => detail().createComment("ws", "A", "x", { comment_html: "<p>changed</p>" }),
  commentEdit: () => detail().updateComment("ws", "A", "x", "comment", { comment_html: "<p>changed</p>" }),
  commentRemove: () => detail().removeComment("ws", "A", "x", "comment"),
  reactionAdd: () => detail().createReaction("ws", "A", "x", "128077"),
  reactionRemove: () => detail().removeReaction("ws", "A", "x", "128077"),
  commentReactionAdd: () => detail().createCommentReaction("ws", "A", "x", "comment", "128077"),
  commentReactionRemove: () => detail().removeCommentReaction("ws", "A", "x", "comment", "128077"),
};
const names = Object.keys(commands) as (keyof typeof commands)[];
const child = { ...issueDetailOf("child"), parent_id: "x" };
const parent = { ...issueDetailOf("x"), sub_issues_count: 1, link_count: 1 };
function facts() {
  return JSON.stringify({
    parent: tab.app.issue.issues.getIssueById("x"),
    child: tab.app.issue.issues.getIssueById("child"),
    links: detail().link.getLinkById("x", "link"),
    comments: detail().comment.getCommentById("x", "comment"),
    reactions: detail().reaction.getReactionsByIssueId("x"),
    relations: detail().relation.getRelationsByIssueId("x"),
    children: detail().subIssues.subIssuesByIssueId("x"),
  });
}
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await issueStoreTab();
  runInAction(() => {
    tab.app.user.data = accountOf("actor");
  });
  for (const issue of [parent, child, issueDetailOf("other")])
    tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method !== "GET") return;
    const data = call.path.endsWith("/links")
      ? [issueLinkOf("link")]
      : call.path.endsWith("/comments")
        ? [issueCommentOf("comment")]
        : call.path.endsWith("/reactions")
          ? [issueReactionOf("reaction")]
          : call.path.endsWith("/sub-issues")
            ? [child]
            : call.path.endsWith("/relations")
              ? [
                  {
                    id: "relation",
                    relation_type: "blocking",
                    issue: issueDetailOf("other"),
                    created_at: parent.created_at,
                  },
                ]
              : [];
    call.answer(json(200, call.path === "/api/v0/issues/x" ? parent : { data, next_cursor: null }));
  };
  await detail().link.fetchLinks("ws", "A", "x");
  await detail().comment.fetchComments("ws", "A", "x");
  await detail().reaction.fetchReactions("ws", "A", "x");
  await detail().relation.fetchRelations("ws", "A", "x");
  await detail().subIssues.fetchSubIssues("ws", "A", "x");
  tab.nerve.calls.length = 0;
});
afterEach(() => {
  tab.app.dispose();
  vi.useRealTimers();
});
it.each(names)("%s changes no held facts before confirmation or after a native refusal", async (name) => {
  const before = facts();
  const sent = commands[name]();
  const result = track<unknown>(sent);
  await until(() => tab.nerve.calls.length === 1 || result.settled, "mutation request");
  expect(result.settled).toBe(false);
  expect(facts()).toBe(before);
  expect(tab.nerve.calls[0]!.method).not.toBe("GET");
  tab.nerve.calls[0]!.answer(problem(403, "forbidden"));
  await expect(sent).rejects.toMatchObject({ status: 403 });
  expect(facts()).toBe(before);
  expect(tab.nerve.calls).toHaveLength(1);
});
for (const boundary of ["retire", "permission ABA"] as const) {
  it.each(names)(`queued %s sends nothing after ${boundary}`, async (name) => {
    const held = heldChange<void>();
    const first = tab.app.issue.enqueueIssueChanges(["x", "child"], () => held.sent);
    const waiting = commands[name]();
    const result = track<unknown>(waiting);
    await vi.advanceTimersByTimeAsync(1);
    expect(tab.nerve.calls).toHaveLength(0);
    if (boundary === "retire") tab.app.issue.retire();
    else {
      const readHandler = tab.nerve.respond;
      tab.nerve.respond = undefined;
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
      tab.nerve.respond = readHandler;
      expect(tab.app.user.permission.getProjectRoleByWorkspaceSlugAndProjectId("ws", "A")).toBeDefined();
      // A new owner may have reloaded canonical before the old queue is released.
      for (const issue of [parent, child, issueDetailOf("other")])
        tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
    }
    const before = tab.nerve.calls.length;
    held.answer();
    await first;
    await vi.advanceTimersByTimeAsync(1);
    expect(tab.nerve.calls).toHaveLength(before);
    await until(() => result.settled, "retired queued command");
    await expect(waiting).rejects.toBeInstanceOf(SessionChangedError);
    expect(tab.nerve.calls).toHaveLength(before);
  });
}

it("a queued command is independent of permission changes in another workspace", async () => {
  const responder = tab.nerve.respond;
  tab.nerve.respond = undefined;
  const workspace = workspaceOf("second");
  const project = projectOf("B", workspace.id, { id: "B" });
  await loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [tab.workspace, workspace]);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, workspace, [project]);
  const hold = heldChange<void>();
  const first = tab.app.issue.enqueueIssueChanges(["x"], () => hold.sent);
  const next = commands.linkAdd();
  const result = track(next);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, workspace, []);
  await loadProjects(tab.nerve, tab.app.projectRoot.project, workspace, [project]);
  tab.nerve.respond = responder;
  const count = tab.nerve.calls.length;
  hold.answer();
  await first;
  await until(() => tab.nerve.calls.length > count || result.settled, "independent POST");
  expect(result.settled).toBe(false);
  expect([tab.nerve.calls[count]!.method, tab.nerve.calls[count]!.path]).toEqual(["POST", "/api/v0/issues/x/links"]);
  tab.nerve.calls[count]!.answer(json(201, issueLinkOf("new")));
  await next;
  expect(detail().link.getLinksByIssueId("x")).toContain("new");
});
it("another root with the same IDs cannot retire or accept this root's queued command", async () => {
  const other = await issueStoreTab();
  try {
    other.app.issue.issues.acceptRead(parent, other.app.issue.captureRead());
    const hold = heldChange<void>();
    const first = tab.app.issue.enqueueIssueChanges(["x"], () => hold.sent);
    const next = commands.linkAdd();
    const result = track(next);
    other.app.issue.retire();
    const count = tab.nerve.calls.length;
    hold.answer();
    await first;
    await until(() => tab.nerve.calls.length > count || result.settled, "independent root POST");
    expect(result.settled).toBe(false);
    tab.nerve.calls[count]!.answer(json(201, issueLinkOf("new")));
    await next;
    expect(detail().link.getLinksByIssueId("x")).toContain("new");
    expect(other.app.issue.issueDetail.link.getLinksByIssueId("x")).toBeUndefined();
  } finally {
    other.app.dispose();
  }
});

it("description, link, comment and a parent batch reserve the same issue before a later child edit", async () => {
  const initial = commands.description();
  const firstResult = track(initial);
  await until(() => tab.nerve.calls.length === 1 || firstResult.settled, "description PATCH");
  const rest = [
    commands.linkAdd(),
    commands.commentAdd(),
    commands.subAdd(),
    tab.app.issue.updateIssue("child", () => ({ name: "edited child" })),
  ];
  const outcomes = rest.map((sent) => track<unknown>(sent));
  await vi.advanceTimersByTimeAsync(1);
  expect(tab.nerve.calls).toHaveLength(1);
  tab.nerve.calls[0]!.answer(
    json(200, { ...parent, description_html: "<p>changed</p>", updated_at: "2026-10-31T00:00:00Z" })
  );
  await initial;
  const expected = [
    ["POST", "/api/v0/issues/x/links"],
    ["POST", "/api/v0/issues/x/comments"],
    ["POST", "/api/v0/issues/x/sub-issues"],
    ["PATCH", "/api/v0/issues/child"],
  ];
  for (const [index, endpoint] of expected.entries()) {
    // oxlint-disable-next-line no-await-in-loop
    await until(
      () => tab.nerve.calls.filter((call) => call.method !== "GET").length === index + 2,
      "next reserved write"
    );
    const writes = tab.nerve.calls.filter((call) => call.method !== "GET");
    expect([writes.at(-1)!.method, writes.at(-1)!.path]).toEqual(endpoint);
    expect(outcomes.slice(index).every((outcome) => !outcome.settled)).toBe(true);
    writes.at(-1)!.answer(problem(403, "forbidden"));
  }
  await Promise.all(rest.map((sent) => sent.catch((error) => error)));
  expect(outcomes.every((outcome) => outcome.settled)).toBe(true);
});
