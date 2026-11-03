import { addProjectMembers, createProject, inviteAndAccept } from "../../fixtures/api";
import { assetRows, imageHTML } from "../../fixtures/assert/storage";
import { bearer } from "../../fixtures/auth";
import { fileStory } from "../../fixtures/file-story";
import { fileWorld } from "../../fixtures/file-world";
import { draftPublished, draftRead, draftUpdated, issueRead } from "../../fixtures/issues";
import { completeFile, filePixel, prepareFile, readFile, storedFile, uploadFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

fileStory(
  "F10 (API): draft saves and moves preserve every object; publication rejects pending files, then migrates the full live set once without exposing unused images",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    await inviteAndAccept(api, w.owner.token, w.workspace.slug, w.other, 15);
    await addProjectMembers(api, w.owner.token, w.project.id, [{ member_id: w.other.id, role: 15 }]);
    const second = await createProject(api, w.owner.token, w.workspace.slug, {
      name: "Move target",
      identifier: "MOVE",
      network: 0,
    });
    const scope = { kind: "draft", id: w.draft.id } as const;
    const image = () =>
      storedFile(
        api,
        w.owner.token,
        "DRAFT_ISSUE_DESCRIPTION",
        scope,
        filePixel,
        "image/png",
        "image.png",
        uploadStatus
      );
    const history = await image(),
      current = await image(),
      unused = await image();
    const bytes = Buffer.from("draft attachment");
    const attachment = await storedFile(
      api,
      w.owner.token,
      "DRAFT_ISSUE_ATTACHMENT",
      scope,
      bytes,
      "text/plain",
      "draft.txt",
      uploadStatus
    );
    const pending = await prepareFile(api, w.owner.token, {
      purpose: "DRAFT_ISSUE_ATTACHMENT",
      scope,
      name: "later.txt",
      size: 5,
      content_type: "text/plain",
    });
    const ids = [history.id, current.id, unused.id, attachment.id, pending.asset.id];
    expect(
      await draftUpdated(api, w.owner.token, w.draft.id, {
        payload: { name: "Published files", description_html: imageHTML(history.id) },
      })
    ).toMatchObject({ status: 200 });
    const payload = { name: "Published files", description_html: imageHTML(current.id) };
    expect(await draftUpdated(api, w.owner.token, w.draft.id, { payload })).toMatchObject({ status: 200 });
    expect(await draftRead(api, w.owner.token, w.draft.id)).toMatchObject({ status: 200, draft: { payload } });
    // A->B->A exercises both UUID directions regardless of their creation order.
    for (const project of [second.id, w.project.id, null, w.project.id]) {
      // eslint-disable-next-line no-await-in-loop -- Each move depends on the preceding persisted scope.
      expect(await draftUpdated(api, w.owner.token, w.draft.id, { project_id: project })).toMatchObject({
        status: 200,
      });
      // eslint-disable-next-line no-await-in-loop
      const rows = await assetRows(db, ids);
      for (const row of rows)
        expect(row).toMatchObject({ draft_issue_id: w.draft.id, project_id: project, issue_id: null });
    }
    const before = await assetRows(db, ids);
    const draftBefore = await draftRead(api, w.owner.token, w.draft.id);
    const issueCount = await db.query("SELECT count(*)::int AS n FROM issues WHERE project_id=$1", [w.project.id]);
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({
      status: 409,
      code: "asset.not_uploaded",
    });
    expect(await assetRows(db, ids)).toEqual(before);
    expect(await draftRead(api, w.owner.token, w.draft.id)).toEqual(draftBefore);
    expect(await db.query("SELECT count(*)::int AS n FROM issues WHERE project_id=$1", [w.project.id])).toEqual(
      issueCount
    );
    await uploadFile(pending.upload, Buffer.from("later"), uploadStatus);
    await completeFile(api, w.owner.token, pending.asset.id);
    const origins = await assetRows(db, ids);
    const published = await draftPublished(api, w.owner.token, w.draft.id);
    expect(published.status).toBe(201);
    if (!("issue" in published)) throw new Error("publication returned no issue");
    const issue = published.issue;
    expect(issue.attachment_count).toBe(2);
    expect(issue.description_html).toBe(payload.description_html);
    const migrated = await assetRows(db, ids);
    for (const row of migrated) {
      const old = origins.find(({ id }) => id === row.id)!;
      expect(row).toMatchObject({
        draft_issue_id: null,
        issue_id: issue.id,
        project_id: w.project.id,
        created_at: old.created_at,
        created_by_id: old.created_by_id,
        bound_at: old.bound_at,
        storage_metadata: old.storage_metadata,
      });
      expect(row.entity_type).toBe(
        row.id === attachment.id || row.id === pending.asset.id ? "ISSUE_ATTACHMENT" : "ISSUE_DESCRIPTION"
      );
    }
    const publicIds = [history.id, current.id, attachment.id, pending.asset.id];
    for (const id of publicIds) {
      // eslint-disable-next-line no-await-in-loop
      expect(await readFile(api, w.other.token, id)).toEqual(
        id === attachment.id ? bytes : id === pending.asset.id ? Buffer.from("later") : filePixel
      );
    }
    expect(migrated.find(({ id }) => id === unused.id)?.bound_at).toBeNull();
    expect(
      (
        await api.GET("/api/v0/assets/{id}/content", {
          params: { path: { id: unused.id } },
          headers: bearer(w.other.token),
        })
      ).response.status
    ).toBe(404);
    expect(await readFile(api, w.owner.token, unused.id)).toEqual(filePixel);
    expect(await issueRead(api, w.other.token, issue.id)).toMatchObject({ issue: { attachment_count: 2 } });
    const list = await api.GET("/api/v0/issues/{issue_id}/attachments", {
      params: { path: { issue_id: issue.id } },
      headers: bearer(w.other.token),
    });
    expect(list.response.status).toBe(200);
    expect(list.data?.data.map(({ id }) => id)).toEqual([attachment.id, pending.asset.id].toSorted());
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({ status: 404 });
    expect(await draftRead(api, w.owner.token, w.draft.id)).toMatchObject({ status: 404 });
    expect(await assetRows(db, ids)).toEqual(migrated);
    expect(await db.query("SELECT count(*)::int AS n FROM issues WHERE project_id=$1", [w.project.id])).toEqual([
      { n: issueCount[0]!.n + 1 },
    ]);
  }
);
