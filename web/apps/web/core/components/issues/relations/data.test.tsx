// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import React, { act, useState } from "react";
import { createRoot, type Root } from "react-dom/client";
import { beforeEach, afterEach, expect, it, vi } from "vitest";
import { configure } from "mobx";
import type { Issue } from "@nerve/api-client";
import type { IssuesModalProps } from "../issue-modal/modal";
import { issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { json, problem } from "@/lib/auth/fake-nerve";
import { RelationsCollapsibleContent } from "../issue-detail-widgets/relations/content";
import { RelationIssueListItem } from "./issue-list-item";
import { loadProjects, projectOf } from "@/store/project/fake-projects";
import { SubIssuesCollapsibleContent } from "../issue-detail-widgets/sub-issues/content";
import { IssueDetailRoot } from "../issue-detail/root";

configure({ safeDescriptors: false });
const binding = vi.hoisted(() => ({ tab: undefined as Awaited<ReturnType<typeof issueStoreTab>> | undefined }));
vi.stubGlobal("localStorage", { getItem: () => null, setItem: () => {}, removeItem: () => {} });
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/store-context", () => ({ rootStore: undefined }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => binding.tab!.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => binding.tab!.app.projectRoot.project }));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ issueDetailSidebarCollapsed: false }) }));
vi.mock("@/hooks/store/user", () => ({ useUserPermissions: () => ({ allowPermissions: () => true }) }));
vi.mock("react-router", () => ({ useNavigate: () => () => {} }));
vi.mock("../issue-detail/main-content", () => ({ IssueMainContent: () => <div data-detail>Detail content</div> }));
vi.mock("../issue-detail/sidebar", () => ({ IssueDetailsSidebar: () => null }));
vi.mock("../peek-overview", () => ({ IssuePeekOverview: () => null }));
vi.mock("@/components/common/empty-state", () => ({ EmptyState: () => <div data-empty>Missing issue</div> }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/use-issue-peek-overview-redirection", () => ({ default: () => ({ handleRedirection: () => {} }) }));
vi.mock("../issue-detail/issue-identifier", () => ({ IssueIdentifier: () => null }));
vi.mock("./properties", () => ({ RelationIssueProperty: () => null }));
vi.mock("../issue-detail-widgets/sub-issues/issues-list/root", () => ({
  SubIssuesListRoot: () => <div data-subitems>Loaded children</div>,
}));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock("@nerve/ui", () => ({
  ControlLink: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  CustomMenu: Object.assign(({ children }: { children: React.ReactNode }) => <div>{children}</div>, {
    MenuItem: ({
      children,
      onClick,
    }: {
      children: React.ReactNode;
      onClick: React.MouseEventHandler<HTMLButtonElement>;
    }) => <button onClick={onClick}>{children}</button>,
  }),
}));
vi.mock("@makeplane/propel/components/collapsible", () => ({
  Collapsible: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));
vi.mock("../delete-issue-modal", () => ({ DeleteIssueModal: () => null }));
vi.mock("./issue-list", () => ({
  RelationIssueList: ({
    handleIssueCrudState,
  }: {
    handleIssueCrudState: (key: string, id: string, issue: Issue) => void;
  }) => (
    <button
      onClick={() => handleIssueCrudState("update", "target", binding.tab!.app.issue.issues.getIssueById("target")!)}
    >
      Edit target
    </button>
  ),
}));
// The child modal boundary emits a confirmed entity, as its onSubmit contract promises.
// The actual parent consumer must not turn that notification into another write.
vi.mock("../issue-modal/modal", () => ({
  CreateUpdateIssueModal: (props: IssuesModalProps) => {
    const [error, setError] = useState(false);
    const submit = async () => {
      try {
        const answer = await binding.tab!.app.issue.updateIssue("target", () => ({ name: "Edited title" }));
        if (!props.isDraft) await props.onSubmit?.(answer);
        props.onClose();
      } catch {
        setError(true);
      }
    };
    return (
      <div role="dialog">
        <input aria-label="name" defaultValue="Edited title" />
        <button onClick={() => void submit()}>Save</button>
        {error && <span role="alert">refused</span>}
      </div>
    );
  },
}));
let renderer: Root;
let container: HTMLElement;
beforeEach(async () => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.useFakeTimers();
  binding.tab = await issueStoreTab();
  const { app, nerve } = binding.tab;
  for (const id of ["source", "target"]) app.issue.issues.acceptRead(issueDetailOf(id), app.issue.captureRead());
  vi.spyOn(app.issue.issueDetail.activity, "fetchActivities").mockResolvedValue([]);
  nerve.respond = (call) => {
    if (call.path.endsWith("/relations"))
      call.answer(
        json(200, {
          data: [
            {
              id: "relation",
              relation_type: "blocking",
              issue: issueDetailOf("target"),
              created_at: "2026-10-29T00:00:00Z",
            },
          ],
        })
      );
  };
  await app.issue.issueDetail.fetchRelations("ws", "A", "source");
  nerve.calls.length = 0;
  container = document.createElement("div");
  document.body.append(container);
  renderer = createRoot(container);
  await act(async () =>
    renderer.render(<RelationsCollapsibleContent workspaceSlug="ws" issueId="source" disabled={false} />)
  );
  await act(async () => (container.querySelector("button") as HTMLButtonElement).click());
});
afterEach(async () => {
  await act(async () => renderer?.unmount());
  container?.remove();
  binding.tab?.app.dispose();
  vi.restoreAllMocks();
  vi.useRealTimers();
});
const save = () => [...container.querySelectorAll("button")].find((button) => button.textContent === "Save")!.click();
it("submitsOneMutationFromTheRelationEditor", async () => {
  const { nerve } = binding.tab!;
  await act(async () => save());
  expect(nerve.calls.map((call) => [call.method, call.path, call.body])).toEqual([
    ["PATCH", "/api/v0/issues/target", { name: "Edited title" }],
  ]);
  await act(async () =>
    nerve.calls[0]!.answer(
      json(200, { ...issueDetailOf("target"), name: "Edited title", updated_at: "2026-10-29T00:00:00Z" })
    )
  );
  expect(nerve.calls.filter((call) => call.method === "PATCH")).toHaveLength(1);
  expect(container.querySelector('[role="dialog"]')).toBeNull();
});
it("keeps the editor's input on a refused mutation", async () => {
  const { nerve } = binding.tab!;
  await act(async () => save());
  await act(async () => nerve.calls[0]!.answer(problem(403, "forbidden")));
  expect(container.querySelector<HTMLInputElement>("input")?.value).toBe("Edited title");
  expect(container.querySelector('[role="alert"]')?.textContent).toBe("refused");
  expect(nerve.calls).toHaveLength(1);
});
it("offers a working read-only retry when confirmed relations make endpoint details uncertain", async () => {
  const { app, nerve } = binding.tab!;
  nerve.respond = undefined;
  let saving!: Promise<void>;
  await act(async () => {
    saving = app.issue.issueDetail.createRelation("ws", "A", "source", "blocking", ["target"]);
  });
  await act(async () =>
    nerve.calls[0]!.answer(
      json(200, {
        data: [
          {
            id: "relation",
            relation_type: "blocking",
            issue: issueDetailOf("target"),
            created_at: "2026-10-29T00:00:00Z",
          },
        ],
      })
    )
  );
  await act(async () => {
    nerve.to("/api/v0/issues/source")[0]!.answer(problem(500, "unavailable"));
    nerve.to("/api/v0/issues/target")[0]!.answer(problem(500, "unavailable"));
    await saving;
  });
  expect(app.issue.issues.getIssueById("source")).toBeUndefined();
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  const retry = [...container.querySelectorAll("button")].find((button) => button.textContent === "common.retry")!;
  await act(async () => retry.click());
  expect(nerve.to("/api/v0/issues/source")).toHaveLength(2);
  await act(async () => {
    nerve.to("/api/v0/issues/source")[1]!.answer(json(200, issueDetailOf("source")));
    nerve.to("/api/v0/issues/target")[1]!.answer(json(200, issueDetailOf("target")));
  });
  expect(nerve.calls.filter((call) => call.method === "POST")).toHaveLength(1);
  expect(container.querySelector('[role="alert"]')).toBeNull();
});
it("the actual cross-project row removes its relation in the source project's scope", async () => {
  const { app, nerve, project, workspace } = binding.tab!;
  await act(async () =>
    loadProjects(nerve, app.projectRoot.project, workspace, [project, projectOf("B", workspace.id, { id: "B" })])
  );
  nerve.respond = (call) => {
    if (call.method === "GET" && call.path.endsWith("/relations"))
      call.answer(
        json(200, {
          data: [
            {
              id: "relation-cross",
              relation_type: "blocking",
              issue: issueDetailOf("cross", "B"),
              created_at: "2026-10-29T00:00:00Z",
            },
          ],
        })
      );
    if (call.path === "/api/v0/workspaces/ws/states" || /\/projects\/B\/(members|labels)$/.test(call.path))
      call.answer(json(200, { data: [] }));
  };
  await act(async () => app.issue.issueDetail.fetchRelations("ws", "A", "source"));
  nerve.calls.length = 0;
  await act(async () =>
    renderer.render(
      <RelationIssueListItem
        workspaceSlug="ws"
        issueId="source"
        relationKey="blocking"
        relationIssueId="cross"
        disabled={false}
        handleIssueCrudState={() => {}}
      />
    )
  );
  const button = [...container.querySelectorAll("button")].find(
    (item) => item.textContent === "common.actions.remove_relation"
  )!;
  await act(async () => button.click());
  expect(nerve.calls.map((call) => [call.method, call.path])).toEqual([
    ["DELETE", "/api/v0/issue-relations/relation-cross"],
  ]);
  await act(async () => nerve.calls[0]!.answer(problem(403, "forbidden")));
  expect(app.issue.issueDetail.relation.getRelationsByIssueId("source")?.blocking).toEqual(["cross"]);
});
it("a failed child read offers retry and displays the children after that read succeeds", async () => {
  const { nerve } = binding.tab!;
  nerve.respond = (call) => {
    if (call.path === "/api/v0/projects/A/states") call.answer(json(200, { data: [] }));
    if (call.path === "/api/v0/issues/source") call.answer(json(200, issueDetailOf("source")));
  };
  await act(async () =>
    renderer.render(
      <SubIssuesCollapsibleContent workspaceSlug="ws" projectId="A" parentIssueId="source" disabled={false} />
    )
  );
  await act(async () => nerve.to("/api/v0/issues/source/sub-issues")[0]!.answer(problem(500, "unavailable")));
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  expect(container.querySelector("[data-subitems]")).toBeNull();
  await act(async () =>
    [...container.querySelectorAll("button")].find((item) => item.textContent === "common.retry")!.click()
  );
  await act(async () =>
    nerve
      .to("/api/v0/issues/source/sub-issues")[1]!
      .answer(json(200, { data: [{ ...issueDetailOf("target"), parent_id: "source" }] }))
  );
  expect(container.querySelector('[role="alert"]')).toBeNull();
  expect(container.querySelector("[data-subitems]")).not.toBeNull();
});
it("the real detail host still offers resource retry while its canonical issue is awaiting verification", async () => {
  const { app, nerve } = binding.tab!;
  nerve.respond = undefined;
  await act(async () => renderer.render(<IssueDetailRoot workspaceSlug="ws" projectId="A" issueId="source" />));
  let saving!: Promise<void>;
  await act(async () => {
    saving = app.issue.issueDetail.createRelation("ws", "A", "source", "blocking", ["target"]);
  });
  await act(async () =>
    nerve.calls[0]!.answer(
      json(200, {
        data: [
          {
            id: "relation",
            relation_type: "blocking",
            issue: issueDetailOf("target"),
            created_at: "2026-10-29T00:00:00Z",
          },
        ],
      })
    )
  );
  await act(async () => {
    nerve.to("/api/v0/issues/source")[0]!.answer(problem(500, "unavailable"));
    nerve.to("/api/v0/issues/target")[0]!.answer(problem(500, "unavailable"));
    await saving;
  });
  expect(container.querySelector('[role="alert"]')).not.toBeNull();
  expect(container.querySelector("[data-empty]")).toBeNull();
  await act(async () =>
    [...container.querySelectorAll("button")].find((item) => item.textContent === "common.retry")!.click()
  );
  await act(async () => {
    nerve.to("/api/v0/issues/source")[1]!.answer(json(200, issueDetailOf("source")));
    nerve.to("/api/v0/issues/target")[1]!.answer(json(200, issueDetailOf("target")));
  });
  expect(container.querySelector("[data-detail]")).not.toBeNull();
  expect(nerve.calls.filter((call) => call.method === "POST")).toHaveLength(1);
});
