/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueCreate, IssueDetail as IssueDetailAnswer, IssueUpdate } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { oneAtATime } from "@/lib/one-at-a-time";
import type { ConfirmedIssueChange } from "@/lib/paged-groups";
import type { IssueListBinding } from "./helpers/issue-list-binding";
import type { IssueChange } from "@nerve/types";
import { changeBody, createBody, updateBody } from "./issue-write-fields";
import { IssueWriteConfirmation, type WritesRoot } from "./issue-write-confirm";
import type { IssueWriteScope } from "./root.store";

export type IssueDrop = NonNullable<ConfirmedIssueChange["drop"]> & { binding?: IssueListBinding };

/**
 * A change sent to nerve and, once nerve confirms it, written into every held copy: one queue per issue and the seven
 * commands with their request bodies (P6b R1); the writes in flight and the confirmation are its confirmation's
 * (issue-write-confirm.ts), whose two public members it forwards.
 */
export class IssueWrites {
  private readonly queues = new Map<string, Promise<void>>();
  private readonly dragQueue = oneAtATime();
  private readonly confirmation: IssueWriteConfirmation;
  constructor(private readonly root: WritesRoot) {
    this.confirmation = new IssueWriteConfirmation(root);
  }
  get pendingIDs(): ReadonlySet<string> {
    return this.confirmation.pendingIDs;
  }
  /** Keeps a read that arrived while a write to its issue is out, for that write's confirmation; false if none is out. */
  observePendingRead(after: Issue): boolean {
    return this.confirmation.observePendingRead(after);
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
  createIssue(scope: IssueWriteScope, input: IssueCreate): Promise<IssueDetailAnswer> {
    const body = createBody(input);
    return this.confirmCreation(scope, () => this.root.api.create(scope.projectID, body));
  }
  /** Freeze existing list evidence before sending, then confirm the new issue in every held view. */
  confirmCreation(
    scope: Pick<IssueWriteScope, "workspaceSlug" | "workspaceID">,
    send: () => Promise<IssueDetailAnswer>
  ): Promise<IssueDetailAnswer> {
    return this.confirmation.write(undefined, scope, send).then((answer) => answer!);
  }
  updateIssue(
    id: string,
    makeBody: (held: Issue) => IssueUpdate,
    drop: IssueDrop | undefined,
    operationScope: IssueWriteScope
  ): Promise<IssueDetailAnswer> {
    const root = this.root;
    const dragPermission = drop ? root.permissionReads.capture() : undefined;
    const update = () =>
      this.enqueueIssueChanges([id], async () => {
        root.assertActive();
        if (
          dragPermission !== undefined &&
          !root.permissionReads.current(dragPermission, operationScope.workspaceID, [operationScope.projectID], true)
        )
          throw new SessionChangedError();
        if (!root.issues.getIssueById(id)) throw new Error("Issue has not been read");
        return (await this.confirmation.write(
          id,
          operationScope,
          (held) => {
            if (!held) throw new Error("Issue has not been read");
            return root.api.update(id, updateBody(makeBody(held)));
          },
          drop
        ))!;
      });
    return drop ? this.dragQueue(update) : update();
  }
  /**
   * Sends a change of the work item (P7a spec 2.2, R3): its modules' additions and removals made, in its turn, into the
   * whole set from the set it then holds, so after a change still out it is that change's answer, and after a refusal
   * the set as it was; ids no page knows, archived modules too, stay. Its other fields go as given.
   */
  changeIssue(id: string, change: IssueChange, scope: IssueWriteScope): Promise<IssueDetailAnswer> {
    return this.updateIssue(id, (held) => changeBody(change, held.module_ids), undefined, scope);
  }
  deleteIssue(id: string, scope: IssueWriteScope): Promise<void> {
    return this.enqueueIssueChanges(
      [id],
      async () => {
        await this.confirmation.write(id, scope, async () => {
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
      async () => (await this.confirmation.write(id, scope, () => this.root.api.archive(id)))!,
      scope
    );
  }
  unarchiveIssue(id: string, scope: IssueWriteScope): Promise<IssueDetailAnswer> {
    return this.enqueueIssueChanges(
      [id],
      async () => (await this.confirmation.write(id, scope, () => this.root.api.unarchive(id)))!,
      scope
    );
  }
}
