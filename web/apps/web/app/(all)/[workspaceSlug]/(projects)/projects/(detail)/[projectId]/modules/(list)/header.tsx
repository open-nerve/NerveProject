/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { useNavigate } from "react-router";
import { ModuleOutline } from "@makeplane/propel/icons";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { ModuleViewHeader } from "@/components/modules/module-view-header";
// hooks
import { useCommandPalette } from "@/hooks/store/use-command-palette";
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
import { usePlanningReads } from "@/hooks/use-planning-reads";

type TProps = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The header of a project's modules list (P6a spec 2.6, R5), of the route's project: its crumbs; the view of the list
 * (its search, order, filters and layout, ModuleViewHeader) for everyone who sees it, while the page shows it; and a
 * new module, by the project's dialogs' host (ProjectLevelModals), for those who write the project's modules
 * (canWriteModules: its admins and members, not one nerve shows only his own work items, and no one while they are
 * turned off).
 */
export const ModulesListHeader = observer(function ModulesListHeader(props: TProps) {
  const { workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  // store hooks
  const { toggleCreateModuleModal } = useCommandPalette();
  const { canWriteModules } = useUserPermissions();
  const { loader } = useProject();
  const { t } = useTranslation();
  const { listed } = usePlanningReads(workspaceSlug, projectId, "modules");

  const canWrite = canWriteModules(workspaceSlug, projectId);

  return (
    <Header>
      <Header.LeftItem>
        <Breadcrumbs onBack={() => navigate(-1)} isLoading={loader === "init-loader"}>
          <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
          <Breadcrumbs.Item
            component={
              <BreadcrumbLink
                label={t("modules")}
                href={`/${workspaceSlug}/projects/${projectId}/modules`}
                icon={<ModuleOutline className="h-4 w-4 text-tertiary" />}
                isLast
              />
            }
            isLast
          />
        </Breadcrumbs>
      </Header.LeftItem>
      <Header.RightItem>
        {listed && <ModuleViewHeader workspaceSlug={workspaceSlug} projectId={projectId} />}
        {canWrite && (
          <Button variant="primary" size="lg" onClick={() => toggleCreateModuleModal(true)}>
            <div className="block sm:hidden">{t("add")}</div>
            <div className="hidden sm:block">{t("project_module.add_module")}</div>
          </Button>
        )}
      </Header.RightItem>
    </Header>
  );
});
