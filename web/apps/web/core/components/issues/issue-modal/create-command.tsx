/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { Issue, Draft, IssueDraftPayload } from "@nerve/api-client";

import type { IssueFormData, IssueDescriptionLoadState, IssueFormSubmitOptions } from "./types";
import { useContext } from "react";
import { pick } from "lodash-es";
import { useParams } from "react-router";
// Nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import type { EIssuesStoreType } from "@nerve/types";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { sessionGuard } from "@/lib/in-session";
import { planningOn, type PlanningDomain } from "@/lib/planning-switch";
import { StoreContext } from "@/lib/store-context";
import { draftInput } from "@/services/issue/draft-input";
import { ContentCopy, CopyUnconfirmedError } from "./content-copy";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// hooks
import { useProject } from "@/hooks/store/use-project";
import { useWorkspaceDraftIssues } from "@/hooks/store/workspace-draft";
import { isPlanningPageStoreType, PLANNING_PAGE_DOMAINS } from "@/hooks/planning-page-domains";
import type { IssueActions } from "@/hooks/use-issues-actions";
import { usePlanningListsAnswered } from "@/hooks/use-planning-names";
// local imports
import { CreateIssueToastActionItems } from "../create-issue-toast-action-items";

type CreateCommandInput = {
  workspaceSlug: string | undefined;
  /** The seed of a seeded draft form; undefined for every other form. */
  draftSeed: IssueDraftPayload | undefined;
  copy: ContentCopy | undefined;
  formScope: () => () => boolean;
  /** The page's creation of what the form gives (createIssueAsGiven), nothing of the page added. */
  createIssue: IssueActions["createIssueAsGiven"];
  descriptionLoadState: IssueDescriptionLoadState;
};

type CreateStartInput = {
  workspaceSlug: string | undefined;
  storeType: EIssuesStoreType;
  isDraft: boolean;
  /** The project the form opens with. */
  projectId: string | null;
  /** What the form's caller gave; a work item's or a draft's own once it has an id. */
  data: IssueFormData | undefined;
};

/**
 * A new work item's form as the modal starts it (P7a spec 2.6, R2; re-72 F2, F7). Before the form mounts its cycle and
 * modules start with those the caller gave and, on a cycle's or a module's page whose creation sent now would join it
 * (joinable), in the page's project, the page's own (the domain's start): merged into the form's default values, so
 * that the form holds them whatever fields it shows, and sends what it holds (useCreateCommand). As it opens with its
 * project it reads the project's two lists of what is not archived, the options; until both have answered its fields
 * show loading, holding their values. A work item's edit and a draft's form start as they are and read nothing here.
 */
export function useCreateStart({ workspaceSlug, storeType, isDraft, projectId, data }: CreateStartInput) {
  const root = useContext(StoreContext);
  const params = useParams();
  const creating = !isDraft && !data?.id;
  const answered = usePlanningListsAnswered(workspaceSlug, creating ? (projectId ?? undefined) : undefined);
  const start = { data, planningLoading: !answered };
  if (!creating || !isPlanningPageStoreType(storeType)) return start;
  const domain = PLANNING_PAGE_DOMAINS[storeType];
  const entityID = params[domain.param];
  if (!entityID || !params.projectId || projectId !== params.projectId) return start;
  const page = { projectID: params.projectId, entityID };
  if (!domain.joinable(root, page)) return start;
  return { ...start, data: { ...data, ...domain.start(pick(data ?? {}, ["cycle_id", "module_ids"]), page) } };
}

/** The modal's create: a work item, or a draft when the form saves one. */
export function useCreateCommand({
  workspaceSlug,
  draftSeed,
  copy,
  formScope,
  createIssue,
  descriptionLoadState,
}: CreateCommandInput) {
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const draftIssues = useWorkspaceDraftIssues();
  const { getProjectById } = useProject();

  return async (
    payload: IssueFormData,
    is_draft_issue: boolean = false,
    options?: IssueFormSubmitOptions
  ): Promise<Issue | Draft | undefined> => {
    if (!workspaceSlug || (!is_draft_issue && !payload.project_id) || descriptionLoadState !== "ready") return;
    // A seeded draft form submits only its edits; the seed holds the rest, images included.
    const input = draftInput(payload, draftSeed);
    const copying =
      copy && ContentCopy.needed(input.payload.description_html) && draftIssues.canRead(workspaceSlug)
        ? copy
        : undefined;
    if (copying?.blocked) throw new CopyUnconfirmedError();

    const session = sessionGuard();
    const current = formScope();
    const inSession = () => session() && current();
    try {
      let response: Issue | Draft | undefined;
      if (copying) {
        if (!is_draft_issue && !payload.name?.trim()) return;
        response = await copying.run(input, !is_draft_issue);
      } else if (is_draft_issue) {
        response = await draftIssues.createIssue(workspaceSlug, input);
      } else if (createIssue) {
        if (!payload.name?.trim()) return;
        // The form's cycle and modules go as it shows them: none while empty, or while its project's switch hides them.
        const project = getProjectById(payload.project_id ?? undefined);
        const shows = (domain: PlanningDomain) => !!project && planningOn(project, domain);
        response = await createIssue(payload.project_id!, {
          name: payload.name,
          description_html: payload.description_html,
          ...(payload.state_id ? { state_id: payload.state_id } : {}),
          priority: payload.priority,
          assignee_ids: payload.assignee_ids,
          label_ids: payload.label_ids,
          ...(payload.start_date != null ? { start_date: payload.start_date } : {}),
          ...(payload.target_date != null ? { target_date: payload.target_date } : {}),
          ...(payload.parent_id != null ? { parent_id: payload.parent_id } : {}),
          ...(payload.cycle_id && shows("cycles") ? { cycle_id: payload.cycle_id } : {}),
          ...(payload.module_ids?.length && shows("modules") ? { module_ids: payload.module_ids } : {}),
        });
      }

      if (!inSession()) throw new SessionChangedError();
      if (!response) throw new Error("Missing confirmed work item");
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("success"),
        message: `${is_draft_issue ? t("draft_created") : t("issue_created_successfully")} `,
        actionItems: !is_draft_issue && response?.project_id && (
          <CreateIssueToastActionItems workspaceSlug={workspaceSlug} issueId={response.id} />
        ),
      });
      return response;
    } catch (error) {
      if (inSession()) (options?.onRefused ?? refuse)(error);
      throw error;
    }
  };
}
