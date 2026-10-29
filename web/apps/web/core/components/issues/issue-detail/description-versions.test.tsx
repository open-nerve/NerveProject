// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act, type ComponentProps } from "react";
import { createRoot, type Root } from "react-dom/client";
import { SWRConfig, type Cache } from "swr";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { json, problem } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import { descriptionVersionOf, issueDetailOf, issueStoreTab, requestedIssue } from "@/store/issue/fake-issues";
import { loadProjects } from "@/store/project/fake-projects";
import type { DescriptionVersionsRoot } from "@/components/core/description-versions";
import { IssueDescriptionVersions } from "./description-versions";
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: "one" }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-member", () => ({ useMember: () => ({ getUserDetails: () => undefined }) }));
vi.mock("@/components/core/description-versions", () => ({
  DescriptionVersionsRoot: (props: ComponentProps<typeof DescriptionVersionsRoot>) => {
    view = props;
    return <span data-versions>{props.versions?.map((row) => row.id).join(",")}</span>;
  },
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let view: ComponentProps<typeof DescriptionVersionsRoot>;
let renderer: Root;
let container: HTMLElement;
let cache: Cache<unknown>;
const editor = { current: { setEditorValue: vi.fn() } };
const route = "/api/v0/issues/x/description-versions";
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  tab = await issueStoreTab();
  cache = new Map();
  editor.current.setEditorValue.mockClear();
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
  await loadCore();
  tab.nerve.calls.length = 0;
});
afterEach(async () => {
  await act(async () => renderer.unmount());
  container.remove();
  tab.app.dispose();
  vi.useRealTimers();
});
async function loadCore() {
  const sent = await requestedIssue(tab.nerve, () => tab.app.issue.issueDetail.issue.fetchIssueCore("ws", "A", "x"), [
    "GET",
    "/api/v0/issues/x",
  ]);
  sent.call.answer(json(200, issueDetailOf("x")));
  await sent.promise;
}
const render = (disabled = false) =>
  act(async () =>
    renderer.render(
      <SWRConfig value={{ provider: () => cache }}>
        <IssueDescriptionVersions
          workspaceSlug="ws"
          projectId="A"
          issueId="x"
          editorRef={editor}
          isRestoreDisabled={disabled}
        />
      </SWRConfig>
    )
  );
async function loaded(disabled = false) {
  await render(disabled);
  await act(async () => until(() => tab.nerve.to(route).length === 1, "version list"));
  await act(async () =>
    tab.nerve.to(route)[0]!.answer(json(200, { data: [descriptionVersionOf("v")], next_cursor: null }))
  );
}
it.each([200, 403, 422])(
  "M4 restoration updates the editor only with confirmed HTML and never emits another write (%s)",
  async (status) => {
    await loaded();
    const sent = await requestedIssue(tab.nerve, () => view.handleRestore("selected"), ["PATCH", "/api/v0/issues/x"]);
    expect(editor.current.setEditorValue).not.toHaveBeenCalled();
    await act(async () =>
      sent.call.answer(
        status === 200
          ? json(200, { ...issueDetailOf("x"), description_html: "sanitized" })
          : problem(status, "forbidden")
      )
    );
    if (status === 200) {
      await sent.promise;
      expect(editor.current.setEditorValue).toHaveBeenCalledWith("sanitized", false);
    } else {
      await expect(sent.promise).rejects.toMatchObject({ status });
      expect(editor.current.setEditorValue).not.toHaveBeenCalled();
    }
    expect(tab.nerve.calls.map((row) => row.method)).toEqual(["GET", "PATCH"]);
  }
);
it("a read-only archived view still lists versions but disables restoration", async () => {
  await loaded(true);
  expect(view.entityInformation.isRestoreDisabled).toBe(true);
  expect(view.versions).toEqual([{ id: "v", ownerId: "actor", lastSavedAt: descriptionVersionOf("v").last_saved_at }]);
});
it("guest pages send no version HTTP", async () => {
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, member_role: 5 }]);
  tab.nerve.calls.length = 0;
  await render();
  expect(tab.nerve.calls).toHaveLength(0);
  expect(container.querySelector("[data-versions]")).toBeNull();
});
it("a permission ABA changes the SWR key and requires a fresh list", async () => {
  await loaded();
  const first = view.scopeKey;
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, []);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [tab.project]);
    await loadCore();
  });
  await act(async () => until(() => tab.nerve.to(route).length === 2, "fresh version list"));
  expect(view.scopeKey).not.toBe(first);
  expect(container.textContent).not.toContain("v");
  await act(async () =>
    tab.nerve.to(route)[1]!.answer(json(200, { data: [descriptionVersionOf("fresh")], next_cursor: null }))
  );
  expect(container.textContent).toBe("fresh");
});
