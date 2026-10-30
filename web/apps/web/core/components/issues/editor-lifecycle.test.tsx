// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, forwardRef, useImperativeHandle, useState, type ComponentProps } from "react";
import { createRoot } from "react-dom/client";
import { SWRConfig } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { toasts } from "@/lib/fake-toast";
import type { RichTextEditor } from "@/components/editor/rich-text";
import type { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { IssueMainContent } from "./issue-detail/main-content";
import { useIssueOperations } from "./issue-detail/use-issue-operations";

vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "editors" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "editors" }) }));
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
vi.mock("@/components/core/description-versions", () => ({
  DescriptionVersionsRoot: (props: ComponentProps<typeof DescriptionVersionsRoot>) => (
    <button onClick={() => void props.handleRestore("restored").catch(() => undefined)}>restore version</button>
  ),
}));
vi.mock("@/components/editor/rich-text", () => ({
  RichTextEditor: forwardRef((props: ComponentProps<typeof RichTextEditor>, ref) => {
    const [value, setValue] = useState(props.initialValue);
    // Match the editor contract: initialValue initializes; imperative replacement does not emit a change.
    useImperativeHandle(ref, () => ({ setEditorValue: (next: string) => setValue(next) }));
    return (
      <textarea
        aria-label="description"
        value={value}
        onChange={(event) => {
          setValue(event.target.value);
          props.onChange?.({}, event.target.value);
        }}
      />
    );
  }),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let renderer: ReturnType<typeof createRoot>;
function Host() {
  const issueOperations = useIssueOperations("ws", "A", "x");
  return (
    <IssueMainContent
      workspaceSlug="ws"
      projectId="A"
      issueId="x"
      isEditable
      isArchived={false}
      issueOperations={issueOperations}
    />
  );
}
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
  tab.app.issue.issues.acceptRead(issueDetailOf("x"), tab.app.issue.captureRead());
  tab.nerve.respond = (call) => {
    if (call.method === "GET") call.answer(json(200, { data: [], next_cursor: null }));
  };
  toasts.length = 0;
  host = document.createElement("div");
  document.body.append(host);
  renderer = createRoot(host);
  await act(async () =>
    renderer.render(
      <SWRConfig value={{ provider: () => new Map() }}>
        <Host />
      </SWRConfig>
    )
  );
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  host.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
const writes = () => tab.nerve.calls.filter((call) => call.method === "PATCH");
const edit = (index: number, value: string) =>
  act(async () => {
    const input = host.querySelectorAll("textarea")[index]!;
    Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, "value")!.set!.call(input, value);
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
const leave = () => act(async () => renderer.render(null));
const answer = (index: number, html: string) =>
  act(async () =>
    writes()[index]!.answer(
      json(200, {
        ...issueDetailOf("x"),
        description_html: html,
        updated_at: "2026-10-30T10:00:00Z",
      })
    )
  );
for (const index of [0, 1]) {
  it(`flushes the last unattempted ${index === 0 ? "title" : "description"} through the real command when the entire host leaves`, async () => {
    await edit(index, "last edit");
    expect(writes()).toHaveLength(0);
    await leave();
    await act(async () => until(() => writes().length === 1, "unmount PATCH"));
    expect(writes()[0]!.body).toMatchObject(index === 0 ? { name: "last edit" } : { description_html: "last edit" });
    await answer(0, "last edit");
    expect(toasts).toHaveLength(0);
  });
  it(`does not revive an unattempted ${index === 0 ? "title" : "description"} after permission loss and recovery`, async () => {
    await edit(index, "obsolete");
    tab.nerve.respond = undefined;
    await act(async () => {
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
      await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    });
    await leave();
    await act(async () => vi.advanceTimersByTimeAsync(2000));
    expect(writes()).toHaveLength(0);
    expect(toasts).toHaveLength(0);
  });
}
for (const prior of ["debouncing", "refused", "in-flight"] as const) {
  it(`restoring a version supersedes ${prior} description input without a later retry or flush overwriting it`, async () => {
    await edit(1, "obsolete edit");
    if (prior !== "debouncing") {
      await act(async () => vi.advanceTimersByTimeAsync(1500));
      await act(async () => until(() => writes().length === 1, "autosave PATCH"));
      if (prior === "refused") {
        await act(async () => writes()[0]!.answer(problem(403, "forbidden")));
        expect(host.querySelector('[role="alert"]')).not.toBeNull();
      }
    }
    await act(async () =>
      [...host.querySelectorAll("button")].find((button) => button.textContent === "restore version")!.click()
    );
    if (prior === "in-flight") await answer(0, "obsolete edit");
    const count = prior === "debouncing" ? 1 : 2;
    await act(async () => until(() => writes().length === count, "restore PATCH"));
    await act(async () => vi.advanceTimersByTimeAsync(2000));
    expect(writes()).toHaveLength(count);
    await answer(count - 1, "restored");
    await act(async () => vi.advanceTimersByTimeAsync(2000));
    expect(host.querySelectorAll("textarea")[1]!.value).toBe("restored");
    expect(host.querySelector('[role="alert"]')).toBeNull();
    expect(writes()).toHaveLength(count);
    await leave();
    expect(writes()).toHaveLength(count);
  });
}

it("a refused version restoration retains unsaved input for an explicit retry without an automatic write", async () => {
  await edit(1, "keep this input");
  await act(async () =>
    [...host.querySelectorAll("button")].find((button) => button.textContent === "restore version")!.click()
  );
  await act(async () => until(() => writes().length === 1, "restore PATCH"));
  await act(async () => writes()[0]!.answer(problem(403, "forbidden")));
  await act(async () => vi.advanceTimersByTimeAsync(2000));
  expect(host.querySelectorAll("textarea")[1]!.value).toBe("keep this input");
  expect(writes()).toHaveLength(1);
  await act(async () =>
    [...host.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!.click()
  );
  await act(async () => until(() => writes().length === 2, "explicit local input retry"));
  expect(writes()[1]!.body).toMatchObject({ description_html: "keep this input" });
  await answer(1, "keep this input");
});
