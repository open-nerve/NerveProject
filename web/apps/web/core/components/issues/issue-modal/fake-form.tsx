/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import {
  act,
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
  type ComponentProps,
  type PropsWithChildren,
} from "react";
import { createRoot } from "react-dom/client";
import { useFormContext } from "react-hook-form";
import { afterEach, beforeEach, vi, type Mock } from "vitest";
import type { Draft } from "@nerve/api-client";
import { EIssuesStoreType } from "@nerve/types";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { issueDetailOf, issueStoreTab, draftOf, requestedIssue } from "@/store/issue/fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { toasts } from "@/lib/fake-toast";
import { CreateUpdateIssueModalBase } from "./base";
import { IssueFormRoot, type IssueFormProps } from "./form";
import type { IssueFormValues } from "./types";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "form-login" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws", projectId: "A" }) }));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({ allowedProjectIds: ["A"], selectedParentIssue: null, setSelectedParentIssue: () => {} }),
}));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => tab.app.state }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/workspace-draft", () => ({ useWorkspaceDraftIssues: () => tab.app.issue.workspaceDraftIssues }));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/hooks/store/user", () => ({ useUserProfile: () => ({ data: { start_of_the_week: 1 } }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.PROJECT }));
vi.mock("@/hooks/use-issues-actions", () => ({
  useIssuesActions: () => ({ createIssue: create, updateIssue: update }),
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
// Adjacent properties are covered by I1/I2. Keep the real field Controllers here.
vi.mock("./components/default-properties", async () => ({
  IssueDefaultProperties: (await import("./components/date-fields")).IssueDateFields,
}));
vi.mock("@nerve/ui", async (original) => ({
  ...(await original<typeof import("@nerve/ui")>()),
  ModalCore: ({ children, isOpen }: PropsWithChildren<{ isOpen: boolean }>) =>
    isOpen ? <div role="dialog">{children}</div> : null,
  ModalTitle: ({ children }: PropsWithChildren) => <h3>{children}</h3>,
}));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: forwardRef((props: ComponentProps<typeof RichTextEditor>, ref) => {
    const [value, setValue] = useState(props.initialValue);
    const {
      formState: { isDirty },
      getValues,
    } = useFormContext<IssueFormValues>();
    useImperativeHandle(ref, () => ({ isEditorReadyToDiscard: () => true, clearEditor: () => setValue("") }));
    useEffect(() => {
      if (props.value != null) setValue(props.value);
    }, [props.value]);
    const migrate = () => {
      const html = value.replace("<p>", '<p data-id="migrated">');
      setValue(html);
      props.onChange?.({}, html, { isMigrationUpdate: true });
    };
    return (
      <>
        <textarea
          aria-label="description"
          value={value}
          onChange={(event) => {
            setValue(event.target.value);
            props.onChange?.({}, event.target.value);
          }}
        />
        <button type="button" onClick={migrate}>
          migrate description
        </button>
        <output data-form-dirty>{String(isDirty)}</output>
        <output data-form-html>{getValues("description_html")}</output>
      </>
    );
  }),
}));

export let tab: Awaited<ReturnType<typeof issueStoreTab>>;
export let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
export const close: Mock<() => void> = vi.fn();
export const confirmed: Mock<() => Promise<void>> = vi.fn();
export const create =
  vi.fn<
    (
      project: string,
      body: Parameters<typeof tab.app.issue.projectIssues.createIssue>[2]
    ) => ReturnType<typeof tab.app.issue.projectIssues.createIssue>
  >();
const update =
  vi.fn<
    (project: string, id: string, body: Parameters<typeof tab.app.issue.projectIssues.updateIssue>[3]) => Promise<void>
  >();
export const original = () => ({ ...issueDetailOf("x"), name: "Existing", description_html: "<p>Original</p>" });
export const writes = () => tab.nerve.calls.filter((call) => call.method !== "GET");
const button = (name: string) =>
  [...host.querySelectorAll("button")].find((node) => node.textContent?.trim() === name)!;
