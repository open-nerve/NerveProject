import { randomUUID } from "node:crypto";

import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { bearer } from "../../fixtures/auth";
import { commentUpdated, createIssueComment } from "../../fixtures/comments";
import { fileStory } from "../../fixtures/file-story";
import { fileWorld } from "../../fixtures/file-world";
import { activityPage, issueRead } from "../../fixtures/issues";
import { filePixel, readFile, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

fileStory(
  "F9 (API): failed comment sends preserve private preparations for retry; only the sent set acquires the final comment and host readers",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
    await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
    const scope = { kind: "new_comment", id: w.issue.id } as const;
    const image = await storedFile(
      api,
      w.owner.token,
      "COMMENT_DESCRIPTION",
      scope,
      filePixel,
      "image/png",
      "image.png",
      uploadStatus
    );
    const bytes = Buffer.from("comment attachment bytes");
    const attachment = await storedFile(
      api,
      w.owner.token,
      "COMMENT_ATTACHMENT",
      scope,
      bytes,
      "text/plain",
      "comment.txt",
      uploadStatus
    );
    const spare = await storedFile(
      api,
      w.owner.token,
      "COMMENT_ATTACHMENT",
      scope,
      bytes,
      "text/plain",
      "spare.txt",
      uploadStatus
    );
    const ids = [image.id, attachment.id, spare.id];
    const origins = await assetRows(db, ids);
    const content = (id: string) =>
      api.GET("/api/v0/assets/{id}/content", { params: { path: { id } }, headers: bearer(w.other.token) });
    const missing = await content(randomUUID());
    for (const id of ids) {
      // eslint-disable-next-line no-await-in-loop
      const hidden = await content(id);
      expect(hidden.response.status).toBe(404);
      expect(hidden.error).toEqual(missing.error);
    }
    const beforeComments = await db.query(
      "SELECT to_jsonb(c) AS row FROM issue_comments c WHERE issue_id=$1 ORDER BY id",
      [w.issue.id]
    );
    const refused = await api.POST("/api/v0/issues/{issue_id}/comments", {
      params: { path: { issue_id: w.issue.id } },
      headers: bearer(w.owner.token),
      body: { comment_html: imageHTML(image.id), attachment_ids: [attachment.id, randomUUID()] },
    });
    expect(refused.response.status).toBe(404);
    expect(refused.error?.code).toBe("asset.not_found");
    expect(await assetRows(db, ids)).toEqual(origins);
    expect(
      await db.query("SELECT to_jsonb(c) AS row FROM issue_comments c WHERE issue_id=$1 ORDER BY id", [w.issue.id])
    ).toEqual(beforeComments);
    const comment = await createIssueComment(api, w.owner.token, w.issue.id, imageHTML(image.id), [attachment.id]);
    const bound = await assetRows(db, [image.id, attachment.id]);
    for (const row of bound) {
      const old = origins.find(({ id }) => id === row.id)!;
      expect(row).toMatchObject({
        comment_id: comment.id,
        issue_id: w.issue.id,
        created_at: old.created_at,
        created_by_id: old.created_by_id,
        storage_metadata: old.storage_metadata,
      });
      expect(row.bound_at).not.toBeNull();
      expect(row.bound_at).toBe(row.updated_at);
    }
    expect(
      await db.query(
        "SELECT count(*)::int AS n FROM file_assets WHERE id=ANY($1::uuid[]) AND bound_at=$2 AND updated_at=$2",
        [[image.id, attachment.id], comment.created_at]
      )
    ).toEqual([{ n: 2 }]);
    expect(await assetRows(db, [spare.id])).toEqual(origins.filter(({ id }) => id === spare.id));
    expect((await content(spare.id)).response.status).toBe(404);
    expect(await readFile(api, w.other.token, image.id)).toEqual(filePixel);
    expect(await readFile(api, w.other.token, attachment.id)).toEqual(bytes);
    const list = await api.GET("/api/v0/issue-comments/{comment_id}/attachments", {
      params: { path: { comment_id: comment.id } },
      headers: bearer(w.other.token),
    });
    expect(list.response.status).toBe(200);
    expect(list.data?.data.map(({ id }) => id)).toEqual([attachment.id]);
    expect(await issueRead(api, w.other.token, w.issue.id)).toMatchObject({ issue: { attachment_count: 0 } });
    expect(
      (await activityPage(api, w.other.token, w.issue.id, undefined, 100)).data.filter(
        ({ field }) => field === "attachment"
      )
    ).toEqual([]);
    const reclaim = await api.POST("/api/v0/issues/{issue_id}/comments", {
      params: { path: { issue_id: w.issue.id } },
      headers: bearer(w.owner.token),
      body: { comment_html: "<p>Wrong composer</p>", attachment_ids: [attachment.id] },
    });
    expect(reclaim.response.status).toBe(404);
    expect(await assetRows(db, [image.id, attachment.id])).toEqual(bound);
    const next = await storedFile(
      api,
      w.owner.token,
      "COMMENT_DESCRIPTION",
      { kind: "comment", id: comment.id },
      filePixel,
      "image/png",
      "next.png",
      uploadStatus
    );
    expect(await commentUpdated(api, w.owner.token, comment.id, imageHTML(next.id))).toMatchObject({ status: 200 });
    expect(await readFile(api, w.other.token, image.id)).toEqual(filePixel);
    expect(await readFile(api, w.other.token, next.id)).toEqual(filePixel);
    expect(await assetRows(db, [image.id, attachment.id])).toEqual(bound);
  }
);
