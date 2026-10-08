/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { enableStaticRendering } from "mobx-react";
// nerve imports
import type { ApiClient } from "@nerve/api-client";
import type { IWorkItemFilterStore } from "@nerve/shared-state";
import { WorkItemFilterStore } from "@nerve/shared-state";
// store
import type { IPowerKStore } from "@/store/power-k.store";
import { PowerKStore } from "@/store/power-k.store";
import type { IStateStore } from "@/store/state.store";
import { StateStore } from "@/store/state.store";
import type { ICommandPaletteStore } from "@/store/command-palette.store";
import { CommandPaletteStore } from "@/store/command-palette.store";
import { WorkspaceRootStore } from "@/store/workspace";
// stores
import type { ICycleStore } from "./cycle.store";
import { CycleStore } from "./cycle.store";
import type { ICycleFilterStore } from "./cycle_filter.store";
import { CycleFilterStore } from "./cycle_filter.store";
import type { IEditorAssetStore } from "./editor/asset.store";
import { EditorAssetStore } from "./editor/asset.store";
import type { IFavoriteStore } from "./favorite.store";
import { FavoriteStore } from "./favorite.store";
import type { IGlobalViewStore } from "./global-view.store";
import { GlobalViewStore } from "./global-view.store";
import type { IProjectInboxStore } from "./inbox/project-inbox.store";
import { ProjectInboxStore } from "./inbox/project-inbox.store";
import type { IInstanceStore } from "./instance.store";
import { InstanceStore } from "./instance.store";
import type { IIssueRootStore } from "./issue/root.store";
import { IssueRootStore } from "./issue/root.store";
import type { ILabelStore } from "./label.store";
import { LabelStore } from "./label.store";
import type { IMemberRootStore } from "./member";
import { MemberRootStore } from "./member";
import type { IModuleStore } from "./module.store";
import { ModulesStore } from "./module.store";
import type { IModuleFilterStore } from "./module_filter.store";
import { ModuleFilterStore } from "./module_filter.store";
import type { IWorkspaceNotificationStore } from "./notifications/workspace-notifications.store";
import { WorkspaceNotificationStore } from "./notifications/workspace-notifications.store";
import type { IProjectRootStore } from "./project";
import { ProjectRootStore } from "./project";
import type { IProjectViewStore } from "./project-view.store";
import { ProjectViewStore } from "./project-view.store";
import type { IRouterStore } from "./router.store";
import { RouterStore } from "./router.store";
import type { IThemeStore } from "./theme.store";
import { ThemeStore } from "./theme.store";
import type { IUserStore } from "./user";
import { UserStore } from "./user";
import type { IWorkspaceRootStore } from "./workspace";

enableStaticRendering(typeof window === "undefined");

export class RootStore {
  workspaceRoot: IWorkspaceRootStore;
  projectRoot: IProjectRootStore;
  memberRoot: IMemberRootStore;
  cycle: ICycleStore;
  cycleFilter: ICycleFilterStore;
  module: IModuleStore;
  moduleFilter: IModuleFilterStore;
  projectView: IProjectViewStore;
  globalView: IGlobalViewStore;
  issue: IIssueRootStore;
  state: IStateStore;
  label: ILabelStore;
  router: IRouterStore;
  commandPalette: ICommandPaletteStore;
  theme: IThemeStore;
  instance: IInstanceStore;
  user: IUserStore;
  projectInbox: IProjectInboxStore;
  workspaceNotification: IWorkspaceNotificationStore;
  favorite: IFavoriteStore;
  editorAssetStore: IEditorAssetStore;
  workItemFilters: IWorkItemFilterStore;
  powerK: IPowerKStore;

  /**
   * The stores of one session, the one api is bound to: store-context.tsx builds a RootStore for each session
   * of the tab. The services the stores build on nerve's API send through api, and a store reaches the others
   * through its own RootStore, so the stores act for that session only, even after the tab has moved on to
   * another (M2 design 7.1). The instance's information, the address's parameters and the sidebars are the
   * page's, not the account's: a RootStore built for the next session goes on with those of `before`. Nothing
   * would fetch the instance's information again until the next page load.
   */
  constructor(api: ApiClient, before?: Pick<RootStore, "instance" | "router" | "theme">) {
    this.router = before?.router ?? new RouterStore();
    this.commandPalette = new CommandPaletteStore();
    this.instance = before?.instance ?? new InstanceStore();
    this.user = new UserStore(this, api);
    this.theme = before?.theme ?? new ThemeStore();
    this.workspaceRoot = new WorkspaceRootStore(this, api);
    this.projectRoot = new ProjectRootStore(this, api);
    this.memberRoot = new MemberRootStore(this, api);
    this.cycle = new CycleStore(this);
    this.cycleFilter = new CycleFilterStore(this);
    this.module = new ModulesStore(this);
    this.moduleFilter = new ModuleFilterStore(this);
    this.projectView = new ProjectViewStore(this);
    this.globalView = new GlobalViewStore(this);
    this.issue = new IssueRootStore(this, api);
    this.state = new StateStore(this, api);
    this.label = new LabelStore(this, api);
    this.projectInbox = new ProjectInboxStore(this);
    this.workspaceNotification = new WorkspaceNotificationStore(this);
    this.favorite = new FavoriteStore(this);
    this.editorAssetStore = new EditorAssetStore();
    this.workItemFilters = new WorkItemFilterStore();
    this.powerK = new PowerKStore();
  }

  /**
   * Releases the project filters' reaction to the address's workspace: it is registered on the page's RouterStore,
   * which goes on with the next session (v0 design 7.7, M3 design 7.1). store-context.tsx calls it as the next
   * session's RootStore takes over, so that a retired session's project filters no longer follow the address. It
   * releases nothing else yet: the cycle and module filters' reactions and the issue root's autorun also follow the
   * RouterStore, and run on in a retired session until M6 and M4 release them here (M3 design 7.1, 13.2).
   */
  dispose(): void {
    this.projectRoot.projectFilter.dispose();
  }
}
