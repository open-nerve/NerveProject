/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { RefObject } from "react";
import { observer } from "mobx-react";
// nerve imports
import type { View } from "@nerve/api-client";
// hooks
import { useMember } from "@/hooks/store/use-member";
// store
import type { ViewContainer } from "@/store/views/view-container";
// local imports
import { ButtonAvatars } from "../dropdowns/member/avatar";
import { ViewQuickActions } from "./quick-actions";

type Props = {
  parentRef: RefObject<HTMLElement | null>;
  view: View;
  workspaceSlug: string;
  /** The view's container, a project or the workspace, whose views the list shows. */
  container: ViewContainer;
};

/**
 * The actions of a view's row in its container's list, a project's or a workspace's own views (P6a spec 2.3, R17;
 * M7/P6b spec 2.11): its owner, and its quick actions, the row's one menu (its button, and the row's right-click menu).
 */
export const ViewListItemAction = observer(function ViewListItemAction(props: Props) {
  const { parentRef, view, workspaceSlug, container } = props;
  // store hooks
  const { getUserDetails } = useMember();
  const owner = getUserDetails(view.owned_by_id);

  return (
    <>
      <ButtonAvatars userIds={owner?.id ?? []} />
      <ViewQuickActions parentRef={parentRef} viewId={view.id} container={container} workspaceSlug={workspaceSlug} />
    </>
  );
});
