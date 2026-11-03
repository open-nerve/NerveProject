import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { bearer } from "../../fixtures/auth";
import { commentDeleted, createIssueComment } from "../../fixtures/comments";
import { fileWorld } from "../../fixtures/file-world";
import { draftDeleted } from "../../fixtures/issues";
import { fileContent, readFile, storedFile } from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F14 (API): comment and draft deletion stamp their files, preparation deletion preserves recoverable bytes, and live neighbors stay unchanged", async ({
  api,
  db,
}, info) => {
  const w = await fileWorld(api, info);
  await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
  await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
  const comment = await createIssueComment(api, w.other.token, w.issue.id, "<p>Another author's file</p>");
  const commentAsset = await storedFile(api, w.other.token, "COMMENT_ATTACHMENT", { kind: "comment", id: comment.id });
  const draftAsset = await storedFile(api, w.owner.token, "DRAFT_ISSUE_ATTACHMENT", { kind: "draft", id: w.draft.id });
  const neighbor = await storedFile(api, w.owner.token, "ISSUE_ATTACHMENT", { kind: "issue", id: w.issue.id });
  const snapshot = () => db.query("SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=$1", [neighbor.id]);
  const before = await snapshot();
  const capability = await fileContent(api, w.other.token, commentAsset.id);
  expect(await commentDeleted(api, w.owner.token, comment.id)).toEqual({ status: 204 });
  expect(await draftDeleted(api, w.owner.token, w.draft.id)).toEqual({ status: 204 });
  expect(
    await db.query(
      "SELECT a.created_by_id=$2 AS original_author, a.deleted_at=c.deleted_at AND a.updated_at=c.deleted_at AND a.deleted_by_id=c.updated_by_id AND a.updated_by_id=c.updated_by_id AS original_stamp, a.updated_by_id=$3 AS by_admin FROM file_assets a JOIN issue_comments c ON c.id=a.comment_id WHERE a.id=$1",
      [commentAsset.id, w.other.id, w.owner.id]
    )
  ).toEqual([{ original_author: true, original_stamp: true, by_admin: true }]);
  expect(
    await db.query(
      "SELECT a.deleted_at=d.deleted_at AND a.updated_at=d.deleted_at AND a.updated_by_id=d.updated_by_id AS original_stamp FROM file_assets a JOIN draft_issues d ON d.id=a.draft_issue_id WHERE a.id=$1",
      [draftAsset.id]
    )
  ).toEqual([{ original_stamp: true }]);
  await Promise.all(
    (
      [
        [commentAsset, w.other.token],
        [draftAsset, w.owner.token],
      ] as const
    ).map(async ([asset, token]) => {
      const result = await api.GET("/api/v0/assets/{id}", {
        params: { path: { id: asset.id } },
        headers: bearer(token),
      });
      expect(result.response.status).toBe(404);
    })
  );
  const stillStored = await fetch(capability.url, { credentials: "omit" });
  expect(stillStored.status).toBe(200);
  await stillStored.arrayBuffer();
  expect(await snapshot()).toEqual(before);
  expect(await readFile(api, w.owner.token, neighbor.id)).toEqual(Buffer.from("real attachment bytes"));
  const headers = bearer(w.owner.token);
  expect(
    (await api.DELETE("/api/v0/assets/{id}", { params: { path: { id: neighbor.id } }, headers })).response.status
  ).toBe(204);
  const deleted = await snapshot();
  expect(
    (await api.DELETE("/api/v0/assets/{id}", { params: { path: { id: neighbor.id } }, headers })).response.status
  ).toBe(404);
  expect(await snapshot()).toEqual(deleted);
  expect(
    await db.query(
      "SELECT is_deleted, deleted_at IS NOT NULL AS tombstone, storage_metadata->'final' IS DISTINCT FROM 'null'::jsonb AS keeps_recovery FROM file_assets WHERE id=$1",
      [neighbor.id]
    )
  ).toEqual([{ is_deleted: true, tombstone: true, keeps_recovery: true }]);
});
