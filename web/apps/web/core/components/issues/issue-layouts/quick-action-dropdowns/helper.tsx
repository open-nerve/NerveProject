/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { useMemo } from "react";
import {
  ArchiveOutline,
  CloseCircleOutline,
  CopyOutline,
  DeleteOutline,
  EditOutline,
  LinkOutline,
  NewTabOutline,
  RestoreOutline,
} from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { TContextMenuItem } from "@nerve/ui";
import { generateWorkItemLink } from "@nerve/utils";
import { useCopyLink } from "@/hooks/use-copy-link";

// Generic helper function to handle optional function calls gracefully
// Overload for functions without parameters
function handleOptionalAction(optionalFn: (() => void) | (() => Promise<void>) | undefined, actionName: string): void;

// Overload for functions with one parameter
function handleOptionalAction<T>(
  optionalFn: ((param: T) => void) | ((param: T) => Promise<void>) | undefined,
  actionName: string,
  param: T
): void;

// Implementation
function handleOptionalAction<T>(
  optionalFn: (() => void) | (() => Promise<void>) | ((param: T) => void) | ((param: T) => Promise<void>) | undefined,
  actionName: string,
  param?: T
): void {
  if (optionalFn) {
    if (param !== undefined) {
      (optionalFn as (param: T) => void | Promise<void>)(param);
    } else {
      (optionalFn as () => void | Promise<void>)();
    }
  } else {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: "Action not available",
      message: `${actionName} action is not implemented.`,
    });
  }
}

/**
 * How the menu's edit and copy open the work item's dialog. A list's rows name the dialog's kind (openDialog) to the
 * list's command host, which holds the dialog while the row may leave the page (P5b T13). A page holding a dialog of
 * its own, the work item's detail or the archive, sets the work item to edit, if any, and then opens the dialog.
 */
type DialogOpening =
  | {
      openDialog: (kind: "edit" | "copy") => void;
      setIssueToEdit?: never;
      setCreateUpdateIssueModal?: never;
    }
  | {
      openDialog?: never;
      setIssueToEdit: (issue: Issue | undefined) => void;
      setCreateUpdateIssueModal: (open: boolean) => void;
    };

export type MenuItemFactoryProps = DialogOpening & {
  issue: Issue;
  workspaceSlug?: string;
  projectIdentifier?: string;
  isEditingAllowed: boolean;
  isArchivingAllowed?: boolean;
  isDeletingAllowed: boolean;
  isRestoringAllowed?: boolean;
  isInArchivableGroup?: boolean;
  // Action handlers
  setDeleteIssueModal: (open: boolean) => void;
  setArchiveIssueModal?: (open: boolean) => void;
  /** Removes the work item from the page's cycle (P5b spec 2.6, R6); the menu offers it while given. */
  handleRemoveFromCycle?: () => void;
  /** The caller owns native refusal/success and its UI lifetime; the menu only consumes the promise. */
  handleRestore?: () => Promise<void>;
};

/** Opens the work item's dialog of kind as the menu's caller holds it (DialogOpening). */
function openDialogOf(props: MenuItemFactoryProps, kind: "edit" | "copy") {
  if (props.openDialog) {
    props.openDialog(kind);
    return;
  }
  if (kind === "edit") props.setIssueToEdit(props.issue);
  props.setCreateUpdateIssueModal(true);
}

// Common action handlers hook
const useIssueActionHandlers = (props: MenuItemFactoryProps) => {
  const { issue, workspaceSlug, projectIdentifier, handleRestore } = props;
  const copyLink = useCopyLink();
  const { t } = useTranslation();

  const workItemLink = useMemo(
    () =>
      generateWorkItemLink({
        workspaceSlug,
        projectId: issue?.project_id,
        issueId: issue?.id,
        projectIdentifier,
        sequenceId: issue?.sequence_id,
      }),
    [workspaceSlug, projectIdentifier, issue]
  );

  const handleCopyIssueLink = () => copyLink(workItemLink, t("common.copied_to_clipboard"));

  const handleOpenInNewTab = () => window.open(workItemLink, "_blank", "noopener,noreferrer");

  const handleIssueRestore = async () => {
    if (!handleRestore) {
      handleOptionalAction(handleRestore, "Restore");
      return;
    }
    // Presentation owners already report confirmed outcomes. Never turn their refusal into a success toast.
    await handleRestore().catch(() => undefined);
  };

  return {
    handleCopyIssueLink,
    handleOpenInNewTab,
    handleIssueRestore,
  };
};

