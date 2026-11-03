import { bearer } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import { fileContent, filePixel, storedFile, type Asset } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F1 (API): ten purposes and both future hosts prepare, upload, confirm and read actual bytes with truthful metadata and download disposition", async ({
  api,
}, info) => {
  const w = await fileWorld(api, info);
  const cases: [Asset["purpose"], Asset["scope"], boolean][] = [
    ["USER_AVATAR", { kind: "user" }, false],
    ["USER_COVER", { kind: "user" }, false],
    ["WORKSPACE_LOGO", { kind: "workspace", id: w.workspace.id }, false],
    ["PROJECT_COVER", { kind: "project", id: w.project.id }, false],
    ["PROJECT_COVER", { kind: "new_project", id: w.workspace.id }, false],
    ["ISSUE_ATTACHMENT", { kind: "issue", id: w.issue.id }, true],
    ["ISSUE_DESCRIPTION", { kind: "issue", id: w.issue.id }, false],
    ["COMMENT_ATTACHMENT", { kind: "comment", id: w.comment.id }, true],
    ["COMMENT_DESCRIPTION", { kind: "comment", id: w.comment.id }, false],
    ["COMMENT_ATTACHMENT", { kind: "new_comment", id: w.issue.id }, false],
    ["COMMENT_DESCRIPTION", { kind: "new_comment", id: w.issue.id }, false],
    ["DRAFT_ISSUE_ATTACHMENT", { kind: "draft", id: w.draft.id }, true],
    ["DRAFT_ISSUE_DESCRIPTION", { kind: "draft", id: w.draft.id }, false],
  ];
  for (const [purpose, scope, bound] of cases) {
    // One outcome at a time keeps the story below the production concurrent-transfer bound.
    // eslint-disable-next-line no-await-in-loop
    const asset = await storedFile(api, w.owner.token, purpose, scope, filePixel, "image/png", "same.png");
    const dimension = purpose.endsWith("ATTACHMENT") ? null : 1;
    expect(asset).toMatchObject({
      purpose,
      scope,
      name: "same.png",
      size: filePixel.length,
      width: dimension,
      height: dimension,
      is_bound: bound,
    });
    // eslint-disable-next-line no-await-in-loop
    const content = await fileContent(api, w.owner.token, asset.id);
    // eslint-disable-next-line no-await-in-loop
    const response = await fetch(content.url, { credentials: "omit" });
    expect(response.status).toBe(200);
    expect(response.headers.get("content-disposition")).toContain(
      purpose.endsWith("ATTACHMENT") ? "attachment;" : "inline;"
    );
    // eslint-disable-next-line no-await-in-loop
    expect(Buffer.from(await response.arrayBuffer())).toEqual(filePixel);
    const seconds = (Date.parse(content.expires_at) - Date.now()) / 1000;
    expect(seconds).toBeGreaterThan(290);
    expect(seconds).toBeLessThanOrEqual(65 * 60);
    // eslint-disable-next-line no-await-in-loop
    const metadata = await api.GET("/api/v0/assets/{id}", {
      params: { path: { id: asset.id } },
      headers: bearer(w.owner.token),
    });
    expect(metadata.data).toEqual(asset);
    expect(JSON.stringify(metadata.data)).not.toMatch(/storage_metadata|staging|final|object_key|signature/);
  }
});
