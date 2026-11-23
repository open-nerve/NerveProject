/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The intake's create modal in a test of its own (P5b spec 2.1): the dialog as the intake page's header opens it, by
// its "Add work item" button, ModalCore real (Headless UI's dialog, in a portal: a test reads the document), the stores
// real over a fake nerve, given through the store context, under fake timers. The description's editor stands in: its
// ref is null (the editor's chunk not built yet), ready, or built but not ready to discard, as createModal.editor says
// at the moment it is asked; as it is built it may write the empty paragraph a real editor writes. The properties stand
// in as one button each, or are the real ones (createModal.realProperties). @nerve/i18n's text is its key; the toasts
// are kept (fake-toast). Each test ends with no console error: the form logs none, its failures said by toasts.

import { act, useEffect, useRef, useState, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, vi, type Mock } from "vitest";
import type { Call } from "@/lib/auth/fake-nerve";
import { FakeNerve } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import { StoreContext } from "@/lib/store-context";
import { RootStore } from "@/store/root.store";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { loadWorkspaces, workspaceOf } from "@/store/workspace/fake-workspaces";
import type { InboxIssueDescription } from "./issue-description";
import type { InboxIssueProperties } from "./issue-properties";
import { InboxIssueCreateModalRoot } from "./modal";

/**
 * What the modal works with: the session's stores and nerve; the description editor's ref (null: its chunk not built
 * yet; ready; unready: built, not ready to discard), and whether it writes the empty paragraph as it is built; the real
 * properties or their stand-ins. And what it did: navigated, closed (handleModalClose), cleared the editor.
 */
type CreateModal = {
  app: RootStore;
  nerve: FakeNerve;
  editor: "null" | "ready" | "unready";
  emptyParagraph: boolean;
  realProperties: boolean;
  navigate: Mock<(to: string) => void>;
  close: Mock<() => void>;
  clear: Mock<() => void>;
};
const state: CreateModal = vi.hoisted(
  () =>
    ({
      editor: "ready",
      emptyParagraph: false,
      realProperties: false,
      navigate: vi.fn(),
      close: vi.fn(),
      clear: vi.fn(),
    }) as unknown as CreateModal
);

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "intake-create" } }, publicClient: {} }));
vi.mock("@/lib/store-context", async () => ({
  get rootStore() {
    return state.app;
  },
  StoreContext: (await import("react")).createContext<RootStore | undefined>(undefined),
}));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-router")>()),
  useNavigate: () => state.navigate,
  useParams: () => ({ workspaceSlug: "ws", projectId: "p-a" }),
}));
vi.mock("./issue-description", () => ({ InboxIssueDescription: Description }));
vi.mock("./issue-properties", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./issue-properties")>();
  return {
    InboxIssueProperties: (props: ComponentProps<typeof InboxIssueProperties>) =>
      state.realProperties ? <actual.InboxIssueProperties {...props} /> : <Properties {...props} />,
  };
});

/** The description's editor: a field; its ref as createModal.editor says. */
function Description(props: ComponentProps<typeof InboxIssueDescription>) {
  const { data, editorRef, handleData } = props;
  // whether its chunk is built, as the form opens; once built, it is so for good
  const [built] = useState(() => state.editor !== "null");
  const done = useRef(false);
  useEffect(() => {
    if (!built || done.current) return;
    done.current = true;
    Object.assign(editorRef, {
      current: { isEditorReadyToDiscard: () => state.editor === "ready", clearEditor: state.clear },
    });
    if (state.emptyParagraph) handleData("description_html", "<p></p>");
  }, [built, editorRef, handleData]);
  return (
    <textarea
      aria-label="description"
      value={data.description_html}
      onChange={(event) => handleData("description_html", event.target.value)}
    />
  );
}

/** The properties: a button for each, which chooses a value. */
function Properties({ handleData }: ComponentProps<typeof InboxIssueProperties>) {
  const choices = [
    ["priority", () => handleData("priority", "high")],
    ["assignees", () => handleData("assignee_ids", ["u-1"])],
    ["labels", () => handleData("label_ids", ["l-1"])],
    ["due date", () => handleData("target_date", "2026-12-01")],
  ] as const;
  return (
    <>
      {choices.map(([name, choose]) => (
        <button key={name} type="button" onClick={choose}>
          {name}
        </button>
      ))}
    </>
  );
}

export const createModal = state;

const ws = workspaceOf("ws");
const project = projectOf("A", ws.id);
/** The address of a work item in the intake, as the modal navigates to it once nerve has it. */
export const selectedAddress = (id: string) =>
  `/${ws.slug}/projects/${project.id}/intake?currentTab=open&inboxIssueId=${id}`;

let host: HTMLDivElement;
let root: Root;