const useMenuItemFactory = (props: MenuItemFactoryProps) => {
  const { t } = useTranslation();
  const actionHandlers = useIssueActionHandlers(props);

  const {
    isEditingAllowed,
    isArchivingAllowed = false,
    isDeletingAllowed,
    isRestoringAllowed = false,
    isInArchivableGroup = false,
    setDeleteIssueModal,
    setArchiveIssueModal,
    handleRemoveFromCycle,
  } = props;

  const createEditMenuItem = (): TContextMenuItem => ({
    key: "edit",
    title: t("common.actions.edit"),
    icon: EditOutline,
    action: () => openDialogOf(props, "edit"),
    shouldRender: isEditingAllowed,
  });

  const createCopyMenuItem = (): TContextMenuItem => ({
    key: "make-a-copy",
    title: t("common.actions.make_a_copy"),
    icon: CopyOutline,
    action: () => openDialogOf(props, "copy"),
    shouldRender: isEditingAllowed,
  });

  const createOpenInNewTabMenuItem = (): TContextMenuItem => ({
    key: "open-in-new-tab",
    title: t("common.actions.open_in_new_tab"),
    icon: NewTabOutline,
    action: actionHandlers.handleOpenInNewTab,
  });

  const createCopyLinkMenuItem = (): TContextMenuItem => ({
    key: "copy-link",
    title: t("common.actions.copy_link"),
    icon: LinkOutline,
    action: actionHandlers.handleCopyIssueLink,
  });

  const createRemoveFromCycleMenuItem = (): TContextMenuItem => ({
    key: "remove-from-cycle",
    title: t("issue.remove.cycle.label"),
    icon: CloseCircleOutline,
    action: () => handleRemoveFromCycle?.(),
    shouldRender: handleRemoveFromCycle !== undefined,
  });

  const createArchiveMenuItem = (): TContextMenuItem => ({
    key: "archive",
    title: t("common.actions.archive"),
    icon: ArchiveOutline,
    className: "items-start",
    iconClassName: "mt-1",
    action: () => handleOptionalAction(setArchiveIssueModal, "Archive", true),
    shouldRender: isArchivingAllowed && isInArchivableGroup,
  });

  const createRestoreMenuItem = (): TContextMenuItem => ({
    key: "restore",
    title: "Restore",
    icon: RestoreOutline,
    action: actionHandlers.handleIssueRestore,
    shouldRender: isRestoringAllowed,
  });

  const createDeleteMenuItem = (): TContextMenuItem => ({
    key: "delete",
    title: t("common.actions.delete"),
    icon: DeleteOutline,
    action: () => {
      setDeleteIssueModal(true);
    },
    shouldRender: isDeletingAllowed,
  });

  return {
    createEditMenuItem,
    createCopyMenuItem,
    createOpenInNewTabMenuItem,
    createCopyLinkMenuItem,
    createRemoveFromCycleMenuItem,
    createArchiveMenuItem,
    createRestoreMenuItem,
    createDeleteMenuItem,
  };
};

// Predefined menu item sets for different contexts
export const useProjectIssueMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return useMemo(
    () => [
      factory.createEditMenuItem(),
      factory.createCopyMenuItem(),
      factory.createOpenInNewTabMenuItem(),
      factory.createCopyLinkMenuItem(),
      factory.createRemoveFromCycleMenuItem(),
      factory.createArchiveMenuItem(),
      factory.createDeleteMenuItem(),
    ],
    [factory]
  );
};

export const useWorkItemDetailMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return useMemo(
    () => [
      factory.createCopyMenuItem(),
      factory.createOpenInNewTabMenuItem(),
      factory.createArchiveMenuItem(),
      factory.createRestoreMenuItem(),
      factory.createDeleteMenuItem(),
    ],
    // oxlint-disable-next-line eslint-plugin-react-hooks/exhaustive-deps
    [factory]
  );
};

export const useArchivedIssueMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return useMemo(
    () => [
      factory.createRestoreMenuItem(),
      factory.createOpenInNewTabMenuItem(),
      factory.createCopyLinkMenuItem(),
      factory.createDeleteMenuItem(),
    ],
    [factory]
  );
};
