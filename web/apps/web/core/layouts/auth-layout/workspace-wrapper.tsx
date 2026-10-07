/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { useParams, Link } from "react-router";
import useSWR from "swr";
// ui
import { LogOutOutline } from "@makeplane/propel/icons";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { getButtonStyling } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { Tooltip } from "@makeplane/propel/components/tooltip";
import { cn } from "@nerve/utils";
// assets
import WorkSpaceNotAvailable from "@/app/assets/workspace/workspace-not-available.png?url";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { LogoSpinner } from "@/components/common/logo-spinner";
import { NerveLogo } from "@/components/common/nerve-logo";
// constants
import {
  WORKSPACE_PARTIAL_PROJECTS,
  WORKSPACE_PROJECTS_ROLES_INFORMATION,
  WORKSPACE_FAVORITE,
  WORKSPACE_STATES,
} from "@nerve/constants";
// hooks
import { useFavorite } from "@/hooks/store/use-favorite";
import { useProject } from "@/hooks/store/use-project";
import { useProjectState } from "@/hooks/store/use-project-state";
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserPermissions } from "@/hooks/store/user";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local imports
import { useWorkspaceFetch } from "./use-workspace-fetch";

interface IWorkspaceAuthWrapper {
  children: ReactNode;
}

export const WorkspaceAuthWrapper = observer(function WorkspaceAuthWrapper(props: IWorkspaceAuthWrapper) {
  const { children } = props;
  // router params
  const { workspaceSlug } = useParams();
  // translation
  const { t } = useTranslation();
  // store hooks
  const { signOut, data: currentUser } = useUser();
  const { fetchPartialProjects } = useProject();
  const { fetchFavorite } = useFavorite();
  const { workspaces, getWorkspaceBySlug } = useWorkspace();
  const { isMobile } = usePlatformOS();
  const { fetchUserProjectPermissions, allowPermissions } = useUserPermissions();
  const { fetchWorkspaceStates } = useProjectState();
  // derived values
  const canPerformWorkspaceMemberActions = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE
  );
  // The caller's workspaces decide whether he may see this one, and his role in it (M3 design 7.2).
  const currentWorkspace = workspaceSlug ? getWorkspaceBySlug(workspaceSlug) : null;

  // the workspace side of what every page of a workspace fetches (M3 design 7.1)
  const listed = useWorkspaceFetch(workspaceSlug);
  useSWR(
    workspaceSlug && currentWorkspace ? WORKSPACE_PROJECTS_ROLES_INFORMATION(workspaceSlug) : null,
    workspaceSlug && currentWorkspace ? () => fetchUserProjectPermissions(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  // fetching workspace projects
  useSWR(
    workspaceSlug && currentWorkspace ? WORKSPACE_PARTIAL_PROJECTS(workspaceSlug) : null,
    workspaceSlug && currentWorkspace ? () => fetchPartialProjects(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  // fetch workspace favorite
  useSWR(
    workspaceSlug && currentWorkspace && canPerformWorkspaceMemberActions ? WORKSPACE_FAVORITE(workspaceSlug) : null,
    workspaceSlug && currentWorkspace && canPerformWorkspaceMemberActions ? () => fetchFavorite(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );
  // fetch workspace states
  useSWR(
    workspaceSlug ? WORKSPACE_STATES(workspaceSlug) : null,
    workspaceSlug ? () => fetchWorkspaceStates(workspaceSlug) : null,
    { revalidateIfStale: false, revalidateOnFocus: false }
  );

  const handleSignOut = async () => {
    await signOut().catch(() =>
      setToast({
        type: TOAST_TYPE.ERROR,
        title: "Error!",
        message: "Failed to sign out. Please try again.",
      })
    );
  };

  // nerve could not be reached: the page says so, and tries again when asked (M2 design 7.1)
  if (listed.error) return <SessionUnavailable autoRetry={false} onRetry={() => void listed.mutate()} />;

  // if list of workspaces are not there then we have to render the spinner
  if (workspaces === undefined) {
    return (
      <div className="grid h-full place-items-center rounded-lg border border-subtle p-4">
        <div className="flex flex-col items-center gap-3 text-center">
          <LogoSpinner />
        </div>
      </div>
    );
  }

  // a workspace that is not among the caller's: it does not exist, or he is not a member of it (M3 design 8.3)
  if (!currentWorkspace) {
    return (
      <div className="relative flex h-full w-full flex-col items-center justify-center bg-surface-2">
        <div className="relative container mx-auto flex h-full w-full flex-col overflow-hidden overflow-y-auto px-5 py-14 md:px-0">
          <div className="relative flex flex-shrink-0 items-center justify-between gap-4">
            <div className="z-10 flex-shrink-0 bg-surface-2 py-4">
              <NerveLogo className="h-9 w-auto" />
            </div>
            <div className="relative flex items-center gap-2">
              <div className="text-13 font-medium">{currentUser?.email}</div>
              <button
                type="button"
                aria-label={t("sign_out")}
                className="relative flex h-6 w-6 flex-shrink-0 cursor-pointer items-center justify-center overflow-hidden rounded-sm hover:bg-layer-1"
                onClick={handleSignOut}
              >
                <Tooltip label={t("sign_out")} alignOffset={8} disabled={isMobile}>
                  <LogOutOutline width={14} height={14} />
                </Tooltip>
              </button>
            </div>
          </div>
          <div className="relative flex h-full w-full flex-grow flex-col items-center justify-center space-y-3">
            <div className="relative flex-shrink-0">
              <img src={WorkSpaceNotAvailable} className="h-[220px] object-contain object-center" alt="" />
            </div>
            <h3 className="text-center text-16 font-semibold">Workspace not found</h3>
            <p className="text-center text-13 text-secondary">
              No workspace found with the URL. It may not exist or you lack authorization to view it.
            </p>
            <div className="flex items-center justify-center gap-2 pt-4">
              {workspaces.length > 0 && (
                <Link to="/" className={cn(getButtonStyling("primary", "base"))}>
                  Go Home
                </Link>
              )}
              {workspaces.length > 0 && (
                <Link to="/settings/profile/general" className={cn(getButtonStyling("secondary", "base"))}>
                  Visit Profile
                </Link>
              )}
              {workspaces.length === 0 && (
                <Link to="/create-workspace" className={cn(getButtonStyling("secondary", "base"))}>
                  Create new workspace
                </Link>
              )}
            </div>
          </div>

          <div className="absolute top-0 bottom-0 left-4 w-0 bg-layer-1 md:w-0.5" />
        </div>
      </div>
    );
  }

  return <>{children}</>;
});
