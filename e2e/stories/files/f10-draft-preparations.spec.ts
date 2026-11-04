import { assetRows } from "../../fixtures/assert/storage";
import { bearer } from "../../fixtures/auth";
import { fileStory } from "../../fixtures/file-story";
import { fileWorld } from "../../fixtures/file-world";
import { createDraft, draftPublished } from "../../fixtures/issues";
import { completeFile, prepareFile, readFile, storedFile, uploadFile } from "../../fixtures/storage";
import { expect } from "../../fixtures/test";

fileStory(
  "F10 (API): a lost prepare answer is recovered by the private list and explicit ID deletion; same-name preparations survive and publication rechecks them",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    const scope = { kind: "draft", id: w.draft.id } as const;
    const body = {
      purpose: "DRAFT_ISSUE_ATTACHMENT",
      scope,
      name: "same.txt",
      size: 5,
      content_type: "text/plain",
    } as const;
    const neighbor = await prepareFile(api, w.owner.token, body);
    const otherDraft = await createDraft(api, w.owner.token, w.workspace.slug, { payload: {} });
    const elsewhere = await prepareFile(api, w.owner.token, {
      ...body,
      scope: { kind: "draft", id: otherDraft.id },
    });
    // The caller discards the answer's data: it learns the unknown ID only below.
    expect((await api.POST("/api/v0/assets", { body, headers: bearer(w.owner.token) })).response.status).toBe(201);
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({
      status: 409,
      code: "asset.not_uploaded",
    });
    const listed = await api.GET("/api/v0/drafts/{draft_id}/asset-preparations", {
      params: { path: { draft_id: w.draft.id } },
      headers: bearer(w.owner.token),
    });
    expect(listed.response.status).toBe(200);
    expect(listed.response.headers.get("cache-control")).toBe("no-store");
    expect(listed.data?.data).toHaveLength(2);
    const recovered = listed.data?.data.find(({ id }) => id !== neighbor.asset.id);
    if (!recovered) throw new Error("the unknown preparation was not discoverable");
    expect(recovered).toMatchObject({ name: "same.txt", state: "pending", is_bound: false, scope });
    expect(recovered.id).not.toBe(elsewhere.asset.id);
    expect(Object.keys(recovered).toSorted()).toEqual(
      [
        "id",
        "purpose",
        "scope",
        "name",
        "size",
        "content_type",
        "state",
        "is_bound",
        "created_at",
        "updated_at",
        "width",
        "height",
      ].toSorted()
    );
    const refused = await api.GET("/api/v0/drafts/{draft_id}/asset-preparations", {
      params: { path: { draft_id: w.draft.id } },
      headers: bearer(w.other.token),
    });
    expect(refused.response.status).toBe(404);
    expect(refused.error?.code).toBe("asset.not_found");
    expect(JSON.stringify(refused.error)).not.toContain("same.txt");
    expect(
      (
        await api.DELETE("/api/v0/assets/{id}", {
          params: { path: { id: recovered.id } },
          headers: bearer(w.owner.token),
        })
      ).response.status
    ).toBe(204);
    const deleted = await assetRows(db, [recovered.id]);
    expect(deleted[0]?.deleted_at).not.toBeNull();
    expect(
      (
        await api.DELETE("/api/v0/assets/{id}", {
          params: { path: { id: recovered.id } },
          headers: bearer(w.owner.token),
        })
      ).response.status
    ).toBe(204);
    expect(await assetRows(db, [recovered.id])).toEqual(deleted);
    const remaining = await api.GET("/api/v0/drafts/{draft_id}/asset-preparations", {
      params: { path: { draft_id: w.draft.id } },
      headers: bearer(w.owner.token),
    });
    expect(remaining.data?.data.map(({ id }) => id)).toEqual([neighbor.asset.id]);
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({
      status: 409,
      code: "asset.not_uploaded",
    });
    await uploadFile(neighbor.upload, Buffer.from("other"), uploadStatus);
    await completeFile(api, w.owner.token, neighbor.asset.id);
    const complete = await assetRows(db, [neighbor.asset.id]);
    await completeFile(api, w.owner.token, neighbor.asset.id);
    expect(await assetRows(db, [neighbor.asset.id])).toEqual(complete);
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({
      status: 201,
      issue: { attachment_count: 1 },
    });
    expect(await readFile(api, w.owner.token, neighbor.asset.id)).toEqual(Buffer.from("other"));
    const untouched = await api.GET("/api/v0/drafts/{draft_id}/asset-preparations", {
      params: { path: { draft_id: otherDraft.id } },
      headers: bearer(w.owner.token),
    });
    expect(untouched.data?.data.map(({ id }) => id)).toEqual([elsewhere.asset.id]);
  }
);

fileStory(
  "F10 (API): image validation fails after duplicate created an unknown target; its preparation can be found and discarded without changing source bytes",
  async ({ api, db, uploadStatus }, info) => {
    const w = await fileWorld(api, info);
    // Ordinary attachments may hold arbitrary bytes. Copying this one as a
    // description image fails real image validation after target preparation.
    const bytes = Buffer.from("not an image");
    const source = await storedFile(
      api,
      w.owner.token,
      "ISSUE_ATTACHMENT",
      { kind: "issue", id: w.issue.id },
      bytes,
      "image/png",
      "source.png",
      uploadStatus
    );
    const failed = await api.POST("/api/v0/assets/{id}/duplicate", {
      params: { path: { id: source.id } },
      body: { purpose: "DRAFT_ISSUE_DESCRIPTION", scope: { kind: "draft", id: w.draft.id } },
      headers: bearer(w.owner.token),
    });
    expect(failed.response.status).toBe(422);
    expect(failed.error?.code).toBe("asset.content_invalid");
    expect(failed.data).toBeUndefined();
    const listed = await api.GET("/api/v0/drafts/{draft_id}/asset-preparations", {
      params: { path: { draft_id: w.draft.id } },
      headers: bearer(w.owner.token),
    });
    expect(listed.response.status).toBe(200);
    expect(listed.data?.data).toHaveLength(1);
    const target = listed.data?.data[0];
    if (!target) throw new Error("failed duplicate's unknown target was not discoverable");
    expect(target.id).not.toBe(source.id);
    expect(target).toMatchObject({ name: "source.png", state: "pending", is_bound: false });
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({
      status: 409,
      code: "asset.not_uploaded",
    });
    expect(await readFile(api, w.owner.token, source.id)).toEqual(bytes);
    expect(
      (await api.DELETE("/api/v0/assets/{id}", { params: { path: { id: target.id } }, headers: bearer(w.owner.token) }))
        .response.status
    ).toBe(204);
    expect(await draftPublished(api, w.owner.token, w.draft.id)).toMatchObject({
      status: 201,
      issue: { attachment_count: 0 },
    });
    expect(await readFile(api, w.owner.token, source.id)).toEqual(bytes);
    expect((await assetRows(db, [target.id]))[0]?.deleted_at).not.toBeNull();
  }
);
