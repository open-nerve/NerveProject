/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act } from "react";
import type { Project, ProjectUpdate } from "@nerve/api-client";
import { profilePage } from "@/components/assets/fake-profile-page";
import { json, type Call } from "@/lib/auth/fake-nerve";
import { loadProjects, loadArchivedProjects } from "@/store/project/fake-projects";
import { loadWorkspaces } from "@/store/workspace/fake-workspaces";

export async function coverPage() {
  const tab = await profilePage();
  let project: Project = {
    ...tab.project,
    cover_image_asset_id: "old-cover",
    cover_image_url: "/static/old-cover.png",
    cover_image_url_expires_at: new Date(Date.now() + 60_000).toISOString(),
  };
  const answer = (call: Call) => {
    if (call.path === `/api/v0/projects/${project.id}`) {
      if (call.method === "PATCH") {
        const patch = call.body as ProjectUpdate;
        project = { ...project, ...patch };
        if (patch.cover_image_asset_id) project.cover_builtin_id = null;
        if (patch.cover_builtin_id) project.cover_image_asset_id = null;
        if ("cover_image_asset_id" in patch || "cover_builtin_id" in patch) {
          project.cover_image_url = project.cover_image_asset_id ? `/static/${project.cover_image_asset_id}.png` : null;
          project.cover_image_url_expires_at = project.cover_image_asset_id
            ? new Date(Date.now() + 60_000).toISOString()
            : null;
        }
      }
      call.answer(json(200, project));
    } else tab.answer(call);
  };
  const directory = async (rows: Project[]) => {
    const previous = tab.nerve.respond;
    tab.nerve.respond = undefined;
    await act(async () => loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, rows));
    tab.nerve.respond = previous;
  };
  tab.nerve.respond = undefined;
  await loadProjects(tab.nerve, tab.app.projectRoot.project, tab.workspace, [project]);
  tab.nerve.respond = answer;
  const roles = async (workspaceRole: 5 | 15 | 20, memberRole: 5 | 15 | 20 | null, archived = false) => {
    tab.nerve.respond = undefined;
    project = { ...project, member_role: memberRole, archived_at: archived ? new Date().toISOString() : null };
    await act(async () =>
      loadWorkspaces(tab.nerve, tab.app.workspaceRoot, [{ ...tab.workspace, role: workspaceRole }])
    );
    await act(async () =>
      (archived ? loadArchivedProjects : loadProjects)(tab.nerve, tab.app.projectRoot.project, tab.workspace, [project])
    );
    tab.nerve.respond = answer;
  };
  const name = async (value: string) => {
    const input = document.querySelector<HTMLInputElement>('input[name="name"]');
    if (!input) throw new Error("project name field absent");
    await act(async () => {
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value")?.set?.call(input, value);
      input.dispatchEvent(new Event("input", { bubbles: true }));
    });
  };
  return {
    ...tab,
    answer,
    directory,
    roles,
    name,
    project: () => project,
    writes: () =>
      tab.nerve.calls.filter((call) => call.method === "PATCH" && call.path.startsWith("/api/v0/projects/")),
  };
}
