import { createHash } from "node:crypto";

import { bearer } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import {
  completeFile,
  duplicateFile,
  filePixel,
  prepareFile,
  readFile,
  sendFile,
  uploadFile,
} from "../../fixtures/storage";
import { expect, test } from "../../fixtures/test";

test("F3 (API): confirmation is idempotent, old PUT cannot replace a final, copies own independent bytes and fake image declarations fail", async ({
  api,
  db,
}, info) => {
  const w = await fileWorld(api, info);
  const bytes = Buffer.from("same-name, independent bytes");
  const prepared = await prepareFile(api, w.owner.token, {
    purpose: "ISSUE_ATTACHMENT",
    scope: { kind: "issue", id: w.issue.id },
    name: "same.txt",
    size: bytes.length,
    content_type: "text/plain",
  });
  const pending = await api.GET("/api/v0/assets/{id}/content", {
    params: { path: { id: prepared.asset.id } },
    headers: bearer(w.owner.token),
  });
  expect(pending.response.status).toBe(409);
  expect(pending.error?.code).toBe("asset.not_uploaded");
  await uploadFile(prepared.upload, bytes);
  const completed = await completeFile(api, w.owner.token, prepared.asset.id);
  const rows = () => db.query("SELECT to_jsonb(a) AS row FROM file_assets a WHERE id=$1", [completed.id]);
  const before = await rows();
  expect(await completeFile(api, w.owner.token, completed.id)).toEqual(completed);
  expect(await rows()).toEqual(before);
  const rejected = await sendFile(prepared.upload, Buffer.alloc(bytes.length, 120));
  expect(rejected.status).toBe(409);
  await rejected.arrayBuffer();
  expect(await readFile(api, w.owner.token, completed.id)).toEqual(bytes);
  const first = await duplicateFile(api, w.owner.token, completed.id, {
    purpose: "COMMENT_ATTACHMENT",
    scope: { kind: "comment", id: w.comment.id },
  });
  const second = await duplicateFile(api, w.owner.token, completed.id, {
    purpose: "DRAFT_ISSUE_ATTACHMENT",
    scope: { kind: "draft", id: w.draft.id },
    name: "same.txt",
  });
  expect(new Set([completed.id, first.id, second.id]).size).toBe(3);
  expect(await readFile(api, w.owner.token, first.id)).toEqual(bytes);
  expect(await readFile(api, w.owner.token, second.id)).toEqual(bytes);
  const objects = await db.query<{ key: string; hash: string }>(
    "SELECT storage_metadata->'final'->>'key' AS key, storage_metadata->'final'->>'sha256' AS hash FROM file_assets WHERE id=ANY($1::uuid[])",
    [[completed.id, first.id, second.id]]
  );
  expect(new Set(objects.map(({ key }) => key)).size).toBe(3);
  expect(objects.map(({ hash }) => hash)).toEqual(Array(3).fill(createHash("sha256").update(bytes).digest("hex")));
  expect(await rows()).toEqual(before);
  const invalid = await prepareFile(api, w.owner.token, {
    purpose: "USER_AVATAR",
    scope: { kind: "user" },
    name: "fake.png",
    content_type: "image/png",
    size: filePixel.length,
  });
  await uploadFile(invalid.upload, Buffer.alloc(filePixel.length, 120));
  const failure = await api.POST("/api/v0/assets/{id}/complete", {
    params: { path: { id: invalid.asset.id } },
    body: {},
    headers: bearer(w.owner.token),
  });
  expect(failure.response.status).toBe(422);
  expect(failure.error?.code).toBe("asset.content_invalid");
  expect(
    await db.query(
      "SELECT is_uploaded, storage_metadata->'final' AS final, jsonb_array_length(storage_metadata->'candidates') AS candidates FROM file_assets WHERE id=$1",
      [invalid.asset.id]
    )
  ).toEqual([{ is_uploaded: false, final: null, candidates: 1 }]);
});
