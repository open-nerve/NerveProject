// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A row's quick actions on a module's page (P6b spec 2.5, 2.7, R5, R7, R13, R15): the project page's, and the removal
// of the work item from the module, offered in each layout to a member while the module takes work items, whatever
// its status; none once it is archived, nor to a guest. The removal is one change of the work item's modules, worked
// out as it is sent: the modules it holds but the page's, an archived one kept. Once nerve confirms it the row leaves
// the list, the page says so and reads the module's progress again, once; a refusal is said, the row stays and nothing
// is read. While it is out, the row's removal again sends nothing, and neither follows once the page is another
// module's or another session's. The edit dialog changes what its form changes, never the work item's modules; the
// copy joins the page's module alone. The layouts, the menus, the dialog's form, the stores and HTTP are real.
import { act, createElement, forwardRef, useImperativeHandle, type ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Issue, IssueDetail, Module } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueDetailOf, listIssue } from "@/store/issue/fake-issues";
import { loadModule, moduleRequests } from "@/store/planning/fake-modules";
import { moduleOf } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { M1, layouts, modulePreferencesOf, planningPageTab, type Layout } from "./fake-planning-page";

const bound = vi.hoisted(() => ({
  app: undefined as RootStore | undefined,
  params: {} as Record<string, string>,
  session: { loginId: "one" },
}));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: bound.session }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => bound.params,
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
// jsdom has no viewport geometry, dialog transitions, editor or font loading; rows, cards, days, the form's own fields
// and HTTP are real.
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children, as }: { children: ReactNode; as?: string }) =>
    as ? createElement(as, null, children) : children,
}));
vi.mock("@nerve/propel/emoji-icon-picker", () => ({ Logo: () => null }));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ModalCore: (props: { isOpen: boolean; children: ReactNode }) =>
    props.isOpen ? <div role="dialog">{props.children}</div> : null,
  ModalTitle: (props: { children: ReactNode }) => <h3>{props.children}</h3>,
}));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: forwardRef(function Editor(_props, ref) {
    useImperativeHandle(ref, () => ({ isEditorReadyToDiscard: () => true, clearEditor: () => {} }));
    return <textarea aria-label="description" />;
  }),
}));
vi.mock("@/components/issues/issue-modal/components/default-properties", () => ({
  IssueDefaultProperties: () => null,
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));

/** Another module of the project, and an archived one, which the work item is in too. */
const M2 = "019a0000-0000-7000-8000-0000000000d2";
const M3 = "019a0000-0000-7000-8000-0000000000d3";
const LATER = "2026-12-31T13:00:00Z";
const shownFields = { state_id: "state", priority: "high", target_date: "2026-12-31" } as const;
/** The page's work item "first", in the page's module, another one and an archived one. */
const held: Issue = listIssue("first", 1000, { module_ids: [M1, M2, M3], ...shownFields });
let page: Awaited<ReturnType<typeof planningPageTab>>;
beforeEach(() => {
  bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M1 };
  bound.session.loginId = "one";
  toasts.length = 0;
});
afterEach(async () => page.close());

/**
 * Shows the page of the module as its fields say to the caller of its role, its one work item "first" in the layout;
 * the tab holds the other module and the archived one as nerve gives them. nerve's list of the page.
 */
async function shown(pageFields: Parameters<typeof planningPageTab>[1] = {}, layout: Layout = "list") {
  page = await planningPageTab(EIssuesStoreType.MODULE, { role: 15, ...pageFields });
  bound.app = page.app;
  await loadModule(page.nerve, page.app.planning.modules, moduleOf(page.project, "Other", { id: M2 }));
  const old = { id: M3, status: "completed" as const, archived_at: "2026-12-01T09:00:00Z" };
  await loadModule(page.nerve, page.app.planning.modules, moduleOf(page.project, "Old", old));
  page.nerve.calls.length = 0;
  const list = layouts[layout].page(true, held);
  await page.show(modulePreferencesOf(layouts[layout].display), list);
  await until(() => row() !== null, "the row");
  return list;
}
/** The row's quick actions, by their names, as its menu offers them. */
const offered = () =>
  [...document.querySelectorAll('[data-context-menu="true"] [role="menuitem"]')].map((node) => node.textContent);
