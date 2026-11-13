/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { maxBy } from "lodash-es";
import { computedFn } from "mobx-utils";
// nerve imports
import type { ApiClient, Module, ModuleLink, ModuleListItem, Progress, Project } from "@nerve/api-client";
// lib
import { ReconciledByKey } from "@/lib/reconciled";
// services
import { ModulesService } from "@/services/planning/modules.service";
// store
import type { RootStore } from "@/store/root.store";
import type { IModuleChanges } from "./module-changes";
import { ModuleChanges } from "./module-changes";

/** A copy of a module the store holds, and when nerve answered it (Reconciled.answeredAt). */
type Copy<T extends Module> = { module: T; answeredAt: number };

export interface IModuleStore extends IModuleChanges {
  // computed actions
  getModuleById: (moduleId: string | null | undefined) => Module | undefined;
  getModuleRow: (moduleId: string | null | undefined) => ModuleListItem | undefined;
  getProjectModuleIds: (projectId: string | null | undefined) => string[] | undefined;
  getArchivedModuleIds: (projectId: string | null | undefined) => string[] | undefined;
  getModuleProgress: (moduleId: string | null | undefined) => Progress | undefined;
  getModuleLinks: (moduleId: string | null | undefined) => ModuleLink[] | undefined;
  // fetch actions
  fetchModules: (projectId: string) => Promise<ModuleListItem[] | undefined>;
  fetchArchivedModules: (projectId: string) => Promise<ModuleListItem[] | undefined>;
  fetchModule: (moduleId: string) => Promise<Module | null | undefined>;
  fetchProgress: (moduleId: string) => Promise<Progress | undefined>;
  fetchLinks: (moduleId: string) => Promise<ModuleLink[] | undefined>;
}

/**
 * The modules of the projects of a session (M6 design 3.2, 3.5–3.8, P6a spec 2.2): each project's two lists, by its
 * id, the archived modules and the others; each module's own read, its progress and its links, by its id. A module's
 * fields show as nerve last answered them, of the copies the store holds (v0 design 7.7); its six counts come from its
 * list's row alone, which only a list gives. Every reader takes its project or module from its caller (the route's),
 * never from the address in the router store; the modules of a project the project store no longer gives (deleted,
 * left, or of a workspace no longer the caller's) do not show, nor do their progress and links. Its service sends with
 * the session's client; fetches do not queue. Its changes go one at a time and write nerve's answers into every copy
 * that holds the module or the link (module-changes.ts, v0 design 7.7).
 */
export class ModuleStore implements IModuleStore {
  /** Each project's modules that are not archived, in nerve's order, reconciled between fetches and changes. */
  readonly unarchived = new ReconciledByKey<ModuleListItem[]>();
  /** Each project's archived modules, the latest archived first. */
  readonly archived = new ReconciledByKey<ModuleListItem[]>();
  /** Each module as nerve last answered it alone; null once deleted here (a fetch out meanwhile answers null too). */
  readonly details = new ReconciledByKey<Module | null>();
  /** Each module's live progress and burndown. */
  readonly progress = new ReconciledByKey<Progress>();
  /** Each module's links, the latest added first. */
  readonly links = new ReconciledByKey<ModuleLink[]>();
  // services
  private readonly service: ModulesService;
  /** The changes of the modules and their links, which write nerve's answers into the values above. */
  private readonly changes: ModuleChanges;
  // stores
  /** The project as the caller sees it, by the project store (ProjectStore.getProjectById). */
  private readonly projectOf: (projectId: string | undefined | null) => Project | undefined;

  constructor(rootStore: RootStore, api: ApiClient) {
    this.service = new ModulesService(api);
    this.projectOf = rootStore.projectRoot.project.getProjectById;
    this.changes = new ModuleChanges(this, this.service);
  }

  /**
   * @description the module's fields as nerve last answered them, of its own read and its project's lists' rows;
   * nothing until one of them has it, or once the project store no longer gives its project
   */
  getModuleById = computedFn((moduleId: string | null | undefined): Module | undefined => {
    if (!moduleId) return undefined;
    const read = this.details.get(moduleId);
    const copies: Copy<Module>[] = [
      ...this.rowsOf(moduleId),
      ...(read ? [{ module: read, answeredAt: this.details.answeredAt(moduleId) }] : []),
    ];
    const newest = maxBy(copies, "answeredAt")?.module;
    return newest && this.projectOf(newest.project_id) ? newest : undefined;
  });

