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

/** The toast of a menu item whose caller gave it no action, in the page's language (P8 spec 2.3). */
type UnavailableToast = { title: string; message: string };

/** The UnavailableToast of the action that actionKey names, in t's language. */
const unavailable = (t: ReturnType<typeof useTranslation>["t"], actionKey: string): UnavailableToast => ({
  title: t("issue.action_unavailable.title"),
  message: t("issue.action_unavailable.message", { action: t(actionKey) }),
});

// Generic helper function to handle optional function calls gracefully: without the function, the toast its caller
// translated
// Overload for functions without parameters
function handleOptionalAction(
  optionalFn: (() => void) | (() => Promise<void>) | undefined,
  toast: UnavailableToast
): void;

// Overload for functions with one parameter
function handleOptionalAction<T>(
  optionalFn: ((param: T) => void) | ((param: T) => Promise<void>) | undefined,
  toast: UnavailableToast,
  param: T
): void;

// Implementation
function handleOptionalAction<T>(
  optionalFn: (() => void) | (() => Promise<void>) | ((param: T) => void) | ((param: T) => Promise<void>) | undefined,
  toast: UnavailableToast,
  param?: T
): void {
  if (optionalFn) {
    if (param !== undefined) {
      (optionalFn as (param: T) => void | Promise<void>)(param);
    } else {
      (optionalFn as () => void | Promise<void>)();
    }
  } else {
    setToast({ type: TOAST_TYPE.ERROR, ...toast });
  }
}

/**
 * How the menu's edit and copy open the work item's dialog. A list's rows name the dialog's kind (openDialog) to the
 * list's command host, which holds the dialog while the row may leave the page (P5b T13). A page holding a dialog of
 * its own, the work item's detail, sets the work item to edit, if any, and then opens the dialog.
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

/** The menu's edit and copy: offered while the caller may edit the work item, each opening its dialog. */
type Editing = DialogOpening & { isEditingAllowed: boolean };

/**
 * What a menu whose items open no dialog is given (the archive's row menu, P7c spec 2.6): what every item reads but
 * the edit and the copy (Editing).
 */
export type ArchivedIssueMenuProps = {
  issue: Issue;
  workspaceSlug?: string;
  projectIdentifier?: string;
  isArchivingAllowed?: boolean;
  isDeletingAllowed: boolean;
  isRestoringAllowed?: boolean;
  isInArchivableGroup?: boolean;
  // Action handlers
  setDeleteIssueModal: (open: boolean) => void;
  setArchiveIssueModal?: (open: boolean) => void;
  /**
   * The removal of the work item from the page's cycle or module (P5b spec 2.6, R6; P6b spec 2.7): the label its domain
   * names it by (an i18n key), and its sending; the menu offers it while given.
   */
  removeFromView?: { label: string; remove: () => void };
  /** The caller owns native refusal/success and its UI lifetime; the menu only consumes the promise. */
  handleRestore?: () => Promise<void>;
};

/** What a menu that edits and copies the work item is given. */
export type MenuItemFactoryProps = Editing & ArchivedIssueMenuProps;

/** Opens the dialog of kind for issue as the menu's caller holds it (DialogOpening). */
function openDialogOf(dialog: DialogOpening, issue: Issue, kind: "edit" | "copy") {
  if (dialog.openDialog) {
    dialog.openDialog(kind);
    return;
  }
  if (kind === "edit") dialog.setIssueToEdit(issue);
  dialog.setCreateUpdateIssueModal(true);
}

// Common action handlers hook
const useIssueActionHandlers = (props: ArchivedIssueMenuProps) => {
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
      handleOptionalAction(handleRestore, unavailable(t, "common.actions.restore"));
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

const useMenuItemFactory = (props: ArchivedIssueMenuProps) => {
  const { t } = useTranslation();
  const actionHandlers = useIssueActionHandlers(props);

  const {
    issue,
    isArchivingAllowed = false,
    isDeletingAllowed,
    isRestoringAllowed = false,
    isInArchivableGroup = false,
    setDeleteIssueModal,
    setArchiveIssueModal,
    removeFromView,
  } = props;

  const createEditMenuItem = (editing: Editing): TContextMenuItem => ({
    key: "edit",
    title: t("common.actions.edit"),
    icon: EditOutline,
    action: () => openDialogOf(editing, issue, "edit"),
    shouldRender: editing.isEditingAllowed,
  });

  const createCopyMenuItem = (editing: Editing): TContextMenuItem => ({
    key: "make-a-copy",
    title: t("common.actions.make_a_copy"),
    icon: CopyOutline,
    action: () => openDialogOf(editing, issue, "copy"),
    shouldRender: editing.isEditingAllowed,
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

  const createRemoveFromViewMenuItem = (): TContextMenuItem => ({
    key: "remove-from-view",
    title: removeFromView ? t(removeFromView.label) : "",
    icon: CloseCircleOutline,
    action: () => removeFromView?.remove(),
    shouldRender: removeFromView !== undefined,
  });

  const createArchiveMenuItem = (): TContextMenuItem => ({
    key: "archive",
    title: t("common.actions.archive"),
    icon: ArchiveOutline,
    className: "items-start",
    iconClassName: "mt-1",
    action: () => handleOptionalAction(setArchiveIssueModal, unavailable(t, "common.actions.archive"), true),
    shouldRender: isArchivingAllowed && isInArchivableGroup,
  });

  const createRestoreMenuItem = (): TContextMenuItem => ({
    key: "restore",
    title: t("common.actions.restore"),
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
    createRemoveFromViewMenuItem,
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
      factory.createEditMenuItem(props),
      factory.createCopyMenuItem(props),
      factory.createOpenInNewTabMenuItem(),
      factory.createCopyLinkMenuItem(),
      factory.createRemoveFromViewMenuItem(),
      factory.createArchiveMenuItem(),
      factory.createDeleteMenuItem(),
    ],
    [factory, props]
  );
};

export const useWorkItemDetailMenuItems = (props: MenuItemFactoryProps): TContextMenuItem[] => {
  const factory = useMenuItemFactory(props);

  return useMemo(
    () => [
      factory.createCopyMenuItem(props),
      factory.createOpenInNewTabMenuItem(),
      factory.createArchiveMenuItem(),
      factory.createRestoreMenuItem(),
      factory.createDeleteMenuItem(),
    ],
    [factory, props]
  );
};

export const useArchivedIssueMenuItems = (props: ArchivedIssueMenuProps): TContextMenuItem[] => {
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
