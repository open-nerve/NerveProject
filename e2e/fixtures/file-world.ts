import type { TestInfo } from "@playwright/test";

import { createProject, createWorkspace, slugFor, type Api } from "./api";
import { newAccount } from "./auth";
import { createIssueComment } from "./comments";
import { createDraft, createIssue } from "./issues";

/** Actual owner endpoints furnish the six host kinds; no database writes. */
export async function fileWorld(api: Api, info: TestInfo) {
  const owner = await newAccount(api, info, "file-owner");
  const other = await newAccount(api, info, "file-other");
  const workspace = await createWorkspace(api, owner.token, { name: "Files", slug: slugFor(info, "files") });
  const project = await createProject(api, owner.token, workspace.slug, {
    name: "Files",
    identifier: "FILE",
    network: 0,
  });
  const issue = await createIssue(api, owner.token, project.id, { name: "Uploaded files" });
  const comment = await createIssueComment(api, owner.token, issue.id, "<p>File discussion</p>");
  const draft = await createDraft(api, owner.token, workspace.slug, {
    project_id: project.id,
    payload: { name: "Draft with files" },
  });
  return { owner, other, workspace, project, issue, comment, draft };
}
