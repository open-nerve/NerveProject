// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// A draft's cycle and modules in its form (M6 P7a spec 2.8, R6, R17; re-72 F6). The form shows those the draft holds,
// an ended cycle and an archived module too; what the form changed of them goes as a draft keeps them: no cycle as no
// key (never null, never ""), no module as an empty set, which is how a draft's modules are taken out (left out, the
// set it held would stay). A draft moved to another project leaves its cycle and modules, which are its project's,
// whatever it held as the form saved it. nerve's refusal to publish it names the cycle or a module under its field; one
// that names neither is said; either way the draft stays, a copy's draft too. The modal, the form and its fields, the
// stores and HTTP are real.
import React, { act } from "react";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Draft, DraftCreate, DraftUpdate, IssueCreate, IssueDraftPayload, Project } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RootStore } from "@/store/root.store";
import { StoreContext } from "@/lib/store-context";
import { contentAssetOf } from "@/store/asset/fake-assets";
import { rowOf } from "@/store/planning/fake-cycles";
import { moduleRowOf } from "@/store/planning/fake-modules";
import { cycleOf, loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
import { json, problem, type Call } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { draftInput } from "@/services/issue/draft-input";
import { cycleField, form, moduleField, pick } from "../issue-layouts/roots/fake-planning-page";
import { editor } from "../fake-body-images";
import { CreateUpdateIssueModalBase } from "./base";
import { fieldRefused, refusedAs } from "./fake-dialog";
import { draftFilesTab } from "./fake-draft-files";
import type { IssuesModalProps } from "./modal";

type Tab = Awaited<ReturnType<typeof draftFilesTab>>;
const m = vi.hoisted(() => ({ projects: [] as string[], tab: undefined as unknown }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "draft-planning" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "draft-planning" }) }));
vi.mock("@/lib/store-context", async () => ({
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/components/editor/rich-text", () => import("@/components/editor/rich-text/editor-implementation"));
vi.mock("react-router", async (original) => ({
  ...(await original<typeof import("react-router")>()),
  useParams: () => ({ workspaceSlug: "ws" }),
}));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.WORKSPACE_DRAFT }));
vi.mock("@/hooks/use-issues-actions", () => ({
  useIssuesActions: () => ({
    createIssueAsGiven: (project: string, body: IssueCreate) =>
      (m.tab as Tab).app.issue.projectIssues.createIssue("ws", project, body),
  }),
}));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({ allowedProjectIds: m.projects, selectedParentIssue: null, setSelectedParentIssue: () => {} }),
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: ({ value, onChange }: { value?: string | null; onChange: (value: string | null) => void }) => (
    <select aria-label="project" value={value ?? ""} onChange={(event) => onChange(event.target.value || null)}>
      <option value="">No project</option>
      {m.projects.map((project) => (
        <option key={project} value={project}>
          {project}
        </option>
      ))}
    </select>
  ),
}));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ...(await import("@/components/issues/issue-modal/fake-dialog")).modal,
}));

let tab: Tab;
let projectB: Project;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  );
  const unlaidOut = document.createElement("div");
  // oxlint-disable-next-line no-extend-native -- declares what the test environment lacks; tests only
  Object.defineProperties(Range.prototype, {
    getClientRects: { configurable: true, value: () => unlaidOut.getClientRects() },
    getBoundingClientRect: { configurable: true, value: () => unlaidOut.getBoundingClientRect() },
  });
  toasts.length = 0;
  tab = await draftFilesTab();
  m.tab = tab;
  projectB = projectOf("B", tab.workspace.id, { id: "B", member_role: 20 });
  // the workspace's second project, listed before the tab answers the rest of what the modal reads
  const respond = tab.nerve.respond;
  tab.nerve.respond = undefined;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, projectB]);
  tab.nerve.respond = respond;
  tab.nerve.calls.length = 0;
  m.projects = [tab.project.id, projectB.id];
  answeringPlanningLists();
});
afterEach(async () => {
  await tab.dispose();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  vi.useRealTimers();
});

