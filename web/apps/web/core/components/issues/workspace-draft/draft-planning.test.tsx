// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, 2.8, R5, R6, R14, R20: a draft's cycle and modules on its row of the workspace's drafts, against
// the session's stores and a fake nerve. Each shows while the draft has a project with the domain on, named by its
// row; an archived one by its name and its caption, each project's lists read once for all its drafts, as its options
// are as a row's list opens. A cycle picked goes as the draft's payload with it, No cycle as none; a module picked is
// one addition or removal, applied in the draft's turn to the payload it then holds, the rest of which it keeps. The
// drafts page, its rows, dropdowns, stores and HTTP are real.
import { act } from "react";
import { MemoryRouter } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Draft, DraftUpdate, IssueDraftPayload, Project } from "@nerve/api-client";
import type { RootStore } from "@/store/root.store";
import { json, type Call } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { issueSelectionTab } from "@/components/core/modals/fake-issue-selection";
import { draftOf } from "@/store/issue/fake-issues";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
import { WorkspaceDraftIssuesRoot } from "./root";

const bound = vi.hoisted(() => ({ app: undefined as RootStore | undefined }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return bound.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "draft-rows" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "draft-rows" }) }));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws" }),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
// The draft's own dialog is the modal's tests' (issue-modal/draft-planning.test.tsx); its rows here.
vi.mock("@/components/issues/issue-modal/modal", () => ({ CreateUpdateIssueModal: () => null }));

const ARCHIVED = "2026-10-20T00:00:00Z";
let tab: Awaited<ReturnType<typeof issueSelectionTab>>;
let projectB: Project;
let drafts: Draft[];
/** What a test holds of nerve's answers: the saves of a draft, which it then answers itself. */
let holding: (call: Call) => boolean;
beforeEach(async () => {
  tab = await issueSelectionTab();
  bound.app = tab.app;
  projectB = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, projectB]);
  drafts = [];
  holding = () => false;
  const directory = { A: directoryOf(tab.project), B: directoryOf(projectB) };
  const states = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    const lists = /^\/api\/v0\/projects\/([^/]+)\/(cycles|modules)$/.exec(call.path);
    const of = lists && directory[lists[1] as "A" | "B"]?.[lists[2] as "cycles" | "modules"];
    if (call.method === "GET" && of)
      call.answer(json(200, { data: call.query.archived === "true" ? of.archived : of.listed }));
    else if (call.method === "GET" && call.path === "/api/v0/workspaces/ws/drafts")
      call.answer(json(200, { data: drafts, total: drafts.length, next_cursor: null }));
    else if (call.method === "PATCH" && call.path.startsWith("/api/v0/drafts/") && !holding(call)) saved(call);
    else states?.(call);
  };
  tab.nerve.calls.length = 0;
  toasts.length = 0;
});
afterEach(async () => tab.close());

/** A project's cycles Now (under way) and Old (archived), its modules Alpha and Beta, and Gamma (archived). */
const planning = (project: Project) => ({
  now: cycleOf(project, "Now", { status: "current" }),
  old: cycleOf(project, "Old", { status: "completed", archived_at: ARCHIVED }),
  alpha: moduleOf(project, "Alpha", { status: "in-progress" }),
  beta: moduleOf(project, "Beta"),
  gamma: moduleOf(project, "Gamma", { status: "completed", archived_at: ARCHIVED }),
});
function directoryOf(project: Project) {
  const { now, old, alpha, beta, gamma } = planning(project);
  return {
    cycles: { listed: [rowOf(now)], archived: [rowOf(old)] },
    modules: { listed: [moduleRowOf(alpha), moduleRowOf(beta)], archived: [moduleRowOf(gamma)] },
  };
}
/** The draft id of the project (none for null), whose payload holds its cycle and modules as fields say. */
const draftIn = (id: string, project: string | null, fields: IssueDraftPayload): Draft => {
  const draft = draftOf(id, tab.workspace.id);
  return { ...draft, project_id: project, payload: { ...draft.payload, ...fields } };
};
/** nerve saves the draft as the call sends it, and answers with it. */
function saved(call: Call) {
  const body = call.body as DraftUpdate;
  const id = call.path.split("/").at(-1)!;
  const row = { ...drafts.find((draft) => draft.id === id)!, payload: body.payload! };
  drafts = drafts.map((draft) => (draft.id === id ? row : draft));
  call.answer(json(200, row));
}
/** The workspace's drafts page, nerve listing rows. */
async function draftsPage(rows: Draft[]) {
  drafts = rows;
  await tab.render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter>
        <WorkspaceDraftIssuesRoot workspaceSlug="ws" />
      </MemoryRouter>
    </SWRConfig>
  );
  await until(() => rows.every((row) => document.getElementById(`issue-${row.id}`) !== null), "the drafts");
  await settle();
}
/** The row's control of the field, by the name its row gives it (R20). */
const control = (field: "common.cycle" | "common.modules", draft: string) =>
  [...document.querySelectorAll<HTMLButtonElement>("button")].find(
    (button) => button.textContent?.startsWith(`${field}: `) && button.textContent.endsWith(`, ${draft}`)
  );
/** The reads of the project's list of the domain so far: "listed" for the list, "archived" for its archive. */
const readsOf = (project: string, domain: "cycles" | "modules") =>
  tab.nerve.calls
    .filter((call) => call.method === "GET" && call.path === `/api/v0/projects/${project}/${domain}`)
    .map((call) => (call.query.archived === "true" ? "archived" : "listed"));
