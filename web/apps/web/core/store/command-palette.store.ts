/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observable, action, computed, makeObservable, runInAction } from "mobx";
import { computedFn } from "mobx-utils";
// plane imports
import type { TCreateModalStoreTypes } from "@plane/constants";
import type { TProfileSettingsTabs } from "@plane/types";
import { EIssuesStoreType } from "@plane/types";
// lib
import { store } from "@/lib/store-context";

export interface ICommandPaletteStore {
  // computed
  isAnyModalOpen: boolean;
  // observables
  isCreateProjectModalOpen: boolean;
  isCreateCycleModalOpen: boolean;
  isCreateModuleModalOpen: boolean;
  isCreateViewModalOpen: boolean;
  isCreateIssueModalOpen: boolean;
  isDeleteIssueModalOpen: boolean;
  isBulkDeleteIssueModalOpen: boolean;
  createIssueStoreType: TCreateModalStoreTypes;
  createWorkItemAllowedProjectIds: string[] | undefined;
  profileSettingsModal: {
    activeTab: TProfileSettingsTabs | null;
    isOpen: boolean;
  };
  projectListOpenMap: Record<string, boolean>;
  getIsProjectListOpen: (projectId: string) => boolean;
  // toggle actions
  toggleCreateProjectModal: (value?: boolean) => void;
  toggleCreateCycleModal: (value?: boolean) => void;
  toggleCreateViewModal: (value?: boolean) => void;
  toggleCreateIssueModal: (value?: boolean, storeType?: TCreateModalStoreTypes, allowedProjectIds?: string[]) => void;
  toggleCreateModuleModal: (value?: boolean) => void;
  toggleDeleteIssueModal: (value?: boolean) => void;
  toggleBulkDeleteIssueModal: (value?: boolean) => void;
  toggleProjectListOpen: (projectId: string, value?: boolean) => void;
  toggleProfileSettingsModal: (value: { activeTab?: TProfileSettingsTabs | null; isOpen?: boolean }) => void;
}

export class CommandPaletteStore implements ICommandPaletteStore {
  // observables
  isCreateProjectModalOpen: boolean = false;
  isCreateCycleModalOpen: boolean = false;
  isCreateModuleModalOpen: boolean = false;
  isCreateViewModalOpen: boolean = false;
  isCreateIssueModalOpen: boolean = false;
  isDeleteIssueModalOpen: boolean = false;
  isBulkDeleteIssueModalOpen: boolean = false;
  createIssueStoreType: TCreateModalStoreTypes = EIssuesStoreType.PROJECT;
  createWorkItemAllowedProjectIds: ICommandPaletteStore["createWorkItemAllowedProjectIds"] = undefined;
  profileSettingsModal: ICommandPaletteStore["profileSettingsModal"] = {
    activeTab: "general",
    isOpen: false,
  };
  projectListOpenMap: Record<string, boolean> = {};

  constructor() {
    makeObservable(this, {
      // observable
      isCreateProjectModalOpen: observable.ref,
      isCreateCycleModalOpen: observable.ref,
      isCreateModuleModalOpen: observable.ref,
      isCreateViewModalOpen: observable.ref,
      isCreateIssueModalOpen: observable.ref,
      isDeleteIssueModalOpen: observable.ref,
      isBulkDeleteIssueModalOpen: observable.ref,
      createIssueStoreType: observable,
      createWorkItemAllowedProjectIds: observable,
      profileSettingsModal: observable,
      projectListOpenMap: observable,
      // toggle actions
      toggleCreateProjectModal: action,
      toggleCreateCycleModal: action,
      toggleCreateViewModal: action,
      toggleCreateIssueModal: action,
      toggleCreateModuleModal: action,
      toggleDeleteIssueModal: action,
      toggleBulkDeleteIssueModal: action,
      toggleProjectListOpen: action,
      toggleProfileSettingsModal: action,
      isAnyModalOpen: computed,
    });
  }

  get isAnyModalOpen(): boolean {
    return Boolean(this.getCoreModalsState());
  }