/** Chooses the row's quick action of name. */
const choose = (name: string) =>
  act(async () =>
    [...document.querySelectorAll<HTMLButtonElement>('[data-context-menu="true"] [role="menuitem"]')]
      .find((node) => node.textContent === name)
      ?.click()
  );
const REMOVE = "issue.remove.module.label";
const row = () => document.querySelector('[id^="issue-first"]');
const writesOf = (id: string) => page.nerve.to(`/api/v0/issues/${id}`).filter((call) => call.method === "PATCH");
const creates = () => page.nerve.to("/api/v0/projects/A/issues").filter((call) => call.method === "POST");
const progressReads = (id = M1) => page.nerve.to(moduleRequests.progress({ id }));
/** The module's work item "first" as nerve answers a write or a read of it, with fields. */
const first = (changes: Partial<IssueDetail> = {}): IssueDetail => ({
  ...issueDetailOf("first"),
  ...held,
  description_html: "<p>Server detail</p>",
  updated_at: LATER,
  ...changes,
});
/** Removes the row's work item from the module by its quick action; nerve holds the change. */
async function removal() {
  await choose(REMOVE);
  await until(() => writesOf("first").length === 1, "the removal");
  expect(writesOf("first")[0]!.body).toEqual({ module_ids: [M2, M3] });
  return writesOf("first")[0]!;
}

it.each(Object.keys(layouts) as Layout[])(
  "offers a member the removal of a row from a module in the %s layout, with the project page's actions",
  async (layout) => {
    await shown({}, layout);
    expect(offered()).toEqual(expect.arrayContaining(["common.actions.edit", "common.actions.make_a_copy", REMOVE]));
  }
);

it("offers a member the removal of a row from a completed module, whose status takes nothing from it", async () => {
  await shown({ module: { status: "completed" } });
  expect(offered()).toEqual(expect.arrayContaining([REMOVE]));
});

// nerve archives only completed or cancelled modules: being archived alone makes the list read-only
const archived: Partial<Module> = { status: "completed", archived_at: "2026-12-01T09:00:00Z" };
it.each([
  { who: "a member", page: "an archived module's", fields: { module: archived } },
  { who: "a guest", page: "a module's", fields: { role: 5 as const } },
])("offers $who no removal from the module on $page page", async ({ fields }) => {
  await shown(fields);
  expect(offered()).toEqual(["common.actions.open_in_new_tab", "common.actions.copy_link"]);
});

it("removes a work item from the module in one change of the modules it holds but the page's; once nerve confirms, the row leaves, the page says so and reads the progress again", async () => {
  await shown();
  const change = await removal();
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(toasts, "nothing said before nerve confirms").toEqual([]);
  expect(progressReads()).toEqual([]);
  await act(async () => change.answer(json(200, first({ module_ids: [M2, M3] }))));
  await until(() => row() === null, "the row off the list");
  expect(toasts).toEqual([
    expect.objectContaining({ type: "success", title: "common.success", message: "issue.remove.module.success" }),
  ]);
  await until(() => progressReads().length === 1, "the progress again");
  expect(progressReads()[0]).toMatchObject({ method: "GET" });
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(progressReads(), "once").toHaveLength(1);
  expect(writesOf("first")).toHaveLength(1);
});

it("works the removal out as it is sent, from the modules nerve gave the work item in its answer to the change before it", async () => {
  await shown();
  // a change of the work item, sent elsewhere on the page, is out as the removal is chosen
  const change = page.app.issue.moduleIssues.updateIssue("ws", "A", "first", { priority: "low" });
  void change.catch(() => undefined);
  await until(() => writesOf("first").length === 1, "the change");
  await choose(REMOVE);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(writesOf("first"), "the removal waits for the change before it").toHaveLength(1);
  // meanwhile the work item left the archived module elsewhere: nerve's answer says so
  await act(async () => writesOf("first")[0]!.answer(json(200, first({ priority: "low", module_ids: [M1, M2] }))));
  await until(() => writesOf("first").length === 2, "the removal");
  expect(writesOf("first")[1]!.body).toEqual({ module_ids: [M2] });
});