/** The intake page's header: its "Add work item" button and the modal it opens; handleModalClose closes it. */
function Header() {
  const [open, setOpen] = useState(false);
  return (
    <main>
      <button type="button" onClick={() => setOpen(true)}>
        add_work_item
      </button>
      <InboxIssueCreateModalRoot
        workspaceSlug={ws.slug}
        projectId={project.id}
        modalState={open}
        handleModalClose={() => {
          state.close();
          setOpen(false);
        }}
      />
    </main>
  );
}

/** Sets the modal's session up before each test and takes it down after. */
export function useCreateModalFixture() {
  let errors: { mock: { calls: unknown[][] } };
  beforeEach(async () => {
    errors = vi.spyOn(console, "error");
    vi.useFakeTimers();
    vi.spyOn(Element.prototype, "getAnimations").mockReturnValue([]);
    vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
    globalThis.IS_REACT_ACT_ENVIRONMENT = true;
    Object.assign(state, { editor: "ready", emptyParagraph: false, realProperties: false });
    state.navigate.mockReset();
    state.close.mockReset();
    state.clear.mockReset();
    toasts.length = 0;
    const nerve = new FakeNerve();
    const app = new RootStore(nerve.client());
    Object.assign(state, { app, nerve });
    await loadWorkspaces(nerve, app.workspaceRoot, [ws]);
    await loadProjects(nerve, app.projectRoot.project, ws, [project]);
    nerve.calls.length = 0;
    host = document.createElement("div");
    document.body.append(host);
    root = createRoot(host);
  });
  afterEach(async () => {
    try {
      await act(async () => root.unmount());
    } finally {
      // what a failed test left in the document, a dialog's portal, is no later test's
      document.body.replaceChildren();
      createModal.app.dispose();
      const logged = errors.mock.calls.map((call) => String(call[0]));
      vi.restoreAllMocks();
      vi.unstubAllGlobals();
      vi.useRealTimers();
      expect(logged).toEqual([]);
    }
  });
}

/** Moves the fake time on, so that what the page does next is done: a dialog's transition among it. */
export const settled = () => act(async () => void (await vi.advanceTimersByTimeAsync(50)));

/** The open dialogs, outermost first. */
export const dialogs = () => [...document.querySelectorAll<HTMLElement>('[role="dialog"][aria-modal="true"]')];

/** A dialog's name: the text of what names it (aria-labelledby). */
export const nameOf = (dialog: HTMLElement) =>
  document.getElementById(dialog.getAttribute("aria-labelledby") ?? "")?.textContent ?? null;

/** The button within scope whose text or label is name. */
export const button = (name: string, scope: ParentNode = document) =>
  [...scope.querySelectorAll<HTMLButtonElement>("button")].find(
    (item) => item.textContent?.trim() === name || item.getAttribute("aria-label") === name
  );

/** The form's name field. */
export const nameField = () => document.querySelector<HTMLInputElement>("input[name=name]")!;

/** Opens the modal from the header's button, which has the focus. */
export async function openModal() {
  await act(async () =>
    root.render(
      <StoreContext.Provider value={createModal.app}>
        <Header />
      </StoreContext.Provider>
    )
  );
  await act(async () => button("add_work_item")!.focus());
  await click("add_work_item");
  expect(dialogs()).toHaveLength(1);
}

/** Takes the page down while the modal is open: Back or Forward leaves the intake page. */
export const leavePage = () => act(async () => root.render(<></>));

/** Presses the button within scope named name. */
export async function click(name: string, scope?: ParentNode) {
  await act(async () => button(name, scope)!.click());
  await settled();
}

/** Types value into a field, as its whole value. */
export async function type(field: HTMLInputElement | HTMLTextAreaElement, value: string) {
  await act(async () => {
    const proto = field.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value")!.set!.call(field, value);
    field.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

/** Presses Escape where the focus is (or on target); composing, as an input method's composition ends. */
export async function escape(target = document.activeElement as HTMLElement, composing = false) {
  await act(async () =>
    target.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", isComposing: composing, bubbles: true, cancelable: true })
    )
  );
  await settled();
}

/** Presses the pointer on the outer dialog's backdrop: down, then up (Headless UI's click outside). */
export async function pressBackdrop() {
  const backdrop = dialogs()[0]!.firstElementChild!;
  await act(async () => {
    backdrop.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true, cancelable: true }));
    backdrop.dispatchEvent(new MouseEvent("pointerup", { bubbles: true, cancelable: true }));
  });
  await settled();
}

/** Submits the form as its button does (requestSubmit: the browser's own validation first). */
export async function submit() {
  await act(async () => nameField().form!.requestSubmit());
  await settled();
}

/** The submissions nerve has been sent. */
export const submissions = () => createModal.nerve.calls.filter((call) => call.method === "POST");

/** The submission, once it is out. */
export async function submission(): Promise<Call> {
  await act(async () => until(() => submissions().length === 1, "the submission"));
  return submissions()[0]!;
}

/** nerve answers call with response, and the form follows it. */
export async function answer(call: Call, response: Response) {
  await act(async () => call.answer(response));
  await settled();
}
