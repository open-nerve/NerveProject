/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { RootStore } from "@/store/root.store";
import type { AssetReadScope } from "./scope-lifetime";

export type ProfileImageSource =
  | { kind: "self"; image: "avatar" | "cover" }
  | { kind: "workspace"; workspaceID: string }
  | { kind: "project"; workspaceID: string; projectID: string }
  | { kind: "member"; workspaceID: string; memberID: string };

/** A projection of canonical owners, never a second entity cache or a permission observer. */
export class ImageSource {
  readonly source: ProfileImageSource;
  readonly group: string;
  readonly entry: string;
  readonly projection: string;
  readonly scope: AssetReadScope;
  constructor(
    private readonly root: RootStore,
    source: ProfileImageSource
  ) {
    this.source = Object.freeze(
      source.kind === "member" && source.memberID === root.user.data?.id
        ? { kind: "self", image: "avatar" }
        : { ...source }
    );
    const current = this.source;
    this.group =
      current.kind === "self"
        ? "self"
        : current.kind === "workspace"
          ? "workspace-directory"
          : JSON.stringify([current.kind, current.kind === "project" ? current.projectID : current.workspaceID]);
    this.entry = JSON.stringify([this.group, current.kind === "self" ? null : current.workspaceID]);
    this.projection = current.kind === "self" ? current.image : current.kind === "member" ? current.memberID : "image";
    this.scope =
      current.kind === "self"
        ? {}
        : {
            workspaceID: current.workspaceID,
            ...(current.kind === "project" ? { projectID: current.projectID } : {}),
            member: current.kind === "member",
          };
  }
  get workspace() {
    const source = this.source;
    return source.kind === "self"
      ? undefined
      : this.root.workspaceRoot.workspaces?.find((w) => w.id === source.workspaceID);
  }
  get allowed(): boolean {
    if (!this.root.issue.active) return false;
    const source = this.source;
    if (source.kind === "self") return !!this.root.user.data;
    const workspace = this.workspace;
    if (!workspace) return false;
    if (source.kind === "member") return workspace.role === 15 || workspace.role === 20;
    if (source.kind === "project")
      return this.root.projectRoot.project.getProjectById(source.projectID)?.workspace_id === workspace.id;
    return true;
  }
  get fact(): { url: string | null; expiresAt: string | null } | undefined {
    if (!this.allowed) return undefined;
    const source = this.source;
    if (source.kind === "self") {
      const user = this.root.user.data;
      return (
        user &&
        (source.image === "avatar"
          ? { url: user.avatar_url, expiresAt: user.avatar_url_expires_at }
          : { url: user.cover_image_url, expiresAt: user.cover_image_url_expires_at })
      );
    }
    if (source.kind === "workspace") {
      const workspace = this.workspace;
      return workspace && { url: workspace.logo_url, expiresAt: workspace.logo_url_expires_at };
    }
    if (source.kind === "project") {
      const project = this.root.projectRoot.project.getProjectById(source.projectID);
      return project && !project.cover_builtin_id
        ? { url: project.cover_image_url, expiresAt: project.cover_image_url_expires_at }
        : undefined;
    }
    const workspace = this.workspace;
    const member =
      workspace && this.root.memberRoot.workspace.getMemberships(workspace.slug)?.[source.memberID]?.member;
    return member && { url: member.avatar_url, expiresAt: member.avatar_url_expires_at };
  }
  read(): Promise<unknown> {
    const source = this.source;
    if (source.kind === "self") return this.root.user.fetchCurrentUser();
    if (source.kind === "workspace") return this.root.workspaceRoot.fetchWorkspaces();
    if (source.kind === "project") return this.root.projectRoot.project.fetchProject(source.projectID);
    const workspace = this.workspace;
    return workspace ? this.root.memberRoot.workspace.fetchWorkspaceMembers(workspace) : Promise.resolve(undefined);
  }
}
