/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { runInAction } from "mobx";
// nerve imports
import type {
  Module,
  ModuleCreate,
  ModuleLink,
  ModuleLinkCreate,
  ModuleLinkPatch,
  ModuleListItem,
  ModuleUpdate,
} from "@nerve/api-client";
// lib
import { oneAtATime } from "@/lib/one-at-a-time";
import type { Change, ReconciledByKey } from "@/lib/reconciled";
import { dropped, prepended, replaced } from "@/lib/reconciled";
// services
import type { ModulesService } from "@/services/planning/modules.service";

/** Whom a change of a module's members adds and whom it removes: one click of a multiple choice, or a command (R6). */
type MembersChange = { add: string[]; remove: string[] };

export interface IModuleChanges {
  // changes
  create: (projectId: string, body: ModuleCreate) => Promise<Module>;
  update: (moduleId: string, body: ModuleUpdate) => Promise<Module>;
  changeMembers: (moduleId: string, change: MembersChange) => Promise<Module>;
  remove: (moduleId: string) => Promise<void>;
  archive: (moduleId: string) => Promise<Module>;
  unarchive: (moduleId: string) => Promise<Module>;
  addLink: (moduleId: string, body: ModuleLinkCreate) => Promise<ModuleLink>;
  changeLink: (linkId: string, body: ModuleLinkPatch) => Promise<ModuleLink>;
  removeLink: (linkId: string) => Promise<void>;
}

/** What the changes write nerve's answers into and find their module or link in: the module store's (ModuleStore). */
type HeldModules = {
  readonly unarchived: ReconciledByKey<ModuleListItem[]>;
  readonly archived: ReconciledByKey<ModuleListItem[]>;
  readonly details: ReconciledByKey<Module | null>;
  readonly links: ReconciledByKey<ModuleLink[]>;
  getModuleById: (moduleId: string) => Module | undefined;
};

/** A module nerve has just created, as its project's list gives it: it has no work items yet, so these are exact. */
const newRow = (module: Module): ModuleListItem => ({
  ...module,
  total_issues: 0,
  backlog_issues: 0,
  unstarted_issues: 0,
  started_issues: 0,
  completed_issues: 0,
  cancelled_issues: 0,
});

/** The module's row, where the list has it, with its fields as nerve answered them: its counts stay its own (R9). */
const written =
  (module: Module): Change<ModuleListItem[]> =>
  (list) =>
    list.map((row) => (row.id === module.id ? { ...row, ...module } : row));

/**
 * The module moved into a list, with the row it left (its counts): first, as nerve lists it, the latest archived first
 * in the archive; where the list has it already (a repeat), written in its place. Without a row it adds nothing: the
 * list's next fetch brings it.
 */
const landed =
  (module: Module, row: ModuleListItem | undefined): Change<ModuleListItem[]> =>
  (list) =>
    !row || list.some((listed) => listed.id === module.id) ? written(module)(list) : [{ ...row, ...module }, ...list];

/** The module's own read as nerve answered the module; a deleted one's stays deleted. */
const reread =
  (module: Module): Change<Module | null> =>
  (read) =>
    read && module;

/**
 * The members, all of them, that a change makes of those nerve last answered: in their order, but whom it removes,
 * then whom it adds; one who has left the project since stays, as nerve keeps him (member_ids replaces the whole set).
 */
const membersAfter = (members: string[], { add, remove }: MembersChange): string[] => {
  const after = new Set(members);
  for (const memberId of remove) after.delete(memberId);
  for (const memberId of add) after.add(memberId);
  return [...after];
};

/**
 * The changes of the modules and their links (P6a spec 2.2), one at a time (v0 design 7.7). Once nerve confirms a
 * change, its answer is written into every copy the store holds: a module's into its project's lists' rows and its own
 * read, a link's into its module's links; nerve's answers have no counts, so a row keeps its own (R9). A change of the
 * members reckons its whole set in its turn, from the members nerve last answered (R6); a deletion finds its module,
 * and a change or removal of a link its link, in its turn. A refusal fails the change and writes nothing (R11).
 */
export class ModuleChanges implements IModuleChanges {
  /** The changes of the modules and their links, sent one at a time. */
  private readonly queue = oneAtATime();

  constructor(
    private readonly modules: HeldModules,
    private readonly service: ModulesService
  ) {}

  /**
   * @description creates a module: first of its project's list, as nerve lists it (its sort order the lowest), of no
   * work items. Fails, changing nothing, when nerve refuses.
   */
  create = (projectId: string, body: ModuleCreate): Promise<Module> =>
    this.queue(async () => {
      const module = await this.service.create(projectId, body);
      this.modules.unarchived.confirm(module.project_id, prepended([newRow(module)]));
      return module;
    });

