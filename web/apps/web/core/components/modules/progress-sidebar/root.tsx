/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { observer } from "mobx-react";
// nerve imports
import { Loader } from "@nerve/ui";
// hooks
import { useModules } from "@/hooks/store/use-modules";
import { useUserPermissions } from "@/hooks/store/user";
// local imports
import { ModuleProgress } from "./issue-progress";
import { ModuleSidebarDetails } from "./sidebar-details";
import { ModuleSidebarHeader } from "./sidebar-header";
import { ModuleSidebarLinks } from "./sidebar-links";

type Props = {
  handleClose: () => void;
  moduleId: string;
  projectId: string;
  workspaceSlug: string;
};

/**
 * A module's progress sidebar (P6a spec 2.6), of the module as the store shows it (its page or its preview reads it,
 * an archived one's too, R12): its head, its details, its progress and its links. Its numbers are nerve's progress of
 * the module (R8). The caller changes its status, days, lead and members here while he writes the project's modules
 * (canWriteModules, R5) and the module is not archived; an archived module's sidebar is read-only.
 */
export const ModuleDetailsSidebar = observer(function ModuleDetailsSidebar(props: Props) {
  const { handleClose, moduleId, projectId, workspaceSlug } = props;
  // store hooks
  const { getModuleById, getModuleProgress } = useModules();
  const { canWriteModules } = useUserPermissions();
  const module = getModuleById(moduleId);
  const progress = getModuleProgress(moduleId);

  if (!module)
    return (
      <Loader className="px-5">
        <div className="space-y-2">
          <Loader.Item height="15px" width="50%" />
          <Loader.Item height="15px" width="30%" />
        </div>
        <div className="mt-8 space-y-3">
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
          <Loader.Item height="30px" />
        </div>
      </Loader>
    );

  const editable = canWriteModules(workspaceSlug, projectId) && module.archived_at === null;
  return (
    <div className="relative pb-2">
      <div className="flex w-full flex-col gap-5">
        <ModuleSidebarHeader module={module} editable={editable} handleClose={handleClose} />
        <ModuleSidebarDetails module={module} projectId={projectId} progress={progress} editable={editable} />
      </div>
      <ModuleProgress module={module} projectId={projectId} progress={progress} />
      <ModuleSidebarLinks moduleId={module.id} editable={editable} />
    </div>
  );
});
