/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ComponentType } from "react";
import { observable } from "mobx";
import { MemoryRouter, Route, Routes, useParams } from "react-router";
import { SWRConfig } from "swr";
import { Where } from "@/components/common/fake-quick-actions";
import Layout from "./layout";
import Page from "./page";

// A project's views list page as its tests open it (P6a spec 2.3): its layout (its header, its small screen's header
// and the page's state) and its page, at the route's address, with SWR's cache of the test's own; the project's
// features and its other pages elsewhere. And a stand-in for the member store, whose people and projects' members the
// list's owners are.

/** A route module's component, given its route's params as the router gives them. */
type RouteProps = { params: Readonly<Record<string, string | undefined>> };
const routeOf = (Component: unknown) => Component as ComponentType<RouteProps>;
function RouteOf({ Component }: { Component: ComponentType<RouteProps> }) {
  const params = useParams();
  return <Component params={params} />;
}

/** A project's views list at path, as the app's routes lay it out. */
export function ListPages({ path }: { path: string }) {
  return (
    <SWRConfig value={{ provider: () => new Map() }}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path=":workspaceSlug/projects/:projectId/views" element={<RouteOf Component={routeOf(Layout)} />}>
            <Route index element={<RouteOf Component={routeOf(Page)} />} />
          </Route>
          <Route path=":workspaceSlug/settings/projects/:projectId/features/views" element={<p>the features</p>} />
          <Route path="*" element={<p>elsewhere</p>} />
        </Routes>
        <Where />
      </MemoryRouter>
    </SWRConfig>
  );
}

/** The people of acme by their account ids: the caller (u-me) and the projects' members, Gus a guest of them. */
const PEOPLE: Record<string, string> = {
  "u-me": "Me",
  "u-ann": "Ann",
  "u-bob": "Bob",
  "u-cat": "Cat",
  "u-dee": "Dee",
  "u-gus": "Gus",
};
const person = (id: string) => (PEOPLE[id] ? { id, display_name: PEOPLE[id] } : undefined);

/**
 * A stand-in for the member store: the people's details; each project's members (the guest Gus too, when asked),
 * once they have arrived (arrived), and what the page asked for (asked: the project, whether with its guests).
 */
export function standInMembers() {
  const arrived = observable.box(true);
  const asked: [string, boolean][] = [];
  return {
    arrived,
    asked,
    getUserDetails: person,
    project: {
      getProjectMemberIds: (projectId: string, includeGuestUsers: boolean) => {
        asked.push([projectId, includeGuestUsers]);
        if (!arrived.get()) return null;
        const members = ["u-me", "u-ann", "u-bob", "u-cat", "u-dee"];
        return includeGuestUsers ? [...members, "u-gus"] : members;
      },
    },
    workspace: {
      getWorkspaceMemberDetails: (id: string) => {
        const member = person(id);
        return member ? { member } : null;
      },
    },
  };
}
