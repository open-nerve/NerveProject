/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef } from "react";
import { observer } from "mobx-react";
import { Logo } from "@nerve/propel/emoji-icon-picker";
import { ViewsOutline } from "@makeplane/propel/icons";
// nerve imports
import type { View } from "@nerve/api-client";
// components
import { ListItem } from "@/components/core/list";
// hooks
import { usePlatformOS } from "@/hooks/use-platform-os";
// store
import type { ViewContainer } from "@/store/views/view-container";
// local imports
import { ViewListItemAction } from "./view-list-item-action";
import { viewPath } from "./view-paths";

type Props = {
  view: View;
  workspaceSlug: string;
  /** The view's container, a project or the workspace, whose views the list shows. */
  container: ViewContainer;
};

/**
 * A view's row in its container's views list, a project's or a workspace's own (P6a spec 2.3, R17; M7/P6b spec 2.11):
 * its icon, its name, a link to its page, and its description; then its owner and its quick actions
 * (ViewListItemAction).
 */
export const ViewListItem = observer(function ViewListItem(props: Props) {
  const { view, workspaceSlug, container } = props;
  // refs
  const parentRef = useRef<HTMLDivElement>(null);
  // store hooks
  const { isMobile } = usePlatformOS();

  return (
    <ListItem
      prependTitleElement={
        view.logo_props.in_use ? (
          <Logo logo={view.logo_props} size={16} type="lucide" />
        ) : (
          <ViewsOutline className="h-4 w-4 text-tertiary" />
        )
      }
      title={view.name}
      appendTitleElement={
        view.description ? <span className="truncate text-12 text-tertiary">{view.description}</span> : undefined
      }
      rightElementClassName="min-w-0 flex-shrink"
      itemLink={viewPath(workspaceSlug, container, view.id)}
      actionableItems={
        <ViewListItemAction parentRef={parentRef} view={view} workspaceSlug={workspaceSlug} container={container} />
      }
      isMobile={isMobile}
      parentRef={parentRef}
    />
  );
});
