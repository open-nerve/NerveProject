// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
// M6 P7a spec 2.3, 2.4, R3, R12, R13 (M2 closeout 4, M3 closeout 3): a work item's module dropdown, against the
// session's stores and a fake nerve. Its options are read as it opens and are the modules the work item may join, in
// nerve's order; an archived module it is in is listed after them, only to be taken out; a pick is one addition or one
// removal, never the whole set, and the list stays open.
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { Module, Project } from "@nerve/api-client";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { issueStoreTab } from "@/store/issue/fake-issues";
import {
  loadArchivedModules,
  loadModule,
  loadModules,
  moduleRequests,
  moduleRowOf,
} from "@/store/planning/fake-modules";
import { loadProjects, moduleOf, projectOf } from "@/store/project/fake-projects";
import type { TButtonVariants } from "@/components/dropdowns/types";
import { ModuleDropdown } from "./dropdown";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "modules" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n-values"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-modules", () => ({ useModules: () => tab.app.planning.modules }));

let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
let other: Project;
let alpha: Module, beta: Module, delta: Module, gamma: Module, old: Module, elsewhere: Module;
const changed = vi.fn();
const ARCHIVED = "2026-10-20T00:00:00Z";

beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  other = projectOf("B", tab.workspace.id, { id: "B" });
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project, other]);
  alpha = moduleOf(tab.project, "Alpha", { status: "in-progress" });
  beta = moduleOf(tab.project, "Beta");
  delta = moduleOf(tab.project, "Delta", { status: "completed" });
  gamma = moduleOf(tab.project, "Gamma", { status: "completed", archived_at: ARCHIVED });
  old = moduleOf(tab.project, "Old", { archived_at: ARCHIVED });
  elsewhere = moduleOf(other, "Elsewhere");
  await loadArchivedModules(tab.nerve, tab.app.planning.modules, [moduleRowOf(gamma), moduleRowOf(old)], tab.project);
  tab.nerve.calls.length = 0;
  changed.mockClear();
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

/** The modules nerve lists for the project, not archived, in its order. */
const listed = () => [delta, alpha, beta].map((item) => moduleRowOf(item));
const button = () => host.querySelector<HTMLButtonElement>("button")!;
const list = () => document.querySelector('[role="listbox"]');
const options = () => [...document.querySelectorAll('[role="option"]')].map((option) => option.textContent);
const selected = () =>
  [...document.querySelectorAll('[role="option"][aria-selected="true"]')].map((option) => option.textContent);

async function show(
  props: {
    value?: string[];
    projectId?: string;
    disabled?: boolean;
    issueName?: string;
    buttonVariant?: TButtonVariants;
  } = {}
) {
  const { value = [], projectId = tab.project.id, disabled = false, issueName } = props;
  await act(async () =>
    renderer.render(
      <ModuleDropdown
        projectId={projectId}
        value={value}
        onChange={changed}
        disabled={disabled}
        issueName={issueName}
        buttonVariant={props.buttonVariant ?? "border-with-text"}
      />
    )
  );
}
/** Opens the list, which reads the project's modules: the read, which the caller answers, or not. */
async function open(project: Project = tab.project) {
  const at = tab.nerve.calls.length;
  await act(async () => {
    button().click();
    await until(() => tab.nerve.calls.length === at + 1, "the modules' read");
  });
  const read = tab.nerve.calls[at]!;
  expect([read.method, read.path, read.query]).toEqual(["GET", moduleRequests.list(project), {}]);
  return read;
}
async function answer(read: { answer: (response: Response) => void }, rows = listed()) {
  await act(async () => {
    read.answer(json(200, { data: rows }));
    await vi.advanceTimersByTimeAsync(20);
  });
}
/** Picks the option as a pointer does: Headless UI's option selects on the left button's mousedown. */
async function pick(text: string) {
  const option = [...document.querySelectorAll<HTMLElement>('[role="option"]')].find(
    (item) => item.textContent === text
  );
  expect(option, text).toBeDefined();
  await act(async () => {
    option!.dispatchEvent(new MouseEvent("mousedown", { bubbles: true, cancelable: true, button: 0 }));
    await vi.advanceTimersByTimeAsync(20);
  });
}

