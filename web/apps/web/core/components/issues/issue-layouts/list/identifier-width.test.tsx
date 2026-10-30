// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, StrictMode, type ComponentProps, type ReactElement, type ReactNode, type Ref } from "react";
import { createRoot } from "react-dom/client";
import { observer } from "mobx-react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { EIssuesStoreType } from "@nerve/types";
import { issueOf, issueDetailOf, issueStoreTab } from "@/store/issue/fake-issues";
import { loadProjects, projectOf, stateOf } from "@/store/project/fake-projects";
import { json } from "@/lib/auth/fake-nerve";
import { until } from "@/lib/auth/fake-time";
import type { Group } from "@/lib/paged-groups";
import { List } from "./default";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
import { IssueBlockRoot } from "./block-root";
import type { ListGroup } from "./list-group";

const ui = vi.hoisted(() => ({
  login: "one",
  kind: "PROJECT" as EIssuesStoreType,
  hidden: new Set<string>(),
  params: { workspaceSlug: "ws", projectId: "A" as string | undefined, userId: undefined as string | undefined },
}));
vi.mock("@atlaskit/pragmatic-drag-and-drop-auto-scroll/element", () => ({ autoScrollForElements: () => () => {} }));
vi.mock("@/lib/auth/api-client", () => ({ tokenManager: { state: { loginId: "one" } }, publicClient: {} }));
vi.mock("@/lib/auth/use-session", () => ({ useSession: () => ({ status: "signed-in", loginId: ui.login }) }));
vi.mock("@/lib/store-context", () => ({ rootStore: {} }));
vi.mock("@nerve/i18n", () => import("@/lib/fake-i18n"));
vi.mock("@nerve/propel/toast", () => import("@/lib/fake-toast"));
vi.mock("react-router", () => ({ useParams: () => ui.params }));
vi.mock("@/hooks/use-platform-os", () => ({ usePlatformOS: () => ({ isMobile: false }) }));
vi.mock("@/hooks/store/use-app-theme", () => ({ useAppTheme: () => ({ sidebarCollapsed: false }) }));
vi.mock("@/hooks/store/use-issue-detail", () => ({ useIssueDetail: () => tab.app.issue.issueDetail }));
vi.mock("@/hooks/store/use-project", () => ({ useProject: () => tab.app.projectRoot.project }));
vi.mock("@/hooks/use-issue-layout-store", () => ({ useIssueStoreType: () => ui.kind }));
vi.mock("../utils", async (original) => ({
  ...(await original<typeof import("../utils")>()),
  getGroupByColumns: () => [{ id: "first" }, { id: "second" }],
}));
vi.mock("../properties", () => ({ IssueProperties: () => null }));
vi.mock("@/components/issues/issue-detail/identifier-text", () => ({
  IdentifierText: ({ identifier }: { identifier: string }) => <span>{identifier}</span>,
}));
vi.mock("@makeplane/propel/components/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("@nerve/ui", () => ({
  Row: ({ children, className, ref }: { children: ReactNode; className?: string; ref?: Ref<HTMLDivElement> }) => (
    <div ref={ref} className={className}>
      {children}
    </div>
  ),
  ControlLink: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  DropIndicator: () => null,
}));
// Keep the actual List, recursive row owner, row and identifier. Isolate group chrome and viewport visibility.
vi.mock("@/components/core/render-if-visible-HOC", () => ({
  default: ({ children }: { children: ReactElement<{ issueId: string }> }) =>
    ui.hidden.has(children.props.issueId) ? null : children,
}));
vi.mock("./list-group", () => ({
  ListGroup: (props: ComponentProps<typeof ListGroup>) =>
    props.collapsedGroups?.group_by.includes(props.group.id) ? null : (
      <section data-group={props.group.id}>
        {props.groupIssueIds?.map((id) => (
          <IssueBlockRoot
            key={id}
            issueId={id}
            issuesMap={props.issuesMap}
            updateIssue={undefined}
            quickActions={() => <></>}
            canEditProperties={() => false}
            displayProperties={props.displayProperties}
            nestingLevel={0}
            containerRef={props.containerRef}
            groupId={props.group.id}
            isDragAllowed={false}
            canDropOverIssue={false}
          />
        ))}
      </section>
    ),
}));
let tab: Awaited<ReturnType<typeof issueStoreTab>>;
let host: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
beforeEach(async () => {
  vi.useFakeTimers();
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  ui.login = "one";
  ui.kind = EIssuesStoreType.PROJECT;
  ui.hidden.clear();
  ui.params = { workspaceSlug: "ws", projectId: "A", userId: undefined };
  tab = await issueStoreTab();
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
    { ...tab.project, identifier: "WEB" },
    projectOf("B", tab.workspace.id, { id: "B", identifier: "LONGPREFIX" }),
  ]);
  const held = [
    { ...issueOf("nine"), sequence_id: 9 },
    { ...issueOf("ten"), sequence_id: 10 },
    { ...issueOf("hundred"), sequence_id: 100 },
    { ...issueOf("parent"), sequence_id: 9, sub_issues_count: 1 },
    { ...issueOf("long", "B"), sequence_id: 9 },
    { ...issueOf("unknown", "not-fetched"), sequence_id: 100000 },
  ];
  for (const issue of held) tab.app.issue.issues.acceptRead(issue, tab.app.issue.captureRead());
  tab.nerve.calls.length = 0;
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
const groups = (first: string[], second: string[]): Map<string, Group> =>
  new Map(
    [first, second].map((ids, index) => [
      index ? "second" : "first",
      {
        rows: ids.map((id) => ({ id, key: 0, updatedAt: "2026-10-30T00:00:00Z" })),
        total: ids.length,
        nextCursor: null,
        boundary: null,
        cursorBoundary: null,
        stale: false,
      },
    ])
  );
