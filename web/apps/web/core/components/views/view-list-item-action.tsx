/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import React, { useState } from "react";
import { observer } from "mobx-react";
import { GlobeOutline, LockOutline } from "@makeplane/propel/icons";
// nerve imports
import { EUserPermissions, EUserPermissionsLevel, IS_FAVORITE_MENU_OPEN } from "@nerve/constants";
import { useLocalStorage } from "@nerve/hooks";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { IProjectView } from "@nerve/types";
import { EViewAccess } from "@nerve/types";
import { FavoriteStar } from "@nerve/ui";
// hooks
import { useMember } from "@/hooks/store/use-member";
import { useProjectView } from "@/hooks/store/use-project-view";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { ButtonAvatars } from "../dropdowns/member/avatar";
import { DeleteProjectViewModal } from "./delete-view-modal";
import { CreateUpdateProjectViewModal } from "./modal";
import { ViewQuickActions } from "./quick-actions";

type Props = {
  parentRef: React.RefObject<HTMLElement | null>;
  view: IProjectView;
  workspaceSlug: string;
  projectId: string;
};

/**
 * The actions of a view's row, of the project of projectId at its page's address (its row's, M3 handoff §4): whether
 * the caller may star the view follows his role in that project, never in the project the router store holds.
 */
export const ViewListItemAction = observer(function ViewListItemAction(props: Props) {
  const { parentRef, view, workspaceSlug, projectId } = props;
  // states
  const [createUpdateViewModal, setCreateUpdateViewModal] = useState(false);
  const [deleteViewModal, setDeleteViewModal] = useState(false);
  // store
  const { allowPermissions } = useUserPermissions();

  const { addViewToFavorites, removeViewFromFavorites } = useProjectView();
  const { getUserDetails } = useMember();

  // local storage
  const { setValue: toggleFavoriteMenu, storedValue: isFavoriteOpen } = useLocalStorage<boolean>(
    IS_FAVORITE_MENU_OPEN,
    false
  );

  // derived values
  const isEditingAllowed = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.PROJECT,
    workspaceSlug,
    projectId
  );

  const access = view.access;

  // handlers
  const handleAddToFavorites = async () => {
    await addViewToFavorites(workspaceSlug, projectId, view.id);
    if (!isFavoriteOpen) toggleFavoriteMenu(true);
  };

  const handleRemoveFromFavorites = () => {
    removeViewFromFavorites(workspaceSlug, projectId, view.id);
  };

  const ownedByDetails = view.owned_by ? getUserDetails(view.owned_by) : undefined;

  return (
    <>
      <CreateUpdateProjectViewModal
        isOpen={createUpdateViewModal}
        onClose={() => setCreateUpdateViewModal(false)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        data={view}
      />
      <DeleteProjectViewModal data={view} isOpen={deleteViewModal} onClose={() => setDeleteViewModal(false)} />
      <div className="cursor-default text-tertiary">
        <Tooltip label={access === EViewAccess.PUBLIC ? "Public" : "Private"}>
          {access === EViewAccess.PUBLIC ? <GlobeOutline className="h-4 w-4" /> : <LockOutline className="h-4 w-4" />}
        </Tooltip>
      </div>

      {/* created by */}
      <ButtonAvatars userIds={ownedByDetails?.id ?? []} />

      {isEditingAllowed && (
        <FavoriteStar
          onClick={(e) => {
            e.preventDefault();
            e.stopPropagation();
            if (view.is_favorite) handleRemoveFromFavorites();
            else handleAddToFavorites();
          }}
          selected={view.is_favorite}
        />
      )}
      <div className="hidden md:block">
        <ViewQuickActions parentRef={parentRef} projectId={projectId} view={view} workspaceSlug={workspaceSlug} />
      </div>
    </>
  );
});
