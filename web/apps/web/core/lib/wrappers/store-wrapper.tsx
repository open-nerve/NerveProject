/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import { setLanguage } from "@nerve/i18n";
// hooks
import { useAppTheme } from "@/hooks/store/use-app-theme";
import { useRouterParams } from "@/hooks/store/use-router-params";
import { useUser, useUserProfile } from "@/hooks/store/user";
import type { IUserStore } from "@/store/user";
// lib
import { useSession } from "@/lib/auth/use-session";
import { sessionTheme } from "@/lib/wrappers/session-theme";

type TStoreWrapper = {
  children: ReactNode;
};

function StoreWrapper(props: TStoreWrapper) {
  const { children } = props;
  // theme
  const { setTheme } = useTheme();
  // router
  const params = useParams();
  // store hooks
  const { setQuery } = useRouterParams();
  const { sidebarCollapsed, toggleSidebar } = useAppTheme();
  const user = useUser();
  const { data: userProfile } = useUserProfile();
  const { status } = useSession();
  // The session whose profile's theme the page has taken: once for each session (a new one has a new user store).
  const themedBy = useRef<IUserStore | undefined>(undefined);

  /**
   * Sidebar collapsed fetching from local storage
   */
  useEffect(() => {
    const localValue = localStorage && localStorage.getItem("app_sidebar_collapsed");
    const localBoolValue = localValue === "true";
    if (localValue && sidebarCollapsed === undefined) toggleSidebar(localBoolValue);
  }, [sidebarCollapsed, setTheme, toggleSidebar]);

  /**
   * The page's theme follows the tab's session (v0 design 7.7), here and nowhere else (session-theme.ts): without a
   * session (signed out in this tab or another, the account deactivated, a refresh refused), the default; with one,
   * its profile's, once, when the profile first arrives. Later changes are applied by the component that makes them,
   * once nerve holds them (theme-switcher.tsx). Until a new session's profile arrives, the page keeps the theme it
   * shows.
   */
  useEffect(() => {
    const next = sessionTheme(status, user, userProfile?.theme, themedBy.current);
    themedBy.current = next.themedBy;
    if (next.theme) setTheme(next.theme);
  }, [status, user, userProfile?.theme, setTheme]);

  /**
   * The page's language is the profile's of the tab's session now, as nerve answered it: the stores never set it.
   * A refused change leaves the profile, and so the page, as it was; a retired session's store, answered late,
   * changes only its own profile, which nothing here reads. Signed out, or before the profile loads, the page keeps
   * the language it has (a new session starts with the default: store-context.tsx).
   */
  useEffect(() => {
    if (userProfile?.language) void setLanguage(userProfile.language);
  }, [userProfile?.language]);

  useEffect(() => {
    if (!params) return;
    setQuery(params);
  }, [params, setQuery]);

  return <>{children}</>;
}

export default observer(StoreWrapper);