  /**
   * @description the module's row of its project's list answered last (its six counts), with the module's fields as
   * nerve last answered them (getModuleById); nothing until a list has it
   */
  getModuleRow = computedFn((moduleId: string | null | undefined): ModuleListItem | undefined => {
    const fields = this.getModuleById(moduleId);
    const row = fields && maxBy(this.rowsOf(fields.id), "answeredAt")?.module;
    return row && { ...row, ...fields };
  });

  /** @description the project's modules that are not archived, in nerve's order; undefined until fetched */
  getProjectModuleIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.listed(this.unarchived, projectId)?.map((row) => row.id)
  );

  /** @description the project's archived modules, the latest archived first; undefined until fetched */
  getArchivedModuleIds = computedFn((projectId: string | null | undefined): string[] | undefined =>
    this.listed(this.archived, projectId)?.map((row) => row.id)
  );

  /** @description the module's progress, of a module the store shows (getModuleById); undefined until fetched */
  getModuleProgress = computedFn((moduleId: string | null | undefined): Progress | undefined => {
    const shown = this.getModuleById(moduleId);
    return shown && this.progress.get(shown.id);
  });

  /** @description the module's links, of a module the store shows (getModuleById); undefined until fetched */
  getModuleLinks = computedFn((moduleId: string | null | undefined): ModuleLink[] | undefined => {
    const shown = this.getModuleById(moduleId);
    return shown && this.links.get(shown.id);
  });

  /**
   * @description fetches the project's modules that are not archived and shows them with the changes nerve confirmed
   * meanwhile; gives what it shows, or undefined for a fetch a newer one overtook or a change of session cut
   * (Reconciled.fetch)
   */
  fetchModules = (projectId: string): Promise<ModuleListItem[] | undefined> =>
    this.unarchived.fetch(projectId, () => this.service.list(projectId));

  /** @description fetches the project's archived modules, as fetchModules */
  fetchArchivedModules = (projectId: string): Promise<ModuleListItem[] | undefined> =>
    this.archived.fetch(projectId, () => this.service.listArchived(projectId));

  /** @description reads the module alone, an archived one too, as fetchModules; null for one deleted meanwhile */
  fetchModule = (moduleId: string): Promise<Module | null | undefined> =>
    this.details.fetch(moduleId, () => this.service.get(moduleId));

  /** @description reads the module's progress, as fetchModules */
  fetchProgress = (moduleId: string): Promise<Progress | undefined> =>
    this.progress.fetch(moduleId, () => this.service.progress(moduleId));

  /** @description reads the module's links, as fetchModules */
  fetchLinks = (moduleId: string): Promise<ModuleLink[] | undefined> =>
    this.links.fetch(moduleId, () => this.service.links(moduleId));

  // changes (module-changes.ts)
  create: IModuleStore["create"] = (projectId, body) => this.changes.create(projectId, body);
  update: IModuleStore["update"] = (moduleId, body) => this.changes.update(moduleId, body);
  changeMembers: IModuleStore["changeMembers"] = (moduleId, change) => this.changes.changeMembers(moduleId, change);
  remove: IModuleStore["remove"] = (moduleId) => this.changes.remove(moduleId);
  archive: IModuleStore["archive"] = (moduleId) => this.changes.archive(moduleId);
  unarchive: IModuleStore["unarchive"] = (moduleId) => this.changes.unarchive(moduleId);
  addLink: IModuleStore["addLink"] = (moduleId, body) => this.changes.addLink(moduleId, body);
  changeLink: IModuleStore["changeLink"] = (linkId, body) => this.changes.changeLink(linkId, body);
  removeLink: IModuleStore["removeLink"] = (linkId) => this.changes.removeLink(linkId);

  /** The project's list in lists (the unarchived or the archived), once fetched, of a project the project store gives. */
  private listed(lists: ReconciledByKey<ModuleListItem[]>, projectId: string | null | undefined) {
    const project = this.projectOf(projectId);
    return project && lists.get(project.id);
  }

  /** The module's rows in the lists the store holds, the archived ones too, each with when nerve answered its list. */
  private rowsOf(moduleId: string): Copy<ModuleListItem>[] {
    return [this.unarchived, this.archived].flatMap((lists) =>
      lists
        .values()
        .flatMap((rows) => rows.filter((row) => row.id === moduleId))
        .map((row) => ({ module: row, answeredAt: lists.answeredAt(row.project_id) }))
    );
  }
}
