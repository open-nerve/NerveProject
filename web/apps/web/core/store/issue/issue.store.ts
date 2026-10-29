/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { Issue, IssueDetail } from "@nerve/api-client";
import { computed, makeObservable, observable, runInAction } from "mobx";
import { ReconciledByKey } from "@/lib/reconciled";
import { compareIssueTime } from "@/lib/issue-order";
import type { ConfirmedIssueChange } from "@/lib/paged-groups";
import type { IssueReadToken, IssueRootStore } from "./root.store";

type Entity = Issue | IssueDetail | null;
export type IIssueStore = IssueStore;

/** The sole full issue copy. null is reserved for a successful delete in this session. */
export class IssueStore {
  private readonly canonical = new ReconciledByKey<Entity>();
  private readonly permissionProof = observable.map<string, number>();
  readonly issuesIdentifierMap: Record<string, string> = {};
  constructor(private readonly root: IssueRootStore) {
    makeObservable(this, { issuesMap: computed, issuesIdentifierMap: observable });
  }
  get issuesMap(): Readonly<Record<string, Issue | IssueDetail>> {
    return Object.freeze(
      Object.fromEntries(
        this.canonical.values().flatMap((issue) => (issue && this.getIssueById(issue.id) ? [[issue.id, issue]] : []))
      )
    );
  }
  private choose = (held: Entity | undefined, answer: Entity, keepDescription = true): Entity => {
    if (held === null || answer === null) return null;
    if (held && compareIssueTime(answer.updated_at, held.updated_at) < 0) return held;
    const merged =
      keepDescription && held && "description_html" in held && !("description_html" in answer)
        ? { ...answer, description_html: held.description_html }
        : { ...answer };
    Object.freeze(merged.assignee_ids);
    Object.freeze(merged.label_ids);
    return Object.freeze(merged);
  };
  acceptRead(answer: Issue | IssueDetail, token: IssueReadToken): void {
    if (
      !this.root.isReadCurrent(token, [answer.project_id], answer.workspace_id) ||
      !this.root.canSeeIssue(answer) ||
      !this.root.verification.canAccept(answer.id, token)
    )
      return;
    const held = this.canonical.get(answer.id);
    const heldCurrent =
      !!held &&
      this.root.permissionReads.current(this.permissionProof.get(answer.id) ?? 0, held.workspace_id, [held.project_id]);
    if (held === null || (held && heldCurrent && compareIssueTime(answer.updated_at, held.updated_at) < 0)) return;
    this.canonical.receive(answer.id, answer, (before, incoming) =>
      this.choose(heldCurrent ? before : undefined, incoming, heldCurrent)
    );
    runInAction(() => this.permissionProof.set(answer.id, token.permission));
    this.root.observeRead(held ?? undefined, answer);
    this.root.verification.accept(answer.id, token);
    const identifier = this.root.rootStore.projectRoot.project.getProjectIdentifierById(answer.project_id);
    if (identifier) this.addIssueIdentifier(`${identifier}-${answer.sequence_id}`, answer.id);
  }
  confirmChange(change: ConfirmedIssueChange, permission?: number): void {
    if (!this.root.active) return;
    runInAction(() => {
      const held = this.canonical.get(change.id);
      if (
        change.after &&
        held &&
        this.root.permissionReads.current(this.permissionProof.get(change.id) ?? 0, held.workspace_id, [
          held.project_id,
        ]) &&
        !this.root.permissionReads.current(permission ?? -1, change.after.workspace_id, [change.after.project_id])
      )
        return;
      const selected = this.canonical.receive(change.id, change.after, this.choose);
      // Publish content and its proof atomically; a rejected older answer keeps both.
      if (change.after && selected !== held) this.permissionProof.set(change.id, permission ?? -1);
    });
    if (change.after === null) this.root.verification.deleted(change.id);
  }
  /** Internal proof only: accessors below additionally enforce visibility and verification. */
  held(issueId: string): Entity | undefined {
    return this.canonical.get(issueId);
  }
  knownDescendants(parentID: string): Issue[] {
    const ids = new Set([parentID]);
    const found: Issue[] = [];
    let more = true;
    while (more) {
      more = false;
      for (const issue of this.canonical.values())
        if (issue?.parent_id && ids.has(issue.parent_id) && !ids.has(issue.id)) {
          ids.add(issue.id);
          found.push(issue);
          more = true;
        }
    }
    return found;
  }
  getIssueById = (issueId: string): Issue | IssueDetail | undefined => {
    const issue = this.canonical.get(issueId);
    return issue &&
      this.root.canSeeIssue(issue) &&
      this.root.permissionReads.current(this.permissionProof.get(issueId) ?? 0, issue.workspace_id, [
        issue.project_id,
      ]) &&
      !this.root.verification.isUncertain(issueId)
      ? issue
      : undefined;
  };
  getIssueDetailById = (issueId: string): IssueDetail | undefined => {
    const issue = this.getIssueById(issueId);
    return issue && "description_html" in issue ? issue : undefined;
  };
  refreshAuthoritative = (issueId: string): Promise<IssueDetail | undefined> => this.root.verification.refresh(issueId);
  addIssueIdentifier = (identifier: string, id: string): void => {
    if (identifier && id)
      runInAction(() => {
        this.issuesIdentifierMap[identifier] = id;
      });
  };
  getIssueIdByIdentifier = (identifier: string): string | undefined => this.issuesIdentifierMap[identifier];
  getIssuesByIds = (ids: string[], type: "archived" | "un-archived"): (Issue | IssueDetail)[] =>
    ids.flatMap((id) => {
      const issue = this.getIssueById(id);
      return issue && (type === "archived" ? !!issue.archived_at : !issue.archived_at) ? [issue] : [];
    });
}
