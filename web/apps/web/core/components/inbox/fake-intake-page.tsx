/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The intake page in a test of its triage (P5a spec 2.6–2.8): the page at /acme/projects/p-web/intake, under its
// layout's state, in a session whose caller (u-me) is an admin of p-web, its stores, SWR, router and HTTP real (a fake
// nerve; the test's SWR cache), its stores given through the store context. A test file mocks the session's modules
// and the selection's read (useIssueOperations' fetch, which page.reads holds until the test answers it), and draws the
// detail's body as a stand-in; the headers, the sidebar's list, the modals and the action model are real. @nerve/i18n's
// text is its key.

import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation, useNavigate, useNavigationType } from "react-router";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, vi } from "vitest";
import type { IntakeIssue, IssueDetail, Project } from "@nerve/api-client";
import type { Call } from "@/lib/auth/fake-nerve";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { acme, intakeTab, listPath, pageOf } from "@/store/issue/intake/fake-intake";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { IntakePageStateProvider } from "./page-state";
import { InboxIssueRoot } from "./root";

/** A read of the selected work item (useIssueOperations' fetch), held until the test answers it. */
export type Read = { id: string; answer: (detail: IssueDetail) => void };

/** The page under test, and what it has done so far. */
export const page = {
  tab: undefined as unknown as Awaited<ReturnType<typeof intakeTab>>,
  reads: [] as Read[],
  where: undefined as { search: string; type: string } | undefined,
  navigate: undefined as unknown as ReturnType<typeof useNavigate>,
  host: undefined as unknown as HTMLDivElement,
  root: undefined as unknown as Root,
};

/** The page's address, and the one before it in the history, which Back goes to. */
const BEFORE = "/acme/projects/p-web/issues";
const INTAKE = "/acme/projects/p-web/intake";

function Where() {
  const location = useLocation();
  page.where = { search: location.search, type: useNavigationType() };
  page.navigate = useNavigate();
  return null;
}

/**
 * Sets the page's session up before each test and takes it down after: fake timers, the browser's stand-ins. Each test
 * ends with no console error (a React warning among them): the page logs none, its failures said by toasts; the check
 * comes once the page is down, whatever the test did.
 */
export function useIntakePageFixture(project: Partial<Project> = {}) {
  let errors: { mock: { calls: unknown[][] } };
  beforeEach(async () => {
    errors = vi.spyOn(console, "error");
    vi.useFakeTimers();
    vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
    for (const name of ["ResizeObserver", "IntersectionObserver"])
      vi.stubGlobal(
        name,
        class {
          observe() {}
          unobserve() {}
          disconnect() {}
        }
      );
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    toasts.length = 0;
    page.reads = [];
    page.tab = await intakeTab();
    await loadProjects(page.tab.nerve, page.tab.app.projectRoot.project, acme, [
      projectOf("WEB", acme.id, { intake_view: true, member_role: 20, ...project }),
    ]);
    page.tab.nerve.calls.length = 0;
    page.host = document.createElement("div");
    document.body.append(page.host);
    page.root = createRoot(page.host);
  });
  afterEach(async () => {
    try {
      await act(async () => page.root.unmount());
    } finally {
      // what a failed test left in the document, a dialog's portal, is no later test's
      document.body.replaceChildren();
      page.tab.app.dispose();
      const logged = errors.mock.calls.map((call) => String(call[0]));
      vi.restoreAllMocks();
      vi.unstubAllGlobals();
      vi.useRealTimers();
      expect(logged).toEqual([]);
    }
  });
}

/** Moves the fake time on, so that what the page does next is done. */
export const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));

/** The calls nerve has been sent to path, by method. */
export const callsTo = (method: string, path: string) =>
  page.tab.nerve.calls.filter((call) => call.method === method && call.path === path);

/** The n-th call of method to path, once it is out. */
export async function callTo(method: string, path: string, n = 1): Promise<Call> {
  await act(async () => until(() => callsTo(method, path).length >= n, `${method} ${path} (${n})`));
  return callsTo(method, path)[n - 1]!;
}

/** The selection's n-th read of id, once it is out. */
async function readOf(id: string, n = 1): Promise<Read> {
  await act(async () => until(() => page.reads.filter((read) => read.id === id).length >= n, `read ${n} of ${id}`));
  return page.reads.filter((read) => read.id === id)[n - 1]!;
}

