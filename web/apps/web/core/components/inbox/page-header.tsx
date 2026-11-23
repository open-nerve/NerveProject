/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { IntakeOutline } from "@makeplane/propel/icons";
// ui
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { CommonProjectBreadcrumbs } from "@/components/breadcrumbs/common";
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { InboxIssueCreateModalRoot } from "@/components/inbox/modals/create-modal";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useUserPermissions } from "@/hooks/store/user";
// lib
import { intakeEntryOn } from "@/lib/intake-entry";

/**
 * The intake page's header (P5a spec 2.7, R21, R27; HO/M3-closeout.md:64): the route's project's breadcrumbs, the
 * intake's named in the page's language, and "Add work item", which opens the create modal, only where the caller may
 * submit: the project's intake is turned on, he has a role in it (an admin's, a member's or a guest's: intakeEntryOn),
 * and it is not archived (nerve refuses a submission to it, project.archived). By the route's project and the caller's
 * role in it, never the router store's.
 */
export const IntakePageHeader = observer(function IntakePageHeader() {
  // states
  const [createIssueModal, setCreateIssueModal] = useState(false);
  // router
  const { workspaceSlug, projectId } = useParams();
  // store hooks
  const { t } = useTranslation();
  const { getProjectById } = useProject();
  const { getProjectRoleByWorkspaceSlugAndProjectId } = useUserPermissions();

  if (!workspaceSlug || !projectId) return null;

  // derived values
  const project = getProjectById(projectId);
  const canSubmit =
    intakeEntryOn(project, getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId)) &&
    project?.archived_at === null;

  return (
    <Header>
      <Header.LeftItem>
        <div className="flex flex-grow items-center gap-4">
          <Breadcrumbs>
            <CommonProjectBreadcrumbs workspaceSlug={workspaceSlug} projectId={projectId} />
            <Breadcrumbs.Item
              component={
                <BreadcrumbLink
                  label={t("sidebar.intake")}
                  href={`/${workspaceSlug}/projects/${projectId}/intake`}
                  icon={<IntakeOutline className="h-4 w-4 text-tertiary" />}
                  isLast
                />
              }
              isLast
            />
          </Breadcrumbs>
        </div>
      </Header.LeftItem>
      <Header.RightItem>
        {canSubmit && (
          <div className="flex items-center gap-2">
            <InboxIssueCreateModalRoot
              workspaceSlug={workspaceSlug}
              projectId={projectId}
              modalState={createIssueModal}
              handleModalClose={() => setCreateIssueModal(false)}
            />
            <Button variant="primary" size="lg" onClick={() => setCreateIssueModal(true)}>
              {t("add_work_item")}
            </Button>
          </div>
        )}
      </Header.RightItem>
    </Header>
  );
});
