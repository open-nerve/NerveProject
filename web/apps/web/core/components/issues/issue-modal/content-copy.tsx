/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useState } from "react";
import { makeObservable, observable, runInAction } from "mobx";
import { observer } from "mobx-react";
import type { Draft, DraftCreate, IssueDetail } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { joinUrlPath } from "@nerve/utils";
import { ApiError } from "@/lib/api-error";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
import { contentImageIDs, withContentImages, withoutContentImages } from "@/hooks/editor/verify-images";
import type { RootStore } from "@/store/root.store";

/** The outcome of the previous attempt is unconfirmed; nothing is sent until the user has checked. */
export class CopyUnconfirmedError extends Error {
  constructor() {
    super("The previous copy attempt is unconfirmed");
    this.name = "CopyUnconfirmedError";
  }
}
/**
 * One explicit copy whose description holds images. A new work item has no content host, so the copy is a real
 * draft: created once, its images copied on the server in document order, saved whole, and published only for an
 * ordinary copy. A refusal keeps the same target and the finished copies; an unknown creation or publish waits for
 * the user's check of the drafts instead of being sent again or guessed.
 */
export class ContentCopy {
  target: Draft | undefined = undefined;
  unknown: "create" | "publish" | undefined = undefined;
  checked = false;
  /** An unconfirmed publish whose target is no longer a draft: it most likely became a work item. */
  published = false;
  private readonly copies = new Map<string, string>();
  constructor(
    private readonly root: RootStore,
    private readonly slug: string
  ) {
    makeObservable(this, {
      target: observable.ref,
      unknown: observable,
      checked: observable,
      published: observable,
    });
  }
  static needed(html: string | undefined): boolean {
    return contentImageIDs(html ?? "").ids.length > 0;
  }
  get blocked(): boolean {
    return this.published || (this.unknown !== undefined && !this.checked);
  }
  async run(input: DraftCreate, publish: boolean): Promise<Draft | IssueDetail> {
    if (this.blocked) throw new CopyUnconfirmedError();
    const drafts = this.root.issue.workspaceDraftIssues;
    const html = input.payload.description_html ?? "";
    runInAction(() => {
      this.unknown = undefined;
      this.checked = false;
    });
    const target =
      this.target ??
      (await this.attempt("create", () =>
        drafts.createIssue(this.slug, {
          ...input,
          payload: { ...input.payload, description_html: withoutContentImages(html) },
        })
      ));
    runInAction(() => {
      this.target = target;
    });
    await this.copyImages(target, html);
    const saved = { ...input, payload: { ...input.payload, description_html: withContentImages(html, this.copies) } };
    const result = publish
      ? await this.attempt("publish", () => drafts.moveIssue(this.slug, target.id, saved))
      : await drafts.updateIssue(this.slug, target.id, saved);
    this.reset();
    return result;
  }
  private async copyImages(target: Draft, html: string): Promise<void> {
    const view = this.root.asset.openContent({
      kind: "draft",
      workspaceID: target.workspace_id,
      projectID: target.project_id,
      draftID: target.id,
    });
    try {
      for (const source of contentImageIDs(html).ids) {
        if (this.copies.has(source)) continue;
        // One at a time in document order, so a refusal leaves a known prefix that the retry continues from.
        // oxlint-disable-next-line no-await-in-loop
        const copy = await this.root.asset.duplicateImage(view, source, view.signal);
        this.copies.set(source, copy.id);
      }
    } finally {
      // The save and publish that follow must not find this lifetime still open.
      view.dispose();
    }
  }
  private async attempt<T>(step: "create" | "publish", send: () => Promise<T>): Promise<T> {
    try {
      return await send();
    } catch (error) {
      if (!(error instanceof ApiError && error.problem) && !(error instanceof SessionChangedError))
        runInAction(() => {
          this.unknown = step;
        });
      throw error;
    }
  }
  /** Reads the drafts again; the created work item of a lost publish answer is never guessed. */
  async check(): Promise<void> {
    const drafts = this.root.issue.workspaceDraftIssues;
    await drafts.fetchIssues(this.slug);
    runInAction(() => {
      if (this.unknown === "publish" && this.target && !drafts.getIssueById(this.target.id, this.slug))
        this.published = true;
      else this.checked = true;
    });
  }
  allowRetry(): void {
    if (!this.checked) return;
    runInAction(() => {
      this.unknown = undefined;
      this.checked = false;
    });
  }
  private reset(): void {
    this.copies.clear();
    runInAction(() => {
      this.target = undefined;
      this.unknown = undefined;
      this.checked = false;
    });
  }
}
/** Where a kept or unconfirmed copy stands, with the path to the drafts that hold it. */
export const CopyNotice = observer(function CopyNotice({ copy, slug }: { copy: ContentCopy; slug: string }) {
  const { t } = useTranslation();
  const refuse = useRefusalToast();
  const [checking, setChecking] = useState(false);
  const drafts = (
    <a href={joinUrlPath(slug, "drafts")} className="font-medium text-accent-primary">
      {t("content_files.copy.open_drafts")}
    </a>
  );
  if (!copy.unknown)
    return copy.target ? (
      <p role="status">
        {t("content_files.copy.kept")} {drafts}
      </p>
    ) : null;
  return (
    <div role="alert" className="space-y-2">
      <p>
        {copy.published
          ? t("content_files.copy.published")
          : copy.unknown === "create"
            ? t("content_files.copy.unknown_create")
            : t("content_files.copy.unknown_publish")}
      </p>
      <div className="flex items-center gap-2">
        {!copy.published &&
          (copy.checked ? (
            <Button type="button" variant="secondary" onClick={() => copy.allowRetry()}>
              {t("content_files.copy.allow_retry")}
            </Button>
          ) : (
            <Button
              type="button"
              variant="secondary"
              disabled={checking}
              onClick={() => {
                setChecking(true);
                copy
                  .check()
                  .catch(refuse)
                  .finally(() => setChecking(false));
              }}
            >
              {t("content_files.copy.verify")}
            </Button>
          ))}
        {drafts}
      </div>
    </div>
  );
});
