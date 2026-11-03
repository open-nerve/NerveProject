import { createHash } from "node:crypto";
import { createServer, type Server } from "node:http";
import { once } from "node:events";

import { expect, type Page } from "@playwright/test";

import { bearer } from "../../fixtures/auth";
import { fileWorld } from "../../fixtures/file-world";
import { s3Test as test } from "../../fixtures/s3-world";
import { completeFile, duplicateFile, fileContent, prepareFile } from "../../fixtures/storage";

test("F4 (browser): private S3 transfer binds browser File length/type, returns exact bytes, enforces CORS and one storage origin", async ({
  s3World: world,
  page,
}, info) => {
  const w = await fileWorld(world.api, info);
  const bytes = Buffer.from("browser-owned S3 bytes\nwith a second line\n");
  const hash = createHash("sha256").update(bytes).digest("hex");
  const prepared = await prepareFile(world.api, w.owner.token, {
    purpose: "ISSUE_ATTACHMENT",
    scope: { kind: "issue", id: w.issue.id },
    name: "浏览器.txt",
    size: bytes.length,
    content_type: "text/plain",
  });
  await page.goto(world.nerve.baseURL + "/auth/sign-in");
  const requests: { method: string; authorization?: string; cookie?: string; size?: string }[] = [];
  page.on("request", (request) => {
    if (new URL(request.url()).origin === world.storage.endpoint) {
      const headers = request.headers();
      requests.push({
        method: request.method(),
        authorization: headers.authorization,
        cookie: headers.cookie,
        size: headers["content-length"],
      });
    }
  });
  const upload = async (contentType: string, uploadBytes: number[]) =>
    page.evaluate(
      async ({ url, type, data }) => {
        // A real File supplies Content-Length; browsers forbid setting it manually.
        const file = new File([new Uint8Array(data)], "upload.txt", { type });
        const response = await fetch(url, {
          method: "PUT",
          headers: { "Content-Type": type },
          body: file,
          credentials: "omit",
        });
        await response.arrayBuffer();
        return response.status;
      },
      { url: prepared.upload.url, type: contentType, data: uploadBytes }
    );
  expect(prepared.upload.method).toBe("PUT");
  expect(prepared.upload.headers).toEqual([{ name: "Content-Type", value: "text/plain" }]);
  expect(new URL(prepared.upload.url).origin).toBe(world.storage.endpoint);
  expect(await upload("text/plain", [...bytes])).toBe(200);
  expect(await upload("application/octet-stream", [...bytes])).toBe(403);
  expect(await upload("text/plain", [...bytes, 120])).toBe(403);
  const asset = await completeFile(world.api, w.owner.token, prepared.asset.id);
  expect(await completeFile(world.api, w.owner.token, asset.id)).toEqual(asset);
  const content = await fileContent(world.api, w.owner.token, asset.id);
  const downloaded = await browserRead(page, content.url);
  expect(downloaded).toMatchObject({
    status: 200,
    bytes: [...bytes],
    type: "text/plain",
    length: String(bytes.length),
  });
  expect(downloaded.etag).toMatch(/^".+"$/);
  expect(downloaded.disposition).toContain("attachment;");
  expect(decodeURIComponent(downloaded.disposition ?? "")).toContain("浏览器.txt");
  // Content is a GET capability: changing it to HEAD must not reuse that signature.
  expect(
    await page.evaluate(async (url) => (await fetch(url, { method: "HEAD", credentials: "omit" })).status, content.url)
  ).toBe(403);
  const anonymous = new URL(content.url);
  anonymous.search = "";
  expect((await browserRead(page, anonymous.href)).status).toBe(403);
  const tampered = new URL(content.url);
  tampered.searchParams.set("X-Amz-Signature", "0".repeat(64));
  expect((await browserRead(page, tampered.href)).status).toBe(403);
  const copied = await duplicateFile(world.api, w.owner.token, asset.id, {
    purpose: "COMMENT_ATTACHMENT",
    scope: { kind: "comment", id: w.comment.id },
  });
  expect((await browserRead(page, (await fileContent(world.api, w.owner.token, copied.id)).url)).bytes).toEqual([
    ...bytes,
  ]);
  const objects = await world.db.query<{ key: string; hash: string }>(
    "SELECT storage_metadata->'final'->>'key' AS key, storage_metadata->'final'->>'sha256' AS hash FROM file_assets WHERE id=ANY($1::uuid[])",
    [[asset.id, copied.id]]
  );
  expect(new Set(objects.map((o) => o.key)).size).toBe(2);
  for (const object of objects) {
    expect(object.hash).toBe(hash);
    // eslint-disable-next-line no-await-in-loop
    expect(await world.bucket.inspect(object.key)).toMatchObject({
      size: bytes.length,
      sha256: hash,
      content_type: "text/plain",
    });
  }
  expect(requests.filter((r) => r.method === "PUT")).toHaveLength(3);
  expect(requests.every((r) => !r.authorization && !r.cookie)).toBe(true);
  // This second origin intentionally allows CORS and serves known bytes, so
  // a blocked fetch here proves CSP, not an unreachable server or CORS error.
  let hits = 0;
  const second = createServer((req, res) => {
    if (req.url === "/probe") hits++;
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.end(req.url === "/probe" ? "second-origin-bytes" : "<!doctype html><title>Other origin</title>");
  });
  second.listen(0, "127.0.0.1");
  await once(second, "listening");
  try {
    const address = second.address();
    if (!address || typeof address === "string") throw new Error("missing probe port");
    const other = `http://127.0.0.1:${address.port}`;
    expect(await (await fetch(other + "/probe")).text()).toBe("second-origin-bytes");
    hits = 0;
    const blocked = await page.evaluate(async (url) => {
      try {
        await fetch(url);
        return false;
      } catch {
        return true;
      }
    }, other + "/probe");
    expect(blocked, "CSP must block the reachable, CORS-enabled second origin").toBe(true);
    expect(hits).toBe(0);
    const wrongOrigin = await page.context().newPage();
    try {
      await wrongOrigin.goto(other);
      expect(
        await wrongOrigin.evaluate(async (url) => {
          try {
            await fetch(url, { credentials: "omit" });
            return false;
          } catch {
            return true;
          }
        }, content.url),
        "wrong-origin browser cannot read the valid capability"
      ).toBe(true);
      // CORS does not revoke a bearer URL for non-browser clients.
      expect(Buffer.from(await (await fetch(content.url)).arrayBuffer())).toEqual(bytes);
    } finally {
      await wrongOrigin.close();
    }
  } finally {
    await closeServer(second);
  }
  for (const id of [asset.id, copied.id]) {
    // eslint-disable-next-line no-await-in-loop
    const deleted = await world.api.DELETE("/api/v0/assets/{id}", {
      params: { path: { id } },
      headers: bearer(w.owner.token),
    });
    expect(deleted.response.status).toBe(204);
    // eslint-disable-next-line no-await-in-loop
    const read = await world.api.GET("/api/v0/assets/{id}/content", {
      params: { path: { id } },
      headers: bearer(w.owner.token),
    });
    expect(read.response.status).toBe(404);
  }
});

async function browserRead(page: Page, target: string) {
  return page.evaluate(async (url) => {
    const response = await fetch(url, { credentials: "omit" });
    return {
      status: response.status,
      bytes: [...new Uint8Array(await response.arrayBuffer())],
      type: response.headers.get("content-type"),
      length: response.headers.get("content-length"),
      etag: response.headers.get("etag"),
      disposition: response.headers.get("content-disposition"),
    };
  }, target);
}
async function closeServer(server: Server): Promise<void> {
  server.closeAllConnections();
  await new Promise<void>((resolve, reject) =>
    server.close((error) => {
      if (error) {
        reject(error);
        return;
      }
      resolve();
    })
  );
}
