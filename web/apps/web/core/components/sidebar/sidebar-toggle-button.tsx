/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { LeftSidePaneOutline } from "@makeplane/propel/icons";
import { useTranslation } from "@nerve/i18n";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { IconButton } from "@nerve/propel/icon-button";

export const AppSidebarToggleButton = observer(function AppSidebarToggleButton() {
  // store hooks
  const { toggleSidebar, sidebarPeek, toggleSidebarPeek } = useAppTheme();
  const { t } = useTranslation();

  return (
    <IconButton
      size="base"
      variant="ghost"
      icon={LeftSidePaneOutline}
      aria-label={t("aria_labels.app_sidebar.toggle")}
      data-sidebar-toggle=""
      onClick={() => {
        if (sidebarPeek) toggleSidebarPeek(false);
        toggleSidebar();
        // the toggle pressed goes (the sidebar's inert as the sidebar collapses, the header's away as it opens), so the
        // focus goes on to the toggle that comes in its place, not to the page's body
        requestAnimationFrame(() =>
          document.querySelector<HTMLElement>("[data-sidebar-toggle]:not([inert] [data-sidebar-toggle])")?.focus()
        );
      }}
    />
  );
});
