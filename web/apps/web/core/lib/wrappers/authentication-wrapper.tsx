/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { observer } from "mobx-react";
import { Navigate, useLocation, useSearchParams } from "react-router";
// nerve imports
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
import { isOnboarded, useLanding } from "@/lib/use-landing";
import { useSessionSWR } from "@/lib/use-session-swr";

type TAuthenticationWrapper = {
  children: ReactNode;
  pageType?: EPageTypes;
};

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
 * to a valid next_path, else to /onboarding until onboarded, else to the landing (M3 design 3.14), as from
 * the onboarding page once onboarded: useLanding decides it, and the wrapper renders what it says. While nerve
 * cannot be reached the session is kept, and the page says so instead of moving.
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
  // derived values
  const validNextPath = nextPath && isValidNextPath(nextPath) ? nextPath : undefined;
  const onboarded = currentProfile !== undefined && isOnboarded(currentProfile);

  // The account is fetched for each session: a sign-in here or in another tab, or another account (7.1).
  // A fetch cut by a change of session is no failure: fetchCurrentUser gives undefined, and the wrapper
  // renders again for the new session, with its new stores. The workspaces are fetched the same way, when the
  // landing needs them (useLanding).
  const { error, mutate } = useSessionSWR(["CURRENT_USER"], () => fetchCurrentUser());
  const landing = useLanding(pageType, validNextPath);

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

  if (landing.kind === "unavailable") return <SessionUnavailable autoRetry={false} onRetry={landing.retry} />;
  if (landing.kind === "loading") return <Loading />;
  if (landing.kind === "go") return <Navigate to={landing.to} replace />;
  switch (pageType) {
    case EPageTypes.NON_AUTHENTICATED:
      return <Navigate to={validNextPath ?? "/onboarding"} replace />;
    case EPageTypes.ONBOARDING:
      return validNextPath && onboarded ? <Navigate to={validNextPath} replace /> : <>{children}</>;
    default:
      return onboarded ? <>{children}</> : <Navigate to="/onboarding" replace />;
  }
});
