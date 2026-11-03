import { addProjectMembers, inviteAndAccept } from "../../fixtures/api";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { bearer } from "../../fixtures/auth";
import { fileStory } from "../../fixtures/file-story";
import { fileWorld } from "../../fixtures/file-world";
import { activityPage, createIssue, issueRead, listIssues, updateIssue } from "../../fixtures/issues";
import { completeFile, fileContent, filePixel, prepareFile, readFile, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

fileStory(
  "F8 (API): ordinary attachments preserve independent bytes, host access, metadata, count, order and exact add/delete activities",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
    await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
    const zero = await createIssue(api, w.owner.token, w.project.id, { name: "No ordinary attachments" });
    const scope = { kind: "issue", id: w.issue.id } as const;
    const aBytes = Buffer.from("first bytes");
    const bBytes = Buffer.from("other bytes");
    expect(aBytes.length).toBe(bBytes.length);
    expect(aBytes).not.toEqual(bBytes);
    const a = await storedFile(
      api,
      w.owner.token,
      "ISSUE_ATTACHMENT",
      scope,
      aBytes,
      "text/plain",
      "same.txt",
      uploadStatus
    );
    const b = await storedFile(
      api,
      w.owner.token,
      "ISSUE_ATTACHMENT",
      scope,
      bBytes,
      "text/plain",
      "same.txt",
      uploadStatus
    );
    const image = await storedFile(
      api,
      w.owner.token,
      "ISSUE_DESCRIPTION",
      scope,
      filePixel,
      "image/png",
      "image.png",
      uploadStatus
    );
    await updateIssue(api, w.owner.token, w.issue.id, { description_html: imageHTML(image.id) });
    await prepareFile(api, w.owner.token, {
      purpose: "ISSUE_ATTACHMENT",
      scope,
      name: "pending.txt",
      size: 4,
      content_type: "text/plain",
    });
    const before = await assetRows(db, [a.id, b.id]);
    expect(before.every((row) => row.bound_at !== null)).toBe(true);
    expect((before[0]!.storage_metadata as { final: { key: string } }).final.key).not.toBe(
      (before[1]!.storage_metadata as { final: { key: string } }).final.key
    );
    await completeFile(api, w.owner.token, a.id);
    expect(await assetRows(db, [a.id, b.id])).toEqual(before);
    const list = () =>
      api.GET("/api/v0/issues/{issue_id}/attachments", {
        params: { path: { issue_id: w.issue.id } },
        headers: bearer(w.other.token),
      });
    const page = await list();
    expect(page.response.status).toBe(200);
    expect(page.data?.next_cursor).toBeNull();
    expect(page.data?.data.map(({ id }) => id)).toEqual([a.id, b.id].toSorted());
    for (const row of page.data?.data ?? []) {
      expect(Object.keys(row).toSorted()).toEqual([
        "content_type",
        "created_at",
        "created_by_id",
        "id",
        "name",
        "size",
      ]);
      expect(row).toMatchObject({
        name: "same.txt",
        size: aBytes.length,
        content_type: "text/plain",
        created_by_id: w.owner.id,
      });
    }
    expect(await readFile(api, w.other.token, a.id)).toEqual(aBytes);
    expect(await readFile(api, w.other.token, b.id)).toEqual(bBytes);
    expect(await readFile(api, w.other.token, image.id)).toEqual(filePixel);
    for (const [id, disposition] of [
      [a.id, "attachment;"],
      [image.id, "inline;"],
    ]) {
      // eslint-disable-next-line no-await-in-loop -- Each capability is checked independently.
      const content = await fileContent(api, w.other.token, id!);
      expect(Date.parse(content.expires_at) - Date.now()).toBeGreaterThan(290_000);
      expect(Date.parse(content.expires_at) - Date.now()).toBeLessThanOrEqual(65 * 60_000);
      // eslint-disable-next-line no-await-in-loop
      const response = await fetch(content.url, { credentials: "omit", signal: AbortSignal.timeout(15_000) });
      expect(response.status).toBe(200);
      expect(response.headers.get("content-disposition")).toContain(disposition);
      // eslint-disable-next-line no-await-in-loop
      expect(Buffer.from(await response.arrayBuffer())).toEqual(id === a.id ? aBytes : filePixel);
    }
    expect(await issueRead(api, w.other.token, w.issue.id)).toMatchObject({
      status: 200,
      issue: { attachment_count: 2 },
    });
    const sorted = await listIssues(api, w.other.token, w.project.id, { order_by: "-attachment_count" });
    if (!sorted.data) throw new Error("ungrouped list returned no data");
    expect(sorted.data.map(({ id, attachment_count }) => [id, attachment_count])).toEqual([
      [w.issue.id, 2],
      [zero.id, 0],
    ]);
    const added = (await activityPage(api, w.other.token, w.issue.id, undefined, 100)).data.filter(
      ({ field }) => field === "attachment"
    );
    expect(added).toHaveLength(2);
    expect(added.map(({ new_identifier }) => new_identifier).toSorted()).toEqual([a.id, b.id].toSorted());
    for (const activity of added)
      expect(activity).toMatchObject({
        verb: "created",
        actor_id: w.owner.id,
        old_identifier: null,
        old_value: null,
        new_value: null,
      });
    const remove = () =>
      api.DELETE("/api/v0/assets/{id}", { params: { path: { id: a.id } }, headers: bearer(w.other.token) });
    expect((await remove()).response.status).toBe(204);
    const tombstone = await assetRows(db, [a.id]);
    expect(tombstone[0]).toMatchObject({ is_deleted: true, deleted_by_id: w.other.id });
    expect((await remove()).response.status).toBe(204);
    expect(await assetRows(db, [a.id])).toEqual(tombstone);
    expect((await list()).data?.data.map(({ id }) => id)).toEqual([b.id]);
    expect(await issueRead(api, w.other.token, w.issue.id)).toMatchObject({ issue: { attachment_count: 1 } });
    const deleted = (await activityPage(api, w.other.token, w.issue.id, undefined, 100)).data.filter(
      ({ field }) => field === "attachment"
    );
    expect(deleted).toHaveLength(3);
    expect(deleted.filter(({ verb }) => verb === "deleted")).toMatchObject([
      { old_identifier: a.id, new_identifier: null, actor_id: w.other.id, old_value: null, new_value: null },
    ]);
    const refused = await api.GET("/api/v0/assets/{id}/content", {
      params: { path: { id: a.id } },
      headers: bearer(w.owner.token),
    });
    expect(refused.response.status).toBe(404);
    expect(refused.error?.code).toBe("asset.not_found");
  }
);