const settle = () => act(async () => vi.advanceTimersByTimeAsync(50));
const saves = (draft: string) =>
  tab.nerve.calls.filter((call) => call.method === "PATCH" && call.path === `/api/v0/drafts/${draft}`);
/** Opens the row's control of the field; its list shows once the project's list is read again. */
async function openList(field: "common.cycle" | "common.modules", draft: string, project: string) {
  const domain = field === "common.cycle" ? "cycles" : "modules";
  const before = readsOf(project, domain).filter((read) => read === "listed").length;
  await act(async () => control(field, draft)!.click());
  await act(async () =>
    until(() => readsOf(project, domain).filter((read) => read === "listed").length === before + 1, "the options")
  );
  await settle();
}
/** Picks the open list's option of name, as a pointer does. */
async function choose(name: string) {
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === name
  );
  expect(option, name).toBeDefined();
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

it("names the drafts' archived cycle and modules with their caption, each project's lists read once for all its drafts", async () => {
  const a = planning(tab.project);
  const b = planning(projectB);
  await draftsPage([
    draftIn("a1", "A", { cycle_id: a.old.id, module_ids: [a.gamma.id] }),
    draftIn("a2", "A", { cycle_id: a.old.id, module_ids: [a.alpha.id, a.gamma.id] }),
    draftIn("b1", "B", { cycle_id: b.old.id, module_ids: [b.gamma.id] }),
    draftIn("b2", "B", { cycle_id: b.now.id, module_ids: [] }),
  ]);
  await act(async () => until(() => readsOf("B", "modules").length === 2, "B's modules' lists"));
  await settle();
  expect(["a1", "a2", "b1", "b2"].map((draft) => control("common.cycle", draft)?.textContent)).toEqual([
    "common.cycle: Old, common.archived, a1",
    "common.cycle: Old, common.archived, a2",
    "common.cycle: Old, common.archived, b1",
    "common.cycle: Now, b2",
  ]);
  expect(["a1", "a2", "b1", "b2"].map((draft) => control("common.modules", draft)?.textContent)).toEqual([
    "common.modules: Gamma, common.archived, a1",
    "common.modules: module.count: Alpha; Gamma, common.archived, a2",
    "common.modules: Gamma, common.archived, b1",
    "common.modules: module.no_module, b2",
  ]);
  expect(["A", "B"].flatMap((project) => [readsOf(project, "cycles"), readsOf(project, "modules")])).toEqual([
    ["listed", "archived"],
    ["listed", "archived"],
    ["listed", "archived"],
    ["listed", "archived"],
  ]);
});

it("shows neither while the draft's project has the domain off, nor on a draft without a project", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    tab.project,
    { ...projectB, module_view: false },
  ]);
  tab.nerve.calls.length = 0;
  await draftsPage([draftIn("a1", "A", {}), draftIn("b1", "B", {}), draftIn("loose", null, {})]);
  expect(
    ["a1", "b1", "loose"].map((draft) => [
      control("common.cycle", draft) !== undefined,
      control("common.modules", draft) !== undefined,
    ])
  ).toEqual([
    [true, true],
    [true, false],
    [false, false],
  ]);
  expect(readsOf("B", "modules")).toEqual([]);
});

it("sends a cycle picked on a draft's row as its payload with it, from its own project's options; No cycle as none", async () => {
  const b = planning(projectB);
  const row = draftIn("b1", "B", { module_ids: [b.gamma.id] });
  await draftsPage([draftIn("a1", "A", {}), row]);
  await openList("common.cycle", "b1", "B");
  await choose("Now");
  await until(() => saves("b1").length === 1, "the draft's save");
  expect(saves("b1")[0]!.body).toEqual({ payload: { ...row.payload, cycle_id: b.now.id } });
  await settle();
  expect(control("common.cycle", "b1")?.textContent).toBe("common.cycle: Now, b1");
  await openList("common.cycle", "b1", "B");
  await choose("cycle.no_cycle");
  await until(() => saves("b1").length === 2, "the draft's second save");
  expect(saves("b1")[1]!.body).toEqual({ payload: row.payload });
  expect(saves("a1")).toEqual([]);
  expect(toasts).toEqual([]);
});

it("applies each module picked on a draft's row in the draft's turn to the payload it then holds", async () => {
  const a = planning(tab.project);
  const row = draftIn("a1", "A", { module_ids: [a.gamma.id] });
  await draftsPage([row]);
  holding = (call) => call.path === "/api/v0/drafts/a1" && saves("a1").length === 1;
  await openList("common.modules", "a1", "A");
  await choose("Alpha");
  await until(() => saves("a1").length === 1, "the first save");
  await choose("Beta");
  await settle();
  expect(saves("a1").map(({ body }) => body)).toEqual([
    { payload: { ...row.payload, module_ids: [a.gamma.id, a.alpha.id] } },
  ]);
  // nerve's answer holds the payload as it now is, a label of its own beside the first module
  const answer = { ...row.payload, label_ids: ["label", "other"], module_ids: [a.gamma.id, a.alpha.id] };
  drafts = [{ ...row, payload: answer }];
  await act(async () => saves("a1")[0]!.answer(json(200, { ...row, payload: answer })));
  await until(() => saves("a1").length === 2, "the second save");
  expect(saves("a1")[1]!.body).toEqual({ payload: { ...answer, module_ids: [a.gamma.id, a.alpha.id, a.beta.id] } });
  await settle();
  expect(control("common.modules", "a1")?.textContent).toBe(
    "common.modules: module.count: Gamma, common.archived; Alpha; Beta, a1"
  );
  expect(toasts).toEqual([]);
});
