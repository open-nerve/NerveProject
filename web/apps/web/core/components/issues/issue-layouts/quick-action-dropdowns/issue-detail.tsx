/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue } from "@nerve/api-client";

import { useState } from "react";
import { pick } from "lodash-es";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import { ARCHIVABLE_STATE_GROUPS, EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";

import { EIssuesStoreType } from "@nerve/types";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useUserPermissions } from "@/hooks/store/user";
// helper
import { ArchiveIssueModal } from "../../archive-issue-modal";
import { DeleteIssueModal } from "../../delete-issue-modal";
import { CreateUpdateIssueModal } from "../../issue-modal/modal";
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useWorkItemDetailMenuItems } from "./helper";

type TWorkItemDetailQuickActionProps = IQuickActionProps & {
  toggleEditIssueModal?: (value: boolean) => void;
  toggleDeleteIssueModal?: (value: boolean) => void;
  toggleArchiveIssueModal?: (value: boolean) => void;
  isPeekMode?: boolean;
  onArchiveConfirmed?: () => void;
};

export const WorkItemDetailQuickActions = observer(function WorkItemDetailQuickActions(
  props: TWorkItemDetailQuickActionProps
) {
  const { t } = useTranslation();
  const {
    issue,
    handleDelete,
    handleUpdate,
    handleArchive,
    onArchiveConfirmed,
    handleRestore,
    portalElement,
    readOnly = false,
    placements = "bottom-end",
    parentRef,
    toggleEditIssueModal,
    toggleDeleteIssueModal,
    toggleArchiveIssueModal,
    isPeekMode = false,
  } = props;
  // router
  const { workspaceSlug } = useParams();
  // states
  const [createUpdateIssueModal, setCreateUpdateIssueModal] = useState(false);
  const [issueToEdit, setIssueToEdit] = useState<Issue | undefined>(undefined);
  const [deleteIssueModal, setDeleteIssueModal] = useState(false);
  const [archiveIssueModal, setArchiveIssueModal] = useState(false);
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { getStateById } = useProjectState();
  const { getProjectIdentifierById } = useProject();
  // derived values
  const stateDetails = getStateById(issue.state_id);
  const projectIdentifier = getProjectIdentifierById(issue?.project_id);
  // auth
  const isEditingAllowed =
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      issue.project_id ?? undefined
    ) && !readOnly;

  const isArchivingAllowed = !issue.archived_at && isEditingAllowed;
  const isInArchivableGroup = !!stateDetails && ARCHIVABLE_STATE_GROUPS.includes(stateDetails?.group);
  const isRestoringAllowed = !!issue.archived_at && isEditingAllowed;

  const isDeletingAllowed = isEditingAllowed;

  const duplicateIssuePayload = pick(
    {
      ...issue,
      name: `${issue.name} (copy)`,
    },
    [
      "name",
      "project_id",
      "state_id",
      "description_html",
      "parent_id",
      "priority",
      "assignee_ids",
      "label_ids",
      "start_date",
      "target_date",
    ]
  );

  const customEditAction = () => {
    setCreateUpdateIssueModal(true);
    if (toggleEditIssueModal) toggleEditIssueModal(true);
  };

  const customDeleteAction = async () => {
    setDeleteIssueModal(true);
    if (toggleDeleteIssueModal) toggleDeleteIssueModal(true);
  };

  const customArchiveAction = async () => {
    setArchiveIssueModal(true);
    if (toggleArchiveIssueModal) toggleArchiveIssueModal(true);
  };

  const customRestoreAction = async () => {
    if (handleRestore) await handleRestore();
  };

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug,
    projectIdentifier,
    isEditingAllowed,
    isArchivingAllowed,
    isRestoringAllowed,
    isDeletingAllowed,
    isInArchivableGroup,
    setIssueToEdit,
    setCreateUpdateIssueModal: customEditAction,
    setDeleteIssueModal: customDeleteAction,
    setArchiveIssueModal: customArchiveAction,
    handleRestore: customRestoreAction,
  };

  //   const MENU_ITEMS = useWorkItemDetailMenuItems(menuItemProps);
  const baseMenuItems = useWorkItemDetailMenuItems(menuItemProps);

  const MENU_ITEMS = baseMenuItems
    // oxlint-disable-next-line oxc/no-map-spread
    .map((item) => {
      // Customize edit action for work item
      if (item.key === "edit") {
        return {
          ...item,
          shouldRender: isEditingAllowed && !isPeekMode,
        };
      }
      // Customize delete action for work item
      if (item.key === "delete") {
        return {
          ...item,
        };
      }
      // Hide copy link in peek mode
      if (item.key === "copy-link") {
        return {
          ...item,
          shouldRender: !isPeekMode,
        };
      }
      return item;
    })
    .filter(function MENU_ITEMS(item) {
      return item.shouldRender !== false;
    });

  const CONTEXT_MENU_ITEMS = MENU_ITEMS.map(function CONTEXT_MENU_ITEMS(item) {
    return {
      ...item,

      onClick: () => {
        item.action();
      },
    };
  });

  return (
    <>
      {/* Modals */}
      {handleArchive && (
        <ArchiveIssueModal
          data={issue}
          isOpen={archiveIssueModal}
          handleClose={() => {
            setArchiveIssueModal(false);
            if (toggleArchiveIssueModal) toggleArchiveIssueModal(false);
          }}
          onSubmit={handleArchive}
          onArchived={onArchiveConfirmed}
        />
      )}
      <DeleteIssueModal
        data={issue}
        isOpen={deleteIssueModal}
        handleClose={() => {
          setDeleteIssueModal(false);
          if (toggleDeleteIssueModal) toggleDeleteIssueModal(false);
        }}
        onSubmit={handleDelete}
      />
      <CreateUpdateIssueModal
        isOpen={createUpdateIssueModal}
        onClose={() => {
          setCreateUpdateIssueModal(false);
          setIssueToEdit(undefined);
          if (toggleEditIssueModal) toggleEditIssueModal(false);
        }}
        data={issueToEdit ?? duplicateIssuePayload}
        sourceIssueId={issueToEdit ? undefined : issue.id}
        onSubmit={async (data) => {
          if (issueToEdit && handleUpdate) await handleUpdate(data);
        }}
        storeType={EIssuesStoreType.PROJECT}
        fetchIssueDetails={false}
      />

      <ContextMenu parentRef={parentRef} items={CONTEXT_MENU_ITEMS} />
      <CustomMenu
        ellipsis
        placement={placements}
        customButton={<MoreHorizontalOutline className="size-4" />}
        customButtonClassName="grid size-7 place-items-center rounded-sm hover:bg-layer-1"
        portalElement={portalElement}
        ariaLabel={`${t("common.options")}: ${issue.name}`}
        menuItemsClassName="z-20"
        maxHeight="lg"
        closeOnSelect
      >
        {MENU_ITEMS.map((item) => {
          if (item.shouldRender === false) return null;

          // Render regular menu item
          return (
            <CustomMenu.MenuItem
              key={item.key}
              onClick={() => {
                item.action();
              }}
              className={cn(
                "flex items-center gap-2",
                {
                  "text-placeholder": item.disabled,
                },
                item.className
              )}
              disabled={item.disabled}
            >
              {item.icon && <item.icon className={cn("h-3 w-3", item.iconClassName)} />}
              <div>
                <h5>{item.title}</h5>
                {item.description && (
                  <p
                    className={cn("whitespace-pre-line text-tertiary", {
                      "text-placeholder": item.disabled,
                    })}
                  >
                    {item.description}
                  </p>
                )}
              </div>
            </CustomMenu.MenuItem>
          );
        })}
      </CustomMenu>
    </>
  );
});
