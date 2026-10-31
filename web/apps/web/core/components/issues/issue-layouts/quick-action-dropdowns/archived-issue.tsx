/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useTranslation } from "@nerve/i18n";
import { observer } from "mobx-react";
import { useParams } from "react-router";
// ui
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useUser, useUserPermissions } from "@/hooks/store/user";
// local imports
import type { IQuickActionProps } from "../list/list-view-types";
import type { MenuItemFactoryProps } from "./helper";
import { useArchivedIssueMenuItems } from "./helper";
import { useDeleteRequest, useRestoreRequest } from "../command-host";

export const ArchivedIssueQuickActions = observer(function ArchivedIssueQuickActions(props: IQuickActionProps) {
  const { t } = useTranslation();
  const {
    issue,
    handleDelete,
    handleRestore,
    customActionButton,
    portalElement,
    placements = "bottom-end",
    parentRef,
  } = props;
  const requestDelete = useDeleteRequest();
  // router
  const { workspaceSlug } = useParams();
  const requestRestore = useRestoreRequest();
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { data: currentUser } = useUser();

  // derived values
  // auth
  const isProjectMember =
    !!workspaceSlug &&
    allowPermissions(
      [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
      EUserPermissionsLevel.PROJECT,
      workspaceSlug,
      issue.project_id
    );
  const isRestoringAllowed = !!handleRestore && !!requestRestore && isProjectMember;
  const isDeletingAllowed =
    isProjectMember &&
    (issue.created_by_id === currentUser?.id ||
      allowPermissions([EUserPermissions.ADMIN], EUserPermissionsLevel.PROJECT, workspaceSlug, issue.project_id));

  // Menu items and modals using helper
  const menuItemProps: MenuItemFactoryProps = {
    issue,
    workspaceSlug,
    isEditingAllowed: false,
    isDeletingAllowed,
    isRestoringAllowed: !!isRestoringAllowed,
    setIssueToEdit: () => {},
    setCreateUpdateIssueModal: () => {},
    setDeleteIssueModal: () => {
      if (!requestDelete) throw new Error("Archived issue actions require an IssueListCommandHost");
      requestDelete(issue, handleDelete);
    },
    handleRestore: handleRestore && requestRestore ? () => requestRestore(issue, handleRestore) : undefined,
  };

  const MENU_ITEMS = useArchivedIssueMenuItems(menuItemProps);

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
      <ContextMenu parentRef={parentRef} items={CONTEXT_MENU_ITEMS} />
      <CustomMenu
        ellipsis
        customButton={customActionButton}
        portalElement={portalElement}
        placement={placements}
        ariaLabel={`${t("common.options")}: ${issue.name}`}
        menuItemsClassName="z-20"
        maxHeight="lg"
        closeOnSelect
      >
        {MENU_ITEMS.map((item) => {
          if (item.shouldRender === false) return null;
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
