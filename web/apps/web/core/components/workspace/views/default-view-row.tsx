/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useRef } from "react";
import { ViewsOutline } from "@makeplane/propel/icons";
// nerve imports
import type { DefaultWorkspaceViewKey } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// components
import { ListItem } from "@/components/core/list";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { DefaultViewMenu } from "./default-view-menu";

type Props = {
  workspaceSlug: string;
  /** The default view: the key its address names, and its name's (M7/P6b spec 2.5). */
  view: { key: DefaultWorkspaceViewKey; i18n_label: string };
};

/**
 * A default view's row in a workspace's views list (M7/P6b spec 2.10, R12), for anyone of the workspace: its name, a
 * link to its page, and its menu, by its button and by a right-click on the row (parentRef), which opens the page in a
 * new tab and copies its link (DefaultViewMenu), as a saved view's row has its own (ViewListItem).
 */
export function DefaultViewRow(props: Props) {
  const { workspaceSlug, view } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // nerve hooks
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();

  return (
    <ListItem
      prependTitleElement={<ViewsOutline className="h-4 w-4 text-tertiary" />}
      title={t(view.i18n_label)}
      itemLink={`/${workspaceSlug}/workspace-views/${view.key}`}
      actionableItems={<DefaultViewMenu parentRef={parentRef} workspaceSlug={workspaceSlug} view={view} />}
      isMobile={isMobile}
      parentRef={parentRef}
    />
  );
}
