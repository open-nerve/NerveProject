import { randomBytes } from "node:crypto";
import { setTimeout as sleep } from "node:timers/promises";

import { removeS3Container, s3Command, startS3Process } from "./s3-process";

const id = () => randomBytes(16).toString("hex");
interface Connection {
  endpoint: string;
  region: string;
  bucket: string;
  prefix: string;
  access_key_id: string;
  secret_access_key: string;
}
interface S3Object {
  size: number;
  sha256: string;
  content_type: string;
  etag: string;
}
export interface S3Bucket {
  readonly connection: Connection;
  cors(origin: string): Promise<void>;
  inspect(key: string): Promise<S3Object>;
  delete(key: string): Promise<void>;
}
/** Makes buckets on a running service and removes the ones it made when closed. */
export interface S3Storage {
  /** The service's container name. */
  readonly name: string;
  readonly endpoint: string;
  createBucket(): Promise<S3Bucket>;
  /** Whether an owned bucket exists on the service now. */
  exists(bucket: string): Promise<boolean>;
  close(): Promise<void>;
}
/** Owns a service's container; closing it removes the container. */
export interface S3Service {
  readonly name: string;
  readonly containerId: string;
  readonly endpoint: string;
  createBucket(): Promise<S3Bucket>;
  close(): Promise<void>;
}

export interface S3Options {
  /** Removes the owned container once its child has exited. */
  remove?: (name: string) => Promise<void>;
  /** The SeaweedFS identity file; one the child refuses fails startup before any container exists. */
  auth?: string;
}

interface Credentials {
  access_key_id: string;
  secret_access_key: string;
}

/** Bounded fixture commands against one service; closing aborts and joins the ones in flight. */
function s3Client(endpoint: string, credentials: Credentials) {
  const abort = new AbortController();
  const pending = new Set<Promise<unknown>>();
  let closed = false;
  const command = <T>(connection: Connection, action: string, fields: object = {}, budget = 12_000): Promise<T> => {
    if (closed) return Promise.reject(new Error("S3 service is closed"));
    const run = s3Command<T>({ ...connection, action, ...fields }, abort.signal, budget);
    pending.add(run);
    void run.finally(() => pending.delete(run)).catch(() => undefined);
    return run;
  };
  const connectionOf = (bucket: string): Connection => ({
    endpoint,
    region: "us-east-1",
    bucket,
    prefix: `e2e/${id()}/`,
    ...credentials,
  });
  const createBucket = async (connection: Connection, deadline = Date.now() + 12_000): Promise<S3Bucket> => {
    for (;;) {
      const left = deadline - Date.now();
      if (left <= 0) throw new Error("S3 data plane did not become ready");
      try {
        // eslint-disable-next-line no-await-in-loop -- Probe one owned bucket until its bounded readiness deadline.
        await command(connection, "create-bucket", {}, Math.min(12_000, left));
        break;
      } catch (error) {
        if (abort.signal.aborted || Date.now() >= deadline) throw error;
        // eslint-disable-next-line no-await-in-loop -- Back off between attempts; never overlap SDK commands.
        await sleep(100, undefined, { signal: abort.signal });
      }
    }
    return {
      connection,
      cors: async (origin) => {
        await command(connection, "cors", { origin });
      },
      inspect: (key) => command<S3Object>(connection, "inspect", { key }),
      delete: async (key) => {
        await command(connection, "delete", { key });
      },
    };
  };
  return {
    newBucket: () => connectionOf(`nerve-e2e-${id()}`),
    createBucket,
    remove: async (bucket: string) => {
      await command(connectionOf(bucket), "delete-bucket");
    },
    exists: async (bucket: string) => (await command<{ exists: boolean }>(connectionOf(bucket), "head-bucket")).exists,
    close: async () => {
      closed = true;
      abort.abort();
      await Promise.allSettled(pending);
    },
  };
}