/** Project A's cycles Now (under way) and Old (ended), its modules Core and Edge, and its archived module Gone. */
const planning = () => ({
  now: cycleOf(tab.project, "Now", { status: "current" }),
  old: cycleOf(tab.project, "Old", { status: "completed" }),
  core: moduleOf(tab.project, "Core"),
  edge: moduleOf(tab.project, "Edge"),
  gone: moduleOf(tab.project, "Gone", { status: "completed", archived_at: "2026-12-01T09:00:00Z" }),
});
/** From now on nerve lists each project's cycles and modules as planning has A's; B has none. */
function answeringPlanningLists() {
  const { now, old, core, edge, gone } = planning();
  const lists: Record<string, Record<"cycles" | "modules", { listed: unknown[]; archived: unknown[] }>> = {
    A: {
      cycles: { listed: [rowOf(now), rowOf(old)], archived: [] },
      modules: { listed: [moduleRowOf(core), moduleRowOf(edge)], archived: [moduleRowOf(gone)] },
    },
    B: { cycles: { listed: [], archived: [] }, modules: { listed: [], archived: [] } },
  };
  const respond = tab.nerve.respond;
  tab.nerve.respond = (call) => {
    const match = /^\/api\/v0\/projects\/([^/]+)\/(cycles|modules)$/.exec(call.path);
    const of = match && lists[match[1]!]?.[match[2] as "cycles" | "modules"];
    if (call.method === "GET" && of)
      call.answer(json(200, { data: call.query.archived === "true" ? of.archived : of.listed }));
    else respond?.(call);
  };
}
/** A stored draft of project A whose payload holds what fields say beside its name and description. */
const stored = (fields: IssueDraftPayload) =>
  tab.seed({
    project_id: tab.project.id,
    payload: { name: "Draft", description_html: "<p>Draft body</p>", ...fields },
  });
/**
 * The modal as the drafts open it, with an SWR cache of its own: this file's forms read the same lists in the same
 * session, and the cache's dedupe would keep a later test's new stores from them.
 */
