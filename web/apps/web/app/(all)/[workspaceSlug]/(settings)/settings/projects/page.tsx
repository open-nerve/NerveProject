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

function ProjectSettingsPage() {
  // store hooks
  const { resolvedTheme } = useTheme();
  const { toggleCreateProjectModal } = useCommandPalette();
  const { t } = useTranslation();
  // derived values
  const resolvedPath = resolvedTheme === "dark" ? ProjectDarkEmptyState : ProjectLightEmptyState;
  return (
    <div className="mx-auto flex h-full max-w-[480px] flex-col items-center justify-center gap-4">
      <img src={resolvedPath} alt="" />
      <div className="text-16 font-semibold text-tertiary">{t("workspace_projects.empty_state.no_projects.title")}</div>
      <div className="text-center text-13 text-tertiary">
        {t("workspace_projects.empty_state.no_projects.description")}
      </div>
      <Button onClick={() => toggleCreateProjectModal(true)}>
        {t("workspace_projects.empty_state.no_projects.primary_button.text")}
      </Button>
    </div>
  );
}

export default observer(ProjectSettingsPage);