const Harness = observer(function Harness({
  first,
  second = [],
  showKey = true,
  view = "assigned",
}: {
  first: string[];
  second?: string[];
  showKey?: boolean;
  view?: string;
}) {
  return (
    <List
      groups={groups(first, second)}
      issuesMap={tab.app.issue.issues.issuesMap}
      group_by={null}
      orderBy={undefined}
      updateIssue={undefined}
      quickActions={() => <></>}
      displayProperties={{ key: showKey }}
      enableIssueQuickAdd={false}
      showEmptyGroup
      canEditProperties={() => false}
      quickAddCallback={undefined}
      loadMoreIssues={() => {}}
      handleOnDrop={async () => {}}
      handleCollapsedGroups={() => {}}
      collapsedGroups={{ group_by: [], sub_group_by: [] }}
      viewId={view}
    />
  );
});
const render = async (first: string[], second: string[] = [], showKey = true, view = "assigned") =>
  act(async () => root.render(<Harness first={first} second={second} showKey={showKey} view={view} />));
const widths = (container: ParentNode = host) =>
  [...container.querySelectorAll<HTMLElement>('[style*="min-width"]')].map((node) =>
    Number.parseFloat(node.style.minWidth)
  );
it("keepsOneWidthAcrossGroupsAndLoadedRows", async () => {
  ui.hidden.add("hundred");
  await render(["nine", "ten"], ["hundred"]);
  expect(host.textContent).not.toContain("WEB-100");
  expect(widths()).toEqual([49, 49]);
});
it("growsForPagedAndExpandedRows", async () => {
  await render(["nine"]);
  expect(widths()).toEqual([35]);
  await render(["nine", "ten"]);
  expect(widths()).toEqual([42, 42]);
  await render(["parent"]);
  expect(widths()).toEqual([42]);
  tab.nerve.respond = (call) => {
    if (call.path === "/api/v0/projects/A/states")
      call.answer(json(200, { data: [stateOf(tab.project, "Todo", "unstarted", 1000)] }));
    if (call.path === "/api/v0/issues/parent")
      call.answer(json(200, { ...issueDetailOf("parent"), sequence_id: 9, sub_issues_count: 1 }));
    if (call.path === "/api/v0/issues/parent/sub-issues")
      call.answer(json(200, { data: [{ ...issueOf("child"), sequence_id: 1000, parent_id: "parent" }] }));
  };
  await act(async () => host.querySelector<HTMLButtonElement>("button")!.click());
  await act(async () =>
    until(
      () => tab.app.issue.issueDetail.subIssues.subIssuesByIssueId("parent")?.includes("child") ?? false,
      "expanded child"
    )
  );
  expect(widths()).toEqual([56, 56]);
});
it("cross-project profile widths compare complete identifiers instead of combining unrelated maxima", async () => {
  ui.kind = EIssuesStoreType.PROFILE;
  ui.params.projectId = undefined;
  ui.params.userId = "alice";
  await render(["hundred"], ["long"]);
  expect(widths()).toEqual([84, 84]);
});
it("resetsForScopeAndPermissionChanges", async () => {
  await render(["hundred", "nine"]);
  await render(["nine"]);
  expect(widths()).toEqual([49]);
  ui.login = "next";
  await render(["nine"]);
  expect(widths()).toEqual([35]);
  await render(["hundred", "nine"]);
  await render(["nine"], [], true, "created");
  expect(widths()).toEqual([35]);
  await render(["hundred", "nine"], [], true, "created");
  await render(["nine"], [], true, "created");
  await act(async () => {
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [
      { ...tab.project, identifier: "WEB", member_role: 5 },
    ]);
    await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [{ ...tab.project, identifier: "WEB" }]);
    tab.app.issue.issues.acceptRead({ ...issueOf("nine"), sequence_id: 9 }, tab.app.issue.captureRead());
  });
  expect(widths()).toEqual([35]);
});
it("doesNotReadProjectsToSizeIdentifiers", async () => {
  await render(["nine", "unknown"]);
  expect(tab.nerve.calls).toHaveLength(0);
  expect(host.textContent).not.toContain("undefined-");
  expect(widths().every((width) => width === 35)).toBe(true);
  await render(["nine"], [], false);
  expect(widths()).toEqual([0]);
  await render([]);
  expect(widths()).toEqual([]);
});
it("independent layouts retain separate widths when the other owner unmounts", async () => {
  await act(async () =>
    root.render(
      <>
        <div data-owner="old">
          <Harness first={["hundred"]} />
        </div>
        <div data-owner="new">
          <Harness first={["nine"]} />
        </div>
      </>
    )
  );
  expect(widths(host.querySelector('[data-owner="old"]')!)).toEqual([49]);
  expect(widths(host.querySelector('[data-owner="new"]')!)).toEqual([35]);
  await act(async () =>
    root.render(
      <>
        <div data-owner="old" />
        <div data-owner="new">
          <Harness first={["nine"]} />
        </div>
      </>
    )
  );
  expect(widths(host.querySelector('[data-owner="new"]')!)).toEqual([35]);
});

