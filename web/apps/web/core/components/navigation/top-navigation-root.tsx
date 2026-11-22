/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

// components
import { observer } from "mobx-react";
import { useMatch, useParams } from "react-router";
import { useTranslation } from "@nerve/i18n";
import { TopNavPowerK } from "@/components/navigation";
import { HelpMenuRoot } from "@/components/workspace/sidebar/help-section/root";
import { UserMenuRoot } from "@/components/workspace/sidebar/user-menu-root";
import { WorkspaceMenuRoot } from "@/components/workspace/sidebar/workspace-menu-root";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { AppSidebarItem } from "@/components/sidebar/sidebar-item";
import { InboxOutline } from "@makeplane/propel/icons";
import { useUnreadNotificationCount } from "@/hooks/use-unread-notification-count";

/**
 * The bar above every page of a workspace: its menu, the search, the inbox, the help and the caller's menu. The inbox
 * link fetches the caller's unread notifications in the route's workspace (useUnreadNotificationCount, P4 spec R5) and
 * shows a dot while either tab has one: nerve's all counts every one but the mentions, so the two are added; the
 * link's name says how many.
 */
export const TopNavigationRoot = observer(function TopNavigationRoot() {
  // router
  const { workspaceSlug } = useParams();
  const isNotificationsPath = useMatch("/:workspaceSlug/notifications") !== null;
  const { t } = useTranslation();

  const unread = useUnreadNotificationCount();
  const unreadCount = unread ? unread.all + unread.mentions : 0;
  const inboxLabel = t("notification.label");

  return (
    <div className="z-[27] flex min-h-10 w-full items-center bg-canvas px-3.5 transition-all duration-300">
      {/* Workspace Menu */}
      <div className="flex-1 shrink-0">
        <WorkspaceMenuRoot />
      </div>
      {/* Power K Search */}
      <div className="shrink-0">
        <TopNavPowerK />
      </div>
      {/* Additional Actions */}
      <div className="flex flex-1 shrink-0 items-center justify-end gap-1">
        <Tooltip label={inboxLabel} side="bottom">
          <AppSidebarItem
            variant="link"
            item={{
              href: `/${workspaceSlug}/notifications`,
              label:
                unreadCount > 0
                  ? t("notification.a11y.unread_count", { label: inboxLabel, count: unreadCount })
                  : inboxLabel,
              icon: (
                <div className="relative">
                  <InboxOutline className="size-5" />
                  {unreadCount > 0 && <span className="absolute top-0 right-0 size-2 rounded-full bg-danger-primary" />}
                </div>
              ),
              isActive: isNotificationsPath,
            }}
          />
        </Tooltip>
        <HelpMenuRoot />
        <div className="flex size-8 items-center justify-center rounded-md hover:bg-layer-1-hover">
          <UserMenuRoot />
        </div>
      </div>
    </div>
  );
});
