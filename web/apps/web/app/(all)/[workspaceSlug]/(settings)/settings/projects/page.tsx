/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useTheme } from "next-themes";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
// assets
import ProjectDarkEmptyState from "@/app/assets/empty-state/project-settings/no-projects-dark.png?url";
import ProjectLightEmptyState from "@/app/assets/empty-state/project-settings/no-projects-light.png?url";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import type { Route } from "./+types/page";

/**
 * The project settings while the caller is in none of the route's workspace's projects: it says so, and offers a
 * project's creation only to whom nerve lets create one, the workspace's admins and members, not its guests.
 */
function ProjectSettingsPage({ params }: Route.ComponentProps) {
  // router: the route's workspace
  const { workspaceSlug } = params;
  // store hooks
  const { resolvedTheme } = useTheme();
  const { toggleCreateProjectModal } = useCommandPalette();
  const { canCreateProject } = useUserPermissions();
  const { t } = useTranslation();
  // derived values
  const resolvedPath = resolvedTheme === "dark" ? ProjectDarkEmptyState : ProjectLightEmptyState;
  const mayCreateProject = canCreateProject(workspaceSlug);
  return (
    <div className="mx-auto flex h-full max-w-[480px] flex-col items-center justify-center gap-4">
      <img src={resolvedPath} alt="" />
      <div className="text-16 font-semibold text-tertiary">{t("workspace_projects.empty_state.no_projects.title")}</div>
      <div className="text-center text-13 text-tertiary">
        {t("workspace_projects.empty_state.no_projects.description")}
      </div>
      {mayCreateProject && (
        <Button onClick={() => toggleCreateProjectModal(true)}>
          {t("workspace_projects.empty_state.no_projects.primary_button.text")}
        </Button>
      )}
    </div>
  );
}

export default observer(ProjectSettingsPage);