it.each([
  [
    "list kind",
    () => {
      ui.kind = EIssuesStoreType.ARCHIVED;
    },
  ],
  [
    "project",
    () => {
      ui.params.projectId = "B";
    },
  ],
  [
    "profile user",
    () => {
      ui.params.userId = "bob";
    },
  ],
  [
    "workspace",
    () => {
      ui.params.workspaceSlug = "elsewhere";
    },
  ],
] as const)("resets width when %s changes", async (_, change) => {
  await render(["hundred", "nine"]);
  await render(["nine"]);
  expect(widths()).toEqual([49]);
  change();
  await render(["nine"]);
  expect(widths()).toEqual([35]);
});
it("does not invent an identifier when its project directory is absent", async () => {
  await act(async () =>
    root.render(<IssueIdentifier projectId="not-fetched" projectIdentifier={undefined} issueSequenceId={100000} />)
  );
  expect(host.textContent).toBe("");
  expect(tab.nerve.calls).toHaveLength(0);
});
it("registers rows again under StrictMode without stale cleanup affecting the active layout", async () => {
  await act(async () =>
    root.render(
      <StrictMode>
        <Harness first={["nine", "hundred"]} />
      </StrictMode>
    )
  );
  expect(widths()).toEqual([49, 49]);
  ui.login = "next";
  await act(async () =>
    root.render(
      <StrictMode>
        <Harness first={["nine"]} />
      </StrictMode>
    )
  );
  expect(widths()).toEqual([35]);
});