it("reads the project's modules as it opens and lists those a work item may join, whatever their status, in nerve's order, the ones it is in where they are", async () => {
  await show({ value: [beta.id] });
  const read = await open();
  expect(list()?.textContent).toContain("common.loading");
  await answer(read);
  expect(options()).toEqual(["Delta", "Alpha", "Beta"]);
  expect(selected()).toEqual(["Beta"]);
  await act(async () => button().click());
  expect(list()).toBeNull();
  await answer(await open(), [...listed(), moduleRowOf(moduleOf(tab.project, "Later"))]);
  expect(options()).toEqual(["Delta", "Alpha", "Beta", "Later"]);
});

it("does not offer a module archived since its project's list was read", async () => {
  await show({ value: [alpha.id] });
  await answer(await open());
  await act(async () => {
    await loadModule(tab.nerve, tab.app.planning.modules, { ...beta, archived_at: ARCHIVED });
  });
  expect(options()).toEqual(["Delta", "Alpha"]);
});

it("lists an archived module it is in after the options, captioned, as one it is in, which a pick takes it out of; no other archived one", async () => {
  await show({ value: [alpha.id, gamma.id] });
  await answer(await open());
  expect(options()).toEqual(["Delta", "Alpha", "Beta", "Gamma, common.archived"]);
  expect(selected()).toEqual(["Alpha", "Gamma, common.archived"]);
  await pick("Gamma, common.archived");
  expect(changed.mock.calls).toEqual([[{ remove: [gamma.id] }]]);
  await show({ value: [alpha.id] });
  expect(options(), "taken out, it is no option").toEqual(["Delta", "Alpha", "Beta"]);
});

it("gives each pick as one addition or one removal, not the whole set, and stays open", async () => {
  await show({ value: [alpha.id, gamma.id] });
  await answer(await open());
  await pick("Beta");
  expect(changed.mock.calls).toEqual([[{ add: [beta.id] }]]);
  expect(list(), "a pick leaves the list open").not.toBeNull();
  await pick("Alpha");
  expect(changed.mock.calls).toEqual([[{ add: [beta.id] }], [{ remove: [alpha.id] }]]);
  expect(list()).not.toBeNull();
});

it("says No module with none, the module's name with one, an archived one captioned, how many with more, and is named by its field, its value and the row's work item", async () => {
  await loadModules(tab.nerve, tab.app.planning.modules, listed(), tab.project);
  await show();
  expect(button().textContent).toBe("common.modules: module.no_module");
  await show({ value: [beta.id] });
  expect(button().textContent).toBe("common.modules: Beta");
  await show({ value: [gamma.id] });
  expect(button().textContent).toBe("common.modules: Gamma, common.archived");
  await show({ value: [alpha.id, gamma.id] });
  expect(button().textContent).toBe("common.modules: module.count 2: Alpha; Gamma, common.archived");
  await show({ value: [alpha.id, gamma.id], issueName: "Fix the login" });
  expect(button().textContent).toBe("common.modules: module.count 2: Alpha; Gamma, common.archived, Fix the login");
  expect(host.querySelectorAll("button button"), "no button in its button").toHaveLength(0);
  expect(host.querySelectorAll("button")).toHaveLength(1);
  await show({ value: [gamma.id], buttonVariant: "border-without-text" });
  expect(button().textContent, "named the same without its text").toBe("common.modules: Gamma, common.archived");
  await show({ value: [alpha.id, gamma.id], buttonVariant: "border-without-text" });
  expect(button().textContent).toBe("common.modules: module.count 2: Alpha; Gamma, common.archived");
});

it("closes as its project changes and opens next on the new project's modules", async () => {
  await show();
  const late = await open();
  await show({ projectId: other.id });
  expect(list(), "the old project's list closes").toBeNull();
  await answer(late);
  const read = await open(other);
  await answer(read, [moduleRowOf(elsewhere)]);
  expect(options()).toEqual(["Elsewhere"]);
});

it("does not open while disabled", async () => {
  await show({ value: [alpha.id], disabled: true });
  expect(button().disabled).toBe(true);
  await act(async () => {
    button().click();
    await vi.advanceTimersByTimeAsync(50);
  });
  expect(list()).toBeNull();
  expect(tab.nerve.calls).toEqual([]);
  expect(changed).not.toHaveBeenCalled();
});

it("has one native trigger, its search takes the focus, and Escape closes it back to the trigger", async () => {
  await show({ value: [alpha.id] });
  await answer(await open());
  const search = document.querySelector<HTMLInputElement>('input[role="combobox"]')!;
  expect(document.activeElement).toBe(search);
  await act(async () => {
    search.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
    await vi.advanceTimersByTimeAsync(20);
  });
  expect(list()).toBeNull();
  expect(document.activeElement).toBe(button());
});
