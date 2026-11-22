/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { toJS } from "mobx";
import type { Issue, IssueDetail as IssueDetailAnswer } from "@nerve/api-client";
import { problemCode } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { compareIssueTime } from "@/lib/issue-order";
import type { Before, ConfirmedIssueChange } from "@/lib/paged-groups";
import { heldCounterSources } from "@/lib/paged-groups.evidence";
import type { IssueListBinding } from "./helpers/issue-list-binding";
import { membershipFields } from "./issue-write-fields";
import type { IssueDrop } from "./issue-writes";
import type { IssueRootStore, IssueWriteScope } from "./root.store";

type PendingWrite = {
  before: Before;
  sources: Map<IssueListBinding, ReadonlySet<number>>;
  observed: { version: string; fields: string }[];
};
/**
 * What the writes read of their generation's issue root, which passes itself in at construction. A type only: the
 * root imports this unit to build it, so this unit never imports the root as a value.
 */
export type WritesRoot = Pick<
  IssueRootStore,
  | "rootStore"
  | "api"
  | "issues"
  | "issueDetail"
  | "permissionReads"
  | "registeredLists"
  | "assertActive"
  | "scope"
  | "invalidateProjectReads"
  | "intake"
  | "intakeApi"
  | "verification"
>;

/** nerve's refusals of a write of a work item another triaged first (R28): it is read again in the refused write's turn. */
const TRIAGED_ELSEWHERE = new Set(["issue.intake_transition_not_allowed", "issue.intake_closed"]);

/**
 * A change nerve answered, written into every held copy (P6b R1; P5a spec 2.4): the writes in flight, the reads that
 * arrive meanwhile, and the confirmation, by write() or, for a work item in an intake, by the intake's route beside it.
 * The writes' unit (issue-writes.ts) holds it, sends its writes one at a time per issue and forwards its two public
 * members; a type only of that unit's IssueDrop, as that unit imports this one to build it.
 */
