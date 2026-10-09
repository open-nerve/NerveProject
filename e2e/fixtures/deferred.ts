/** A promise and the function that resolves it: what a test holds back until it says so. */
export interface Deferred<T> {
  readonly promise: Promise<T>;
  readonly resolve: (value: T) => void;
}

/** What resolve is for the instant before the promise's executor runs. */
function unset(): void {}

/**
 * A promise the caller resolves when it chooses, as Promise.withResolvers gives one (the e2e package's lib is
 * ES2023, which has no withResolvers). The executor runs at once, so resolve is the promise's from the start.
 */
export function deferred<T = void>(): Deferred<T> {
  let resolve: (value: T) => void = unset;
  const promise = new Promise<T>((settle) => {
    resolve = settle;
  });
  return { promise, resolve };
}
