/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { observer } from "mobx-react";
import { MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve imports
import type { View } from "@nerve/api-client";
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useViewMenuItems } from "@/components/common/quick-actions-helper";
import { useViews } from "@/hooks/store/use-views";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { useCopyLink } from "@/hooks/use-copy-link";
// local imports
import { useViewsPage } from "./actions-host";

type Props = {
  parentRef: RefObject<HTMLElement | null>;
  viewId: string;
  projectId: string;
  workspaceSlug: string;
  customClassName?: string;
};

/**
 * A view's quick actions (P6a spec 2.3, R17): its menu, by its button, named by the view, and by a right-click on its
 * row (parentRef); none while the store does not give the view. Its dialogs are its page's (ViewActionsHost), and what
 * follows a deletion is the page's while it is mounted, though the deletion takes the row away.
 */
export const ViewQuickActions = observer(function ViewQuickActions(props: Props) {
  const { getViewById } = useViews();
  const view = getViewById(props.viewId);
  return view ? <ViewMenu {...props} view={view} /> : null;
});

/**
 * The menu of the view of the project of projectId (useViewMenuItems): what the caller may do with it, as nerve decides
 * it (P6a spec 2.2): its owner changes it while the project's admin or member (View.UpdatableBy, view.update); the
 * project's admins delete it, and so does its owner while the project's admin or member (View.DeletableBy,
 * view.delete); anyone who sees it copies its link and opens it in a new tab.
 */
const ViewMenu = observer(function ViewMenu(props: Props & { view: View }) {
  const { parentRef, view, projectId, workspaceSlug, customClassName } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { data: currentUser } = useUser();
  const { openDialog } = useViewsPage();
  const copyLink = useCopyLink();
  // auth
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const isAdmin = role === EUserPermissions.ADMIN;
  const isOwner = (isAdmin || role === EUserPermissions.MEMBER) && view.owned_by_id === currentUser?.id;

  const link = `/${workspaceSlug}/projects/${projectId}/views/${view.id}`;
  const MENU_ITEMS = useViewMenuItems({
    isOwner,
    isAdmin,
    handleEdit: () => {},
    handleDelete: () => openDialog("delete", view),
    handleCopyLink: () => void copyLink(link, t("common.link_copied_to_clipboard")),
    handleOpenInNewTab: () => window.open(link, "_blank", "noopener,noreferrer"),
  });

  return (
    <>
      <ContextMenu parentRef={parentRef} items={MENU_ITEMS} />
      <CustomMenu
        ariaLabel={t("common.item_options", { name: view.name })}
        customButton={<MoreHorizontalOutline className="size-4" aria-hidden="true" />}
        customButtonClassName={cn(getIconButtonStyling("tertiary", "lg"), customClassName)}
        placement="bottom-end"
        closeOnSelect
        maxHeight="lg"
      >
        {MENU_ITEMS.map((item) => (
          <CustomMenu.MenuItem
            key={item.key}
            onClick={() => item.action()}
            className={cn("flex items-center gap-2", { "text-placeholder": item.disabled }, item.className)}
            disabled={item.disabled}
          >
            {item.icon && <item.icon className={cn("h-3 w-3 flex-shrink-0", item.iconClassName)} />}
            <div>
              <h5>{item.title}</h5>
              {item.description && (
                <p className={cn("whitespace-pre-line text-tertiary", { "text-placeholder": item.disabled })}>
                  {item.description}
                </p>
              )}
            </div>
          </CustomMenu.MenuItem>
        ))}
      </CustomMenu>
    </>
  );
});