it("sends nothing for the row's removal again while its removal is out", async () => {
  await shown();
  const change = await removal();
  await choose(REMOVE);
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(writesOf("first")).toHaveLength(1);
  await act(async () => change.answer(json(200, first({ module_ids: [M2, M3] }))));
  await until(() => progressReads().length === 1, "the progress again");
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(writesOf("first")).toHaveLength(1);
  expect(toasts).toHaveLength(1);
});

it.each([
  { status: 409, code: "planning.modules_disabled" },
  { status: 403, code: "forbidden" },
])("says why nerve refused the removal ($code); the row stays and nothing is read", async ({ status, code }) => {
  await shown();
  const change = await removal();
  await act(async () => change.answer(problem(status, code)));
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(toasts).toEqual([expect.objectContaining({ type: "error", message: `errors.${code.replace(".", "_")}` })]);
  expect(row()).not.toBeNull();
  expect(progressReads()).toEqual([]);
});

it.each(["another module", "another session"])(
  "neither says nor reads anything when nerve confirms the removal once the page is %s's",
  async (ending) => {
    await shown();
    const change = await removal();
    if (ending === "another module") {
      bound.params = { workspaceSlug: "ws", projectId: "A", moduleId: M2 };
      await page.mount();
    } else bound.session.loginId = "two";
    await act(async () => change.answer(json(200, first({ module_ids: [M2, M3] }))));
    await act(async () => vi.advanceTimersByTimeAsync(100));
    expect(toasts).toEqual([]);
    expect(progressReads(M1)).toEqual([]);
    expect(progressReads(M2)).toEqual([]);
  }
);

/**
 * Has nerve answer the reads a row's dialog makes: the work item's (with its description), its resources', and the
 * page's list's again once the check of the work item took the row off; the test answers the rest.
 */
function answerDialogReads(list: unknown) {
  const normal = page.nerve.respond;
  page.nerve.respond = (call: Call) => {
    if (call.method !== "GET") normal?.(call);
    else if (call.path === "/api/v0/issues/first") call.answer(json(200, first({ updated_at: held.updated_at })));
    else if (/\/issues\/first\/(sub-issues|relations|links|reactions|activities|comments)$/.test(call.path))
      call.answer(json(200, { data: [], next_cursor: null }));
    else if (call.path === "/api/v0/projects/A/issues") call.answer(json(200, list));
    else normal?.(call);
  };
}
const form = () => document.querySelector<HTMLFormElement>('[role="dialog"] form');
/** Sends the open dialog's form once it has read its work item, its name typed as name when given. */
async function sendDialog(name?: string) {
  await until(() => form()?.querySelector<HTMLButtonElement>('button[type="submit"]')?.disabled === false, "the form");
  if (name !== undefined)
    await act(async () => {
      const input = form()!.querySelector<HTMLInputElement>('input[name="name"]')!;
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, name);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  await act(async () => form()!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
}

it("changes what the edit dialog's form changes of a work item in three modules, never its modules", async () => {
  const list = await shown();
  answerDialogReads(list);
  await choose("common.actions.edit");
  await sendDialog("Renamed");
  await until(() => writesOf("first").length === 1, "the change");
  const [change] = writesOf("first");
  expect(change!.body).toMatchObject({ name: "Renamed" });
  expect(change!.body).not.toHaveProperty("module_ids");
});

it("joins the copy of a work item in three modules to the page's module alone", async () => {
  const list = await shown();
  answerDialogReads(list);
  await choose("common.actions.make_a_copy");
  await sendDialog();
  await until(() => creates().length === 1, "the copy");
  expect(creates()[0]!.body).toMatchObject({ name: "first (copy)", module_ids: [M1] });
});
