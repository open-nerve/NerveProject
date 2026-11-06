// A separate process lets the owner bound even image pull / Docker startup.
// The parent kills and joins this child before removing its exact owned name.
import { GenericContainer, Wait, type StartedTestContainer } from "testcontainers";

const image = "chrislusf/seaweedfs@sha256:57877e32bb90073b37dc109b438ba66e84fd5ed9ced5f5c04f56444ef243ce59";
class LoopbackContainer extends GenericContainer {
  constructor() {
    super(image);
    this.withExposedPorts(8333);
    this.hostConfig.PortBindings = { "8333/tcp": [{ HostIp: "127.0.0.1", HostPort: "0" }] };
  }
}
let container: StartedTestContainer | undefined;
let closing: Promise<void> | undefined;
function close(): Promise<void> {
  return (closing ??= (async () => {
    if (container) await container.stop({ timeout: 2000 });
    process.disconnect();
  })());
}
process.on("message", async (message: { action: string; name?: string; auth?: string }) => {
  try {
    if (message.action === "close") {
      await close();
      return;
    }
    if (message.action !== "start" || !message.name || !message.auth || container) throw new Error("invalid startup");
    container = await new LoopbackContainer()
      .withName(message.name)
      .withLabels({ "nerve.test-owner": message.name })
      .withTmpFs({ "/data": "rw,size=2g" })
      .withCopyContentToContainer([{ content: message.auth, target: "/etc/nerve-s3.json", mode: 0o600 }])
      .withEntrypoint([
        "/bin/sh",
        "-c",
        'chown seaweed:seaweed /etc/nerve-s3.json && exec /entrypoint.sh "$@"',
        "nerve-s3-test",
      ])
      .withCommand([
        "server",
        "-dir=/data",
        "-ip=127.0.0.1",
        "-ip.bind=0.0.0.0",
        "-s3",
        "-s3.config=/etc/nerve-s3.json",
        "-s3.iam=false",
        "-s3.port.iceberg=0",
        "-s3.port.lance=0",
        // The run's one service holds a bucket per live world. SeaweedFS gives every bucket that holds data about
        // seven volumes and frees them when the bucket is removed: 256 is room for some thirty-six at once.
        "-volume.max=256",
        "-master.volumeSizeLimitMB=512",
        "-master.telemetry=false",
      ])
      .withWaitStrategy(Wait.forHttp("/", 8333).forStatusCodeMatching((code) => code === 400 || code === 403))
      .withStartupTimeout(90_000)
      .start();
    process.send?.({ endpoint: `http://127.0.0.1:${container.getMappedPort(8333)}`, id: container.getId() });
  } catch (error) {
    // Error objects do not cross IPC: the parent rebuilds this child's own reason from its parts.
    const { name, message: reason, stack } = error instanceof Error ? error : new Error(String(error));
    process.send?.({ error: { name, message: reason, stack } });
    process.exitCode = 1;
    await close();
  }
});
// Parent loss must not keep an unowned helper running. Ryuk is the second guard.
process.on("disconnect", () => {
  process.exit(0);
});
