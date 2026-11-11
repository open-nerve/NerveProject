/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// types
import type { Cycle } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import type { IModule, IProjectView, IWorkspaceView } from "@nerve/types";
import type { TContextMenuItem } from "@nerve/ui";
// hooks
import { useQuickActionsFactory } from "@/components/common/quick-actions-factory";

// Types
interface UseCycleMenuItemsProps {
  cycle: Cycle;
  /** The caller writes the cycle's project: he is its admin or its member (P5a spec 2.5). */
  canWrite: boolean;
  /** The caller may delete the cycle: the project's admin, or its creator while the project's admin or member (R8). */
  canDelete: boolean;
  handleEdit: () => void;
  handleArchive: () => void;
  handleRestore: () => void;
  handleDelete: () => void;
  handleCopyLink: () => void;
  handleOpenInNewTab: () => void;
}

interface UseModuleMenuItemsProps {
  moduleDetails: IModule | undefined;
  isEditingAllowed: boolean;
  workspaceSlug: string;
  projectId: string;
  moduleId: string;
  handleEdit: () => void;
  handleArchive: () => void;
  handleRestore: () => void;
  handleDelete: () => void;
  handleCopyLink: () => void;
  handleOpenInNewTab: () => void;
}

interface UseViewMenuItemsProps {
  isOwner: boolean;
  isAdmin: boolean;
  workspaceSlug: string;
  projectId?: string;
  view: IProjectView | IWorkspaceView;
  handleEdit: () => void;
  handleDelete: () => void;
  handleCopyLink: () => void;
  handleOpenInNewTab: () => void;
}

/**
 * A cycle's menu (P5a spec 2.5, R8): changing it, to the caller who writes its project while it has not ended and is
 * not archived; opening it in a new tab and copying its link, to anyone; archiving it, to the caller who writes, while
 * it is not archived, the item disabled with the reason until it has ended; restoring it, to the caller who writes,
 * once archived; deleting it, to the caller who may (canDelete), while it has not ended and is not archived. Its state
 * is the status nerve gives it.
 */
export const useCycleMenuItems = (props: UseCycleMenuItemsProps): TContextMenuItem[] => {
  const factory = useQuickActionsFactory();
  const { t } = useTranslation();
  const { cycle, canWrite, canDelete, ...handlers } = props;

  const isArchived = cycle.archived_at !== null;
  const isCompleted = cycle.status === "completed";

  // Assemble final menu items - order defined here
  const items = [
    factory.createEditMenuItem(handlers.handleEdit, canWrite && !isCompleted && !isArchived),
    factory.createOpenInNewTabMenuItem(handlers.handleOpenInNewTab),
    factory.createCopyLinkMenuItem(handlers.handleCopyLink),
    factory.createArchiveMenuItem(handlers.handleArchive, {
      shouldRender: canWrite && !isArchived,
      disabled: !isCompleted,
      description: isCompleted ? undefined : t("project_cycles.archive.only_completed"),
    }),
    factory.createRestoreMenuItem(handlers.handleRestore, canWrite && isArchived),
    factory.createDeleteMenuItem(handlers.handleDelete, canDelete && !isCompleted && !isArchived),
  ].filter((item) => item.shouldRender !== false);

  return items;
};

export const useModuleMenuItems = (props: UseModuleMenuItemsProps): TContextMenuItem[] => {
  const factory = useQuickActionsFactory();
  const { moduleDetails, isEditingAllowed, ...handlers } = props;

  const isArchived = !!moduleDetails?.archived_at;
  const moduleState = moduleDetails?.status?.toLocaleLowerCase();
  const isInArchivableGroup = !!moduleState && ["completed", "cancelled"].includes(moduleState);

  // Assemble final menu items - order defined here
  const items = [
    factory.createEditMenuItem(handlers.handleEdit, isEditingAllowed && !isArchived),
    factory.createOpenInNewTabMenuItem(handlers.handleOpenInNewTab),
    factory.createCopyLinkMenuItem(handlers.handleCopyLink),
    factory.createArchiveMenuItem(handlers.handleArchive, {
      shouldRender: isEditingAllowed && !isArchived,
      disabled: !isInArchivableGroup,
      description: isInArchivableGroup ? undefined : "Only completed or cancelled modules can be archived",
    }),
    factory.createRestoreMenuItem(handlers.handleRestore, isEditingAllowed && isArchived),
    factory.createDeleteMenuItem(handlers.handleDelete, isEditingAllowed && !isArchived),
  ].filter((item) => item.shouldRender !== false);

  return items;
};

export const useViewMenuItems = (props: UseViewMenuItemsProps): TContextMenuItem[] => {
  const factory = useQuickActionsFactory();
  const { workspaceSlug, isOwner, isAdmin, projectId, view, ...handlers } = props;

  if (!view) return [];

  // Assemble final menu items - order defined here
  const items = [
    factory.createEditMenuItem(handlers.handleEdit, isOwner),
    factory.createOpenInNewTabMenuItem(handlers.handleOpenInNewTab),
    factory.createCopyLinkMenuItem(handlers.handleCopyLink),
    factory.createDeleteMenuItem(handlers.handleDelete, isOwner || isAdmin),
  ].filter((item) => item.shouldRender !== false);

  return items;
};
