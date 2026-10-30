/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useRef, useState } from "react";
import type { Issue } from "@nerve/api-client";

import { observer } from "mobx-react";
import { useTranslation } from "@nerve/i18n";
import { ChevronRightOutline, CloseOutline, DeleteOutline, EditOutline, LinkOutline } from "@makeplane/propel/icons";
// nerve imports
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { TSubIssueOperations } from "@nerve/types";
import { EIssuesStoreType } from "@nerve/types";
import { ControlLink, CustomMenu } from "@nerve/ui";
import { cn, generateWorkItemLink } from "@nerve/utils";
// helpers
import { useSubIssueOperations } from "@/components/issues/issue-detail-widgets/sub-issues/helper";
import { WithDisplayPropertiesHOC } from "@/components/issues/issue-layouts/properties/with-display-properties-HOC";
// hooks
import { useIssueDetail, useIssueDetailModals } from "@/hooks/store/use-issue-detail";
import { useProject } from "@/hooks/store/use-project";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
// components
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// local components
import { SubIssuesListItemProperties } from "./properties";
import { SubIssuesListRoot } from "./root";
import { useDetailScope } from "../../use-detail-scope";
import { DetailResourceError } from "../../resource-error";

type Props = {
  workspaceSlug: string;
  projectId: string;
  parentIssueId: string;
  rootIssueId: string;
  spacingLeft: number;
  canEdit: boolean;
  handleIssueCrudState: (key: "update" | "delete", issueId: string, issue?: Issue | null) => void;
  subIssueOperations: TSubIssueOperations;
  issueId: string;
  storeType?: EIssuesStoreType;
};

