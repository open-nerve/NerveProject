/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, type ReactNode } from "react";
import { observer } from "mobx-react";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import type { Project, View } from "@nerve/api-client";
import { Where, settle } from "@/components/common/fake-quick-actions";
import { useViews } from "@/hooks/store/use-views";
import { type FakeNerve, json } from "@/lib/auth/fake-nerve";
import { viewRequests } from "@/store/views/fake-views";
import { ViewActionsHost } from "./actions-host";
import { ProjectViewListItem } from "./view-list-item";

// The pages of a project's views as a view's quick actions are on them (P6a spec 2.3), for their tests: its list, a
// view's own page, each the page of its ViewActionsHost with a row (ProjectViewListItem) for each view it shows, and
// another page, elsewhere; and nerve's answer to a page's read of a project's views. The views are a views' store's
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
      <ProjectViewListItem view={view} workspaceSlug={workspaceSlug} projectId={projectId} />
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

function Page({ children }: { children: ReactNode }) {
  const { workspaceSlug = "", projectId = "" } = useParams();
  return (
    <ViewActionsHost workspaceSlug={workspaceSlug} projectId={projectId}>
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
