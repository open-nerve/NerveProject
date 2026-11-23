// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { MemoryRouter, Route, Routes, useLocation, useNavigationType, useSearchParams } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { acme, intakeIssueOf, intakeTab, loadList, open, pageOf } from "@/store/issue/intake/fake-intake";
import { IntakePageStateProvider } from "../../page-state";
import { IntakeNavigation, useIntakeArrowKeys } from "./navigation";

// The previous and next work items of the intake page's list (P5a spec 2.7; HO/M3-closeout.md:65): two buttons, named
// in the page's language, and ↑ and ↓, that select the row before or after the selected one in the list the page
// shows (the tab's filters), the first after the last and the last before the first; the user's own move, so a push.
// The keys step only while the focus is outside the editors and the fields (a name, a description, a comment, a
// search), and never a key a control has handled, one pressed with a modifier, or one inside a dialog. The list is the
// intake store's own, as nerve gave it. @nerve/i18n's text is its key.

const held = vi.hoisted(() => ({ where: undefined as { search: string; type: string } | undefined }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@/hooks/store/use-intake", () => ({ useIntake: () => tab.store }));

function Page() {
  const [params] = useSearchParams();
  const location = useLocation();
  held.where = { search: location.search, type: useNavigationType() };
  const selected = params.get("inboxIssueId") ?? undefined;
  useIntakeArrowKeys("p-web", selected);
  return (
    <>
      <IntakeNavigation projectId="p-web" issueId={selected ?? ""} />
      <input aria-label="a field" />
      <textarea aria-label="a name" />
      <div aria-label="an editor" contentEditable suppressContentEditableWarning>
        <p>text</p>
      </div>
      <div role="dialog" aria-label="a dialog">
        <button type="button">in the dialog</button>
      </div>
    </>
  );
}

let tab: Awaited<ReturnType<typeof intakeTab>>;
let host: HTMLDivElement, root: Root;
globalThis.IS_REACT_ACT_ENVIRONMENT = true;
beforeEach(async () => {
  vi.useFakeTimers();
  tab = await intakeTab();
  const rows = ["One", "Two", "Three"].map((name, at) => intakeIssueOf(name, { sequence_id: at + 1 }));
  await loadList(tab.nerve, tab.store, "p-web", open, pageOf(rows));
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});

const render = (selected = "i-Two") =>
  act(async () =>
    root.render(
      <MemoryRouter initialEntries={[`/${acme.slug}/projects/p-web/intake?currentTab=open&inboxIssueId=${selected}`]}>
        <Routes>
          <Route
            path="/:workspaceSlug/projects/:projectId/intake"
            element={
              <IntakePageStateProvider>
                <Page />
              </IntakePageStateProvider>
            }
          />
        </Routes>
      </MemoryRouter>
    )
  );
const selected = () => new URLSearchParams(held.where!.search).get("inboxIssueId");
const named = (name: string) => host.querySelector<HTMLElement>(`[aria-label="${name}"]`)!;
const press = (name: string) => act(async () => named(name).click());
/** A control's own handling of a key, which says the key is taken. */
const handled = (event: Event) => event.preventDefault();
/** A key pressed where the focus is (the page's body unless at says), as the browser sends it. */
function key(name: string, at: Element = document.body, init: KeyboardEventInit = {}) {
  return act(async () => {
    if (at instanceof HTMLElement) at.focus();
    at.dispatchEvent(new KeyboardEvent("keydown", { key: name, bubbles: true, cancelable: true, ...init }));
  });
}

describe("IntakeNavigation", () => {
  it("is two buttons, named the previous and the next work item in the page's language", async () => {
    await render();
    const buttons = [...host.querySelectorAll("button")].filter((button) => !button.closest('[role="dialog"]'));
    expect(buttons.map((button) => [button.getAttribute("type"), button.getAttribute("aria-label")])).toEqual([
      ["button", "inbox_issue.a11y.previous"],
      ["button", "inbox_issue.a11y.next"],
    ]);
  });

  it("selects the next row and the previous one in the list shown, round its ends, each a push", async () => {
    await render();
    await press("inbox_issue.a11y.next");
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Three", type: "PUSH" });
    await press("inbox_issue.a11y.next");
    expect(selected()).toBe("i-One");
    await press("inbox_issue.a11y.previous");
    expect(selected()).toBe("i-Three");
    await press("inbox_issue.a11y.previous");
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Two", type: "PUSH" });
  });

  it.each([
    ["next", "i-One"],
    ["previous", "i-Three"],
  ])("selects, as the %s of a work item the list does not show, its row at that end", async (direction, row) => {
    await render("i-Gone");
    await press(`inbox_issue.a11y.${direction}`);
    expect(selected()).toBe(row);
  });
});

describe("useIntakeArrowKeys", () => {
  it("steps the selection with ↓ and ↑ while the focus is outside the editors and the fields", async () => {
    await render();
    await key("ArrowDown");
    expect(held.where).toEqual({ search: "?currentTab=open&inboxIssueId=i-Three", type: "PUSH" });
    await key("ArrowUp");
    await key("ArrowUp");
    expect(selected()).toBe("i-One");
  });

  it.each(["a field", "a name", "an editor"])("steps nothing while the focus is in %s", async (where) => {
    await render();
    await key("ArrowDown", named(where));
    expect(selected()).toBe("i-Two");
    await key("ArrowUp", named(where).querySelector("p") ?? named(where));
    expect(selected()).toBe("i-Two");
  });

  it("steps nothing for a key a control handled, one pressed with a modifier, or one inside a dialog", async () => {
    await render();
    document.body.addEventListener("keydown", handled, { capture: true });
    await key("ArrowDown");
    document.body.removeEventListener("keydown", handled, { capture: true });
    await key("ArrowDown", document.body, { shiftKey: true });
    await key("ArrowDown", document.body, { metaKey: true });
    await key("ArrowDown", host.querySelector('[role="dialog"] button')!);
    expect(selected()).toBe("i-Two");
  });

  it("steps nothing with nothing selected", async () => {
    await act(async () =>
      root.render(
        <MemoryRouter initialEntries={["/acme/projects/p-web/intake?currentTab=open"]}>
          <Routes>
            <Route
              path="/:workspaceSlug/projects/:projectId/intake"
              element={
                <IntakePageStateProvider>
                  <Page />
                </IntakePageStateProvider>
              }
            />
          </Routes>
        </MemoryRouter>
      )
    );
    await key("ArrowDown");
    expect(selected()).toBeNull();
  });
});
