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
import { WORKSPACE_PROJECTS_ROLES_INFORMATION, WORKSPACE_STATES } from "@nerve/constants";
// hooks
import { useProjectState } from "@/hooks/store/use-project-state";
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
  const { isMobile } = usePlatformOS();
  const { fetchUserProjectPermissions } = useUserPermissions();
  const { fetchWorkspaceStates } = useProjectState();

  // the workspace side of what every page of a workspace fetches (M3 design 7.1), and what the caller's workspaces
  // decide this one is to him (7.2, 8.3)
  const access = useWorkspaceFetch(workspaceSlug);
  const workspace = access.kind === "ready" ? access.workspace : null;
  useSWR(
    workspace ? WORKSPACE_PROJECTS_ROLES_INFORMATION(workspace.slug) : null,
    workspace ? () => fetchUserProjectPermissions(workspace.slug) : null,
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
  if (access.kind === "unavailable") return <SessionUnavailable autoRetry={false} onRetry={access.retry} />;

  // the caller's workspaces are not there yet
  if (access.kind === "loading") {
    return (
      <div className="grid h-full place-items-center rounded-lg border border-subtle p-4">
        <div className="flex flex-col items-center gap-3 text-center">
          <LogoSpinner />
        </div>
      </div>
    );
  }

  // a workspace that is not among the caller's: it does not exist, or he is not a member of it (M3 design 8.3)
  if (access.kind === "not-found") {
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
              {access.hasWorkspaces && (
                <Link to="/" className={cn(getButtonStyling("primary", "base"))}>
                  Go Home
                </Link>
              )}
              {access.hasWorkspaces && (
                <Link to="/settings/profile/general" className={cn(getButtonStyling("secondary", "base"))}>
                  Visit Profile
                </Link>
              )}
              {!access.hasWorkspaces && (
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
