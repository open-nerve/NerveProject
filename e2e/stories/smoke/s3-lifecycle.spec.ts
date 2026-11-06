import { randomBytes } from "node:crypto";
import { execFile } from "node:child_process";
import { access, readFile } from "node:fs/promises";
import { promisify } from "node:util";

import { expect, test } from "@playwright/test";

import { createS3World } from "../../fixtures/s3-world";
import { removeS3Container, startS3Process } from "../../fixtures/s3-process";
import { startS3 } from "../../fixtures/s3";

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
  await expect(createS3World(info, { NERVE_FILES__OPERATION_TIMEOUT: "0s" })).rejects.toThrow(
    "nerve did not become ready"
  );
  // The failed world records its actual resources before starting Nerve. Other
  // worlds may create or release their resources while this one is closing.
  const resources = JSON.parse(await readFile(info.outputPath("s3-resources.json"), "utf8")) as {
    containerName: string;
    secretsDirectory: string;
  };
  expect(resources.containerName).toMatch(/^nerve-s3-e2e-[0-9a-f]{32}$/);
  expect(resources.secretsDirectory).toContain("nerve-s3-e2e-");
  await absent(resources.containerName);
  await expect(access(resources.secretsDirectory)).rejects.toMatchObject({ code: "ENOENT" });
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

test("S3 fixture: a child that refuses its startup reports its own error", async () => {
  const name = "nerve-s3-e2e-" + randomBytes(16).toString("hex");
  const child = startS3Process(name, "");
  const failure = await child.ready.then(
    () => undefined,
    (error: unknown) => error
  );
  await child.close();
  expect(failure).toMatchObject({ cause: { message: "invalid startup" } });
  await absent(name);
});

test("S3 fixture: a failed startup whose cleanup also fails keeps both errors", async () => {
  const names: string[] = [];
  const cleanup = new Error("cleanup refused");
  const failure = await startS3({
    auth: "",
    remove: async (name) => {
      names.push(name);
      await removeS3Container(name);
      throw cleanup;
    },
  }).then(
    async (service) => {
      await service.close();
      return undefined;
    },
    (error: unknown) => error
  );
  expect(names).toHaveLength(1);
  await absent(names[0]!);
  expect(failure).toBeInstanceOf(AggregateError);
  expect((failure as AggregateError).errors).toEqual([
    expect.objectContaining({ cause: expect.objectContaining({ message: "invalid startup" }) }),
    cleanup,
  ]);
});

async function absent(name: string) {
  await expect(promisify(execFile)("docker", ["inspect", name], { timeout: 5000 })).rejects.toMatchObject({
    stderr: expect.stringMatching(/no such (object|container):/i),
  });
}
