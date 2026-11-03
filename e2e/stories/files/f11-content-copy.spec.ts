import { randomUUID } from "node:crypto";

import { addProjectMembers, createProject, inviteAndAccept } from "../../fixtures/api";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { bearer } from "../../fixtures/auth";
import { fileStory } from "../../fixtures/file-story";
import { fileWorld } from "../../fixtures/file-world";
import { createIssue, updateIssue } from "../../fixtures/issues";
import { duplicateFile, filePixel, readFile, storedFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

fileStory(
  "F11 (API): a readable source and writable destination create an independent copy; hidden sources win over forbidden destinations in either project order",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
    await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
    const second = await createProject(api, w.owner.token, w.workspace.slug, {
      name: "Other destination",
      identifier: "COPY",
      network: 0,
    });
    await addProjectMembers(api, w.owner.token, second.id, [{ member_id: w.other.id, role: 15 }]);
    const otherIssue = await createIssue(api, w.owner.token, second.id, { name: "Copy destination" });
    // Both project lock orders get an independently uploaded and bound source.
    for (const [source, target] of [
      [w.issue, otherIssue],
      [otherIssue, w.issue],
    ]) {
      // eslint-disable-next-line no-await-in-loop
      const original = await storedFile(
        api,
        w.owner.token,
        "ISSUE_DESCRIPTION",
        { kind: "issue", id: source!.id },
        filePixel,
        "image/png",
        "source.png",
        uploadStatus
      );
      // eslint-disable-next-line no-await-in-loop
      await updateIssue(api, w.owner.token, source!.id, { description_html: imageHTML(original.id) });
      // eslint-disable-next-line no-await-in-loop
      const before = await assetRows(db, [original.id]);
      // eslint-disable-next-line no-await-in-loop
      const copy = await duplicateFile(api, w.owner.token, original.id, {
        purpose: "ISSUE_DESCRIPTION",
        scope: { kind: "issue", id: target!.id },
      });
      expect(copy.id).not.toBe(original.id);
      expect(copy.is_bound).toBe(false);
      // eslint-disable-next-line no-await-in-loop
      const rows = await assetRows(db, [copy.id]);
      const copiedMetadata = rows[0]?.storage_metadata as { origin: string; final: { key: string; sha256: string } };
      const sourceMetadata = before[0]?.storage_metadata as { final: { key: string; sha256: string } };
      expect(copiedMetadata.origin).toBe("copy");
      expect(copiedMetadata.final.key).not.toBe(sourceMetadata.final.key);
      expect(copiedMetadata.final.sha256).toBe(sourceMetadata.final.sha256);
      // eslint-disable-next-line no-await-in-loop
      expect(await readFile(api, w.owner.token, copy.id)).toEqual(filePixel);
      // eslint-disable-next-line no-await-in-loop
      await updateIssue(api, w.owner.token, target!.id, { description_html: imageHTML(copy.id) });
      // eslint-disable-next-line no-await-in-loop
      expect(await readFile(api, w.other.token, copy.id)).toEqual(filePixel);
      // eslint-disable-next-line no-await-in-loop
      expect(await assetRows(db, [original.id])).toEqual(before);
    }
    const source = await storedFile(
      api,
      w.owner.token,
      "ISSUE_ATTACHMENT",
      { kind: "issue", id: w.issue.id },
      filePixel,
      "image/png",
      "visible.png",
      uploadStatus
    );
    const privateImage = await storedFile(
      api,
      w.owner.token,
      "ISSUE_DESCRIPTION",
      { kind: "issue", id: w.issue.id },
      filePixel,
      "image/png",
      "private.png",
      uploadStatus
    );
    const request = (id: string, forbiddenTarget: boolean) =>
      api.POST("/api/v0/assets/{id}/duplicate", {
        params: { path: { id } },
        headers: bearer(w.other.token),
        body: forbiddenTarget
          ? { purpose: "PROJECT_COVER", scope: { kind: "project", id: second.id } }
          : { purpose: "ISSUE_ATTACHMENT", scope: { kind: "issue", id: w.issue.id } },
      });
    const before = await db.query("SELECT to_jsonb(a) AS row FROM file_assets a ORDER BY id");
    const targetDenied = await request(source.id, true);
    expect(targetDenied.response.status).toBe(403);
    const absent = await request(randomUUID(), true);
    const bothDenied = await request(privateImage.id, true);
    const hiddenSourceOnly = await request(privateImage.id, false);
    expect(absent.response.status).toBe(404);
    expect(absent.error?.code).toBe("asset.not_found");
    for (const hidden of [bothDenied, hiddenSourceOnly]) {
      expect(hidden.response.status).toBe(404);
      expect(hidden.error).toEqual(absent.error);
    }
    expect(await db.query("SELECT to_jsonb(a) AS row FROM file_assets a ORDER BY id")).toEqual(before);
    const copy = await duplicateFile(api, w.other.token, source.id, {
      purpose: "ISSUE_ATTACHMENT",
      scope: { kind: "issue", id: w.issue.id },
    });
    expect(copy.is_bound).toBe(true);
    expect(copy.id).not.toBe(source.id);
    expect(await readFile(api, w.owner.token, copy.id)).toEqual(filePixel);
  }
);
