/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The tab of the tests of the stores of a project's lists, its states and its labels (M3 design 3.16, 3.17), against a
// fake nerve: the caller's workspaces, acme with web and ops, and beta with lab; the tab at web's address; the requests
// of the lists; web's labels as nerve lists them; and nerve's answer to a change of a state or a label.

import type { ApiClient, Label, LabelCreate, Project, State, StateCreate } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { answered } from "@/lib/auth/fake-nerve";
import { fakeRoot } from "@/store/fake-root";
import { LabelStore } from "@/store/label.store";
import { labelOf, projectOf, projectTab } from "@/store/project/fake-projects";
import type { RootStore } from "@/store/root.store";
import { StateStore } from "@/store/state.store";
import { workspaceOf } from "@/store/workspace/fake-workspaces";

export const acme = workspaceOf("acme");
export const beta = workspaceOf("beta");
export const web = projectOf("WEB", acme.id);
export const ops = projectOf("OPS", acme.id);
export const lab = projectOf("LAB", beta.id);

/** The requests of the states: web's, acme's, and a state's. */
export const stateRequests = {
  LIST: `/api/v0/projects/${web.id}/states`,
  ACME: "/api/v0/workspaces/acme/states",
  at: (state: State) => `/api/v0/states/${state.id}`,
};

/** A state created in web's started group, as the page sends it. */
export const blocking: StateCreate = { name: "Blocked", color: "#60646C", group: "started" };

/** The requests of the labels: web's, and a label's. */
export const labelRequests = {
  LIST: `/api/v0/projects/${web.id}/labels`,
  at: (label: Label) => `/api/v0/labels/${label.id}`,
};

const bug = labelOf(web, "bug", 65535);
const feature = labelOf(web, "feature", 75535);
const frontend = labelOf(web, "frontend", 80000, { parent_id: feature.id });
const backend = labelOf(web, "backend", 85000, { parent_id: feature.id });
const docs: LabelCreate = { name: "docs", color: "#3F76FF" };

/** web's labels, and a label created in web. */
export const webLabels = {
  bug,
  feature,
  frontend,
  backend,
  /** web's labels as nerve lists them, by sort order: two at the top, two under feature. */
  listed: [bug, feature, frontend, backend],
  /** A label created in web, as the page sends it, and as nerve answers it: last, at the top. */
  docs,
  created: labelOf(web, "docs", 95000, { color: "#3F76FF" }),
};

/** A time after the fixtures', so that nerve's answer to a change can be told from the request. */
const LATER = "2026-10-08T09:00:00Z";

/**
 * nerve's answer to a change of a state or a label: the fields given, and a change the request does not make, its
 * updated_at.
 */
export const changed = <T extends State | Label>(record: T, fields: Partial<T>): T => ({
  ...record,
  ...fields,
  updated_at: LATER,
});

/**
 * A store of a project's lists (StateStore, LabelStore) on the stores of a tab at web's address, whose caller's acme
 * and beta nerve listed with their projects; the requests so far are forgotten.
 */
export async function listTab<S>(Store: new (root: RootStore, api: ApiClient) => S) {
  const { nerve, api, router, workspaceRoot, projectRoot } = await projectTab(
    { workspace: acme, projects: [web, ops] },
    { workspace: beta, projects: [lab] }
  );
  const store = new Store(fakeRoot({ router, workspaceRoot, projectRoot }), api);
  return { nerve, api, router, projects: projectRoot.project, store };
}

/** The store fetches the project's states, web's unless it says, and nerve lists these. */
export function loadStates(nerve: FakeNerve, store: StateStore, states: State[], project: Project = web) {
  const path = `/api/v0/projects/${project.id}/states`;
  return answered(nerve, () => store.fetchProjectStates(project.id), ["GET", path], { data: states }, "the states");
}

/** The store fetches the project's labels, web's unless it says, and nerve lists these. */
export function loadLabels(nerve: FakeNerve, store: LabelStore, labels: Label[], project: Project = web) {
  const path = `/api/v0/projects/${project.id}/labels`;
  return answered(nerve, () => store.fetchProjectLabels(project.id), ["GET", path], { data: labels }, "the labels");
}

/** A state store of the tab, whose states of web nerve listed as these. */
export async function loadedStates(states: State[]) {
  const tab = await listTab(StateStore);
  await loadStates(tab.nerve, tab.store, states);
  return tab;
}

/** A label store of the tab, whose labels of web nerve listed. */
export async function loadedLabels() {
  const tab = await listTab(LabelStore);
  await loadLabels(tab.nerve, tab.store, webLabels.listed);
  return tab;
}
