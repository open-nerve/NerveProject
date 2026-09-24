/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { Link, useMatch, useParams } from "react-router";
// nerve imports
import type { IWorkspaceSidebarNavigationItem } from "@nerve/constants";
import { EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { joinUrlPath } from "@nerve/utils";
// components
import { SidebarNavItem } from "@/components/sidebar/sidebar-navigation";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useUser, useUserPermissions } from "@/hooks/store/user";
// components
import { getSidebarNavigationItemIcon } from "@/components/workspace/sidebar/helper";

type Props = {
  item: IWorkspaceSidebarNavigationItem;
};

export const SidebarItemBase = observer(function SidebarItemBase({ item }: Props) {
  const { t } = useTranslation();
  const { workspaceSlug } = useParams();
  const { allowPermissions } = useUserPermissions();
  const { data } = useUser();

  const { toggleSidebar } = useAppTheme();

  const handleLinkClick = () => {
    if (window.innerWidth < 768) toggleSidebar();
  };

  const slug = workspaceSlug || "";

  const itemHref =
    item.key === "your_work" && data?.id ? joinUrlPath(slug, item.href, data?.id) : joinUrlPath(slug, item.href);
  const isActive = useMatch({ path: itemHref, end: item.end ?? false }) !== null;

  if (!allowPermissions(item.access, EUserPermissionsLevel.WORKSPACE, slug)) return null;

  const icon = getSidebarNavigationItemIcon(item.key);

  return (
    <Link to={itemHref} onClick={handleLinkClick}>
      <SidebarNavItem isActive={isActive}>
        <div className="flex items-center gap-1.5 py-[1px]">
          {icon}
          <p className="text-13 leading-5 font-medium">{t(item.labelTranslationKey)}</p>
        </div>
      </SidebarNavItem>
    </Link>
  );
});
