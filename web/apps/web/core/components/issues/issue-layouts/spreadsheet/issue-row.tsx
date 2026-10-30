/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, IssueDetail } from "@nerve/api-client";

import type { Dispatch, MouseEvent, MutableRefObject, SetStateAction } from "react";
import { useRef } from "react";
import { observer } from "mobx-react";
import { useParams } from "react-router";
import { ChevronRightOutline, MoreHorizontalOutline } from "@makeplane/propel/icons";
// nerve helpers
// types
import { Tooltip } from "@makeplane/propel/components/tooltip";
import type { IIssueDisplayProperties } from "@nerve/types";
// ui
import { ControlLink, Row } from "@nerve/ui";
import { cn, generateWorkItemLink } from "@nerve/utils";
// components
import RenderIfVisible from "@/components/core/render-if-visible-HOC";
import { IssueIdentifier } from "@/components/issues/issue-detail/issue-identifier";
// hooks
import { useIssueDetail } from "@/hooks/store/use-issue-detail";
import { useIssues } from "@/hooks/store/use-issues";
import { useProject } from "@/hooks/store/use-project";
import useIssuePeekOverviewRedirection from "@/hooks/use-issue-peek-overview-redirection";
import { usePlatformOS } from "@/hooks/use-platform-os";
// local components
import type { TRenderQuickActions } from "../list/list-view-types";
import { isIssueNew } from "../utils";
import { IssueColumn } from "./issue-column";
import { useRowExpansion } from "../row-expansion";
import { useTranslation } from "@nerve/i18n";
import { DetailResourceError } from "../../issue-detail-widgets/resource-error";

interface Props {
  displayProperties: IIssueDisplayProperties;
  quickActions: TRenderQuickActions;
  canEditProperties: (projectId: string | undefined) => boolean;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<IssueDetail>) => Promise<void>) | undefined;
  portalElement: React.MutableRefObject<HTMLDivElement | null>;
  nestingLevel: number;
  issueId: string;
  isScrolled: MutableRefObject<boolean>;
  containerRef: MutableRefObject<HTMLDivElement | null>;
  spreadsheetColumnsList: (keyof IIssueDisplayProperties)[];
  spacingLeft?: number;
  shouldRenderByDefault?: boolean;
}

export const SpreadsheetIssueRow = observer(function SpreadsheetIssueRow(props: Props) {
  const {
    displayProperties,
    issueId,
    nestingLevel,
    portalElement,
    updateIssue,
    quickActions,
    canEditProperties,
    isScrolled,
    containerRef,
    spreadsheetColumnsList,
    spacingLeft = 6,
    shouldRenderByDefault,
  } = props;
  // states
  const [isExpanded, setExpanded] = useRowExpansion(JSON.stringify([issueId, nestingLevel]));
  // store hooks
  const { subIssues: subIssuesStore } = useIssueDetail();
  const { issueMap } = useIssues();

  // derived values
  const issue = issueMap[issueId];
  const subIssues = subIssuesStore.subIssuesByIssueId(issueId);

  if (!issue) return null;

  return (
    <>
      {/* first column/ issue name and key column */}
      <RenderIfVisible
        as="tr"
        root={containerRef}
        placeholderChildren={
          <td
            colSpan={100}
            className="border-[0.5px] border-transparent border-b-subtle-1"
            style={{ height: "calc(2.75rem - 1px)" }}
          />
        }
        classNames="bg-surface-1 transition-[background-color]"
        verticalOffset={100}
        shouldRecordHeights={false}
        defaultValue={shouldRenderByDefault || isIssueNew(issue)}
      >
        <IssueRowDetails
          issueId={issueId}
          displayProperties={displayProperties}
          quickActions={quickActions}
          canEditProperties={canEditProperties}
          nestingLevel={nestingLevel}
          spacingLeft={spacingLeft}
          updateIssue={updateIssue}
          portalElement={portalElement}
          isScrolled={isScrolled}
          isExpanded={isExpanded}
          setExpanded={setExpanded}
          spreadsheetColumnsList={spreadsheetColumnsList}
        />
      </RenderIfVisible>

      {isExpanded &&
        subIssues?.map((subIssueId: string) => (
          <SpreadsheetIssueRow
            key={subIssueId}
            issueId={subIssueId}
            displayProperties={displayProperties}
            quickActions={quickActions}
            canEditProperties={canEditProperties}
            nestingLevel={nestingLevel + 1}
            spacingLeft={spacingLeft + 12}
            updateIssue={updateIssue}
            portalElement={portalElement}
            isScrolled={isScrolled}
            containerRef={containerRef}
            spreadsheetColumnsList={spreadsheetColumnsList}
            shouldRenderByDefault={isExpanded}
          />
        ))}
    </>
  );
});

