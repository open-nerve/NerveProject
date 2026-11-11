/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { CyclesOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { CyclesViewHeader } from "@/components/cycles/cycles-view-header";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";

type TProps = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The header of a project's cycles list (P5a spec 2.5, R8), of the route's project: its crumbs; for those who write
 * the project's cycles (canWriteCycles: its admins and members, not one nerve shows only his own work items, and no one
 * while they are turned off), the list's search and filters, and a new cycle, by the project's dialogs' host
 * (ProjectLevelModals).
 */
export const CyclesListHeader = observer(function CyclesListHeader(props: TProps) {
  const { workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const { toggleCreateCycleModal } = useCommandPalette();
  const { canWriteCycles } = useUserPermissions();
  const { loader } = useProject();
  const { t } = useTranslation();

  const canWrite = canWriteCycles(workspaceSlug, projectId);

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs onBack={() => navigate(-1)} isLoading={loader === "init-loader"}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={t("cycles")}
                href={`/${workspaceSlug}/projects/${projectId}/cycles`}
                icon={<CyclesOutline className="h-4 w-4 text-tertiary" />}
                isLast
              />
            }
            isLast
          />
        </Breadcrumbs>
      </Header.LeftItem>
      {canWrite && (
        <Header.RightItem>
          <CyclesViewHeader projectId={projectId} list="default" />
          <Button variant="primary" size="lg" onClick={() => toggleCreateCycleModal(true)}>
            <div className="block sm:hidden">{t("add")}</div>
            <div className="hidden sm:block">{t("project_cycles.add_cycle")}</div>
          </Button>
        </Header.RightItem>
      )}
    </Header>
  );
});