/** The selection's n-th read of id answers detail, and the page shows it. */
export async function shown(detail: IssueDetail, n = 1) {
  const read = await readOf(detail.id, n);
  await act(async () => read.answer(detail));
  await settled();
}

/** nerve answers the call (once it is out) with response, and the page follows it. */
export async function answer(call: Call | Promise<Call>, response: Response) {
  const out = await call;
  await act(async () => out.answer(response));
  await settled();
}

/**
 * Opens the intake page on its open tab (its address before it the project's work items), its list read as rows,
 * the first of them selected by the page and shown as first.
 */
export async function opened(rows: IntakeIssue[], first: IssueDetail) {
  await act(async () =>
    page.root.render(
      <StoreContext.Provider value={page.tab.app}>
        <SWRConfig value={{ provider: () => new Map() }}>
          <MemoryRouter initialEntries={[BEFORE, `${INTAKE}?currentTab=open`]} initialIndex={1}>
            <Routes>
              <Route path="/:workspaceSlug/projects/:projectId/issues" element={<Where />} />
              <Route
                path="/:workspaceSlug/projects/:projectId/intake"
                element={
                  <IntakePageStateProvider>
                    <Where />
                    <InboxIssueRoot workspaceSlug="acme" projectId="p-web" />
                  </IntakePageStateProvider>
                }
              />
            </Routes>
          </MemoryRouter>
        </SWRConfig>
      </StoreContext.Provider>
    )
  );
  await answer(callTo("GET", listPath("p-web")), json(200, pageOf(rows)));
  await shown(first);
}

/** The work item the address selects. */
export const selected = () => new URLSearchParams(page.where?.search).get("inboxIssueId");

/** The detail's region and the list's. */
export const detailRegion = () => page.host.querySelector<HTMLElement>('section[aria-label="inbox_issue.a11y.detail"]');
export const listRegion = () => page.host.querySelector<HTMLElement>('section[aria-label="inbox_issue.a11y.list"]');
/** The row of the work item id in the list, its link. */
export const rowOf = (id: string) => listRegion()?.querySelector<HTMLElement>(`a[href$="inboxIssueId=${id}"]`) ?? null;

/** The button, anywhere in the document (a dialog's are outside the page), whose text is text. */
export const buttonSaying = (text: string, within: ParentNode = document) =>
  [...within.querySelectorAll<HTMLButtonElement>("button")].find((button) => button.textContent?.trim() === text);
/** The button of the wide header's menu, or of the narrow header's (the second in the detail, as the page draws them). */
export const menuButton = (header: "wide" | "narrow") =>
  detailRegion()?.querySelectorAll<HTMLButtonElement>('button[aria-label="common.options"]')[
    header === "wide" ? 0 : 1
  ] ?? null;

/** The keyboard's press of a button: it has the focus first, as Tab gives it, then Enter clicks it. */
export const pressByKeyboard = (button: HTMLElement | undefined | null) =>
  act(async () => {
    expect(button).toBeTruthy();
    button!.focus();
    button!.click();
  });
/** The pointer's press of a button where a press does not focus it (WebKit's): the focus stays where it was. */
export const pressByPointer = (button: HTMLElement | undefined | null) =>
  act(async () => {
    expect(button).toBeTruthy();
    button!.click();
  });

/**
 * Opens the menu of the header (its button named common.options) and chooses the item saying text by the pointer: the
 * item takes the focus as the pointer presses it (Headless UI), then the click chooses it.
 */
export async function chooseFromMenu(header: "wide" | "narrow", text: string) {
  const menu = menuButton(header);
  expect(menu).toBeTruthy();
  await act(async () => menu!.click());
  await settled();
  const item = buttonSaying(text, document.querySelector('[role="menu"]') ?? document);
  expect(item).toBeTruthy();
  await act(async () => {
    item!.focus();
    item!.click();
  });
  await settled();
}

/** The dialog open, by its name. */
export const dialogNamed = (name: string) =>
  [...document.querySelectorAll<HTMLElement>('[role="dialog"]')].find(
    (dialog) => document.getElementById(dialog.getAttribute("aria-labelledby") ?? "")?.textContent === name
  );

/** Escape, on what has the focus, as a browser gives it. */
export const escape = () =>
  act(async () =>
    (document.activeElement ?? document.body).dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
