import { writeFile } from "node:fs/promises";
import { test, type Page } from "@playwright/test";
import { transitionsEnded } from "./settings-pages";

/** Opt-in observations after real actions. A screenshot is evidence to inspect, never an automatic visual verdict. */
export async function c12Evidence(page: Page, group: string, state: string) {
  if (!process.env.M4_CLOSEOUT_EVIDENCE) return;
  if (!/^B(0[1-9]|1[0-9]|2[01])$/.test(group)) throw new Error(`Unknown C12 group: ${group}`);
  // Let enter classes reach the rendering pipeline, then observe the real finite animations.
  // Do not disable animation: these records must show what the browser actually paints.
  await page.evaluate(
    () => new Promise<void>((resolve) => requestAnimationFrame(() => requestAnimationFrame(() => resolve())))
  );
  await transitionsEnded(page.locator("body"));
  const info = test.info();
  const name = `c12-${group}-${state.replace(/[^a-zA-Z0-9_-]/g, "-")}`;
  const screenshot = info.outputPath(`${name}.png`);
  const observations = await page.evaluate(() => {
    const viewport = { width: window.innerWidth, height: window.innerHeight };
    const describe = (element: Element) => {
      const box = element.getBoundingClientRect();
      const style = getComputedStyle(element);
      return {
        tag: element.tagName,
        role: element.getAttribute("role"),
        name: element.getAttribute("aria-label"),
        text: element.textContent?.trim().slice(0, 240),
        box: { x: box.x, y: box.y, width: box.width, height: box.height },
        withinViewport: box.left >= 0 && box.top >= 0 && box.right <= viewport.width && box.bottom <= viewport.height,
        outline: { style: style.outlineStyle, width: style.outlineWidth, color: style.outlineColor },
        inert: !!element.closest("[inert]"),
        hidden: !!element.closest('[aria-hidden="true"]'),
      };
    };
    return {
      origin: window.location.origin,
      pathname: window.location.pathname,
      secureContext: window.isSecureContext,
      viewport,
      focus: document.activeElement ? describe(document.activeElement) : null,
      panels: Array.from(
        document.querySelectorAll('[role="dialog"], [role="listbox"], [role="menu"], [data-popper-placement]')
      )
        .filter((element) => element.getClientRects().length && getComputedStyle(element).visibility !== "hidden")
        .map(describe),
    };
  });
  await page.screenshot({ path: screenshot });
  const record = info.outputPath(`${name}.json`);
  await writeFile(
    record,
    JSON.stringify(
      {
        group,
        state,
        title: info.title,
        testId: info.testId,
        project: info.project.name,
        browserVersion: page.context().browser()?.version(),
        observations,
        screenshot,
        visualReview: "pending",
      },
      null,
      2
    )
  );
  await info.attach(name, { path: screenshot, contentType: "image/png" });
  await info.attach(`${name}-observations`, { path: record, contentType: "application/json" });
}