  /**
   * Returns whether any base modal is open
   * @protected - allows access from child classes
   */
  protected getCoreModalsState(): boolean {
    return Boolean(
      this.isCreateIssueModalOpen ||
      this.isCreateCycleModalOpen ||
      this.isCreateProjectModalOpen ||
      this.isCreateModuleModalOpen ||
      this.isCreateViewModalOpen ||
      store.powerK.isShortcutsListModalOpen ||
      this.isBulkDeleteIssueModalOpen ||
      this.isDeleteIssueModalOpen
    );
  }
  // computedFn
  getIsProjectListOpen = computedFn((projectId: string) => this.projectListOpenMap[projectId]);

  /**
   * Toggles the project list open state
   * @param projectId
   * @param value
   */
  toggleProjectListOpen = (projectId: string, value?: boolean) => {
    if (value !== undefined) this.projectListOpenMap[projectId] = value;
    else this.projectListOpenMap[projectId] = !this.projectListOpenMap[projectId];
  };

  /**
   * Toggles the create project modal
   * @param value
   * @returns
   */
  toggleCreateProjectModal = (value?: boolean) => {
    if (value !== undefined) {
      this.isCreateProjectModalOpen = value;
    } else {
      this.isCreateProjectModalOpen = !this.isCreateProjectModalOpen;
    }
  };

  /**
   * Toggles the create cycle modal
   * @param value
   * @returns
   */
  toggleCreateCycleModal = (value?: boolean) => {
    if (value !== undefined) {
      this.isCreateCycleModalOpen = value;
    } else {
      this.isCreateCycleModalOpen = !this.isCreateCycleModalOpen;
    }
  };

  /**
   * Toggles the create view modal
   * @param value
   * @returns
   */
  toggleCreateViewModal = (value?: boolean) => {
    if (value !== undefined) {
      this.isCreateViewModalOpen = value;
    } else {
      this.isCreateViewModalOpen = !this.isCreateViewModalOpen;
    }
  };

  /**
   * Toggles the create issue modal
   * @param value
   * @param storeType
   * @returns
   */
  toggleCreateIssueModal = (value?: boolean, storeType?: TCreateModalStoreTypes, allowedProjectIds?: string[]) => {
    if (value !== undefined) {
      this.isCreateIssueModalOpen = value;
      this.createIssueStoreType = storeType || EIssuesStoreType.PROJECT;
      this.createWorkItemAllowedProjectIds = allowedProjectIds ?? undefined;
    } else {
      this.isCreateIssueModalOpen = !this.isCreateIssueModalOpen;
      this.createIssueStoreType = EIssuesStoreType.PROJECT;
      this.createWorkItemAllowedProjectIds = undefined;
    }
  };

  /**
   * Toggles the delete issue modal
   * @param value
   * @returns
   */
  toggleDeleteIssueModal = (value?: boolean) => {
    if (value !== undefined) {
      this.isDeleteIssueModalOpen = value;
    } else {
      this.isDeleteIssueModalOpen = !this.isDeleteIssueModalOpen;
    }
  };

  /**
   * Toggles the create module modal
   * @param value
   * @returns
   */
  toggleCreateModuleModal = (value?: boolean) => {
    if (value !== undefined) {
      this.isCreateModuleModalOpen = value;
    } else {
      this.isCreateModuleModalOpen = !this.isCreateModuleModalOpen;
    }
  };

  /**
   * Toggles the bulk delete issue modal
   * @param value
   * @returns
   */
  toggleBulkDeleteIssueModal = (value?: boolean) => {
    if (value !== undefined) {
      this.isBulkDeleteIssueModalOpen = value;
    } else {
      this.isBulkDeleteIssueModalOpen = !this.isBulkDeleteIssueModalOpen;
    }
  };

  /**
   * Toggles the profile settings modal
   * @param value
   * @returns
   */
  toggleProfileSettingsModal: ICommandPaletteStore["toggleProfileSettingsModal"] = (payload) => {
    const updatedSettings: ICommandPaletteStore["profileSettingsModal"] = {
      ...this.profileSettingsModal,
      ...payload,
    };

    runInAction(() => {
      this.profileSettingsModal = updatedSettings;
    });
  };
}
