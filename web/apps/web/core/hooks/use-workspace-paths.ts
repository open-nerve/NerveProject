/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "next/navigation";
import { useLocation } from "react-router";

/**
 * Custom hook to detect different workspace paths
 * @returns Object containing boolean flags for different workspace paths
 */
export const useWorkspacePaths = () => {
  const { workspaceSlug } = useParams();
  const { pathname } = useLocation();

  const isSettingsPath = pathname.includes(`/${workspaceSlug}/settings`);
  const isProjectsPath = pathname.includes(`/${workspaceSlug}/`) && !isSettingsPath;
  const isNotificationsPath = pathname.includes(`/${workspaceSlug}/notifications`);

  return {
    isSettingsPath,
    isProjectsPath,
    isNotificationsPath,
  };
};
