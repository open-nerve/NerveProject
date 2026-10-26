import { createApi, type Api } from "./api";
import { createDatabase, openDatabase, templateDatabase, type Database } from "./db";
import { startNerve, type Nerve } from "./server";

interface ClockRun {
  nerve: Nerve;
  api: Api;
  /** Real time recorded before starting this generation's process. */
  startedAt: Date;
}

export interface ClockWorld {
  db: Database;
  current: ClockRun;
  advance(offset: string): Promise<ClockRun>;
  close(): Promise<void>;
}

/** A database and a sequence of nerve processes owned by one clock test. */
export async function createClockWorld(options: {
  databaseName: string;
  logFile: (generation: number) => string;
}): Promise<ClockWorld> {
  await createDatabase(options.databaseName, templateDatabase);
  const db = openDatabase(options.databaseName);
  let generation = 0;
  let current: ClockRun | undefined;
  let closed = false;

  async function start(offset?: string): Promise<ClockRun> {
    const startedAt = new Date();
    const nerve = await startNerve(
      db.url,
      options.logFile(generation++),
      offset === undefined ? {} : { NERVE_CLOCK__OFFSET: offset }
    );
    return { nerve, api: createApi(nerve.baseURL), startedAt };
  }

  async function close(): Promise<void> {
    if (closed) return;
    closed = true;
    const active = current;
    current = undefined;
    try {
      await active?.nerve.stop();
    } finally {
      await db.close();
    }
  }

  async function fail(error: unknown): Promise<never> {
    const errors: unknown[] = [error];
    // A failed start/advance keeps its own database available for a dump,
    // even when the child was killed by startNerve before it became ready.
    try {
      await db.dump(`${options.logFile(generation - 1)}.sql`);
    } catch (dumpError) {
      errors.push(dumpError);
    }
    try {
      await close();
    } catch (closeError) {
      errors.push(closeError);
    }
    throw errors.length === 1 ? error : new AggregateError(errors, "clock world failed and cleanup also failed");
  }

  try {
    current = await start();
  } catch (error) {
    return fail(error);
  }

  return {
    db,
    get current() {
      if (!current) throw new Error("clock world is closed");
      return current;
    },
    async advance(offset) {
      if (closed || !current) throw new Error("clock world is closed");
      try {
        await current.nerve.stop();
        current = undefined;
        current = await start(offset);
        return current;
      } catch (error) {
        return fail(error);
      }
    },
    close,
  };
}
