/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
// components
import { PageHead } from "@/components/core/page-title";
import { ArchivedCycleLayoutRoot } from "@/components/cycles/archived-cycles";
import { ArchivedCyclesHeader } from "@/components/cycles/archived-cycles/header";
// hooks
import { useProject } from "@/hooks/store/use-project";
import type { Route } from "./+types/page";

/** A project's archive of cycles (P5a spec 2.5), of the route's project. */
function ProjectArchivedCyclesPage({ params }: Route.ComponentProps) {
  const { workspaceSlug, projectId } = params;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectById } = useProject();
  // derived values
  const project = getProjectById(projectId);

  return (
    <>
      <PageHead title={project ? `${project.name} - ${t("project_cycles.archived_cycles")}` : undefined} />
      <div className="relative flex h-full w-full flex-col overflow-hidden">
        <ArchivedCyclesHeader projectId={projectId} />
        <ArchivedCycleLayoutRoot workspaceSlug={workspaceSlug} projectId={projectId} />
      </div>
    </>
  );
}

export default observer(ProjectArchivedCyclesPage);
