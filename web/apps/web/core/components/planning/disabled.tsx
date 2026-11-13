/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { useTheme } from "next-themes";
// nerve imports
import { EUserPermissions } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// assets
import darkCycles from "@/app/assets/empty-state/disabled-feature/cycles-dark.webp?url";
import lightCycles from "@/app/assets/empty-state/disabled-feature/cycles-light.webp?url";
import darkModules from "@/app/assets/empty-state/disabled-feature/modules-dark.webp?url";
import lightModules from "@/app/assets/empty-state/disabled-feature/modules-light.webp?url";
// components
import { DetailedEmptyState } from "@/components/empty-state/detailed-empty-state-root";
// hooks
import { useUserPermissions } from "@/hooks/store/user";
// lib
import type { PlanningDomain } from "@/lib/planning-switch";

type Props = { workspaceSlug: string; projectId: string; domain: PlanningDomain };

/** What says each domain turned off: its texts, its picture in each theme. */
const TURNED_OFF = {
  cycles: {
    title: "disabled_project.empty_state.cycle.title",
    description: "disabled_project.empty_state.cycle.description",
    action: "disabled_project.empty_state.cycle.primary_button.text",
    light: lightCycles,
    dark: darkCycles,
  },
  modules: {
    title: "disabled_project.empty_state.module.title",
    description: "disabled_project.empty_state.module.description",
    action: "disabled_project.empty_state.module.primary_button.text",
    light: lightModules,
    dark: darkModules,
  },
} as const satisfies Record<PlanningDomain, Record<string, string>>;

/**
 * A project's cycles or modules turned off (P5a spec 2.5, P5b spec 2.1, P6a spec 2.5): it says so in the domain's own
 * words and picture, and only the project's admins may go to its features of the domain to turn it on. The domain's
 * list, and a cycle's work items, show it in their place.
 */
export const PlanningDisabled = observer(function PlanningDisabled({ workspaceSlug, projectId, domain }: Props) {
  const navigate = useNavigate();
  const { resolvedTheme } = useTheme();
  const { t } = useTranslation();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const role = getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId);
  const off = TURNED_OFF[domain];
  return (
    <div className="flex h-full w-full items-center justify-center">
      <DetailedEmptyState
        title={t(off.title)}
        description={t(off.description)}
        assetPath={resolvedTheme === "light" ? off.light : off.dark}
        primaryButton={{
          text: t(off.action),
          onClick: () => navigate(`/${workspaceSlug}/settings/projects/${projectId}/features/${domain}`),
          disabled: role !== EUserPermissions.ADMIN,
        }}
      />
    </div>
  );
});
