/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { toJS } from "mobx";
import type { Issue, IssueDetail as IssueDetailAnswer } from "@nerve/api-client";
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
>;

/**
 * A change nerve answered, written into every held copy (P6b R1; P5a spec 2.4): the writes in flight, the reads that
 * arrive meanwhile, and the confirmation. The writes' unit (issue-writes.ts) holds it, sends its writes one at a time
 * per issue and forwards its two public members; a type only of that unit's IssueDrop, as that unit imports this one
 * to build it.
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
}