interface IssueRowDetailsProps {
  displayProperties: IIssueDisplayProperties;
  quickActions: TRenderQuickActions;
  canEditProperties: (projectId: string | undefined) => boolean;
  updateIssue: ((projectId: string | null, issueId: string, data: Partial<IssueDetail>) => Promise<void>) | undefined;
  portalElement: React.MutableRefObject<HTMLDivElement | null>;
  nestingLevel: number;
  issueId: string;
  isScrolled: MutableRefObject<boolean>;
  isExpanded: boolean;
  setExpanded: Dispatch<SetStateAction<boolean>>;
  spreadsheetColumnsList: (keyof IIssueDisplayProperties)[];
  spacingLeft?: number;
}

const IssueRowDetails = observer(function IssueRowDetails(props: IssueRowDetailsProps) {
  const {
    displayProperties,
    issueId,
    nestingLevel,
    portalElement,
    updateIssue,
    quickActions,
    canEditProperties,
    isScrolled,
    isExpanded,
    setExpanded,
    spreadsheetColumnsList,
    spacingLeft = 6,
  } = props;
  // states
  // refs
  const cellRef = useRef(null);
  // router
  const { workspaceSlug } = useParams();
  const { t } = useTranslation();
  // hooks
  const { getProjectIdentifierById } = useProject();
  const { getIsIssuePeeked, peekIssue } = useIssueDetail();
  const { handleRedirection } = useIssuePeekOverviewRedirection();
  const { isMobile } = usePlatformOS();

  // handlers
  const handleIssuePeekOverview = (issue: Issue, trigger: HTMLElement) =>
    handleRedirection(workspaceSlug, issue, isMobile, nestingLevel, trigger);

  const { subIssues: subIssuesStore, issue } = useIssueDetail();

  const issueDetail = issue.getIssueById(issueId);

  const subIssueIndentation = `${spacingLeft}px`;

  const customActionButton = <MoreHorizontalOutline className="h-3.5 w-3.5" />;
  if (!issueDetail) return null;

  const handleToggleExpand = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    e.preventDefault();
    if (nestingLevel >= 3) {
      handleIssuePeekOverview(issueDetail, e.currentTarget);
    } else {
      if (!isExpanded && workspaceSlug && issueDetail.project_id)
        void subIssuesStore
          .fetchSubIssues(workspaceSlug, issueDetail.project_id, issueDetail.id)
          .catch(() => undefined);
      setExpanded(!isExpanded);
    }
  };

  const disableUserActions = !canEditProperties(issueDetail.project_id ?? undefined);
  const subIssuesCount = issueDetail?.sub_issues_count ?? 0;
  const projectIdentifier = getProjectIdentifierById(issueDetail.project_id);

  const workItemLink = generateWorkItemLink({
    workspaceSlug,
    projectId: issueDetail?.project_id,
    issueId,
    projectIdentifier,
    sequenceId: issueDetail?.sequence_id,
  });

  return (
    <>
      {/* Single sticky column containing both identifier and workitem */}
      <td
        id={`issue-${issueId}`}
        ref={cellRef}
        tabIndex={0}
        className="group/list-block relative left-0 z-10 max-w-lg bg-surface-1 md:sticky"
      >
        <ControlLink
          href={workItemLink}
          onClick={(event) => handleIssuePeekOverview(issueDetail, event.currentTarget)}
          className="outline-none"
        >
          <Row
            className={cn(
              "group clickable z-10 flex h-11 w-full cursor-pointer items-center border-r-[0.5px] border-subtle-1 bg-transparent text-13 after:absolute",
              {
                "border-b-[0.5px]": !getIsIssuePeeked(issueDetail.id),
                "border border-accent-strong hover:border-accent-strong":
                  getIsIssuePeeked(issueDetail.id) && nestingLevel === peekIssue?.nestingLevel,
                "shadow-[8px_22px_22px_10px_rgba(0,0,0,0.05)]": isScrolled.current,
              }
            )}
          >
            {/* Identifier section - conditionally rendered */}
            {displayProperties?.key && (
              <div className="flex h-full min-w-24 flex-shrink-0 items-center">
                <div className="relative flex cursor-pointer items-center text-11 hover:text-primary">
                  {issueDetail.project_id && (
                    <IssueIdentifier
                      issueId={issueDetail.id}
                      projectId={issueDetail.project_id}
                      size="xs"
                      variant="tertiary"
                      displayProperties={displayProperties}
                    />
                  )}
                </div>
              </div>
            )}

            {/* Workitem section */}
            <div
              className={cn("flex flex-grow items-center gap-0.5 py-2", {
                "min-w-[360px]": !displayProperties?.key,
                "min-w-60": displayProperties?.key,
              })}
            >
              {/* sub issues indentation */}
              {nestingLevel !== 0 && <div style={{ width: subIssueIndentation }} />}

              {/* sub-issues chevron */}
              <div className="grid size-4 place-items-center">
                {subIssuesCount > 0 && (
                  <button
                    type="button"
                    className="grid size-4 place-items-center rounded-xs text-placeholder hover:text-tertiary"
                    aria-expanded={isExpanded}
                    aria-label={`${t("common.sub_work_items")}: ${issueDetail.name}`}
                    onClick={handleToggleExpand}
                  >
                    <ChevronRightOutline
                      className={cn("size-4", {
                        "rotate-90": isExpanded,
                      })}
                    />
                  </button>
                )}
              </div>

              <div className="my-auto flex h-full w-full items-center justify-between gap-2 truncate">
                <div className="line-clamp-1 w-full text-14 text-primary">
                  <div className="w-full overflow-hidden">
                    <Tooltip label={issueDetail.name} layout="stacked" disabled={isMobile}>
                      <div
                        className="h-full w-full cursor-pointer truncate pr-4 text-left text-13 text-primary focus:outline-none"
                        tabIndex={-1}
                      >
                        {issueDetail.name}
                      </div>
                    </Tooltip>
                  </div>
                </div>
                <div className="rounded-sm p-1 opacity-0 transition-opacity group-hover:opacity-100 focus-within:opacity-100 has-[[aria-expanded=true]]:bg-layer-1 has-[[aria-expanded=true]]:opacity-100">
                  {quickActions({
                    issue: issueDetail,
                    parentRef: cellRef,
                    customActionButton,
                    portalElement: portalElement.current,
                  })}
                </div>
              </div>
            </div>
          </Row>
        </ControlLink>
        {isExpanded && workspaceSlug && (
          <>
            {subIssuesStore.getLoading(issueId) && <div role="status">{t("common.loading")}</div>}
            <DetailResourceError
              error={subIssuesStore.getError(issueId)}
              retry={() => subIssuesStore.retry(workspaceSlug, issueDetail.project_id, issueId)}
              workspaceSlug={workspaceSlug}
              projectId={issueDetail.project_id}
              issueId={issueId}
            />
          </>
        )}
      </td>
      {/* Rest of the columns */}
      {spreadsheetColumnsList.map((property) => (
        <IssueColumn
          key={property}
          displayProperties={displayProperties}
          issueDetail={issueDetail}
          disableUserActions={disableUserActions}
          property={property}
          updateIssue={updateIssue}
        />
      ))}
    </>
  );
});
