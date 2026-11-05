/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
/** Input only: never retain a prepared write, timer, request, owner DTO or authorization. */
export type AutosaveSnapshot<T> = {
  value: T;
  confirmed: T;
  revision: number;
  attempted: number;
  dirty: boolean;
  failure?: { error: unknown };
};
export interface AutosaveRetention<T> {
  read(): AutosaveSnapshot<T> | undefined;
  store(snapshot: AutosaveSnapshot<T>): void;
  readonly canFlushOnUnmount: boolean;
}
