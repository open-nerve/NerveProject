/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import type { Project, View, Workspace } from "@nerve/api-client";
import { Where, settle } from "@/components/common/fake-quick-actions";
import { useViews } from "@/hooks/store/use-views";
import { type FakeNerve, json } from "@/lib/auth/fake-nerve";
import { viewRequests } from "@/store/views/fake-views";
import type { ViewContainer } from "@/store/views/view-container";
import { ViewActionsHost, useViewsPage } from "./actions-host";
import { ViewListItem } from "./view-list-item";

// The pages of a project's views as a view's quick actions are on them (P6a spec 2.3), for their tests: its list, a
// view's own page, each the page of its ViewActionsHost with a row (ViewListItem) for each view it shows, and
// another page, elsewhere, and what the host of the page shown gives it (hostPage); and nerve's answer to a page's
// read of a project's views. The views are a views' store's
// (fake-views.ts), which a test file gives the pages by mocking @/hooks/store/use-views with it.

/** nerve answers the page's last read of the project's views with these. */
export async function listViews(nerve: FakeNerve, project: Project, views: View[]) {
  await act(async () =>
    nerve
      .to(viewRequests.list(project))
      .at(-1)
      ?.answer(json(200, { data: views }))
  );
  await settle();
}

/** The row of the view, if its page shows it: the row its right-click menu is on. */
export const viewRow = (id: string) => document.querySelector<HTMLElement>(`[data-row="${id}"] > div`);

/** A row of a page: the view's, as its list shows it. */
function Row({ view }: { view: View }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  return (
    <div data-row={view.id}>
      <ViewListItem view={view} workspaceSlug={workspaceSlug} container={{ kind: "project", projectId }} />
    </div>
  );
}

/** The project's views, as the store gives them. */
const List = observer(function List() {
  const { projectId } = useParams();
  const { getProjectViews } = useViews();
  return (getProjectViews(projectId) ?? []).map((view) => <Row key={view.id} view={view} />);
});

/** A view's own page: its row. */
const OwnPage = observer(function OwnPage() {
  const { viewId } = useParams();
  const view = useViews().getViewById(viewId);
  return view ? <Row view={view} /> : <p>no view</p>;
});

/** What the host of the page shown last gives its page (useViewsPage). */
export const hostPage: { current?: ReturnType<typeof useViewsPage> } = {};
function HostPage() {
  hostPage.current = useViewsPage();
  return null;
}

function Page({ children }: { children: ReactNode }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  return (
    <ViewActionsHost workspaceSlug={workspaceSlug} container={{ kind: "project", projectId }}>
      <HostPage />
      {children}
    </ViewActionsHost>
  );
}

/** The pages of the project's views, at path. */
export function ViewPages({ path }: { path: string }) {
  return (
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path=":workspaceSlug/projects/:projectId/views"
          element={
            <Page>
              <List />
            </Page>
          }
        />
        <Route
          path=":workspaceSlug/projects/:projectId/views/:viewId"
          element={
            <Page>
              <OwnPage />
            </Page>
          }
        />
        <Route path="*" element={<p>elsewhere</p>} />
      </Routes>
      <Where />
    </MemoryRouter>
  );
}

// The pages of a workspace's own views (M7/P6b spec 2.11), at the app's own routes (app/routes/core.ts): its list and
// a view's own page, whose view is the route's globalViewId (re-97 I5), each the page of its ViewActionsHost in the
// workspace, with a row for each of the views it shows that the store gives.

/** A row of a workspace's page: the view's, in the workspace (container), of the route's workspace. */
function WorkspaceRow({ view, container }: { view: View; container: ViewContainer }) {
  const { workspaceSlug = "" } = useParams();
  return (
    <div data-row={view.id}>
      <ViewListItem view={view} workspaceSlug={workspaceSlug} container={container} />
    </div>
  );
}

/** The workspace's list: the views it was given, as the store gives them. */
const WorkspaceList = observer(function WorkspaceList(props: { container: ViewContainer; listed: View[] }) {
  const { getViewById } = useViews();
  return props.listed.flatMap(({ id }) => {
    const view = getViewById(id);
    return view ? [<WorkspaceRow key={id} view={view} container={props.container} />] : [];
  });
});

/** A workspace's view's own page, of the route's globalViewId: its row. */
const WorkspaceOwnPage = observer(function WorkspaceOwnPage({ container }: { container: ViewContainer }) {
  const { globalViewId } = useParams();
  const view = useViews().getViewById(globalViewId);
  return view ? <WorkspaceRow view={view} container={container} /> : <p>no view</p>;
});

function WorkspacePage({ container, children }: { container: ViewContainer; children: ReactNode }) {
  const { workspaceSlug = "" } = useParams();
  return (
    <ViewActionsHost workspaceSlug={workspaceSlug} container={container}>
      <HostPage />
      {children}
    </ViewActionsHost>
  );
}

/** The pages of the workspace's own views, at path; its list shows those of listed the store gives. */
export function WorkspaceViewPages({
  path,
  workspace,
  listed,
}: {
  path: string;
  workspace: Workspace;
  listed: View[];
}) {
  const container: ViewContainer = { kind: "workspace", workspaceId: workspace.id, workspaceSlug: workspace.slug };
  return (
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route
          path=":workspaceSlug/workspace-views"
          element={
            <WorkspacePage container={container}>
              <WorkspaceList container={container} listed={listed} />
            </WorkspacePage>
          }
        />
        <Route
          path=":workspaceSlug/workspace-views/:globalViewId"
          element={
            <WorkspacePage container={container}>
              <WorkspaceOwnPage container={container} />
            </WorkspacePage>
          }
        />
        <Route path="*" element={<p>elsewhere</p>} />
      </Routes>
      <Where />
    </MemoryRouter>
  );
}
