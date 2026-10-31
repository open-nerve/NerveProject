// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useRef, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { useForm } from "react-hook-form";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { EditorRefApi } from "@nerve/editor";
import { EFileAssetType } from "@nerve/types";
import type { RichTextEditor } from "@/components/editor/rich-text";
import { DescriptionInput } from "@/components/editor/rich-text/description-input";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json } from "@/lib/auth/fake-nerve";
import { IssueMainContent } from "./issue-detail/main-content";
import { PeekOverviewIssueDetails } from "./peek-overview/issue-detail";
import { useIssueOperations } from "./issue-detail/use-issue-operations";
import { IssueDescriptionEditor } from "./issue-modal/components/description-editor";
import type { IssueFormValues } from "./issue-modal/types";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "capabilities" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "capabilities" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-editor-asset", () => ({ useEditorAsset: () => ({}) }));
vi.mock("@/hooks/store/use-workspace", () => ({ useWorkspace: () => tab.app.workspaceRoot }));
vi.mock("@/hooks/store/user", () => ({ useUser: () => ({ data: undefined }) }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/hooks/use-window-size", () => ({ default: () => [1024, 800] }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("./issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("./issue-detail/issue-activity", () => ({ IssueActivity: () => null }));
vi.mock("./issue-detail/parent", () => ({ IssueParentDetail: () => null }));
vi.mock("./issue-detail/reactions", () => ({ IssueReaction: () => null }));
vi.mock("./issue-detail-widgets", () => ({ IssueDetailWidgets: () => null }));
vi.mock("./peek-overview/properties", () => ({ PeekOverviewProperties: () => null }));
vi.mock("./title-input", () => ({ IssueTitleInput: () => null }));
vi.mock("@/components/core/description-versions", () => ({ DescriptionVersionsRoot: () => null }));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: (props: ComponentProps<typeof RichTextEditor>) => {
    rendered = props;
    return <div data-editor />;
  },
}));
let rendered: ComponentProps<typeof RichTextEditor> | undefined;
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
const noop = () => {};
type HostKind = "detail" | "peek" | "create" | "edit" | "draft" | "shared";
function Host({ kind }: { kind: HostKind }) {
  const issueOperations = useIssueOperations("ws", "A", "x");
  const editorRef = useRef<EditorRefApi>(null);
  const submitBtnRef = useRef<HTMLButtonElement>(null);
  const { control } = useForm<IssueFormValues>({ defaultValues: { ...issueDetailOf("x"), project_id: "A" } });
  const context = { workspaceSlug: "ws", projectId: "A", issueId: "x", issueOperations, isArchived: false };
  if (kind === "detail") return <IssueMainContent {...context} isEditable />;
  if (kind === "peek")
    return (
      <PeekOverviewIssueDetails
        {...context}
        editorRef={editorRef}
        disabled={false}
        isSubmitting="saved"
        setTitleStatus={noop}
        setDescriptionStatus={noop}
      />
    );
  if (kind === "shared")
    return (
      <DescriptionInput
        workspaceSlug="ws"
        projectId="A"
        entityId="x"
        fileAssetType={EFileAssetType.ISSUE_DESCRIPTION}
        initialValue="<p>Other module</p>"
        onSubmit={async () => {}}
        setIsSubmitting={noop}
      />
    );
  return (
    <IssueDescriptionEditor
      control={control}
      isDraft={kind === "draft"}
      descriptionHtmlData="<p>Form</p>"
      editorRef={editorRef}
      submitBtnRef={submitBtnRef}
      workspaceSlug="ws"
      projectId={kind === "draft" ? null : "A"}
      handleDescriptionHTMLDataChange={noop}
    />
  );
}
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method === "GET") call.answer(json(200, { data: [], next_cursor: null }));
  };
  rendered = undefined;
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
async function render(kind: HostKind) {
  await act(async () =>
    renderer.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <Host kind={kind} />
      </SWRConfig>
    )
  );
  expect(host.querySelector("[data-editor]")).not.toBeNull();
  return rendered!;
}
it.each(["detail", "peek", "create", "edit", "draft"] as const)(
  "disables images at the actual %s description owner",
  async (kind) => {
    expect((await render(kind)).disabledExtensions ?? []).toContain("image");
  }
);
it("keeps the shared description default available to its other consumers", async () => {
  expect((await render("shared")).disabledExtensions ?? []).not.toContain("image");
});
