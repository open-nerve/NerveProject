import { generateKeyPairSync } from "node:crypto";
import { mkdir, readFile as readPhysicalFile, writeFile } from "node:fs/promises";
import path from "node:path";

import { bearer, login } from "../../fixtures/auth";
import { archiveWatermark, completedPeriodicJobs } from "../../fixtures/auto-archive";
import { fileWorld } from "../../fixtures/file-world";
import { fileContent, prepareFile, readFile, sendFile, storedFile, waitForFileSafety } from "../../fixtures/storage";
import { clockTest as test, expect } from "../../fixtures/test";

test.use({
  // oxlint-disable-next-line no-empty-pattern -- Playwright reads fixture dependencies from this pattern
  clockEnv: async ({}, use, info) => {
    // This key is newly generated test material, used only by this world's processes.
    // Stable configured signing keys are required for a pre-restart capability to survive.
    const filename = info.outputPath("file-signing.pem");
    await mkdir(path.dirname(filename), { recursive: true });
    const { privateKey } = generateKeyPairSync("ed25519");
    await writeFile(filename, privateKey.export({ type: "pkcs8", format: "pem" }), { mode: 0o600 });
    await use({
      NERVE_FILES__UPLOAD_TTL: "10s",
      NERVE_FILES__CLEANUP_INTERVAL: "1m",
      NERVE_FILES__CLEANUP_BATCH: "2",
      NERVE_AUTH__JWT__PRIVATE_KEY_FILE: filename,
    });
  },
});

test("F15 (API): +61 business days cannot expire a runtime upload; process restarts retain the root, safety deadlines and recoverable bytes before physical purge", async ({
  clockWorld: world,
}, info) => {
  test.setTimeout(120_000);
  const db = world.db;
  let api = world.current.api;
  const w = await fileWorld(api, info);
  const root = world.current.nerve.storageRoot;
  const old = await storedFile(
    api,
    w.owner.token,
    "ISSUE_ATTACHMENT",
    { kind: "issue", id: w.issue.id },
    Buffer.from("old deleted bytes")
  );
  const oldRead = await fileContent(api, w.owner.token, old.id);
  const unbound = await storedFile(
    api,
    w.owner.token,
    "ISSUE_ATTACHMENT",
    { kind: "issue", id: w.issue.id },
    Buffer.from("seven day preparation")
  );
  const originalKeys = await db.query<{ id: string; key: string }>(
    "SELECT id, storage_metadata->'final'->>'key' AS key FROM file_assets WHERE id=ANY($1::uuid[]) ORDER BY id",
    [[old.id, unbound.id]]
  );
  expect(
    (await api.DELETE("/api/v0/assets/{id}", { params: { path: { id: old.id } }, headers: bearer(w.owner.token) }))
      .response.status
  ).toBe(204);
  const pending = await prepareFile(api, w.owner.token, {
    purpose: "ISSUE_ATTACHMENT",
    scope: { kind: "issue", id: w.issue.id },
    name: "late.txt",
    size: 3,
    content_type: "text/plain",
  });
  await world.advance("1464h");
  api = world.current.api;
  w.owner.token = (await login(api, w.owner.email)).access_token;
  expect(world.current.nerve.storageRoot).toBe(root);
  const rebase = (url: string) => {
    const u = new URL(url);
    return new URL(u.pathname + u.search, world.current.nerve.baseURL).href;
  };
  const resumedUpload = { ...pending.upload, url: rebase(pending.upload.url) };
  const uploaded = await sendFile(resumedUpload, Buffer.from("old"));
  expect(uploaded.status, await uploaded.text()).toBe(204);
  expect(await db.query("SELECT id FROM file_assets WHERE id=$1", [old.id])).toEqual([{ id: old.id }]);
  for (const row of originalKeys) {
    // eslint-disable-next-line no-await-in-loop
    expect((await readPhysicalFile(path.join(root, "objects", row.key, "data"))).length).toBeGreaterThan(0);
  }
  await expect
    .poll(() => Date.now(), { timeout: 15_000, intervals: [100, 250] })
    .toBeGreaterThan(Date.parse(pending.upload.expires_at));
  const expired = await sendFile(resumedUpload, Buffer.from("old"));
  expect(expired.status).toBe(403);
  expect(await expired.json()).toMatchObject({ code: "asset.invalid_signature" });
  const recent = await storedFile(
    api,
    w.owner.token,
    "ISSUE_ATTACHMENT",
    { kind: "issue", id: w.issue.id },
    Buffer.from("recent live bytes")
  );
  await waitForFileSafety(db, [old.id, unbound.id, pending.asset.id]);
  const watermark = await archiveWatermark(db);
  await world.advance("1464h");
  api = world.current.api;
  w.owner.token = (await login(api, w.owner.email)).access_token;
  expect(world.current.nerve.storageRoot).toBe(root);
  await expect
    .poll(async () => (await completedPeriodicJobs(db, "asset.cleanup", watermark, world.current.startedAt)).length, {
      timeout: 15_000,
    })
    .toBeGreaterThan(0);
  await expect
    .poll(() => db.query("SELECT id FROM file_assets WHERE id=$1", [old.id]), { timeout: 15_000 })
    .toEqual([]);
  const gone = await fetch(rebase(oldRead.url), { credentials: "omit" });
  expect(gone.status).toBe(404);
  await gone.arrayBuffer();
  expect(
    await db.query(
      "SELECT is_deleted, deleted_by_id, storage_metadata->'staging' AS staging, storage_metadata->'final' IS DISTINCT FROM 'null'::jsonb AS final FROM file_assets WHERE id=$1",
      [unbound.id]
    )
  ).toEqual([{ is_deleted: true, deleted_by_id: null, staging: null, final: true }]);
  const kept = originalKeys.find(({ id }) => id === unbound.id);
  if (!kept) throw new Error("missing unbound final proof");
  expect(await readPhysicalFile(path.join(root, "objects", kept.key, "data"))).toEqual(
    Buffer.from("seven day preparation")
  );
  expect(await readFile(api, w.owner.token, recent.id)).toEqual(Buffer.from("recent live bytes"));
  expect(
    (await api.GET("/api/v0/assets/{id}", { params: { path: { id: recent.id } }, headers: bearer(w.owner.token) })).data
  ).toEqual(recent);
  await info.attach("asset-cleanup-attempts", {
    body: JSON.stringify(
      await db.query("SELECT id,state,metadata,errors FROM river_job WHERE kind='asset.cleanup' ORDER BY id")
    ),
    contentType: "application/json",
  });
});
