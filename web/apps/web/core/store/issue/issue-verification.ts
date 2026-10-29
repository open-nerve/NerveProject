/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueDetail } from "@nerve/api-client";
import { observable, runInAction } from "mobx";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import type { IssueListBinding } from "./helpers/issue-list-binding";
import type { IssueReadToken, IssueRootStore } from "./root.store";

type Verification = {
  id: string;
  round: number;
  status: "pending" | "missing" | "error" | "unconfirmed";
  error?: unknown;
  freshnessFloor: number;
  identity: string;
};
export type VerificationRound = {
  id: number;
  users: number;
  ids: Map<string, VerificationRequest>;
  scopes: Set<IssueListBinding>;
};
type VerificationRequest = { identity: string; promise: Promise<IssueDetail | undefined> };

/** Root-owned metadata only. All entity evidence stays in IssueStore. */
export class IssueVerification {
  private readonly records = observable.map<string, Verification>({}, { deep: false });
  private readonly affected = new Map<string, Set<IssueListBinding>>();
  private readonly inFlight = observable.map<string, VerificationRequest>({}, { deep: false });
  private current?: VerificationRound;
  private sequence = 0;
  constructor(private readonly root: IssueRootStore) {}
  begin(fresh = false): VerificationRound {
    // Explicit recovery owns new completed-id dedupe and scope budgets even if an old read is still out.
    if (fresh) this.current = undefined;
    const round = (this.current ??= { id: ++this.sequence, users: 0, ids: new Map(), scopes: new Set() });
    ++round.users;
    return round;
  }
  end(round: VerificationRound): void {
    if (--round.users === 0 && this.current === round) this.current = undefined;
  }
  isUncertain(id: string): boolean {
    return this.records.has(id);
  }
  deleted(id: string): void {
    runInAction(() => this.records.delete(id));
    this.affected.delete(id);
  }
  isVerifying(list: IssueListBinding): boolean {
    return [...this.inFlight.keys()].some((id) => this.affected.get(id)?.has(list) || this.belongs(id, list));
  }
  hasRefusal(list: IssueListBinding): boolean {
    return [...this.records.values()].some(
      (record) => !!record.error && (this.affected.get(record.id)?.has(list) || this.belongs(record.id, list))
    );
  }
  canAccept(id: string, token: IssueReadToken): boolean {
    return token.serial > (this.records.get(id)?.freshnessFloor ?? -1);
  }
  accept(id: string, token: IssueReadToken): void {
    if (this.canAccept(id, token)) {
      const error = this.records.get(id)?.error;
      if (error) for (const list of this.affected.get(id) ?? []) list.clearVerificationError(error);
      runInAction(() => this.records.delete(id));
    }
  }
  blocksPage(list: IssueListBinding, token: IssueReadToken): boolean {
    return [...this.records.values()].some(
      (record) =>
        (this.affected.get(record.id)?.has(list) || this.belongs(record.id, list)) &&
        token.serial <= record.freshnessFloor
    );
  }
  private belongs(id: string, list: IssueListBinding): boolean {
    const issue = this.root.issues.held(id);
    return (
      !!issue &&
      list.covers({
        projectID: issue.project_id,
        workspaceID: issue.workspace_id,
        workspaceSlug: list.scope.workspaceSlug,
      })
    );
  }
  require(id: string, list: IssueListBinding): void {
    if (!this.root.active) return;
    const lists = this.affected.get(id) ?? new Set<IssueListBinding>();
    lists.add(list);
    this.affected.set(id, lists);
    const round = this.begin();
    if (round.ids.get(id)?.identity === this.identity(id) && !this.records.has(id)) {
      this.end(round);
      return;
    }
    if (!this.records.has(id))
      runInAction(() =>
        this.records.set(id, {
          id,
          round: round.id,
          status: "pending",
          freshnessFloor: this.root.readSerial,
          identity: this.identity(id),
        })
      );
    // Start after synchronous confirmation has published the authoritative write answer.
    void this.request(id, round, false)
      .catch(() => {})
      .finally(() => this.end(round));
  }
  async drain(list: IssueListBinding, round: VerificationRound): Promise<void> {
    await Promise.all(
      [...this.affected]
        .filter(
          ([id, lists]) => (lists.has(list) || this.belongs(id, list)) && (this.records.has(id) || round.ids.has(id))
        )
        .map(([id, lists]) => {
          lists.add(list);
          return this.request(id, round, false);
        })
    );
  }
  async retry(list: IssueListBinding): Promise<boolean> {
    const ids = [...this.records.keys()].filter((id) => this.affected.get(id)?.has(list) || this.belongs(id, list));
    if (!ids.length) return false;
    const round = this.begin(true);
    try {
      await Promise.all(ids.map((id) => this.request(id, round, true)));
      return true;
    } finally {
      this.end(round);
    }
  }
  async refresh(id: string, retryStaleDetail = true): Promise<IssueDetail | undefined> {
    this.root.assertActive();
    const lists = this.affected.get(id) ?? new Set<IssueListBinding>();
    for (const list of this.root.listsForIssue(id)) lists.add(list);
    this.affected.set(id, lists);
    const round = this.begin(true);
    try {
      return await this.request(id, round, true, retryStaleDetail);
    } finally {
      this.end(round);
    }
  }
  private identity(id: string): string {
    const held = this.root.issues.held(id);
    return this.root.permissionReads.identity(held?.workspace_id, held ? [held.project_id] : []);
  }
  private request(
    id: string,
    round: VerificationRound,
    explicit: boolean,
    retryStaleDetail = true
  ): Promise<IssueDetail | undefined> {
    const held = this.root.issues.held(id);
    if (held && !this.root.canSeeIssue(held)) return Promise.resolve(undefined);
    const identity = this.identity(id);
    const existing = [this.inFlight.get(id), round.ids.get(id)].find((request) => request?.identity === identity);
    if (existing) {
      // A genuinely pending request can span rounds; its completion still dedupes this joining round.
      round.ids.set(id, existing);
      return existing.promise;
    }
    const record = this.records.get(id);
    if (record && record.identity === identity && record.status !== "pending" && !explicit)
      return record.error ? Promise.reject(record.error) : Promise.resolve(undefined);
    if (!record || record.identity !== identity || explicit)
      runInAction(() =>
        this.records.set(id, {
          id,
          round: round.id,
          status: "pending",
          error: record?.identity === identity ? record.error : undefined,
          freshnessFloor: this.root.readSerial,
          identity,
        })
      );
    const request: VerificationRequest = { identity, promise: undefined as never };
    const job = Promise.resolve()
      .then(() => this.read(id, round, retryStaleDetail, identity))
      .finally(() => {
        if (this.inFlight.get(id) === request) runInAction(() => this.inFlight.delete(id));
      });
    request.promise = job;
    round.ids.set(id, request);
    runInAction(() => this.inFlight.set(id, request));
    return job;
  }
  private async read(
    id: string,
    round: VerificationRound,
    retryStaleDetail: boolean,
    identity: string
  ): Promise<IssueDetail | undefined> {
    this.root.assertActive();
    if (this.identity(id) !== identity) return undefined;
    const held = this.root.issues.held(id);
    if (held && !this.root.canSeeIssue(held)) return undefined;
    if (!held) {
      const { workspaceSlug, projectId } = this.root;
      if (
        !workspaceSlug ||
        !projectId ||
        this.root.rootStore.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspaceSlug, projectId) ===
          undefined
      )
        return undefined;
    }
    const projects = held ? [held.project_id] : [];
    const token = this.root.captureRead();
    const lists = () => [...(this.affected.get(id) ?? [])];
    for (const list of lists()) list.markVerificationStale(id);
    const current = () => this.root.isReadCurrent(token, projects, held?.workspace_id);
    let answer: IssueDetail;
    try {
      answer = await this.root.api.get(id);
    } catch (error) {
      if (!current() || error instanceof SessionChangedError) return undefined;
      const missing = error instanceof ApiError && error.status === 404;
      runInAction(() =>
        this.records.set(id, {
          id,
          round: round.id,
          status: missing ? "missing" : "error",
          error: missing ? undefined : error,
          freshnessFloor: this.root.readSerial,
          identity: this.identity(id),
        })
      );
      for (const list of lists()) list.markVerificationStale(id, missing ? undefined : error);
      if (missing) {
        await this.refreshScopes(lists(), round);
        return undefined;
      }
      throw error;
    }
    if (!this.root.isReadCurrent(token, [answer.project_id], answer.workspace_id)) return undefined;
    this.root.issues.acceptRead(answer, token);
    const uncertain = this.records.get(id);
    if (uncertain) {
      const latest = this.root.issues.held(id);
      if (retryStaleDetail && latest && !("description_html" in latest) && this.root.canSeeIssue(latest))
        return this.read(id, round, false, identity);
      const error = new Error("The issue detail was superseded; retry the read");
      runInAction(() => this.records.set(id, { ...uncertain, status: "unconfirmed", error }));
      for (const list of lists()) list.markVerificationStale(id, error);
      return undefined;
    }
    await this.refreshScopes(lists(), round);
    return this.root.issues.getIssueDetailById(id);
  }
  private async refreshScopes(lists: IssueListBinding[], round: VerificationRound): Promise<void> {
    await Promise.all(
      lists.map(async (list) => {
        if (!this.root.active || !list.isVisible || round.scopes.has(list)) return;
        round.scopes.add(list);
        await list.refreshVerification(round);
      })
    );
  }
}
