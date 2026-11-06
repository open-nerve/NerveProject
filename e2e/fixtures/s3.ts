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

export async function startS3({ remove = removeS3Container, auth }: S3Options = {}): Promise<S3Service> {
  const name = `nerve-s3-e2e-${id()}`;
  const access = id(),
    secret = id() + id();
  const deadline = Date.now() + 90_000;
  const process = startS3Process(
    name,
    auth ??
      JSON.stringify({
        identities: [
          {
            name: "nerve-test",
            credentials: [{ accessKey: access, secretKey: secret }],
            actions: ["Admin", "Read", "Write", "List", "Tagging"],
          },
        ],
      }),
    remove
  );
  const abort = new AbortController();
  const pending = new Set<Promise<unknown>>();
  let closing: Promise<void> | undefined;
  let timer: NodeJS.Timeout | undefined;
  const close = () =>
    (closing ??= (async () => {
      abort.abort();
      await Promise.allSettled(pending);
      await process.close();
    })());
  try {
    const ready = await Promise.race([
      process.ready,
      new Promise<never>((_, reject) => {
        timer = setTimeout(() => reject(new Error("S3 startup exceeded 90s")), Math.max(1, deadline - Date.now()));
      }),
    ]);
    const command = <T>(connection: Connection, action: string, fields: object = {}, budget = 12_000): Promise<T> => {
      if (closing) return Promise.reject(new Error("S3 service is closed"));
      const run = s3Command<T>({ ...connection, action, ...fields }, abort.signal, budget);
      pending.add(run);
      void run.finally(() => pending.delete(run)).catch(() => undefined);
      return run;
    };
    const createBucket = async (startupDeadline = Date.now() + 12_000): Promise<S3Bucket> => {
      const connection = {
        endpoint: ready.endpoint,
        region: "us-east-1",
        bucket: `nerve-e2e-${id()}`,
        prefix: `e2e/${id()}/`,
        access_key_id: access,
        secret_access_key: secret,
      };
      for (;;) {
        const left = startupDeadline - Date.now();
        if (left <= 0) throw new Error("S3 data plane did not become ready");
        try {
          // eslint-disable-next-line no-await-in-loop -- Probe one owned bucket until its bounded readiness deadline.
          await command(connection, "create-bucket", {}, Math.min(12_000, left));
          break;
        } catch (error) {
          if (abort.signal.aborted || Date.now() >= startupDeadline) throw error;
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
    // Transport readiness alone is insufficient: real create/PUT/GET/DELETE.
    await createBucket(deadline);
    return { name, containerId: ready.id, endpoint: ready.endpoint, createBucket: () => createBucket(), close };
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
