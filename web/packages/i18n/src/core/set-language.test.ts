/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeAll, describe, expect, it, vi } from "vitest";
import { LANGUAGE_STORAGE_KEY } from "../constants/language";

// setLanguage against a real i18next instance and its real resources, in a page that is a stub: the stored language
// and <html lang>. i18next applies only the language asked for last; the page must end up with the one it applied,
// whichever call's continuation runs last. The order is forced by holding the reads of one language's resources until
// the test releases them.

const stored = new Map<string, string>();
const html = { lang: "" };
vi.stubGlobal("window", {});
vi.stubGlobal("localStorage", {
  getItem: (key: string) => stored.get(key) ?? null,
  setItem: (key: string, value: string) => void stored.set(key, value),
});
vi.stubGlobal("document", { documentElement: html });

type Read = (language: string, namespace: string, callback: (error: unknown, data?: unknown) => void) => void;

/** A page with a fresh i18next instance, which has loaded English's resources, and nothing stored. */
async function load() {
  vi.resetModules();
  stored.clear();
  html.lang = "";
  const { i18nInstance, initPromise } = await import("./instance");
  const { setLanguage } = await import("./set-language");
  await initPromise;
  /** The page's language: i18next's, the stored one and <html lang>. */
  const page = () => ({ applied: i18nInstance.language, stored: stored.get(LANGUAGE_STORAGE_KEY), html: html.lang });
  /** Holds every read of language's resources from now on until release() is called. */
  const holdReads = (language: string) => {
    const backend = (i18nInstance.services.backendConnector as { backend: { read: Read } }).backend;
    const read = backend.read;
    let release!: () => void;
    const released = new Promise<void>((resolve) => {
      release = resolve;
    });
    let held = 0;
    backend.read = (lng, namespace, callback) => {
      if (lng !== language) return read.call(backend, lng, namespace, callback);
      held++;
      void released.then(() => read.call(backend, lng, namespace, callback));
    };
    return { held: () => held, release };
  };
  return { setLanguage, page, holdReads };
}

// The first import compiles the modules, with a deadline of its own rather than a test's; each load() then evaluates
// them afresh.
beforeAll(async () => {
  await load();
}, 10_000);

describe("setLanguage", () => {
  it("writes the language it applied", async () => {
    const { setLanguage, page } = await load();
    expect(page()).toEqual({ applied: "en", stored: undefined, html: "" });

    await setLanguage("zh-CN");

    expect(page()).toEqual({ applied: "zh-CN", stored: "zh-CN", html: "zh-CN" });
  });

  it("leaves the page at the language i18next applied when an earlier call's resources load last", async () => {
    const { setLanguage, page, holdReads } = await load();
    const gate = holdReads("zh-CN");

    // To Chinese, whose resources i18next reads first; then back to English, whose it has.
    const toChinese = setLanguage("zh-CN");
    await vi.waitFor(() => expect(gate.held()).toBeGreaterThan(0), { timeout: 2_000 });
    await setLanguage("en");
    expect(page()).toEqual({ applied: "en", stored: "en", html: "en" });

    // Chinese's resources arrive: i18next keeps English, asked for last, and so does the page.
    gate.release();
    await toChinese;
    expect(page()).toEqual({ applied: "en", stored: "en", html: "en" });
  });
});
