/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useMemo, useState } from "react";
import { observer } from "mobx-react";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { usePowerK } from "@/hooks/store/use-power-k";
import { useParams, useNavigate } from "react-router";
// components
import { ProjectLevelModals } from "@/components/modals/project-level";
import { WorkItemLevelModals } from "@/components/modals/work-item-level";
import { WorkspaceLevelModals } from "@/components/modals/workspace-level";
// local imports
import { useProjectsAppPowerKCommands } from "./config/commands";
import type { TPowerKCommandConfig, TPowerKContext } from "./core/types";
import { GlobalShortcutsProvider } from "./global-shortcuts";
import { ProjectsAppPowerKCommandsList } from "./ui/modal/commands-list";
import { ProjectsAppPowerKModalWrapper } from "./ui/modal/wrapper";

/**
 * Projects App PowerK provider
 */
export const ProjectsAppPowerKProvider = observer(function ProjectsAppPowerKProvider() {
  // router
  const navigate = useNavigate();
  const params = useParams();
  const { workspaceSlug, projectId: routerProjectId, workItem: workItemIdentifier } = params;
  // states
  const [activeCommand, setActiveCommand] = useState<TPowerKCommandConfig | null>(null);
  const [shouldShowContextBasedActions, setShouldShowContextBasedActions] = useState(true);
  // store hooks
  const { activeContext, isPowerKModalOpen, togglePowerKModal, setActivePage } = usePowerK();
  // The store outlives this host and the host survives project/item route changes. Retire its palette before the
  // new route paints, including Back to the previous route. Use route identity, not asynchronously loaded detail.
  useLayoutEffect(
    () => () => {
      togglePowerKModal(false);
      setActivePage(null);
    },
    [workspaceSlug, routerProjectId, workItemIdentifier, togglePowerKModal, setActivePage]
  );
  // derived values
  const {
    issue: { getIssueById, getIssueIdByIdentifier },
  } = useIssueDetail();
  // derived values
  const workItemId = workItemIdentifier ? getIssueIdByIdentifier(workItemIdentifier) : undefined;
  const workItemDetails = workItemId ? getIssueById(workItemId) : undefined;
  const projectId = routerProjectId ?? workItemDetails?.project_id ?? undefined;
  const commands = useProjectsAppPowerKCommands();
  // Build command context from props and store
  const context: TPowerKContext = useMemo(
    () => ({
      activeCommand,
      activeContext,
      shouldShowContextBasedActions,
      setShouldShowContextBasedActions,
      params: {
        ...params,
        projectId,
      },
      navigate,
      closePalette: () => togglePowerKModal(false),
      setActiveCommand,
      setActivePage,
    }),
    [
      activeCommand,
      activeContext,
      shouldShowContextBasedActions,
      params,
      projectId,
      navigate,
      togglePowerKModal,
      setActivePage,
    ]
  );

  return (
    <>
      <GlobalShortcutsProvider context={context} commands={commands} />
      {workspaceSlug && <WorkspaceLevelModals workspaceSlug={workspaceSlug} />}
      {/* keyed by its project: the host closes its dialogs as it goes (ProjectLevelModals) */}
      {workspaceSlug && projectId && (
        <ProjectLevelModals key={projectId} workspaceSlug={workspaceSlug} projectId={projectId} />
      )}
      {workspaceSlug && <WorkItemLevelModals workItemIdentifier={workItemIdentifier} />}
      <ProjectsAppPowerKModalWrapper
        commandsListComponent={ProjectsAppPowerKCommandsList}
        context={context}
        isOpen={isPowerKModalOpen}
        onClose={() => togglePowerKModal(false)}
      />
    </>
  );
});
