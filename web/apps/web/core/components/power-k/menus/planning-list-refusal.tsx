/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { problemMessageKey } from "@/lib/error-messages";
import { planningListRefusal, type PlanningDomain } from "@/lib/planning-switch";
// local imports
import { PowerKMenuEmptyState } from "./empty-state";

type Props = {
  /** The address's workspace, whose role in the project the reason asks of. */
  workspaceSlug: string | undefined;
  projectId: string | undefined;
  domain: PlanningDomain;
};

/**
 * Why a page of the palette lists none of a project's cycles or modules while the gate of its read is closed
 * (usePlanningReads's listed, P7c spec 2.2, R2): no role in the project, the project archived, the domain turned off,
 * in planningListRefusal's order, the facts usePlanningReads asks. Said at once, before any list the store may still
 * hold, waiting for nothing; a page renders it only while its gate is closed, so that the role is asked only then.
 */
export const PowerKPlanningListRefusal = observer(function PowerKPlanningListRefusal(props: Props) {
  const { workspaceSlug, projectId, domain } = props;
  // store hooks
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();
  const { t } = useTranslation();
  const role = workspaceSlug ? getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) : undefined;
  const refusal = planningListRefusal(getProjectById(projectId), role, domain);
  return <PowerKMenuEmptyState emptyText={t(problemMessageKey(refusal))} />;
});
