/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { SessionChangedError } from "@/lib/auth/token-manager";
/** A small per-editor scheduler. Aborted work releases its slot and never starts from the queue. */
export class ImageQueue {
  private active = 0;
  private waiting: (() => void)[] = [];
  run<T>(signal: AbortSignal, work: () => Promise<T>): Promise<T> {
    return new Promise((resolve, reject) => {
      let started = false;
      let ended = false;
      const finish = (error?: unknown, value?: T) => {
        if (ended) return;
        ended = true;
        signal.removeEventListener("abort", abort);
        this.waiting = this.waiting.filter((item) => item !== start);
        if (started) this.active--;
        if (error !== undefined) reject(error);
        else resolve(value!);
        this.drain();
      };
      const abort = () => finish(new SessionChangedError());
      const start = () => {
        if (signal.aborted) return abort();
        started = true;
        this.active++;
        void Promise.resolve()
          .then(() => {
            if (signal.aborted) throw new SessionChangedError();
            return work();
          })
          .then(
            (value) => finish(undefined, value),
            (error: unknown) => finish(error)
          );
      };
      signal.addEventListener("abort", abort, { once: true });
      if (signal.aborted) return abort();
      this.waiting.push(start);
      this.drain();
    });
  }
  private drain() {
    while (this.active < 2 && this.waiting.length) this.waiting.shift()!();
  }
}