  /**
   * @description changes the fields body names as the caller gives them (a form's member_ids, the whole set); the
   * store then shows nerve's answer. Fails, changing nothing, when refused.
   */
  update = (moduleId: string, body: ModuleUpdate): Promise<Module> =>
    this.queue(async () => this.changed(await this.service.update(moduleId, body)));

  /**
   * @description adds and removes members of a module: its whole set is reckoned in the change's turn, from the members
   * nerve last answered, so two quick changes both hold. Fails, changing nothing, as remove.
   */
  changeMembers = (moduleId: string, change: MembersChange): Promise<Module> =>
    this.inTurn(moduleId, async (module) =>
      this.changed(await this.service.update(module.id, { member_ids: membersAfter(module.member_ids, change) }))
    );

  /**
   * @description archives a completed or cancelled module: its row moves first into its project's archive. One
   * archived already is answered as nerve holds it, which the store writes. Fails, changing nothing, when refused.
   */
  archive = (moduleId: string): Promise<Module> =>
    this.queue(async () => this.moved(await this.service.archive(moduleId), "unarchived", "archived"));

  /** @description unarchives a module: its row moves first into its project's list, as archive */
  unarchive = (moduleId: string): Promise<Module> =>
    this.queue(async () => this.moved(await this.service.unarchive(moduleId), "archived", "unarchived"));

  /**
   * @description deletes a module, which the store then no longer gives, nor its progress and links. Fails, changing
   * nothing, when nerve refuses, or asking nerve nothing when the store does not give the module in its turn.
   */
  remove = (moduleId: string): Promise<void> =>
    this.inTurn(moduleId, async (module) => {
      await this.service.remove(module.id);
      runInAction(() => {
        this.modules.unarchived.confirm(module.project_id, dropped(module.id));
        this.modules.archived.confirm(module.project_id, dropped(module.id));
        this.modules.details.confirm(module.id, () => null);
      });
    });

  /**
   * @description adds a link to a module: first of its links, as nerve lists them, its metadata empty until nerve has
   * read it (R10). Fails, changing nothing, when refused.
   */
  addLink = (moduleId: string, body: ModuleLinkCreate): Promise<ModuleLink> =>
    this.queue(async () => {
      const link = await this.service.addLink(moduleId, body);
      this.modules.links.confirm(link.module_id, prepended([link]));
      return link;
    });

  /**
   * @description changes a link's address or title; its module's links then show nerve's answer (a new address's
   * metadata empty). Fails, changing nothing, when nerve refuses, or asking nerve nothing when the store does not give
   * the link in its turn.
   */
  changeLink = (linkId: string, body: ModuleLinkPatch): Promise<ModuleLink> =>
    this.linkInTurn(linkId, async (link) => {
      const changed = await this.service.changeLink(link.id, body);
      this.modules.links.confirm(changed.module_id, replaced(changed));
      return changed;
    });

  /** @description removes a link from its module's links, as changeLink */
  removeLink = (linkId: string): Promise<void> =>
    this.linkInTurn(linkId, async (link) => {
      await this.service.removeLink(link.id);
      this.modules.links.confirm(link.module_id, dropped(link.id));
    });

  /** Sends a change of a module in its turn, with the module as the store gives it then; fails when it gives none. */
  private inTurn<T>(moduleId: string, send: (module: Module) => Promise<T>): Promise<T> {
    return this.queue(() => {
      const module = this.modules.getModuleById(moduleId);
      if (!module) throw new Error("Module not found");
      return send(module);
    });
  }

  /**
   * Sends a change of a link in its turn, with the link as its module's links hold it then; fails when the store holds
   * it no longer, or no longer gives its module.
   */
  private linkInTurn<T>(linkId: string, send: (link: ModuleLink) => Promise<T>): Promise<T> {
    return this.queue(() => {
      const link = this.modules.links
        .values()
        .flat()
        .find((held) => held.id === linkId);
      if (!link || !this.modules.getModuleById(link.module_id)) throw new Error("Link not found");
      return send(link);
    });
  }

  /** Writes the module nerve answered into its rows, in both of its project's lists, and its own read. */
  private changed(module: Module): Module {
    runInAction(() => {
      this.modules.unarchived.confirm(module.project_id, written(module));
      this.modules.archived.confirm(module.project_id, written(module));
      this.modules.details.confirm(module.id, reread(module));
    });
    return module;
  }

  /** Moves the module nerve archived or unarchived from one of its project's lists to the other, with its row. */
  private moved(module: Module, from: "unarchived" | "archived", to: "unarchived" | "archived"): Module {
    runInAction(() => {
      const row = this.modules[from].get(module.project_id)?.find((listed) => listed.id === module.id);
      this.modules[from].confirm(module.project_id, dropped(module.id));
      this.modules[to].confirm(module.project_id, landed(module, row));
      this.modules.details.confirm(module.id, reread(module));
    });
    return module;
  }
}
