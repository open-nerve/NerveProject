// @vitest-environment jsdom
/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { act, useEffect } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { useNotificationLanding, type NotificationLanding } from "./landing";
import type { NotificationSelection } from "./page-state";
import { NotificationPageStateProvider, useNotificationPageState } from "./page-state";

// Where the notifications page's focus lands as its preview leaves the notification selected (P5b spec 2.5, R26; re-95
// I3, O7): the selection cleared, a microtask after the commit that clears it (after Headless UI gives the focus back
// to what had it as a dialog unmounts), on the main button of the notification's card, its first; the card gone (the
// tab or the filters changed meanwhile), on the list's region; the list not drawn (its read failed, or is out), on the
// preview's region; never on the page's body. A leave of a notification no longer selected does nothing. The page is a
// stand-in here: the list's region (data-notification-list) and its cards (data-notification-id), the preview's region,
// and in it, the notification selected, whose control has the focus as it leaves, with a dialog whose unmount gives the
// focus back to a button outside, as Headless UI does.

type Shown = "cards" | "no cards" | "no list";
const held = {
  landing: undefined as NotificationLanding | undefined,
  state: undefined as ReturnType<typeof useNotificationPageState> | undefined,
  shown: "cards" as Shown,
};
globalThis.IS_REACT_ACT_ENVIRONMENT = true;

const one: NotificationSelection = { notificationId: "n-one", issueId: "i-web-1", projectId: "p-web" };
const two: NotificationSelection = { notificationId: "n-two", issueId: "i-web-2", projectId: "p-web" };

/** A dialog of the selected notification's: as it unmounts, a microtask later, it gives the focus back to its opener. */
function Dialog() {
  useEffect(
    () => () => queueMicrotask(() => document.querySelector<HTMLElement>('[aria-label="opener"]')?.focus()),
    []
  );
  return null;
}

function Page() {
  const landing = useNotificationLanding();
  const state = useNotificationPageState();
  held.landing = landing;
  held.state = state;
  const { selected } = state;
  return (
    <>
      <button type="button" aria-label="opener" />
      {held.shown !== "no list" && (
        <div data-notification-list tabIndex={-1} aria-label="list">
          {held.shown === "cards" && (
            <ul>
              {[one, two].map(({ notificationId }) => (
                <li key={notificationId} data-notification-id={notificationId}>
                  <button type="button" aria-label={`${notificationId} main`} />
                  <button type="button" aria-label={`${notificationId} menu`} />
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
      <section ref={landing.preview} tabIndex={-1} aria-label="preview">
        {selected && (
          <>
            <button type="button" aria-label="close" />
            <Dialog />
          </>
        )}
      </section>
    </>
  );
}

let host: HTMLDivElement, root: Root;
beforeEach(() => {
  held.shown = "cards";
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(async () => {
  await act(async () => root.unmount());
  host.remove();
});

/** Draws the page with the notification selected, its close having the focus. */
async function selectedWith(selection: NotificationSelection) {
  await act(async () =>
    root.render(
      <NotificationPageStateProvider>
        <Page />
      </NotificationPageStateProvider>
    )
  );
  await act(async () => held.state!.select(selection));
  host.querySelector<HTMLElement>('[aria-label="close"]')!.focus();
}
/** The preview leaves the notification, and the page settles: its commit, then its microtasks. */
async function leave(notificationId: string) {
  await act(async () => held.landing!.leave(notificationId));
  await act(async () => {
    await Promise.resolve();
  });
}
/** What has the focus: the page's body, or an element's name. */
const focused = () =>
  document.activeElement === document.body ? "body" : document.activeElement?.getAttribute("aria-label");

describe("useNotificationLanding (R26)", () => {
  it("clears the selection, then lands on the main button of the notification's card, after the dialog gives it back", async () => {
    await selectedWith(one);
    await leave("n-one");
    expect(held.state?.selected).toBeUndefined();
    expect(focused()).toBe("n-one main");
  });

  it("lands on the list's region once the card is gone (another tab, other filters)", async () => {
    held.shown = "no cards";
    await selectedWith(one);
    await leave("n-one");
    expect(held.state?.selected).toBeUndefined();
    expect(focused()).toBe("list");
  });

  it("lands on the preview's region once the list is not drawn (its read failed, or is out): never on the body", async () => {
    held.shown = "no list";
    await selectedWith(one);
    await leave("n-one");
    expect(held.state?.selected).toBeUndefined();
    expect(focused()).toBe("preview");
  });

  it("does nothing for a notification no longer selected: the selection and the focus stay", async () => {
    await selectedWith(two);
    await leave("n-one");
    expect(held.state?.selected).toEqual(two);
    expect(focused()).toBe("close");
  });
});
