/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Navigate, useLocation, useSearchParams } from "react-router";
import useSWR from "swr";
// plane imports
import { isValidNextPath } from "@nerve/utils";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useWorkspace } from "@/hooks/store/use-workspace";
import { useUser, useUserProfile, useUserSettings } from "@/hooks/store/user";

type TPageType = EPageTypes;

type TAuthenticationWrapper = {
  children: ReactNode;
  pageType?: TPageType;
};

export const AuthenticationWrapper = observer(function AuthenticationWrapper(props: TAuthenticationWrapper) {
  const { pathname } = useLocation();
  const [searchParams] = useSearchParams();
  const nextPath = searchParams.get("next_path")?.trim();
  // props
  const { children, pageType = EPageTypes.AUTHENTICATED } = props;
  // hooks
  const { isLoading: isUserLoading, data: currentUser, fetchCurrentUser } = useUser();
  const { data: currentUserProfile } = useUserProfile();
  const { data: currentUserSettings } = useUserSettings();
  const { loader: workspacesLoader, workspaces } = useWorkspace();

  const { isLoading: isUserSWRLoading } = useSWR("USER_INFORMATION", async () => await fetchCurrentUser(), {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });

  const isUserOnboard =
    currentUserProfile?.is_onboarded ||
    (currentUserProfile?.onboarding_step?.profile_complete &&
      currentUserProfile?.onboarding_step?.workspace_create &&
      currentUserProfile?.onboarding_step?.workspace_invite &&
      currentUserProfile?.onboarding_step?.workspace_join) ||
    false;

  const getWorkspaceRedirectionUrl = (): string => {
    let redirectionRoute = "/create-workspace";

    // validating the nextPath from the router query
    if (nextPath && isValidNextPath(nextPath)) {
      redirectionRoute = nextPath;
      return redirectionRoute;
    }

    // validate the last and fallback workspace_slug
    const currentWorkspaceSlug =
      currentUserSettings?.workspace?.last_workspace_slug || currentUserSettings?.workspace?.fallback_workspace_slug;

    // validate the current workspace_slug is available in the user's workspace list
    const isCurrentWorkspaceValid = Object.values(workspaces || {}).findIndex(
      (workspace) => workspace.slug === currentWorkspaceSlug
    );

    if (isCurrentWorkspaceValid >= 0) redirectionRoute = `/${currentWorkspaceSlug}`;

    return redirectionRoute;
  };

  if ((isUserSWRLoading || isUserLoading || workspacesLoader) && !currentUser?.id)
    return (
      <div className="relative flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );

  if (pageType === EPageTypes.PUBLIC) return <>{children}</>;

  if (pageType === EPageTypes.NON_AUTHENTICATED) {
    if (!currentUser?.id) return <>{children}</>;
    else {
      if (currentUserProfile?.id && isUserOnboard) {
        const currentRedirectRoute = getWorkspaceRedirectionUrl();
        return <Navigate to={currentRedirectRoute} replace />;
      } else {
        return <Navigate to="/onboarding" replace />;
      }
    }
  }

  if (pageType === EPageTypes.ONBOARDING) {
    if (!currentUser?.id) {
      return <Navigate to={`/?next_path=${pathname}`} replace />;
    } else {
      if (currentUser && currentUserProfile?.id && isUserOnboard) {
        const currentRedirectRoute = getWorkspaceRedirectionUrl();
        return <Navigate to={currentRedirectRoute} replace />;
      } else return <>{children}</>;
    }
  }

  if (pageType === EPageTypes.AUTHENTICATED) {
    if (currentUser?.id) {
      if (currentUserProfile && currentUserProfile?.id && isUserOnboard) return <>{children}</>;
      else {
        return <Navigate to="/onboarding" replace />;
      }
    } else {
      return <Navigate to={`/?next_path=${pathname}`} replace />;
    }
  }

  return <>{children}</>;
});
