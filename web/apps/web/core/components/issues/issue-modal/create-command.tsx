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
import { cycleOptions, moduleOptions } from "@/lib/planning-directory";
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
import { usePlanningListReads } from "@/hooks/use-planning-names";
import type { RootStore } from "@/store/root.store";
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

type PlanningValues = Pick<IssueFormData, "cycle_id" | "module_ids">;

/**
 * Of the cycle and modules a new work item's caller gave (a row's copy, its source's: P7a spec 2.6, 2.7, R15; re-72
 * F8), those that are its options in the project (cycleOptions, moduleOptions, as the store holds the project's lists):
 * none of a domain the project has turned off, nor of one whose list the store does not hold (its read refused: the
 * form then holds none of them, and sends none; the modal keeps what was given until a retry answers). nerve refuses a
 * creation that names any other (planning.cycle_completed, planning.archived).
 */
function givenOptions(root: RootStore, projectId: string, given: PlanningValues): Required<PlanningValues> {
  const project = root.projectRoot.project.getProjectById(projectId);
  const { cycles, modules } = root.planning;
  const on = (domain: PlanningDomain) => !!project && planningOn(project, domain);
  const cycleIds = (on("cycles") && cycleOptions(cycles.getProjectCycleIds(projectId), cycles.getCycleById)) || [];
  const moduleIds =
    (on("modules") && moduleOptions(modules.getProjectModuleIds(projectId), modules.getModuleById)) || [];
  return {
    cycle_id: cycleIds.find((cycle) => !!cycle && cycle.id === given.cycle_id)?.id ?? null,
    module_ids: (given.module_ids ?? []).filter((id) => moduleIds.some((item) => item.id === id)),
  };
}

/**
 * A new work item's form as the modal starts it (P7a spec 2.6, R2, R15; re-72 F2, F7, F8). Before the form mounts its
 * cycle and modules start with those the caller gave and, on a cycle's or a module's page whose creation sent now would
 * join it (joinable), in the page's project, the page's own beside them (the domain's start): merged into the form's
 * default values, so that the form holds them whatever fields it shows, and sends what it holds (useCreateCommand). As
 * it opens with its project it reads the project's two lists of what is not archived, the options; while either is out
 * its fields show loading, holding what was given; then of those it keeps the options (givenOptions), which the form
 * takes as its defaults, not as an edit. A list nerve refused its field says so, with a retry (planningRefusals). A
 * work item's edit and a draft's form start as they are and read nothing here.
 */
export function useCreateStart({ workspaceSlug, storeType, isDraft, projectId, data }: CreateStartInput) {
  const root = useContext(StoreContext);
  const params = useParams();
  const creating = !isDraft && !data?.id;
  const reads = usePlanningListReads(workspaceSlug, creating ? (projectId ?? undefined) : undefined);
  const read = { planningLoading: reads.out, planningRefusals: reads.refusals };
  if (!creating || !projectId) return { data, ...read };
  const given = pick(data ?? {}, ["cycle_id", "module_ids"]);
  const kept = reads.out ? given : givenOptions(root, projectId, given);
  const start = { data: { ...data, ...kept }, ...read };
  if (!isPlanningPageStoreType(storeType)) return start;
  const domain = PLANNING_PAGE_DOMAINS[storeType];
  const entityID = params[domain.param];
  if (!entityID || !params.projectId || projectId !== params.projectId) return start;
  const page = { projectID: params.projectId, entityID };
  if (!domain.joinable(root, page)) return start;
  return { ...start, data: { ...start.data, ...domain.start(kept, page) } };
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
