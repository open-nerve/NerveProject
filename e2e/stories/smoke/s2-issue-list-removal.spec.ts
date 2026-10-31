import type { Page, Request } from "@playwright/test";
import { writeFile } from "node:fs/promises";
import { createProject, createWorkspace, slugFor } from "../../fixtures/api";
import { emailFor } from "../../fixtures/auth";
import { expectQuietConsole, requestsElsewhere, watchPage } from "../../fixtures/browser";
import { APP, PROJECT, PROJECT_MEMBER, WORKSPACE, valued } from "../../fixtures/mounts";
import { registerOnboarded } from "../../fixtures/settings-pages";
import { expect, test } from "../../fixtures/test";

const missing = "0199f1c2-7a1b-7c3d-8e4f-5a6b7c8d9e0f";
const entries = [
  { name: "cycle", path: `cycles/${missing}`, domain: `cycles/${missing}/`, empty: "Cycle does not exist" },
  { name: "module", path: `modules/${missing}`, domain: `modules/${missing}/`, empty: "Module does not exist" },
  { name: "project view", path: `views/${missing}`, domain: `views/${missing}/`, empty: "View does not exist" },
  { name: "workspace view", path: "workspace-views/all-issues", domain: "views/" },
  { name: "archived cycles", path: "archives/cycles", domain: "archived-cycles/", tab: "Cycles" },
  { name: "archived modules", path: "archives/modules", domain: "archived-modules/", tab: "Modules" },
] as const;

/** A separate observation of the original events: watchPage's arrays are never filtered or reset. */
function observe(page: Page) {
  const requests: {
    method: string;
    url: string;
    status?: number;
    ended: boolean;
    failure?: string;
  }[] = [];
  const byRequest = new Map<Request, (typeof requests)[number]>();
  const console: {
    type: string;
    text: string;
    location: ReturnType<import("@playwright/test").ConsoleMessage["location"]>;
  }[] = [];
  page.on("request", (request) => {
    const record = { method: request.method(), url: request.url(), ended: false };
    requests.push(record);
    byRequest.set(request, record);
  });
  page.on("response", (response) => {
    const record = byRequest.get(response.request());
    if (record) record.status = response.status();
  });
  page.on("requestfinished", (request) => {
    const record = byRequest.get(request);
    if (record) record.ended = true;
  });
  page.on("requestfailed", (request) => {
    const record = byRequest.get(request);
    if (record) {
      record.ended = true;
      record.failure = request.failure()?.errorText;
    }
  });
  page.on("console", (message) => {
    console.push({ type: message.type(), text: message.text(), location: message.location() });
  });
  const api = () => requests.filter((request) => new URL(request.url).pathname.startsWith("/api/"));
  return { requests, console, api };
}

