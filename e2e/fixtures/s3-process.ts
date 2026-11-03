import { execFile, fork, type ChildProcess } from "node:child_process";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";

/** Only a test-generated container name is accepted, never a shared Docker resource. */
export async function removeS3Container(name: string, timeout = 10_000): Promise<void> {
  if (!/^nerve-s3-e2e-[a-f0-9]{32}$/.test(name)) throw new Error("invalid owned container name");
  try {
    const { stdout } = await promisify(execFile)(
      "docker",
      ["inspect", "--format", '{{index .Config.Labels "nerve.test-owner"}}', name],
      { timeout, maxBuffer: 4096 }
    );
    if (stdout.trim() !== name) throw new Error("S3 fixture ownership differs");
  } catch (error) {
    if (/no such (object|container):/i.test(String((error as { stderr?: string }).stderr))) return;
    throw error;
  }
  await promisify(execFile)("docker", ["rm", "--force", "--volumes", name], { timeout, maxBuffer: 4096 });
}

export function startS3Process(name: string, auth: string) {
  const child = fork(fileURLToPath(new URL("./s3-container.ts", import.meta.url)), [], {
    execArgv: [],
    stdio: ["ignore", "ignore", "ignore", "ipc"],
  });
  const exited = new Promise<void>((resolve, reject) => {
    child.once("error", reject);
    child.once("exit", () => resolve());
  });
  // Always observe an early spawn error, including while waiting on readiness.
  void exited.catch(() => undefined);
  const ready = new Promise<{ endpoint: string; id: string }>((resolve, reject) => {
    child.once("error", reject);
    child.once("exit", () => reject(new Error("S3 child exited before readiness")));
    child.once("message", (message: { endpoint?: string; id?: string; error?: string }) => {
      if (!message.endpoint || !message.id || message.error) reject(new Error("S3 child startup failed"));
      else resolve({ endpoint: message.endpoint, id: message.id });
    });
    child.send({ action: "start", name, auth });
  });
  let closing: Promise<void> | undefined;
  return {
    ready,
    close: () => (closing ??= closeProcess(child, exited, name)),
  };
}

async function closeProcess(child: ChildProcess, exited: Promise<void>, name: string): Promise<void> {
  let timer: NodeJS.Timeout | undefined;
  try {
    if (child.connected) child.send({ action: "close" }, () => undefined);
    timer = setTimeout(() => child.kill("SIGKILL"), 5000);
    await exited;
  } finally {
    clearTimeout(timer);
    // Joining the child first prevents a late container creation after cleanup.
    await removeS3Container(name);
  }
}

const binary = fileURLToPath(new URL("../../bin/nerve-s3-fixture", import.meta.url));
export async function s3Command<T>(request: object, signal: AbortSignal, timeout: number): Promise<T> {
  const input = JSON.stringify(request);
  if (Buffer.byteLength(input) > 1_048_576) throw new Error("S3 fixture input exceeds its envelope");
  const child = execFile(binary, [], { signal, timeout, killSignal: "SIGKILL", maxBuffer: 1_048_576 });
  const result = new Promise<string>((resolve, reject) => {
    let stdout = "";
    let failure: Error | undefined;
    child.stdout?.setEncoding("utf8").on("data", (data: string) => {
      stdout += data;
    });
    child.once("error", (error) => {
      failure = error;
    });
    child.once("close", (code) =>
      code === 0 && !failure ? resolve(stdout) : reject(new Error("S3 fixture command failed"))
    );
  });
  child.stdin?.on("error", () => undefined);
  child.stdin?.end(input);
  // execFile owns maxBuffer/timeout; close means all command pipes are closed.
  return JSON.parse(await result) as T;
}
