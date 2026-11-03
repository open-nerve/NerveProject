import { randomBytes } from "node:crypto";
import { execFile } from "node:child_process";
import { access, readdir } from "node:fs/promises";
import { tmpdir } from "node:os";
import { promisify } from "node:util";

import { expect, test } from "@playwright/test";

import { createS3World } from "../../fixtures/s3-world";
import { removeS3Container, startS3Process } from "../../fixtures/s3-process";

// oxlint-disable-next-line no-empty-pattern -- Playwright requires destructured fixture arguments.
test("S3 fixture: concurrent close removes its container, process, secrets and database connection", async ({}, info) => {
  test.setTimeout(200_000);
  const world = await createS3World(info);
  const name = world.storage.name;
  try {
    const { stdout } = await promisify(execFile)("docker", [
      "inspect",
      "--format",
      "{{json .HostConfig.PortBindings}}",
      name,
    ]);
    const ports = JSON.parse(stdout) as Record<string, { HostIp: string }[]>;
    expect(
      Object.values(ports)
        .flat()
        .every((p) => p.HostIp === "127.0.0.1")
    ).toBe(true);
    const second = await world.storage.createBucket();
    expect(second.connection.bucket).not.toBe(world.bucket.connection.bucket);
    expect(second.connection.prefix).not.toBe(world.bucket.connection.prefix);
    await Promise.all([world.close(), world.close(), world.close()]);
    expect(world.nerve.exitCode).toBe(0);
    await expect(fetch(world.nerve.baseURL + "/healthz", { signal: AbortSignal.timeout(2000) })).rejects.toThrow();
    await expect(world.db.query("SELECT 1")).rejects.toThrow();
    await expect(access(world.secretsDirectory)).rejects.toMatchObject({ code: "ENOENT" });
    await expect(world.storage.createBucket()).rejects.toThrow("closed");
    await absent(name);
  } finally {
    await world.close();
  }
});

// oxlint-disable-next-line no-empty-pattern -- Playwright requires destructured fixture arguments.
test("S3 fixture: failed Nerve startup releases the service and temporary secrets", async ({}, info) => {
  test.setTimeout(200_000);
  const before = await ownedNames();
  const secrets = (await readdir(tmpdir())).filter((name) => name.startsWith("nerve-s3-e2e-")).toSorted();
  await expect(createS3World(info, { NERVE_FILES__OPERATION_TIMEOUT: "0s" })).rejects.toThrow(
    "nerve did not become ready"
  );
  expect(await ownedNames()).toEqual(before);
  expect((await readdir(tmpdir())).filter((name) => name.startsWith("nerve-s3-e2e-")).toSorted()).toEqual(secrets);
});

test("S3 fixture: close during child startup joins it before final owned cleanup", async () => {
  const name = "nerve-s3-e2e-" + randomBytes(16).toString("hex");
  const child = startS3Process(name, "{}");
  const readiness = child.ready.then(
    () => true,
    () => false
  );
  await Promise.all([child.close(), child.close()]);
  expect(await readiness).toBe(false);
  await absent(name);
  // The independent fallback is itself idempotent on exact daemon NotFound.
  await removeS3Container(name);
});

async function absent(name: string) {
  await expect(promisify(execFile)("docker", ["inspect", name], { timeout: 5000 })).rejects.toMatchObject({
    stderr: expect.stringMatching(/no such (object|container):/i),
  });
}
async function ownedNames(): Promise<string[]> {
  const { stdout } = await promisify(execFile)(
    "docker",
    ["ps", "-a", "--filter", "name=nerve-s3-e2e-", "--format", "{{.Names}}"],
    { timeout: 5000 }
  );
  return stdout.trim().split("\n").filter(Boolean).toSorted();
}