export const dirty = () => host.querySelector("[data-form-dirty]")?.textContent;
export const html = () => host.querySelector("[data-form-html]")?.textContent;
export const submit = () =>
  act(async () => host.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true })));
export const click = (name: string) => act(async () => button(name).click());
export function useFormFixture() {
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
    tab = await issueStoreTab();
    tab.app.issue.issues.acceptRead(original(), tab.app.issue.captureRead());
    tab.nerve.calls.length = 0;
    close.mockReset();
    confirmed.mockReset();
    toasts.length = 0;
    create
      .mockReset()
      .mockImplementation((project, body) => tab.app.issue.projectIssues.createIssue("ws", project, body));
    update
      .mockReset()
      .mockImplementation((project, id, body) => tab.app.issue.projectIssues.updateIssue("ws", project, id, body));
    host = document.createElement("div");
    document.body.append(host);
    root = createRoot(host);
  });
  afterEach(async () => {
    await act(async () => root.unmount());
    host.remove();
    tab.app.dispose();
    vi.unstubAllGlobals();
    vi.useRealTimers();
  });
}
export async function mountExisting(kind: "issue" | "draft") {
  let draft: Draft | undefined;
  if (kind === "draft") {
    draft = { ...draftOf("d"), payload: { name: "Existing", description_html: "<p>Original</p>" } };
    const read = await requestedIssue(tab.nerve, () => tab.app.issue.workspaceDraftIssues.fetchIssues("ws"), [
      "GET",
      "/api/v0/workspaces/ws/drafts",
    ]);
    read.call.answer(json(200, { data: [draft], total: 1, next_cursor: null }));
    await read.promise;
    tab.nerve.calls.length = 0;
  }
  await act(async () =>
    root.render(
      kind === "draft" ? (
        <CreateUpdateIssueModalBase isOpen isDraft draft={draft} onClose={close} onSubmit={confirmed} />
      ) : (
        <CreateUpdateIssueModalBase
          isOpen
          data={original()}
          fetchIssueDetails={false}
          onClose={close}
          onSubmit={confirmed}
        />
      )
    )
  );
}
export const mountNew = () =>
  act(async () =>
    root.render(
      <CreateUpdateIssueModalBase
        isOpen
        data={{ name: "New", project_id: "A", description_html: "<p>New</p>" }}
        fetchIssueDetails={false}
        onClose={close}
        onSubmit={confirmed}
      />
    )
  );
function Form({ overrides }: { overrides: Partial<IssueFormProps> }) {
  const title = useRef<HTMLInputElement>(null);
  return (
    <IssueFormRoot
      issueTitleRef={title}
      isCreateMoreToggleEnabled={false}
      onCreateMoreToggleChange={() => {}}
      onAssetUpload={() => {}}
      onClose={close}
      onSubmit={async () => {}}
      workspaceSlug="ws"
      projectId={null}
      isDraft={false}
      isDuplicateModalOpen={false}
      handleDuplicateIssueModal={() => {}}
      data={{ name: "", description_html: "<p></p>", project_id: null }}
      {...overrides}
    />
  );
}
export const mountForm = (overrides: Partial<IssueFormProps> = {}) =>
  act(async () => root.render(<Form overrides={overrides} />));
export const type = (selector: string, value: string) =>
  act(async () => {
    const input = host.querySelector<HTMLInputElement | HTMLTextAreaElement>(selector)!;
    const proto = input.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(proto, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
export function recordFocus() {
  const values: { tag: string; invalid: string | null; description: string }[] = [];
  host.addEventListener("focusin", (event) => {
    const element = event.target as HTMLElement;
    values.push({
      tag: element.tagName,
      invalid: element.getAttribute("aria-invalid"),
      description: (element.getAttribute("aria-describedby")?.split(/\s+/) ?? [])
        .map((id) => document.getElementById(id)?.textContent ?? "")
        .join(" "),
    });
  });
  return values;
}
