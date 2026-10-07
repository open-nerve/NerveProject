/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Navigate, useLocation, useSearchParams } from "react-router";
// nerve imports
import type { Profile } from "@nerve/api-client";
import { isValidNextPath, signInPath } from "@nerve/utils";
// components
import { SessionUnavailable } from "@/components/account/session-unavailable";
import { LogoSpinner } from "@/components/common/logo-spinner";
// helpers
import { EPageTypes } from "@/helpers/authentication.helper";
// hooks
import { useUser, useUserProfile } from "@/hooks/store/user";
// lib
import { tokenManager } from "@/lib/auth/api-client";
import { useSession } from "@/lib/auth/use-session";
import { useSessionSWR } from "@/lib/use-session-swr";

type TAuthenticationWrapper = {
  children: ReactNode;
  pageType?: EPageTypes;
};

const isOnboarded = (profile: Profile) =>
  profile.is_onboarded ||
  (profile.onboarding_step.profile_complete &&
    profile.onboarding_step.workspace_create &&
    profile.onboarding_step.workspace_invite &&
    profile.onboarding_step.workspace_join);

function Loading() {
  return (
    <div className="relative flex h-screen w-full items-center justify-center">
      <LogoSpinner />
    </div>
  );
}

/**
 * Decides who may see a page (M2 design 7.4), from the session, the account, its profile and next_path
 * only, and is the one place that sends the user elsewhere (7.1): signed out on a page that needs an
 * account, to the sign-in page with the page as next_path (3.18); signed in on the sign-in or sign-up page,
 * to a valid next_path, else to /onboarding until onboarded, else to /create-workspace. While nerve cannot
 * be reached the session is kept, and the page says so instead of moving.
 */
export const AuthenticationWrapper = observer(function AuthenticationWrapper(props: TAuthenticationWrapper) {
  const { children, pageType = EPageTypes.AUTHENTICATED } = props;
  const { pathname, search, hash } = useLocation();
  const [searchParams] = useSearchParams();
  const nextPath = searchParams.get("next_path")?.trim();
  // store hooks
  const session = useSession();
  const { data: currentUser, fetchCurrentUser } = useUser();
  const { data: currentProfile } = useUserProfile();

  // The account is fetched for each session: a sign-in here or in another tab, or another account (7.1).
  // A fetch cut by a change of session is no failure: fetchCurrentUser gives undefined, and the wrapper
  // renders again for the new session, with its new stores.
  const { error, mutate } = useSessionSWR(["CURRENT_USER"], () => fetchCurrentUser(), {
    revalidateOnFocus: false,
    shouldRetryOnError: false,
  });

  if (session.status === "starting") return <Loading />;
  if (session.status === "unavailable")
    return <SessionUnavailable autoRetry={session.retryAt !== undefined} onRetry={() => void tokenManager.retry()} />;

  if (session.status === "signed-out") {
    if (pageType === EPageTypes.PUBLIC || pageType === EPageTypes.NON_AUTHENTICATED) return <>{children}</>;
    return <Navigate to={signInPath(pathname + search + hash)} replace />;
  }

  if (error) return <SessionUnavailable autoRetry={false} onRetry={() => void mutate()} />;
  if (!currentUser || !currentProfile) return <Loading />;
  if (pageType === EPageTypes.PUBLIC) return <>{children}</>;

  const validNextPath = nextPath && isValidNextPath(nextPath) ? nextPath : undefined;
  const onboarded = isOnboarded(currentProfile);
  switch (pageType) {
    case EPageTypes.NON_AUTHENTICATED:
      return <Navigate to={validNextPath ?? (onboarded ? "/create-workspace" : "/onboarding")} replace />;
    case EPageTypes.ONBOARDING:
      return onboarded ? <Navigate to={validNextPath ?? "/create-workspace"} replace /> : <>{children}</>;
    default:
      return onboarded ? <>{children}</> : <Navigate to="/onboarding" replace />;
  }
});