export async function startS3({ remove = removeS3Container, auth }: S3Options = {}): Promise<S3Service> {
  const name = `nerve-s3-e2e-${id()}`;
  const credentials = { access_key_id: id(), secret_access_key: id() + id() };
  const deadline = Date.now() + 90_000;
  const process = startS3Process(
    name,
    auth ??
      JSON.stringify({
        identities: [
          {
            name: "nerve-test",
            credentials: [{ accessKey: credentials.access_key_id, secretKey: credentials.secret_access_key }],
            actions: ["Admin", "Read", "Write", "List", "Tagging"],
          },
        ],
      }),
    remove
  );
  let client: ReturnType<typeof s3Client> | undefined;
  let closing: Promise<void> | undefined;
  let timer: NodeJS.Timeout | undefined;
  const close = () =>
    (closing ??= (async () => {
      await client?.close();
      await process.close();
    })());
  try {
    const ready = await Promise.race([
      process.ready,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error("S3 startup exceeded 90s")), Math.max(1, deadline - Date.now()));
      }),
    ]);
    client = s3Client(ready.endpoint, credentials);
    // Transport readiness alone is insufficient: real create/PUT/GET/DELETE. The probe bucket is removed so it
    // keeps none of the service's volumes.
    const probe = client.newBucket();
    await client.createBucket(probe, deadline);
    await client.remove(probe.bucket);
    const { createBucket, newBucket } = client;
    shared.set(name, { name, endpoint: ready.endpoint, ...credentials });
    return {
      name,
      containerId: ready.id,
      endpoint: ready.endpoint,
      createBucket: () => createBucket(newBucket()),
      close,
    };
  } catch (error) {
    // A cleanup that also fails must not hide why startup failed.
    const cleanup = await close().then(
      () => undefined,
      (reason: unknown) => ({ reason })
    );
    if (cleanup) {
      // oxlint-disable-next-line preserve-caught-error -- the caught error is the first of the aggregated errors.
      throw new AggregateError([error, cleanup.reason], `S3 startup and cleanup failed (${name})`);
    }
    throw new Error(`S3 startup failed (${name})`, { cause: error });
  } finally {
    clearTimeout(timer);
  }
}

interface SharedService extends Credentials {
  name: string;
  endpoint: string;
}
const sharedVariable = "NERVE_E2E_S3";
const shared = new Map<string, SharedService>();

/**
 * Makes a started service the run's: global-setup.ts calls this before any browser starts, and the test workers it
 * spawns inherit the variable. A container started or removed while pages load changes the Linux host's network
 * interfaces, and Chromium then fails every request in flight with net::ERR_NETWORK_CHANGED (P8 T14).
 */
export function shareS3(service: S3Service): void {
  const description = shared.get(service.name);
  if (!description) throw new Error("only a service this process started can be shared");
  process.env[sharedVariable] = JSON.stringify(description);
}

/** A client of the run's service: every bucket it makes is removed again when it closes. */
export function sharedS3(): S3Storage {
  const raw = process.env[sharedVariable];
  if (!raw) throw new Error(`${sharedVariable} is unset: global-setup.ts did not share an S3 service`);
  const { name, endpoint, ...credentials } = JSON.parse(raw) as SharedService;
  const client = s3Client(endpoint, credentials);
  const owned: string[] = [];
  const creating = new Set<Promise<unknown>>();
  let closing: Promise<void> | undefined;
  return {
    name,
    endpoint,
    createBucket: () => {
      if (closing) return Promise.reject(new Error("S3 storage is closed"));
      const connection = client.newBucket();
      // Owned before it is created: a creation that fails halfway is still removed.
      owned.push(connection.bucket);
      const created = client.createBucket(connection);
      creating.add(created);
      void created.finally(() => creating.delete(created)).catch(() => undefined);
      return created;
    },
    exists: (bucket) => client.exists(bucket),
    close: () =>
      (closing ??= (async () => {
        // A creation still running finishes first, so no bucket appears after its removal.
        await Promise.allSettled(creating);
        const removed = await Promise.allSettled(owned.map((bucket) => client.remove(bucket)));
        await client.close();
        const errors = removed.flatMap((result) => (result.status === "rejected" ? [result.reason] : []));
        if (errors.length) throw new AggregateError(errors, `S3 bucket cleanup failed (${name})`);
      })()),
  };
}
