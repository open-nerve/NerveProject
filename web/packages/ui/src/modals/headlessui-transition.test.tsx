// @vitest-environment jsdom
/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */
import { createRequire } from "node:module";
import { Transition } from "@headlessui/react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const require = createRequire(import.meta.url);
const entry = require.resolve("@headlessui/react");
const development = require(entry) as { Transition: typeof Transition };
delete require.cache[entry];
vi.stubEnv("NODE_ENV", "production");
const production = require(entry) as { Transition: typeof Transition };
vi.unstubAllEnvs();
delete require.cache[entry];

// jsdom cannot run CSS transitions. Only the browser animation and frame APIs are controlled;
// each entry renders Headless UI's real Transition, including its cancellation and nesting effects.
class ControlledTransition {
  playState: AnimationPlayState = "running";
  finished: Promise<void>;
  private resolve!: () => void;
  private reject!: (reason: Error) => void;

  constructor() {
    this.finished = new Promise<void>((resolve, reject) => {
      this.resolve = resolve;
      this.reject = reject;
    });
  }

  finish() {
    this.playState = "finished";
    this.resolve();
  }

  cancel() {
    this.playState = "idle";
    this.reject(new Error("CSS transition replaced"));
  }
}

let root: Root | undefined;
let container: HTMLElement;
let animations: ControlledTransition[];
let frames: Map<number, FrameRequestCallback>;
let nextFrame: number;
let completedLeaves: string[];

async function frame() {
  await act(async () => {
    const callbacks = [...frames.values()];
    frames.clear();
    for (const callback of callbacks) callback(performance.now());
  });
}

async function advance() {
  // Headless UI prepares, runs on the next double frame, then captures on one further frame.
  await frame();
  await frame();
  await frame();
  await frame();
}

async function settle(action: () => void) {
  await act(async () => action());
}

beforeEach(() => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = true;
  vi.stubGlobal("CSSTransition", ControlledTransition);
  frames = new Map();
  nextFrame = 0;
  animations = [];
  completedLeaves = [];
  vi.stubGlobal("requestAnimationFrame", (callback: FrameRequestCallback) => {
    frames.set(++nextFrame, callback);
    return nextFrame;
  });
  vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id));
  vi.spyOn(Element.prototype, "getAnimations").mockImplementation(() => animations as unknown as Animation[]);
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
});

afterEach(async () => {
  await act(async () => root?.unmount());
  container.remove();
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

for (const [name, Component] of [
  ["ESM import", Transition],
  ["CJS development require", development.Transition],
  ["CJS production require", production.Transition],
] as const) {
  describe(name, () => {
    async function render(show: boolean) {
      await act(async () => {
        root?.render(
          <Component show={show} enter="enter" leave="leave" afterLeave={() => completedLeaves.push("closed")}>
            <div data-testid="panel">Workspace menu</div>
          </Component>
        );
      });
    }

    async function reverse() {
      await render(false);
      const entering = new ControlledTransition();
      animations = [entering];
      await render(true);
      await advance();
      expect(container.querySelector('[data-testid="panel"]')).not.toBeNull();
      await render(false);
      // The closing generation captures A before the browser starts its replacement B.
      await advance();
      const leaving = new ControlledTransition();
      animations = [leaving];
      await settle(() => entering.cancel());
      return leaving;
    }

    it("waits for the replacement transition, then unmounts even if finished animations remain listed", async () => {
      const leaving = await reverse();
      expect(container.querySelector('[data-testid="panel"]')).not.toBeNull();
      await settle(() => leaving.finish());
      expect(container.querySelector('[data-testid="panel"]')).toBeNull();
    });

    it("keeps a reopened panel when an older generation finishes, then closes the new generation", async () => {
      const leaving = await reverse();
      const reopened = new ControlledTransition();
      animations = [reopened];
      await render(true);
      await advance();
      await settle(() => leaving.finish());
      expect(container.querySelector('[data-testid="panel"]')).not.toBeNull();
      await settle(() => reopened.finish());
      expect(completedLeaves).toEqual([]);
      expect(container.querySelector('[data-testid="panel"]')).not.toBeNull();
      await render(false);
      const closing = new ControlledTransition();
      animations = [closing];
      await advance();
      expect(container.querySelector('[data-testid="panel"]')).not.toBeNull();
      await settle(() => closing.finish());
      expect(container.querySelector('[data-testid="panel"]')).toBeNull();
    });

    it("cancels completion after unmount without touching a later panel", async () => {
      const leaving = await reverse();
      await act(async () => root?.unmount());
      root = createRoot(container);
      animations = [];
      await render(true);
      await advance();
      await settle(() => leaving.finish());
      expect(completedLeaves).toEqual([]);
      expect(container.querySelector('[data-testid="panel"]')).not.toBeNull();
      await render(false);
      await advance();
      expect(container.querySelector('[data-testid="panel"]')).toBeNull();
    });
  });
}