export function SubIssuesListItem(props: Props) {
  return (
    <SubIssuesListItemContent
      key={JSON.stringify([props.workspaceSlug, props.projectId, props.parentIssueId, props.issueId])}
      {...props}
    />
  );
}
const SubIssuesListItemContent = observer(function SubIssuesListItemContent(props: Props) {
  const {
    workspaceSlug,
    projectId,
    parentIssueId,
    rootIssueId,
    issueId,
    spacingLeft = 10,
    canEdit,
    handleIssueCrudState,
    subIssueOperations,
    storeType = EIssuesStoreType.PROJECT,
  } = props;
  const { t } = useTranslation();
  const {
    issue: { getIssueById },
    subIssues: {
      filters: { getSubIssueFilters },
    },
  } = useIssueDetail();
  const {
    subIssues: { subIssueHelpersByIssueId, setSubIssueHelpers, getError, retry },
  } = useIssueDetail();
  const { fetchSubIssues } = useSubIssueOperations(workspaceSlug, projectId, issueId);
  const capture = useDetailScope(workspaceSlug, projectId, issueId);
  const pending = useRef(false);
  const [loading, setLoading] = useState(false);
  const { toggleDeleteIssueModal } = useIssueDetailModals();
  const { toggleCreateIssueModal } = useIssueDetailModals();
  const project = useProject();
  const { handleRedirection } = useIssuePeekOverviewRedirection();
  const { isMobile } = usePlatformOS();
  const issue = getIssueById(issueId);

  // derived values
  const projectDetail = (issue && issue.project_id && project.getProjectById(issue.project_id)) || undefined;

  const subIssueHelpers = subIssueHelpersByIssueId(parentIssueId);
  const subIssueCount = issue?.sub_issues_count ?? 0;

  // derived values
  const subIssueFilters = getSubIssueFilters(parentIssueId);
  const displayProperties = subIssueFilters?.displayProperties ?? {};

  //
  const handleIssuePeekOverview = (selected: Issue) => handleRedirection(workspaceSlug, selected, isMobile);

  if (!issue) return <></>;

  // check if current issue is the root issue
  const isCurrentIssueRoot = issueId === rootIssueId;

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issue?.project_id,
    issueId: issue?.id,
    projectIdentifier: projectDetail?.identifier,
    sequenceId: issue?.sequence_id,
  });

  return (
    <div key={issueId}>
      <ControlLink
        id={`issue-${issue.id}`}
        href={workItemLink}
        onClick={() => handleIssuePeekOverview(issue)}
        className="w-full cursor-pointer"
      >
        {issue && (
          <div
            className="group relative flex h-full min-h-11 w-full items-center py-1 pr-2 transition-all hover:bg-surface-2"
            style={{ paddingLeft: `${spacingLeft}px` }}
          >
            <div className="flex size-5 flex-shrink-0 items-center justify-center">
              {/* disable the chevron when current issue is also the root issue*/}
              {subIssueCount > 0 && !isCurrentIssueRoot && (
                <>
                  {loading ? (
                    <div className="flex h-full w-full cursor-not-allowed items-center justify-center rounded-xs bg-layer-1 transition-all">
                      <LinkOutline width={14} height={14} className="animate-spin" />
                    </div>
                  ) : (
                    <button
                      type="button"
                      aria-label={t("common.sub_work_items")}
                      aria-expanded={subIssueHelpers.issue_visibility.includes(issueId)}
                      className="flex h-full w-full cursor-pointer items-center justify-center text-placeholder hover:text-tertiary"
                      onClick={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        if (pending.current) return;
                        const current = capture();
                        pending.current = true;
                        setLoading(true);
                        try {
                          if (!subIssueHelpers.issue_visibility.includes(issueId)) {
                            await fetchSubIssues(workspaceSlug, projectId, issueId);
                          }
                          if (current()) setSubIssueHelpers(parentIssueId, "issue_visibility", issueId);
                        } catch {
                          // The resource retains the refusal for the inline read-only retry.
                        } finally {
                          pending.current = false;
                          if (current()) setLoading(false);
                        }
                      }}
                    >
                      <ChevronRightOutline
                        className={cn("size-3.5 transition-all", {
                          "rotate-90": subIssueHelpers.issue_visibility.includes(issue.id),
                        })}
                      />
                    </button>
                  )}
                </>
              )}
            </div>

            <div className="flex w-full cursor-pointer items-center gap-3 truncate">
              <WithDisplayPropertiesHOC displayProperties={displayProperties || {}} displayPropertyKey="key">
                <div className="flex-shrink-0">
                  {projectDetail && (
                    <IssueIdentifier
                      projectId={projectDetail.id}
                      projectIdentifier={projectDetail.identifier}
                      issueSequenceId={issue.sequence_id}
                      size="xs"
                      variant="secondary"
                    />
                  )}
                </div>
              </WithDisplayPropertiesHOC>
              <Tooltip label={issue.name} layout="stacked" disabled={isMobile}>
                <span className="w-0 flex-1 truncate text-13 text-primary">{issue.name}</span>
              </Tooltip>
            </div>

            <div
              className="flex-shrink-0 text-13"
              role="group"
              onKeyDown={(e) => e.stopPropagation()}
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
              }}
            >
              <SubIssuesListItemProperties
                workspaceSlug={workspaceSlug}
                parentIssueId={parentIssueId}
                issueId={issueId}
                canEdit={canEdit}
                updateSubIssue={subIssueOperations.updateSubIssue}
                displayProperties={displayProperties}
                issue={issue}
              />
            </div>

            <div className="flex-shrink-0 text-13">
              <CustomMenu placement="bottom-end" ellipsis>
                {canEdit && (
                  <CustomMenu.MenuItem
                    onClick={() => {
                      handleIssueCrudState("update", parentIssueId, { ...issue });
                      toggleCreateIssueModal(true);
                    }}
                  >
                    <div className="flex items-center gap-2">
                      <EditOutline className="h-3.5 w-3.5" />
                      <span>{t("issue.edit")}</span>
                    </div>
                  </CustomMenu.MenuItem>
                )}

                <CustomMenu.MenuItem
                  onClick={() => {
                    subIssueOperations.copyLink(workItemLink);
                  }}
                >
                  <div className="flex items-center gap-2">
                    <LinkOutline className="h-3.5 w-3.5" />
                    <span>{t("issue.copy_link")}</span>
                  </div>
                </CustomMenu.MenuItem>

                {canEdit && (
                  <CustomMenu.MenuItem
                    onClick={() => {
                      if (issue.project_id)
                        subIssueOperations.removeSubIssue(workspaceSlug, issue.project_id, parentIssueId, issue.id);
                    }}
                  >
                    <div className="flex items-center gap-2">
                      <CloseOutline className="h-3.5 w-3.5" />
                      {t("issue.remove.parent.label")}
                    </div>
                  </CustomMenu.MenuItem>
                )}

                {canEdit && (
                  <CustomMenu.MenuItem
                    onClick={() => {
                      handleIssueCrudState("delete", parentIssueId, issue);
                      toggleDeleteIssueModal(issue.id);
                    }}
                  >
                    <div className="flex items-center gap-2">
                      <DeleteOutline className="h-3.5 w-3.5" />
                      <span>{t("issue.delete.label")}</span>
                    </div>
                  </CustomMenu.MenuItem>
                )}
              </CustomMenu>
            </div>
          </div>
        )}
      </ControlLink>

      {/* should not expand the current issue if it is also the root issue*/}
      <DetailResourceError
        error={getError(issueId)}
        workspaceSlug={workspaceSlug}
        projectId={projectId}
        issueId={issueId}
        retry={async () => {
          const current = capture();
          await retry(workspaceSlug, projectId, issueId);
          if (
            current() &&
            !getError(issueId) &&
            !subIssueHelpersByIssueId(parentIssueId).issue_visibility.includes(issueId)
          )
            setSubIssueHelpers(parentIssueId, "issue_visibility", issueId);
        }}
      />
      {subIssueHelpers.issue_visibility.includes(issueId) &&
        issue.project_id &&
        subIssueCount > 0 &&
        !isCurrentIssueRoot && (
          <SubIssuesListRoot
            storeType={storeType}
            workspaceSlug={workspaceSlug}
            projectId={issue.project_id}
            parentIssueId={issue.id}
            rootIssueId={rootIssueId}
            spacingLeft={spacingLeft + 22}
            canEdit={canEdit}
            handleIssueCrudState={handleIssueCrudState}
            subIssueOperations={subIssueOperations}
          />
        )}
    </div>
  );
});
