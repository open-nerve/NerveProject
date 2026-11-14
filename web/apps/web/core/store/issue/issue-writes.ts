/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { toJS } from "mobx";
import type { Issue, IssueCreate, IssueDetail as IssueDetailAnswer, IssueUpdate } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
import { compareIssueTime } from "@/lib/issue-order";
import type { Before, ConfirmedIssueChange } from "@/lib/paged-groups";
import { heldCounterSources } from "@/lib/paged-groups.evidence";
import type { IssueListBinding } from "./helpers/issue-list-binding";
import { createBody, membershipFields, updateBody } from "./issue-write-fields";
import type { IssueRootStore, IssueWriteScope } from "./root.store";

export type IssueDrop = NonNullable<ConfirmedIssueChange["drop"]> & { binding?: IssueListBinding };
type PendingWrite = {
  before: Before;
  sources: Map<IssueListBinding, ReadonlySet<number>>;
  observed: { version: string; fields: string }[];
};
/**
 * What the writes read of their generation's issue root, which passes itself in at construction. A type only: the
 * root imports this unit to build it, so this unit never imports the root as a value.
 */
type WritesRoot = Pick<
  IssueRootStore,
  | "rootStore"
  | "api"
  | "issues"
  | "issueDetail"
  | "permissionReads"
  | "registeredLists"
  | "workspaceSlug"
  | "assertActive"
  | "scope"
  | "invalidateProjectReads"
>;

/**
 * A change sent to nerve and, once nerve confirms it, written into every held copy: one queue per issue, the writes in
 * flight, the confirmation and the six commands with their request bodies (P6b R1).
 */
export class IssueWrites {
  private readonly queues = new Map<string, Promise<void>>();
  private readonly dragQueue = oneAtATime();
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
  /** Reserve every affected ID before yielding; queued operations must not enqueue another write. */
  enqueueIssueChanges<T>(
    ids: readonly string[],
    operation: () => Promise<T>,
    requestedScope?: Pick<IssueWriteScope, "workspaceSlug" | "projectID">
  ): Promise<T> {
    const ordered = [...new Set(ids)].toSorted();
    const permission = this.root.permissionReads.capture();
    const scopes = new Map<string, Set<string>>();
    const remember = (workspaceID: string, projectID: string) => {
      const projects = scopes.get(workspaceID) ?? new Set<string>();
      projects.add(projectID);
      scopes.set(workspaceID, projects);
    };
    for (const id of ordered) {
      const issue = this.root.issues.held(id);
      if (issue) remember(issue.workspace_id, issue.project_id);
    }
    if (requestedScope) {
      const workspace = this.root.rootStore.workspaceRoot.getWorkspaceBySlug(requestedScope.workspaceSlug);
      if (!workspace) return Promise.reject(new SessionChangedError());
      remember(workspace.id, requestedScope.projectID);
    }
    const run = Promise.all(ordered.map((id) => this.queues.get(id))).then(() => {
      this.root.assertActive();
      // Preserve the initiating permission lifetime, including an observed loss/recovery while waiting.
      const current = scopes.size
        ? [...scopes].every(([workspace, projects]) =>
            this.root.permissionReads.current(permission, workspace, [...projects], true)
          )
        : this.root.permissionReads.current(permission, undefined, [], true);
      if (!current) throw new SessionChangedError();
      return operation();
    });
    const tail = run.then(
      () => undefined,
      () => undefined
    );
    for (const id of ordered) this.queues.set(id, tail);
    void tail.then(() => {
      for (const id of ordered) if (this.queues.get(id) === tail) this.queues.delete(id);
      return undefined;
    });
    return run;
  }
  private async write(
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
  createIssue(scope: IssueWriteScope, input: IssueCreate): Promise<IssueDetailAnswer> {
    const body = createBody(input);
    return this.confirmCreation(scope, () => this.root.api.create(scope.projectID, body));
  }
  /** Freeze existing list evidence before sending, then confirm the new issue in every held view. */
  confirmCreation(
    scope: Pick<IssueWriteScope, "workspaceSlug" | "workspaceID">,
    send: () => Promise<IssueDetailAnswer>
  ): Promise<IssueDetailAnswer> {
    return this.write(undefined, scope, send).then((answer) => answer!);
  }
  updateIssue(
    id: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop?: IssueDrop,
    operationScope?: IssueWriteScope
  ): Promise<IssueDetailAnswer> {
    const root = this.root;
    const dragPermission = drop ? root.permissionReads.capture() : undefined;
    const initiating = drop ? root.issues.held(id) : undefined;
    const dragWorkspace = operationScope?.workspaceID ?? initiating?.workspace_id;
    const dragProject = operationScope?.projectID ?? initiating?.project_id;
    const update = () =>
      this.enqueueIssueChanges([id], async () => {
        root.assertActive();
        if (
          dragPermission !== undefined &&
          !root.permissionReads.current(dragPermission, dragWorkspace, dragProject ? [dragProject] : [], true)
        )
          throw new SessionChangedError();
        const snapshot = root.issues.getIssueById(id);
        if (!snapshot) throw new Error("Issue has not been read");
        // Without a given scope this falls back to the router's workspace, as it has since M4 (P7 removes it, spec §5).
        const scope = operationScope ?? root.scope(root.workspaceSlug!, snapshot.project_id);
        return (await this.write(
          id,
          scope,
          (held) => {
            if (!held) throw new Error("Issue has not been read");
            return root.api.update(id, updateBody(makeBody(held)));
          },
          drop
        ))!;
      });
    return drop ? this.dragQueue(update) : update();
  }
  deleteIssue(id: string, scope: IssueWriteScope): Promise<void> {
    return this.enqueueIssueChanges(
      [id],
      async () => {
        await this.write(id, scope, async () => {
          await this.root.api.delete(id);
          return null;
        });
      },
      scope
    );
  }
  archiveIssue(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer> {
    return this.enqueueIssueChanges(
      [id],
      async () => (await this.write(id, scope, () => this.root.api.archive(id)))!,
      scope
    );
  }
  unarchiveIssue(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer> {
    return this.enqueueIssueChanges(
      [id],
      async () => (await this.write(id, scope, () => this.root.api.unarchive(id)))!,
      scope
    );
  }
}
