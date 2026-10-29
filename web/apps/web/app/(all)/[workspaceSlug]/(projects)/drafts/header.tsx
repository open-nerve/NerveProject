/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { observer } from "mobx-react";
import { EUserPermissions, EUserPermissionsLevel } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
// ui
import { Button } from "@nerve/propel/button";
import { DraftsOutline } from "@makeplane/propel/icons";
import { EIssuesStoreType } from "@nerve/types";
import { Breadcrumbs, Header } from "@nerve/ui";
// components
import { BreadcrumbLink } from "@/components/common/breadcrumb-link";
import { CountChip } from "@/components/common/count-chip";
import { CreateUpdateIssueModal } from "@/components/issues/issue-modal/modal";

// hooks
import { useParams } from "react-router";
import { useUserPermissions } from "@/hooks/store/user";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";

export const WorkspaceDraftHeader = observer(function WorkspaceDraftHeader() {
  // state
  const [isDraftIssueModalOpen, setIsDraftIssueModalOpen] = useState(false);
  // store hooks
  const { allowPermissions } = useUserPermissions();
  const { getTotal } = useWorkspaceDraftIssues();
  const { workspaceSlug } = useParams();
  const total = workspaceSlug ? getTotal(workspaceSlug) : undefined;

  const { t } = useTranslation();
  // check if user is authorized to create draft work item
  const isAuthorizedUser = allowPermissions(
    [EUserPermissions.ADMIN, EUserPermissions.MEMBER],
    EUserPermissionsLevel.WORKSPACE
  );

  return (
    <>
      <CreateUpdateIssueModal
        isOpen={isDraftIssueModalOpen}
        storeType={EIssuesStoreType.WORKSPACE_DRAFT}
        onClose={() => setIsDraftIssueModalOpen(false)}
        isDraft
      />
      <Header>
        <Header.LeftItem>
          <div className="flex items-center gap-2.5">
            <Breadcrumbs>
              <Breadcrumbs.Item
                component={
                  <BreadcrumbLink label={t("drafts")} icon={<DraftsOutline className="h-4 w-4 text-tertiary" />} />
                }
              />
            </Breadcrumbs>
            {total !== undefined && <CountChip count={total} />}
          </div>
        </Header.LeftItem>

        <Header.RightItem>
          {workspaceSlug && (
            <Button
              variant="primary"
              size="lg"
              className="items-center gap-1"
              onClick={() => setIsDraftIssueModalOpen(true)}
              disabled={!isAuthorizedUser}
            >
              {t("workspace_draft_issues.draft_an_issue")}
            </Button>
          )}
        </Header.RightItem>
      </Header>
    </>
  );
});
