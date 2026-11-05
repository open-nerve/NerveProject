/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { RootStore } from "@/store/root.store";
import { ScopeLifetime, type AssetReadScope } from "./scope-lifetime";
import type { ContentUse, ContentView } from "./content-view";
import { contentTarget } from "./content-target";
import type { UploadPolicy } from "./upload-session";

export type ProfileUploadTarget =
  | { purpose: "USER_AVATAR" | "USER_COVER"; scope: { kind: "user" } }
  | { purpose: "WORKSPACE_LOGO"; scope: { kind: "workspace"; id: string } }
  | { purpose: "PROJECT_COVER"; scope: { kind: "project" | "new_project"; id: string } };

const raster = (file: File) => ["image/jpeg", "image/png", "image/webp", "image/gif"].includes(file.type);

/** The profile adapter retains its existing scope lifetime and exact write rules. */
export function profileUploadPolicy(root: RootStore, input: ProfileUploadTarget): UploadPolicy {
  const target = { ...input };
  target.scope = Object.freeze({ ...target.scope });
  Object.freeze(target);
  const scope: AssetReadScope =
    target.scope.kind === "user"
      ? {}
      : target.scope.kind === "project"
        ? {
            workspaceID: root.projectRoot.project.getProjectById(target.scope.id)?.workspace_id,
            projectID: target.scope.id,
            member: true,
          }
        : { workspaceID: target.scope.id, member: true };
  const allowed = () => {
    if (target.scope.kind === "user") return true;
    const workspace = root.workspaceRoot.workspaces?.find((row) => row.id === scope.workspaceID);
    if (!workspace) return false;
    if (target.scope.kind === "workspace") return workspace.role === 20;
    if (target.scope.kind === "new_project") return workspace.role === 15 || workspace.role === 20;
    const project = root.projectRoot.project.getProjectById(scope.projectID);
    return (
      project?.workspace_id === workspace.id &&
      !project.archived_at &&
      root.user.permission.getProjectRoleByWorkspaceSlugAndProjectId(workspace.slug, scope.projectID) === 20
    );
  };
  let lifetime: ScopeLifetime | undefined;
  return {
    target,
    preview: true,
    maxBytes: () => root.instance.config?.file_size_limit,
    accepts: raster,
    contentType: (file) => file.type,
    current: () => lifetime?.current ?? false,
    subscribeEnd: (end) => {
      lifetime = new ScopeLifetime(root, scope, allowed, end);
      return () => lifetime?.dispose();
    },
  };
}

export function contentUploadPolicy(root: RootStore, view: ContentView, use: ContentUse): UploadPolicy {
  return {
    target: contentTarget(view.host, use),
    preview: use === "description",
    maxBytes: () => root.instance.config?.file_size_limit,
    current: () => view.canWrite(use),
    accepts: (file) =>
      use === "description"
        ? raster(file)
        : !file.type || /^[a-z0-9!#$%&'+.^_`|~-]+\/[a-z0-9!#$%&'+.^_`|~-]+$/.test(file.type),
    contentType: (file) => file.type || "application/octet-stream",
    subscribeEnd: (end) => {
      view.signal.addEventListener("abort", end, { once: true });
      return () => view.signal.removeEventListener("abort", end);
    },
  };
}
