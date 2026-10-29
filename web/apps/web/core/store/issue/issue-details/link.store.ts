/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { IssueLink, IssueLinkCreate, IssueLinkPatch } from "@nerve/api-client";
import { SessionChangedError } from "@/lib/auth/token-manager";
import { dropped, upserted } from "@/lib/reconciled";
import type { IssueReadToken, IssueWriteScope } from "../root.store";
import type { IIssueDetail } from "./root.store";
import { IssueResources } from "./resources";

export type IIssueLinkStore = IssueLinkStore;
export type IIssueLinkStoreActions = Pick<IssueLinkStore, "fetchLinks" | "createLink" | "updateLink" | "removeLink">;
export class IssueLinkStore {
  private readonly resources: IssueResources<IssueLink[]>;
  constructor(readonly rootIssueDetailStore: IIssueDetail) {
    this.resources = new IssueResources(this.root);
  }
  private get root() {
    return this.rootIssueDetailStore.rootIssueStore;
  }
  getLinksByIssueId = (id: string) => this.resources.get(id)?.value?.map((link) => link.id);
  getLinkById = (issueId: string, linkId: string) =>
    this.resources.get(issueId)?.value?.find((link) => link.id === linkId);
  getError = (id: string) => this.resources.get(id)?.error;
  retry = async (ws: string, project: string, id: string) => {
    const resource = this.resources.entry(ws, project, id);
    if (!(await resource.retryConfirmationRead())) await this.fetchLinks(ws, project, id);
  };
  private current(scope: IssueWriteScope, token: IssueReadToken) {
    if (!this.root.isReadCurrent(token, [scope.projectID], scope.workspaceID)) throw new SessionChangedError();
    this.root.assertProjectReadable(scope.workspaceSlug, scope.projectID);
  }
  private validate(answer: IssueLink[], issueId: string, linkId?: string) {
    if (answer.some((link) => link.issue_id !== issueId || (linkId !== undefined && link.id !== linkId)))
      throw new SessionChangedError();
  }
  fetchLinks = async (ws: string, project: string, id: string): Promise<IssueLink[]> => {
    const resource = this.resources.entry(ws, project, id);
    const value = await resource.fetch(async (read) => {
      const answer = await this.root.detailApi.listLinks(id);
      if (!read.isCurrent()) throw new SessionChangedError();
      this.validate(answer.data, id);
      return answer.data;
    });
    if (!value) throw new SessionChangedError();
    return value;
  };
  private write(
    ws: string,
    project: string,
    id: string,
    request: () => Promise<IssueLink | void>,
    removed?: string
  ): Promise<IssueLink | void> {
    return this.root.enqueueIssueChanges([id], async () => {
      const resource = this.resources.entry(ws, project, id);
      const scope = this.root.scope(ws, project);
      const token = this.root.captureRead();
      const answer = await request();
      this.current(scope, token);
      if (answer) {
        this.validate([answer], id);
        resource.receive([answer], token, (held) => upserted(answer)(held ?? []));
      } else if (removed) resource.confirm(dropped(removed), token);
      await resource.refreshAfterConfirmation(async () => {
        this.current(scope, token);
        await this.root.issues.refreshAuthoritative(id, { afterWrite: true });
        this.current(scope, token);
        await this.rootIssueDetailStore.refreshAfterChange(ws, project, id);
      }, token);
      return answer;
    });
  }
  createLink = async (ws: string, project: string, id: string, body: IssueLinkCreate): Promise<IssueLink> => {
    const result = await this.write(ws, project, id, () =>
      this.root.detailApi.createLink(id, {
        url: body.url,
        ...(body.title !== undefined ? { title: body.title } : {}),
      })
    );
    if (!result) throw new Error("Missing confirmed link");
    return result;
  };
  updateLink = async (
    ws: string,
    project: string,
    id: string,
    linkId: string,
    body: IssueLinkPatch
  ): Promise<IssueLink> => {
    const result = await this.write(ws, project, id, async () => {
      if (!this.getLinkById(id, linkId)) throw new Error("Link has not been read");
      const answer = await this.root.detailApi.updateLink(linkId, {
        ...(body.url !== undefined ? { url: body.url } : {}),
        ...(body.title !== undefined ? { title: body.title } : {}),
      });
      this.validate([answer], id, linkId);
      return answer;
    });
    if (!result) throw new Error("Missing confirmed link");
    return result;
  };
  removeLink = async (ws: string, project: string, id: string, linkId: string): Promise<void> => {
    await this.write(
      ws,
      project,
      id,
      async () => {
        if (!this.getLinkById(id, linkId)) throw new Error("Link has not been read");
        await this.root.detailApi.deleteLink(linkId);
      },
      linkId
    );
  };
}
