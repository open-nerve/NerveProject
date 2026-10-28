// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act, useEffect, useRef } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { issueOf } from "@/store/issue/fake-issues";
import type { IssueDescriptionEditor } from "./components/description-editor";
import { CreateUpdateIssueModalBase } from "./base";
const boundary = vi.hoisted(() => ({ fetch: vi.fn(), create: vi.fn(), update: vi.fn() }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ({ workspaceSlug: "ws", projectId: "A" }) }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/context/use-issue-modal", () => ({
  useIssueModal: () => ({ allowedProjectIds: ["A", "B"], selectedParentIssue: null, setSelectedParentIssue: () => {} }),
}));
vi.mock("@/hooks/store/use-issues", () => ({ useIssues: () => ({ issues: {} }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({
  useIssueDetail: () => ({ fetchIssue: boundary.fetch, issue: { getIssueById: () => undefined } }),
}));
vi.mock("@/hooks/store/use-project", () => ({
  useProject: () => ({ getProjectByIdentifier: () => undefined, getProjectById: () => undefined }),
}));
vi.mock("@/hooks/store/use-project-state", () => ({ useProjectState: () => ({ getStateById: () => undefined }) }));
vi.mock("@/hooks/store/workspace-draft", () => ({ useWorkspaceDraftIssues: () => ({ moveIssue: vi.fn() }) }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => EIssuesStoreType.PROJECT }));
vi.mock("@/hooks/use-issues-actions", () => ({
  useIssuesActions: () => ({ createIssue: boundary.create, updateIssue: boundary.update }),
}));
vi.mock("@/components/issues/create-issue-toast-action-items", () => ({ CreateIssueToastActionItems: () => null }));
vi.mock("./components/default-properties", () => ({ IssueDefaultProperties: () => null }));
vi.mock("./components/parent-tag", () => ({ IssueParentTag: () => null }));
vi.mock("@/components/dropdowns/project/dropdown", () => ({
  ProjectDropdown: ({
    disabled,
    value,
    onChange,
  }: {
    disabled?: boolean;
    value: string;
    onChange: (value: string) => void;
  }) => (
    <select aria-label="project" disabled={disabled} value={value} onChange={(e) => onChange(e.target.value)}>
      <option>A</option>
      <option>B</option>
    </select>
  ),
}));
vi.mock("./components/description-editor", () => ({
  IssueDescriptionEditor: (props: React.ComponentProps<typeof IssueDescriptionEditor>) => {
    const { descriptionHtmlData, handleDescriptionHTMLDataChange } = props;
    const updateHtml = useRef(handleDescriptionHTMLDataChange);
    updateHtml.current = handleDescriptionHTMLDataChange;
    useEffect(() => {
      if (descriptionHtmlData !== undefined) updateHtml.current(descriptionHtmlData);
    }, [descriptionHtmlData]);
    useEffect(() => {
      Object.assign(props.editorRef, { current: { isEditorReadyToDiscard: () => true, clearEditor: () => {} } });
    }, [props.editorRef]);
    return <div data-html={props.descriptionHtmlData} />;
  },
}));
vi.mock("@nerve/ui", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@nerve/ui")>()),
  ModalCore: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
class NoResize {
  observe() {}
  unobserve() {}
  disconnect() {}
}
let root: Root;
let container: HTMLElement;
beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal("ResizeObserver", NoResize);
  boundary.fetch.mockReset().mockResolvedValue({ ...issueOf("source"), description_html: "<p>source</p>" });
  boundary.create.mockReset().mockResolvedValue(issueOf("created"));
  boundary.update.mockReset();
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});
afterEach(async () => {
  await act(async () => root.unmount());
  container.remove();
});
it("reads the independent clone source, locks the real project Controller and submits source HTML without local identity", async () => {
  const data = { name: "Cloned title", project_id: "A", state_id: "state" };
  const original = { ...data };
  const props = { data, sourceIssueId: "source", isOpen: true, onClose: vi.fn(), withDraftIssueWrapper: false };
  await act(async () => root.render(<CreateUpdateIssueModalBase {...props} />));
  await act(async () =>
    container.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }))
  );
  expect(boundary.create).toHaveBeenCalledTimes(1);
  const [project, payload] = boundary.create.mock.calls[0]!;
  expect(project).toBe("A");
  expect(payload).toMatchObject({ name: data.name, description_html: "<p>source</p>" });
  expect(payload).not.toHaveProperty("project_id");
  expect(payload).not.toHaveProperty("sourceIssueId");
  expect(payload).not.toHaveProperty("id");
  for (const field of [
    "sequence_id",
    "created_at",
    "updated_at",
    "cycle_id",
    "module_ids",
    "is_draft",
    "pending",
    "tempId",
  ]) {
    expect(payload).not.toHaveProperty(field);
  }
  expect(data).toEqual(original);
  expect(boundary.update).not.toHaveBeenCalled();
  expect(boundary.fetch).toHaveBeenCalledWith("ws", "A", "source");
  expect(container.querySelector<HTMLSelectElement>("select")!.disabled).toBe(true);
});
it.each([
  { data: { name: "new", project_id: "A" }, disabled: false },
  { data: { name: "edit", project_id: "A", id: "existing" }, disabled: true },
  { data: { name: "forced", project_id: "A" }, isProjectSelectionDisabled: true, disabled: true },
])("project selection follows actual FormRoot disabled state ($disabled)", async ({ disabled, ...props }) => {
  await act(async () =>
    root.render(<CreateUpdateIssueModalBase {...props} isOpen onClose={() => {}} withDraftIssueWrapper={false} />)
  );
  expect(container.querySelector<HTMLSelectElement>("select")!.disabled).toBe(disabled);
});