const shown = (props: IssuesModalProps) =>
  act(async () =>
    tab.renderer.render(
      <StoreContext.Provider value={tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <CreateUpdateIssueModalBase
            fetchIssueDetails={false}
            storeType={EIssuesStoreType.WORKSPACE_DRAFT}
            withDraftIssueWrapper={false}
            {...props}
          />
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
/** Opens the stored draft's form as its row does, publishing it from there when publish says so. */
async function opened(draft: Draft, publish = false) {
  const held = tab.app.issue.workspaceDraftIssues.getIssueById(draft.id, "ws");
  await shown({
    isOpen: true,
    isDraft: true,
    draft: held,
    moveToIssue: publish,
    onClose: tab.close,
    onSubmit: tab.confirmed,
  });
  await tab.waitFor(() => !!editor() && cycleField() !== undefined, "the draft's form");
  await act(async () => vi.advanceTimersByTimeAsync(50));
}
const saves = (draft: Draft) => tab.calls("PATCH", `/api/v0/drafts/${draft.id}`);
/** Saves the open form, and what it sent to save the draft. */
async function saved(draft: Draft) {
  await tab.submit();
  await tab.waitFor(() => saves(draft).length === 1, "the draft's save");
  return saves(draft)[0]!.body as DraftUpdate;
}
const held = (draft: Draft) => tab.drafts().find((row) => row.id === draft.id);
/** Picks the option of name in the list a pick left open (the modules' list stays open), as a pointer does. */
async function also(name: string) {
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === name
  );
  expect(option, name).toBeDefined();
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

it("shows the cycle and the modules the draft holds, an ended cycle and an archived module too, by their names", async () => {
  const { old, core, gone } = planning();
  await opened(await stored({ cycle_id: old.id, module_ids: [core.id, gone.id] }));
  await tab.waitFor(() => cycleField()?.textContent === "common.cycle: Old", "the ended cycle's name");
  expect(moduleField()?.textContent).toBe("common.modules: module.count: Core; Gone, common.archived");
});

it("leaves no cycle in the draft for No cycle: its save drops the key, its modules as they were", async () => {
  const { now, core, gone } = planning();
  const draft = await stored({ cycle_id: now.id, module_ids: [core.id, gone.id] });
  await opened(draft);
  await pick(cycleField, "cycle.no_cycle");
  const body = await saved(draft);
  expect(body.payload).not.toHaveProperty("cycle_id");
  expect(body.payload?.module_ids).toEqual([core.id, gone.id]);
  await tab.waitFor(() => held(draft)?.payload.module_ids !== undefined && toasts.length === 1, "the saved draft");
  expect(held(draft)?.payload).not.toHaveProperty("cycle_id");
});

it("takes all its modules out of the draft as an empty set, which nerve keeps; left out, the set it held would stay", async () => {
  const { core, gone } = planning();
  const draft = await stored({ module_ids: [core.id, gone.id] });
  await opened(draft);
  await pick(moduleField, "Core");
  await also("Gone, common.archived");
  expect(moduleField()?.textContent).toBe("common.modules: module.no_module");
  const body = await saved(draft);
  expect(body.payload?.module_ids).toEqual([]);
  await tab.waitFor(() => toasts.length === 1, "the saved draft");
  expect(held(draft)?.payload.module_ids).toEqual([]);
});

it.each([
  { from: "a draft's form", draft: true },
  { from: "a new work item's form, saved to the drafts", draft: false },
])("saves a new draft from $from with no cycle as no key, its modules as an empty set", async ({ draft }) => {
  await shown({
    isOpen: true,
    data: { project_id: tab.project.id },
    onClose: tab.close,
    ...(draft ? { isDraft: true, onSubmit: tab.confirmed } : { storeType: EIssuesStoreType.PROJECT }),
  });
  await tab.waitFor(() => !!editor() && cycleField() !== undefined, "the new form");
  await tab.waitFor(() => cycleField()?.querySelector('[role="status"]') == null, "the form's lists");
  await pick(cycleField, "Now");
  await pick(cycleField, "cycle.no_cycle");
  await act(async () => {
    const input = form()!.querySelector<HTMLInputElement>("#name")!;
    Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")!.set!.call(input, "New");
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  if (draft) await tab.submit();
  else await tab.click("save_to_drafts");
  const creates = () => tab.calls("POST", "/api/v0/workspaces/ws/drafts");
  await tab.waitFor(() => creates().length === 1, "the new draft");
  const body = creates()[0]!.body as DraftCreate;
  expect(body.payload).toMatchObject({ name: "New", module_ids: [] });
  expect(body.payload).not.toHaveProperty("cycle_id");
});

it("takes the draft out of its cycle and its modules as the form moves it to another project", async () => {
  const { now, core } = planning();
  const draft = await stored({ cycle_id: now.id, module_ids: [core.id] });
  await opened(draft);
  await tab.chooseProject(projectB.id);
  await tab.waitFor(() => cycleField()?.textContent === "common.cycle: cycle.label", "B's form");
  expect(moduleField()?.textContent).toBe("common.modules: module.no_module");
  const body = await saved(draft);
  expect(body.project_id).toBe(projectB.id);
  expect(body.payload).not.toHaveProperty("cycle_id");
  expect(body.payload?.module_ids).toEqual([]);
});

it("moved to another project, leaves the cycle and the modules a write gave the draft since its form opened", async () => {
  const { now, core } = planning();
  const draft = await stored({});
  await opened(draft);
  // its row's pick, confirmed while the form is open
  await act(async () =>
    tab.app.issue.workspaceDraftIssues.updateIssue("ws", draft.id, (row) =>
      draftInput({ cycle_id: now.id, module_ids: [core.id] }, row.payload)
    )
  );
  expect(held(draft)?.payload).toMatchObject({ cycle_id: now.id, module_ids: [core.id] });
  tab.nerve.calls.length = 0;
  await tab.chooseProject(projectB.id);
  const body = await saved(draft);
  expect(body.project_id).toBe(projectB.id);
  expect(body.payload).not.toHaveProperty("cycle_id");
  expect(body.payload?.module_ids).toEqual([]);
});

/** Publishes the open form's draft; nerve saves it, then refuses to publish it as answer says. */
async function refusedPublish(draft: Draft, answer: () => Response) {
  tab.control.publish = (call: Call) => call.answer(answer());
  await tab.click("add_to_project");
  await tab.waitFor(() => tab.calls("POST", `/api/v0/drafts/${draft.id}/publish`).length === 1, "the publish");
  await act(async () => vi.advanceTimersByTimeAsync(50));
}

it.each([
  {
    refusal: "of its cycle",
    answer: () => fieldRefused("cycle_id"),
    field: cycleField,
    message: "errors.field.not_allowed",
  },
  {
    refusal: "that its cycle has ended",
    answer: () => problem(409, "planning.cycle_completed"),
    field: cycleField,
    message: "errors.planning_cycle_completed",
  },
  {
    refusal: "of a module",
    answer: () => fieldRefused("module_ids[0]"),
    field: moduleField,
    message: "errors.field.not_allowed",
  },
])(
  "shows nerve's refusal to publish the draft $refusal under its field, which takes the focus; the draft stays",
  async ({ answer, field, message }) => {
    const { old, core } = planning();
    const draft = await stored({ cycle_id: old.id, module_ids: [core.id] });
    await opened(draft, true);
    await refusedPublish(draft, answer);
    expect(refusedAs(field())).toEqual({ invalid: "true", message });
    expect(document.activeElement, "the focus").toBe(field());
    expect(toasts).toEqual([]);
    expect(form()).not.toBeNull();
    expect(tab.app.issue.workspaceDraftIssues.getIssueById(draft.id, "ws")).toBeDefined();
    expect(tab.issues).toEqual([]);
  }
);

it.each(["planning.archived", "planning.cycles_disabled"])(
  "says why nerve refused to publish the draft when it names no field (%s); the draft stays",
  async (code) => {
    const { now, core } = planning();
    const draft = await stored({ cycle_id: now.id, module_ids: [core.id] });
    await opened(draft, true);
    await refusedPublish(draft, () => problem(409, code));
    expect(toasts).toEqual([expect.objectContaining({ type: "error", message: `errors.${code.replace(".", "_")}` })]);
    expect([refusedAs(cycleField()).invalid, refusedAs(moduleField()).invalid]).toEqual([null, null]);
    expect(form()).not.toBeNull();
    expect(tab.app.issue.workspaceDraftIssues.getIssueById(draft.id, "ws")).toBeDefined();
  }
);

// T12's note: a copy with images goes as a draft, which nerve publishes; its cycle is the form's, though the project's
// cycles were turned off as the dialog was open, and nerve refuses to publish it (design 3.3): the copy's draft stays.
it("keeps a copy's draft whose cycle nerve refuses to publish, the project's cycles turned off as its dialog was open", async () => {
  const { now, core } = planning();
  const s1 = "44444444-4444-4444-8444-000000000001";
  tab.assets.set(s1, contentAssetOf(s1, tab.hosts.issue, "description"));
  await shown({
    isOpen: true,
    data: {
      name: "Copy",
      project_id: tab.project.id,
      description_html: `<p data-id="a">First</p><image-component src="${s1}"></image-component>`,
      cycle_id: now.id,
      module_ids: [core.id],
    },
    sourceIssueId: tab.issue.id,
    storeType: EIssuesStoreType.PROJECT,
    onClose: tab.close,
    onSubmit: tab.confirmed,
  });
  await tab.waitFor(() => !!editor() && cycleField()?.textContent === "common.cycle: Now", "the copy's form");
  act(() => tab.app.projectRoot.project.confirmProject(tab.project, (project) => ({ ...project, cycle_view: false })));
  await tab.waitFor(() => cycleField() === undefined, "the cycle's field gone");
  const publish = tab.control.publish;
  tab.control.publish = (call, draft) =>
    draft.payload.cycle_id ? call.answer(problem(409, "planning.cycles_disabled")) : publish(call, draft);
  await tab.submit();
  await tab.waitFor(() => tab.calls("POST", /\/drafts\/[^/]+\/publish$/).length === 1, "the publish");
  await act(async () => vi.advanceTimersByTimeAsync(100));
  expect(toasts.map(({ message }) => message)).toEqual(["errors.planning_cycles_disabled"]);
  expect(tab.drafts()).toEqual([expect.objectContaining({ payload: expect.objectContaining({ cycle_id: now.id }) })]);
  expect(document.body.textContent).toContain("content_files.copy.kept");
  expect(tab.issues).toEqual([]);
  expect(form()).not.toBeNull();
});
