/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { makeObservable, observable, reaction, runInAction } from "mobx";
import { ApiError } from "@/lib/api-error";
import type { IssueRootStore } from "@/store/issue/root.store";
import type { DetailRegistration } from "@/store/issue/issue-details/resource";
import type { AutosaveSnapshot } from "@/components/editor/autosave-retention";
export type RetainedText = AutosaveSnapshot<string>;
export type RetainedForm = {
  html: string;
  baseHTML: string;
  attachmentIDs: string[];
  revision: number;
  rejectedRevision?: number;
  error?: unknown;
  unknown?: boolean;
  verified?: boolean;
  editing?: boolean;
  focusAfterRead?: boolean;
};
type Receipt = { kind: "uploaded" | "deleted"; id: string };
class LocalInput<T> {
  private value: T | undefined;
  constructor(
    private readonly owner: IssueDetailInputState,
    private readonly epoch: number,
    private readonly clone: (input: T) => T
  ) {
    makeObservable<LocalInput<T>, "value">(this, { value: observable.ref });
  }
  get canFlushOnUnmount(): boolean {
    return this.owner.canFlushOnUnmount;
  }
  read(): T | undefined {
    return this.owner.current && this.owner.epoch === this.epoch && this.value !== undefined
      ? this.clone(this.value)
      : undefined;
  }
  store(value: T): void {
    if (this.owner.current && this.owner.epoch === this.epoch)
      runInAction(() => {
        this.value = this.clone(value);
      });
  }
  clear(): void {
    runInAction(() => {
      this.value = undefined;
    });
  }
}
/** Per mounted detail/peek; only canonical owner conclusions and the issue root end its input lifetime. */
export class IssueDetailInputState implements DetailRegistration {
  private active = false;
  private terminal = false;
  private seen = false;
  private readonly permission: number;
  private readonly text = new Map<string, LocalInput<RetainedText>>();
  private readonly forms = new Map<string, LocalInput<RetainedForm>>();
  private confirmed: readonly Receipt[] = [];
  /** The control whose confirmed command the canonical re-read unmounted; its next mount takes the focus back once. */
  private refocus: { owner: string; kind: Receipt["kind"] } | undefined;
  private unregister?: () => void;
  private stopOwner?: () => void;
  epoch = 0;
  constructor(
    private readonly root: IssueRootStore,
    readonly workspaceID: string,
    readonly projectID: string,
    readonly issueID: string
  ) {
    this.permission = root.permissionReads.capture();
    makeObservable<IssueDetailInputState, "active" | "terminal" | "confirmed">(this, {
      active: observable,
      terminal: observable,
      epoch: observable,
      confirmed: observable.ref,
    });
  }
  private permissionCurrent(): boolean {
    return (
      this.root.active && this.root.permissionReads.current(this.permission, this.workspaceID, [this.projectID], true)
    );
  }
  private ownerInvalid(): boolean {
    const error = this.root.verification.getError(this.issueID);
    if (
      this.root.verification.isMissing(this.issueID) ||
      (error instanceof ApiError && [401, 403, 404].includes(error.status))
    )
      return true;
    const issue = this.root.issues.getIssueById(this.issueID);
    if (issue) return issue.workspace_id !== this.workspaceID || issue.project_id !== this.projectID;
    return this.seen && !this.root.verification.isUncertain(this.issueID);
  }
  get current(): boolean {
    return this.active && !this.terminal && this.permissionCurrent() && !this.ownerInvalid();
  }
  get canFlushOnUnmount(): boolean {
    return (
      !this.terminal &&
      this.permissionCurrent() &&
      !this.ownerInvalid() &&
      !this.root.verification.isUncertain(this.issueID)
    );
  }
  get receipts(): readonly Receipt[] {
    return this.current ? this.confirmed : [];
  }
  activate(): void {
    if (this.terminal || this.active) return;
    if (!this.permissionCurrent()) {
      this.retire();
      return;
    }
    runInAction(() => {
      this.active = true;
    });
    this.unregister = this.root.registerResource(this);
    this.stopOwner = reaction(
      () => {
        const issue = this.root.issues.getIssueById(this.issueID);
        return { present: !!issue, invalid: this.ownerInvalid() };
      },
      ({ present, invalid }) => {
        if (invalid) this.retire();
        else if (present) this.seen = true;
      },
      { fireImmediately: true }
    );
    if (this.terminal) {
      this.stopOwner();
      this.stopOwner = undefined;
    }
  }
  private clear(): void {
    for (const slot of this.text.values()) slot.clear();
    for (const form of this.forms.values()) form.clear();
    this.text.clear();
    this.forms.clear();
    this.refocus = undefined;
    runInAction(() => {
      this.confirmed = [];
      ++this.epoch;
    });
  }
  deactivate = (): void => {
    runInAction(() => {
      this.active = false;
    });
    this.stopOwner?.();
    this.stopOwner = undefined;
    this.unregister?.();
    this.unregister = undefined;
    this.clear();
  };
  private retire(): void {
    runInAction(() => {
      this.terminal = true;
    });
    this.deactivate();
  }
  affects(workspace: string, project?: string): boolean {
    return workspace === this.workspaceID && (!project || project === this.projectID);
  }
  invalidate(): void {
    if (!this.permissionCurrent() || this.ownerInvalid()) this.retire();
  }
  dispose(): void {
    this.retire();
  }
  slot(key: string): LocalInput<RetainedText> {
    let slot = this.text.get(key);
    if (!slot) {
      slot = new LocalInput(this, this.epoch, (value) => ({ ...value }));
      this.text.set(key, slot);
    }
    return slot;
  }
  form(key: string): LocalInput<RetainedForm> {
    let form = this.forms.get(key);
    if (!form) {
      form = new LocalInput(this, this.epoch, (value) => ({ ...value, attachmentIDs: [...value.attachmentIDs] }));
      this.forms.set(key, form);
    }
    return form;
  }
  /** owner names the control that sent the command, so that it can take back the focus a remount loses. */
  confirm(receipt: Receipt, owner: string): void {
    if (!this.current) return;
    this.refocus = { owner, kind: receipt.kind };
    runInAction(() => {
      this.confirmed = [
        ...this.confirmed.filter((value) => value.id !== receipt.id),
        { kind: receipt.kind, id: receipt.id },
      ];
    });
  }
  /** The kind of owner's last confirmed command, once: a later mount of the same control gets nothing. */
  takeFocus(owner: string): Receipt["kind"] | undefined {
    const request = this.current && this.refocus?.owner === owner ? this.refocus : undefined;
    if (request) this.refocus = undefined;
    return request?.kind;
  }
}
