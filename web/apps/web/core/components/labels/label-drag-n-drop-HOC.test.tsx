/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { beforeEach, describe, expect, it, vi } from "vitest";
import { labelOf, projectOf } from "@/store/project/fake-projects";
import { LabelDndHOC } from "./label-drag-n-drop-HOC";

// Which grip a label's drag is registered with, as its row renders it (M3 design 3.16, P8b's P7): a row rendered
// anew, once its edit closes, has a new grip, which the drag must take. The app's vitest runs in node, with no DOM, so
// the test models React's hooks itself and runs the HOC's function as React would: useState keeps its value across
// renders, its setter the same function at each; useRef keeps its object; useEffect runs after a commit, only when its
// dependencies changed (Object.is), its last cleanup first; a commit sets the row's ref, and calls the grip's ref when
// the grip or that ref changed; a state the commit changed renders again. The drag adapter's stand-in keeps the grips
// registered.

const hooks = vi.hoisted(() => {
  const queued: (() => void)[] = [];
  return {
    at: 0,
    changed: false,
    states: new Map<number, { value: unknown; set: (next: unknown) => void }>(),
    refs: new Map<number, { current: unknown }>(),
    effects: new Map<number, { deps: readonly unknown[]; cleanup: (() => void) | void }>(),
    queued,
  };
});
vi.mock("react", async (importOriginal) => {
  const react = await importOriginal<typeof import("react")>();
  return {
    ...react,
    useState: (initial: unknown) => {
      const at = hooks.at++;
      const state = hooks.states.get(at) ?? {
        value: initial,
        set: (next: unknown) => {
          if (Object.is(next, state.value)) return;
          state.value = next;
          hooks.changed = true;
        },
      };
      hooks.states.set(at, state);
      return [state.value, state.set];
    },
    useRef: (initial: unknown) => {
      const at = hooks.at++;
      const ref = hooks.refs.get(at) ?? { current: initial };
      hooks.refs.set(at, ref);
      return ref;
    },
    useEffect: (effect: () => (() => void) | void, deps: readonly unknown[]) => {
      const at = hooks.at++;
      const last = hooks.effects.get(at);
      if (last && last.deps.length === deps.length && last.deps.every((dep, i) => Object.is(dep, deps[i]))) return;
      hooks.queued.push(() => {
        hooks.effects.get(at)?.cleanup?.();
        hooks.effects.set(at, { deps, cleanup: effect() });
      });
    },
  };
});
const dnd = vi.hoisted(() => ({ grips: new Set<unknown>() }));
vi.mock("@atlaskit/pragmatic-drag-and-drop/element/adapter", () => ({
  draggable: ({ dragHandle }: { dragHandle?: unknown }) => {
    dnd.grips.add(dragHandle);
    return () => dnd.grips.delete(dragHandle);
  },
  dropTargetForElements: () => () => undefined,
}));
vi.mock("mobx-react", () => ({ observer: <C,>(component: C) => component }));
vi.mock("@nerve/ui", () => ({ DropIndicator: () => null }));

const web = projectOf("WEB", "w-acme");
const bug = labelOf(web, "Bug", 10000);
// stand-ins of the elements React would give the refs, which the HOC only hands on: each its own, by its name
const element = (name: string): HTMLButtonElement => Object.assign(Object.create(null), { name });
const row: HTMLDivElement = Object.create(null);
// one handler for every render, as a list that keeps its handler gives: the drag cannot count on a new one
const onDrop = () => undefined;

/** The grip the last commit gave the grip's ref, and that ref. */
let committed: { grip: HTMLButtonElement | null; ref?: (grip: HTMLButtonElement | null) => void } = { grip: null };

/** Shows the label's row with grip (none: the row renders none) as React would: renders, commits, renders again. */
function show(grip: HTMLButtonElement | null) {
  for (let renders = 0; renders < 5; renders++) {
    hooks.at = 0;
    hooks.changed = false;
    let gripRef: ((grip: HTMLButtonElement | null) => void) | undefined;
    const tree = LabelDndHOC({
      label: bug,
      isGroup: false,
      isChild: false,
      isLastChild: true,
      onDrop,
      children: (_dragging, _child, ref) => {
        gripRef = ref;
        return null;
      },
    });
    tree.props.ref.current = row;
    if (gripRef && (gripRef !== committed.ref || grip !== committed.grip)) {
      if (committed.ref && committed.ref !== gripRef) committed.ref(null);
      gripRef(grip);
      committed = { grip, ref: gripRef };
    }
    for (const run of hooks.queued.splice(0)) run();
    if (!hooks.changed) return;
  }
  throw new Error("the HOC renders without end");
}

beforeEach(() => {
  hooks.states.clear();
  hooks.refs.clear();
  hooks.effects.clear();
  hooks.queued.length = 0;
  dnd.grips.clear();
  committed = { grip: null };
});

describe("LabelDndHOC", () => {
  it("registers the label's drag with the grip its row renders now, and anew when the row renders another", () => {
    const first = element("first");
    show(first);
    expect([...dnd.grips]).toEqual([first]);
    const second = element("second");
    show(second);
    expect([...dnd.grips]).toEqual([second]);
  });

  it("registers no drag while its row renders no grip", () => {
    show(null);
    expect([...dnd.grips]).toEqual([]);
  });
});
