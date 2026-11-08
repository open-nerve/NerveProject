import type { Api } from "./api";
import { issueCreated, issueRead, issueUpdated, type IssueCreate, type IssueUpdate } from "./issues";

// The calls of the stories of a work item's cycle and modules (M6 design G4, G11, G14, M6 P3a spec 2.3).

/** A work item's answer, or a refusal's, as the planning stories compare it. */
type Answered = Awaited<ReturnType<typeof issueRead>>;

/** answered as a planning story compares it: its status and the work item's cycle and modules, or the problem. */
function placed(answered: Answered) {
  return "issue" in answered
    ? { status: answered.status, cycle_id: answered.issue.cycle_id, module_ids: answered.issue.module_ids }
    : answered;
}

/**
 * The calls on work items whose answer a planning story compares, each by the caller of token: its status and the
 * work item's cycle and modules, or the problem's code and fields.
 */
export function issuePlanningCalls(api: Api) {
  return {
    create: async (token: string, projectId: string, body: IssueCreate) =>
      placed(await issueCreated(api, token, projectId, body)),
    update: async (token: string, issueId: string, body: IssueUpdate) =>
      placed(await issueUpdated(api, token, issueId, body)),
    read: async (token: string, issueId: string) => placed(await issueRead(api, token, issueId)),
  };
}

/** The answer of a write refused with validation_failed for problems, each its field and its code, in order. */
export function validationFailed(...problems: [field: string, code: string][]) {
  return { status: 422, code: "validation_failed", errors: problems.map(([field, code]) => ({ field, code })) };
}

/** The ids of rows, ascending: the order a work item answers its modules in. */
export function ascendingIds(...rows: { id: string }[]): string[] {
  return rows.map((r) => r.id).toSorted();
}