export class IssueWriteConfirmation {
  private readonly pending = new Map<string, PendingWrite>();
  constructor(private readonly root: WritesRoot) {}
  get pendingIDs(): ReadonlySet<string> {
    return new Set(this.pending.keys());
  }
  /** Keeps a read that arrived while a write to its issue is out, for that write's confirmation; false if none is out. */
  observePendingRead(after: Issue): boolean {
    const pending = this.pending.get(after.id);
    pending?.observed.push({ version: after.updated_at, fields: membershipFields(after) });
    return pending !== undefined;
  }
  async write(
    id: string | undefined,
    requestedScope: IssueWriteScope | Pick<IssueWriteScope, "workspaceSlug" | "workspaceID">,
    request: (held?: Issue) => Promise<IssueDetailAnswer | null>,
    drop?: IssueDrop
  ): Promise<IssueDetailAnswer | null> {
    const root = this.root;
    root.assertActive();
    const permission = root.permissionReads.capture();
    const initialScope = "projectID" in requestedScope ? requestedScope : undefined;
    let initial = id ? root.issues.getIssueById(id) : undefined;
    while (initial) {
      if (!initialScope) throw new Error("An existing issue needs its project scope");
      for (const list of root.registeredLists)
        if (list.covers(initialScope) && list.reconciled.value !== undefined) {
          // Each directory wait may change canonical; freeze only after this serial readiness pass.
          // oxlint-disable-next-line no-await-in-loop
          await list.prepareCatalog([initial]);
        }
      root.assertActive();
      const latest = id ? root.issues.getIssueById(id) : undefined;
      if (latest === initial) break;
      initial = latest;
    }
    root.assertActive();
    if (
      !root.permissionReads.current(
        permission,
        requestedScope.workspaceID,
        initialScope ? [initialScope.projectID] : [],
        true
      )
    )
      throw new SessionChangedError();
    const held = id ? root.issues.getIssueById(id) : undefined;
    const pending: PendingWrite = {
      before: id
        ? held
          ? { kind: "known", issue: structuredClone(toJS(held)) }
          : { kind: "unknown" }
        : { kind: "absent" },
      sources: new Map([...root.registeredLists].map((list) => [list, heldCounterSources(list.reconciled.value)])),
      observed: [],
    };
    if (id) this.pending.set(id, pending);
    try {
      const after = await request(held);
      root.assertActive();
      if (after && after.workspace_id !== requestedScope.workspaceID) throw new SessionChangedError();
      if (!initialScope && !after) throw new Error("A creation must return its issue");
      const scope: IssueWriteScope = initialScope ?? { ...requestedScope, projectID: after!.project_id };
      const current = () =>
        root.permissionReads.current(
          permission,
          scope.workspaceID,
          after ? [scope.projectID, after.project_id] : [scope.projectID],
          true
        );
      // The write succeeded, but an answer from an invalidated permission scope
      // cannot revise recovered entities or list membership.
      if (!current()) return after;
      const prepared = new Set<IssueListBinding>();
      const unavailable = new Set<IssueListBinding>();
      // A binding can be created while a request or directory is out. Each owns its query and proof.
      for (const list of root.registeredLists)
        if (list.covers(scope) && !prepared.has(list)) {
          prepared.add(list);
          try {
            await list.prepareCatalog([
              ...(pending.before.kind === "known" ? [pending.before.issue] : []),
              ...(after ? [after] : []),
            ]);
          } catch (error) {
            root.assertActive();
            if (!current()) return after;
            if (error instanceof SessionChangedError) throw error;
            // The issue write already succeeded. Keep that fact; only this list needs authoritative replacement.
            unavailable.add(list);
            list.markUnavailable(error);
          }
        }
      root.assertActive();
      if (!current()) return after;
      let before = pending.before;
      if (after && before.kind === "known") {
        const old = before.issue;
        if (
          pending.observed.some(
            (read) =>
              compareIssueTime(read.version, after.updated_at) > 0 ||
              (compareIssueTime(read.version, old.updated_at) > 0 &&
                (compareIssueTime(read.version, after.updated_at) === 0
                  ? read.fields !== membershipFields(after)
                  : read.fields !== membershipFields(old)))
          )
        )
          before = { kind: "unknown" };
      }
      const position = drop
        ? { group: drop.group, subgroup: drop.subgroup, beforeId: drop.beforeId, afterId: drop.afterId }
        : undefined;
      const change: ConfirmedIssueChange = { id: after?.id ?? id!, beforeSnapshot: before, after };
      if (!after) {
        root.invalidateProjectReads(scope.projectID, scope.workspaceID);
        for (const child of root.issues.knownDescendants(change.id)) {
          const deletion = { id: child.id, beforeSnapshot: { kind: "known" as const, issue: child }, after: null };
          for (const list of root.registeredLists)
            if (list.covers(scope)) list.acceptConfirmed(deletion, pending.sources.get(list) ?? new Set());
          root.issues.confirmChange(deletion);
        }
      }
      for (const list of root.registeredLists)
        if (list.covers(scope) && !unavailable.has(list))
          list.acceptConfirmed(
            { ...change, drop: list === drop?.binding ? position : undefined },
            pending.sources.get(list) ?? new Set()
          );
      root.issues.confirmChange(change, permission);
      await root.issueDetail.subIssues.confirmCore(change, scope);
      if (after) await root.issueDetail.refreshAfterChange(scope.workspaceSlug, scope.projectID, after.id);
      return after;
    } catch (error) {
      root.assertActive();
      throw error;
    } finally {
      if (id && this.pending.get(id) === pending) this.pending.delete(id);
    }
  }
  /**
   * A content change of a work item in its project's intake, sent by request on the intake's route (P5a spec 2.4):
   * canonical takes nerve's answer, its detail's activity and comments are read again, its intake lists take it; the
   * counts stay as they are.
   */
  edit(id: string, scope: IssueWriteScope, request: () => Promise<IssueDetailAnswer>): Promise<IssueDetailAnswer> {
    return this.intake(id, scope, request, async (answer, permission) => {
      this.root.issues.confirmChange({ id, beforeSnapshot: { kind: "unknown" }, after: answer }, permission);
      this.root.intake.confirm(answer);
      await this.root.issueDetail.refreshAfterChange(scope.workspaceSlug, scope.projectID, id);
    });
  }
  /**
   * A triage of a work item of an intake, sent by request on the intake's route (P5a spec 2.4, R34): its intake lists
   * take nerve's answer, canonical never does (a triage but an acceptance moves no updated_at, and canonical takes an
   * equal read, so a read sent before would undo it); the work item is read again from after it (readAgain), and the
   * counts are fetched again. Gives that read, or undefined for none.
   */
  async triage(
    id: string,
    scope: IssueWriteScope,
    request: () => Promise<IssueDetailAnswer>
  ): Promise<IssueDetailAnswer | undefined> {
    let read: IssueDetailAnswer | undefined;
    await this.intake(id, scope, request, async (answer) => {
      this.root.intake.confirm(answer);
      read = await this.readAgain(id, scope);
      this.countAgain(scope);
    });
    return read;
  }
  /**
   * A deletion of a work item of an intake, sent by request on the intake's route (P5a spec 2.4, R34): its intake row is
   * gone, so it leaves every intake list; then it is read again in the turn (readAgain), as nerve deletes the intake row
   * alone of one accepted meanwhile. Only a read that found none (the verification's missing, not a read overtaken or
   * undecided, which leaves canonical to the read that overtook it) deletes it from canonical; one read takes its place
   * there. No project list is touched; the counts are fetched again. A work item in an intake is no parent and no child
   * (M7 design), so none goes with it.
   */
  async delete(id: string, scope: IssueWriteScope, request: () => Promise<null>): Promise<void> {
    await this.intake(id, scope, request, async (_, permission) => {
      this.root.intake.drop(scope.projectID, id);
      if (!(await this.readAgain(id, scope)) && this.root.verification.isMissing(id))
        this.root.issues.confirmChange({ id, beforeSnapshot: { kind: "unknown" }, after: null }, permission);
      this.countAgain(scope);
    });
  }
  /**
   * The intake's route (P5a spec 2.4, R3; re-94 S1): a write of a work item of an intake, beside write(), with its
   * permission marks alone. It reads no project list's directory before it is sent or as it is confirmed (the work
   * item is in its project's triage state, which no directory lists, so a list that needs the states would fail it),
   * confirms no project list, makes none unavailable or read again (issueMembership does not read intake: one could
   * take it for a row), and keeps no pending write (only project lists read one). nerve's answer, from the permission
   * scope it was sent from, is confirmed by confirm; a refusal is followed (refused) and handed on. Gives the answer.
   */
  private async intake<T extends IssueDetailAnswer | null>(
    id: string,
    scope: IssueWriteScope,
    request: () => Promise<T>,
    confirm: (answer: T, permission: number) => Promise<void> | void
  ): Promise<T> {
    const root = this.root;
    root.assertActive();
    const permission = root.permissionReads.capture();
    let answer: T;
    try {
      answer = await request();
    } catch (error) {
      await this.refused(id, scope, error);
      throw error;
    }
    root.assertActive();
    if (answer && answer.workspace_id !== scope.workspaceID) throw new SessionChangedError();
    // The write succeeded, but an answer from an invalidated permission scope cannot revise what is held.
    if (root.permissionReads.current(permission, scope.workspaceID, [scope.projectID], true))
      await confirm(answer, permission);
    return answer;
  }
  /**
   * What a refusal of a write of a work item of an intake changes, in the write's turn, before it is handed on (P5a
   * spec 2.4, R28; re-94 m1): a work item gone from its intake (issue.intake_not_found) leaves every intake list; one
   * another triaged first is read again (readAgain), its read confirmed to its intake lists; the counts are fetched
   * again after either. Any other refusal leaves every value as it was.
   */
  private async refused(id: string, scope: IssueWriteScope, error: unknown): Promise<void> {
    this.root.assertActive();
    const code = problemCode(error);
    if (code === "issue.intake_not_found") this.root.intake.drop(scope.projectID, id);
    else if (code !== undefined && TRIAGED_ELSEWHERE.has(code)) {
      const read = await this.readAgain(id, scope);
      if (read) this.root.intake.confirm(read);
    } else return;
    this.countAgain(scope);
  }
  /**
   * Reads the work item again in its write's turn, from after the write (P5a R34, R28): a read sent before is not
   * taken, and canonical takes nerve's read. A work item read as gone, or by a read overtaken and undecided
   * (undefined), leaves every intake list. A failure but a change of session stays the work item's own (its
   * verification keeps it, for the detail to show and retry) and fails no write, which nerve made
   * (subscription.store.ts). Gives the read, or undefined for none.
   */
  private async readAgain(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer | undefined> {
    let read: IssueDetailAnswer | undefined;
    try {
      read = await this.root.issues.refreshAuthoritative(id, { afterWrite: true });
    } catch (error) {
      if (error instanceof SessionChangedError) throw error;
      return undefined;
    }
    if (read === undefined) this.root.intake.drop(scope.projectID, id);
    return read;
  }
  /** Fetches the counts of the scope's workspace again, if this generation holds them (R8; IntakeStore.countAgain). */
  private countAgain(scope: IssueWriteScope): void {
    this.root.intake.countAgain({ id: scope.workspaceID, slug: scope.workspaceSlug });
  }
}
