/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
import { ChevronDownOutline } from "@makeplane/propel/icons";
// nerve imports
import { MODULE_VIEW_LAYOUTS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { CustomMenu, Row } from "@nerve/ui";
// components
import { ModuleLayoutIcon } from "@/components/modules/module-layout-icon";
import { useModuleDisplay } from "@/components/modules/use-module-display";
// hooks
import { usePlanningReads } from "@/hooks/use-planning-reads";

type Props = {
  workspaceSlug: string;
  projectId: string;
};

/**
 * The small screen's header of a project's modules list (P6a spec 2.6, R4), of the route's project: the menu of the
 * list's layouts, which the browser keeps for the project (useModuleDisplay), while the page shows the list.
 */
export const ModulesListMobileHeader = observer(function ModulesListMobileHeader({ workspaceSlug, projectId }: Props) {
  const { t } = useTranslation();
  const { listed } = usePlanningReads(workspaceSlug, projectId, "modules");
  const { setLayout } = useModuleDisplay(projectId);

  if (!listed) return null;

  return (
    <div className="flex justify-start md:hidden">
      <CustomMenu
        maxHeight="md"
        className="flex flex-grow justify-start border-b border-subtle bg-surface-1 py-2 text-13 text-secondary"
        customButton={
          <Row className="flex flex-grow justify-center gap-2 text-13 text-secondary">
            <span>{t("common.layout")}</span>
            <ChevronDownOutline className="my-auto h-4 w-4 text-secondary" aria-hidden="true" />
          </Row>
        }
        customButtonClassName="flex flex-grow justify-center items-center text-secondary text-13"
        closeOnSelect
      >
        {MODULE_VIEW_LAYOUTS.map((layout) => (
          <CustomMenu.MenuItem
            key={layout.key}
            onClick={() => setLayout(layout.key)}
            className="flex items-center gap-2"
          >
            <ModuleLayoutIcon layout={layout.key} />
            <div className="text-tertiary">{t(layout.i18n_title)}</div>
          </CustomMenu.MenuItem>
        ))}
      </CustomMenu>
    </div>
  );
});
