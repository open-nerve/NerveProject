/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

// The modules of the tests of the planning stores (M6 design 3.2, 3.5–3.8), against a fake nerve: a module as nerve's
// lists give it, a module's progress as nerve reads it and a link of it, the requests of the modules, and the fetches
// that load them into a store. The modules themselves are fake-projects.ts's moduleOf.

import type { Module, ModuleLink, ModuleListItem, Progress, Project } from "@nerve/api-client";
import type { FakeNerve } from "@/lib/auth/fake-nerve";
import { answered } from "@/lib/auth/fake-nerve";
import { web } from "@/store/fake-project-lists";
import type { IModuleStore } from "@/store/planning/module.store";

/** A module as a list of nerve's gives it: of no work items, unless fields say so. */
export function moduleRowOf(item: Module, fields: Partial<ModuleListItem> = {}): ModuleListItem {
  return {
    ...item,
    total_issues: 0,
    backlog_issues: 0,
    unstarted_issues: 0,
    started_issues: 0,
    completed_issues: 0,
    cancelled_issues: 0,
    ...fields,
  };
}

/** A module's progress as nerve reads it: of no work items, with no burndown, unless fields say so. */
export function moduleProgressOf(fields: Partial<Progress> = {}): Progress {
  return {
    total_issues: 0,
    backlog_issues: 0,
    unstarted_issues: 0,
    started_issues: 0,
    completed_issues: 0,
    cancelled_issues: 0,
    assignees: [],
    labels: [],
    burndown: null,
    ...fields,
  };
}

/** A link of the module as nerve lists it: url names it in its module; no title, its metadata not read yet. */
export function moduleLinkOf(item: Module, url: string, fields: Partial<ModuleLink> = {}): ModuleLink {
  return {
    id: `ml-${item.id}-${url}`,
    module_id: item.id,
    url,
    title: null,
    metadata: {},
    created_by_id: null,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-01T09:00:00Z",
    ...fields,
  };
}

/** The requests of the modules: a project's lists (the archived one with archived=true); a module, its progress, its links. */
export const moduleRequests = {
  list: (project: Pick<Project, "id">) => `/api/v0/projects/${project.id}/modules`,
  at: (item: Pick<Module, "id">) => `/api/v0/modules/${item.id}`,
  progress: (item: Pick<Module, "id">) => `/api/v0/modules/${item.id}/progress`,
  links: (item: Pick<Module, "id">) => `/api/v0/modules/${item.id}/links`,
};

/** The store fetches the project's modules that are not archived, web's unless it says, and nerve lists these. */
export function loadModules(nerve: FakeNerve, store: IModuleStore, rows: ModuleListItem[], project: Project = web) {
  const path = moduleRequests.list(project);
  return answered(nerve, () => store.fetchModules(project.id), ["GET", path], { data: rows }, "the modules");
}

/** The store fetches the project's archived modules, web's unless it says, and nerve lists these. */
export function loadArchivedModules(
  nerve: FakeNerve,
  store: IModuleStore,
  rows: ModuleListItem[],
  project: Project = web
) {
  const path = moduleRequests.list(project);
  return answered(nerve, () => store.fetchArchivedModules(project.id), ["GET", path], { data: rows }, "the archived");
}

/** The store reads the module alone, and nerve gives it as read. */
export function loadModule(nerve: FakeNerve, store: IModuleStore, read: Module) {
  return answered(nerve, () => store.fetchModule(read.id), ["GET", moduleRequests.at(read)], read, "the module");
}

/** The store reads the module's progress, and nerve gives progress. */
export function loadModuleProgress(nerve: FakeNerve, store: IModuleStore, item: Module, progress: Progress) {
  const path = moduleRequests.progress(item);
  return answered(nerve, () => store.fetchProgress(item.id), ["GET", path], progress, "the progress");
}

/** The store reads the module's links, and nerve lists these. */
export function loadModuleLinks(nerve: FakeNerve, store: IModuleStore, item: Module, links: ModuleLink[]) {
  const path = moduleRequests.links(item);
  return answered(nerve, () => store.fetchLinks(item.id), ["GET", path], { data: links }, "the links");
}
