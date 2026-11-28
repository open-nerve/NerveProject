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
import { useTranslation } from "@nerve/i18n";
import { getIconButtonStyling } from "@nerve/propel/icon-button";
import { ContextMenu, CustomMenu } from "@nerve/ui";
import { cn } from "@nerve/utils";
// hooks
import { useViewMenuItems } from "@/components/common/quick-actions-helper";
import { useViews } from "@/hooks/store/use-views";
import { useCopyLink } from "@/hooks/use-copy-link";
// store
import type { ViewContainer } from "@/store/views/view-container";
// local imports
import { useViewsPage } from "./actions-host";
import { useViewRights } from "./use-view-rights";
import { viewPath } from "./view-paths";

type Props = {
  /** The view's row, whose right-click opens the menu too; none in a header. */
  parentRef?: RefObject<HTMLElement | null>;
  viewId: string;
  /** The view's container, a project or the workspace, whose views the page shows. */
  container: ViewContainer;
  workspaceSlug: string;
  customClassName?: string;
};

/**
 * A view's quick actions (P6a spec 2.3, 2.4, R17; M7/P6b spec 2.11, R15), a project's or a workspace's own: its menu,
 * by its button, named by the view, and in a list by a right-click on its row (parentRef); none while the store does
 * not give the view. Its dialogs, which change it and delete it, are its page's (ViewActionsHost), and what follows
 * them is the page's while it is mounted, though they take the row away.
 */
export const ViewQuickActions = observer(function ViewQuickActions(props: Props) {
  const { getViewById } = useViews();
  const view = getViewById(props.viewId);
  return view ? <ViewMenu {...props} view={view} /> : null;
});

/**
 * The menu of the view of the container (useViewMenuItems): what the caller may do with it, as nerve decides it by his
 * role in the container (useViewRights; P6a spec 2.2, M7/P6b spec 2.7): its owner changes it while the container's
 * admin or member (View.UpdatableBy); the container's admins delete it, and so does its owner while its admin or member
 * (View.DeletableBy); anyone who sees it copies the link of its page and opens it in a new tab.
 */
const ViewMenu = observer(function ViewMenu(props: Props & { view: View }) {
  const { parentRef, view, container, workspaceSlug, customClassName } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { openDialog } = useViewsPage();
  const copyLink = useCopyLink();
  // auth
  const { isAdmin, isOwner } = useViewRights(workspaceSlug, container);

  const link = viewPath(workspaceSlug, container, view.id);
  const MENU_ITEMS = useViewMenuItems({
    isOwner: isOwner(view),
    isAdmin,
    handleEdit: () => openDialog("edit", view),
    handleDelete: () => openDialog("delete", view),
    handleCopyLink: () => void copyLink(link, t("common.link_copied_to_clipboard")),
    handleOpenInNewTab: () => window.open(link, "_blank", "noopener,noreferrer"),
  });

  return (
    <>
      {parentRef && <ContextMenu parentRef={parentRef} items={MENU_ITEMS} />}
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