for (const entry of entries) {
  test(`S2 removal: ${entry.name} mounts without work item lists or their actions`, async ({
    api,
    signedInPage,
  }, testInfo) => {
    const tokens = await registerOnboarded(api, emailFor(testInfo));
    const slug = slugFor(testInfo);
    await createWorkspace(api, tokens.access_token, { name: "Removal workspace", slug });
    // The creator is a real project member; an outer project guard is not positive page coverage.
    const project = await createProject(api, tokens.access_token, slug, {
      name: "Removal project",
      identifier: "REM",
      network: 2,
    });
    const page = await signedInPage(tokens);
    const watch = await watchPage(page);
    const observed = observe(page);
    let beforeProbe: unknown;
    const workspaceView = entry.name === "workspace view";
    const detail = "empty" in entry;
    const path = workspaceView ? `/${slug}/${entry.path}` : `/${slug}/projects/${project.id}/${entry.path}`;
    const domain = workspaceView
      ? `/api/workspaces/${slug}/${entry.domain}`
      : `/api/workspaces/${slug}/projects/${project.id}/${entry.domain}`;
    const expected = [...APP, ...WORKSPACE, ...(detail ? [...PROJECT, ...PROJECT_MEMBER] : [])].map((request) =>
      valued(request, { [slug]: "{slug}", [project.id]: "{project}" })
    );
    expected.push(`GET ${domain}`);
    const whole = () => ({
      pending: observed.api().filter((request) => !request.ended).length,
      requests: observed
        .api()
        .map(({ method, url }) => {
          const { pathname, search } = new URL(url);
          return `${method} ${pathname}${search}`;
        })
        .toSorted(),
    });
    try {
      await page.goto(path);
      await expect(page).toHaveURL(path);
      if (detail) await expect(page.getByText(entry.empty, { exact: true })).toBeVisible();
      else if (workspaceView) {
        // The title comes from the actual child page, whereas the switcher belongs to its parent layout.
        await expect(page).toHaveTitle("Removal workspace - All Views");
        await expect(page.getByRole("button", { name: "All work items" })).toBeVisible();
      } else {
        await expect(
          page
            .locator(`a[href="/${slug}/projects/${project.id}/archives/issues"]`)
            .getByText("Archives", { exact: true })
        ).toBeVisible();
        // Newly created projects disable M6 feature navigation; the direct route still renders its own breadcrumb
        // and directory header. ArchiveTabsList only includes the enabled feature tabs.
        await expect(page.getByText(entry.tab, { exact: true })).toBeVisible();
        await expect(page.getByRole("textbox", { name: "Search", exact: true })).toBeAttached();
      }
      await expect(page.getByRole("button", { name: "Join project", exact: true })).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: /^(Add work item|Add existing work item|Display|Transfer issues)$/i })
      ).toHaveCount(0);
      // An exact complete set excludes generic issues/issues-detail/v2/issues, single workspace-view reads,
      // removed filters and writes too; no old API prefix or class of 4xx is broadly exempted.
      await expect.poll(whole).toEqual({ pending: 0, requests: expected.toSorted() });
      if (!detail && !workspaceView) {
        // M6's failed archive read leaves fetchedMap unset: record the real directory loader, not an empty success.
        await expect(page.getByRole("main").last().locator(".animate-pulse")).toHaveCount(1);
      }
      const origin = new URL(page.url()).origin;
      expect(observed.api().map(({ url }) => new URL(url).origin)).toEqual(expected.map(() => origin));
      const own = observed.api().filter(({ url }) => url === `${origin}${domain}`);
      expect(own).toEqual([{ method: "GET", url: `${origin}${domain}`, status: 404, ended: true }]);
      expect(observed.api().filter(({ status }) => status === undefined || status >= 400)).toEqual(own);
      expect(watch.apiRequests.toSorted()).toEqual(expected.map((request) => request.split("?")[0]).toSorted());
      expect(watch.apiFailures).toEqual([`404 GET ${domain}`]);
      expect(watch.oldApiRequests).toEqual([`GET ${domain}`]);
      expect([watch.pageErrors, watch.cspViolations, requestsElsewhere(page, watch)]).toEqual([[], [], []]);
      const logged = "Failed to load resource: the server responded with a status of 404 (Not Found)";
      // Attribute each resource error to this exact response URL, not just its generic browser text.
      expect(
        observed.console
          .filter(({ type }) => type === "error")
          .map(({ text, location }) => ({ text, url: location.url }))
      ).toEqual([{ text: logged, url: `${origin}${domain}` }]);
      beforeProbe = structuredClone(watch);
      await expectQuietConsole(page, watch, { errors: [logged] });
      expect(whole()).toEqual({ pending: 0, requests: expected.toSorted() });
    } finally {
      const file = testInfo.outputPath("removal-page-observations.json");
      const mounted: { url: string; title?: string; content?: string; diagnosticError?: string } = { url: page.url() };
      const observations = {
        entry,
        path,
        domain,
        expected,
        beforeProbe,
        watch,
        mounted,
        observed: { requests: observed.requests, console: observed.console },
      };
      // Preserve the original events before optional browser diagnostics can fail or time out.
      await writeFile(file, JSON.stringify(observations, null, 2));
      let diagnosticTimeout: ReturnType<typeof setTimeout> | undefined;
      try {
        const [title, content] = await Promise.race([
          Promise.all([page.title(), page.getByRole("main").last().innerHTML({ timeout: 1_000 })]),
          new Promise<never>((_resolve, reject) => {
            diagnosticTimeout = setTimeout(() => reject(new Error("Optional page diagnostics exceeded 1000ms")), 1_000);
          }),
        ]);
        Object.assign(mounted, { title, content });
      } catch (error) {
        mounted.diagnosticError = error instanceof Error ? `${error.name}: ${error.message}` : String(error);
      } finally {
        clearTimeout(diagnosticTimeout);
      }
      await writeFile(file, JSON.stringify(observations, null, 2));
      await testInfo.attach("removal-page-observations", {
        path: file,
        contentType: "application/json",
      });
    }
  });
}
