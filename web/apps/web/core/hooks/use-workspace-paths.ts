/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMatch, useParams } from "react-router";

/**
 * Custom hook to detect different workspace paths
 * @returns Object containing boolean flags for different workspace paths
 */
export const useWorkspacePaths = () => {
  const { workspaceSlug } = useParams();

  const isSettingsPath = useMatch({ path: "/:workspaceSlug/settings", end: false }) !== null;
  const isProjectsPath = workspaceSlug !== undefined && !isSettingsPath;
  const isNotificationsPath = useMatch("/:workspaceSlug/notifications") !== null;

  return {
    isSettingsPath,
    isProjectsPath,
    isNotificationsPath,
  };
};
