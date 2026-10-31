/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import { useCallback, useEffect, useState } from "react";
import type { IFilterOption, TSingleSelectFilterFieldConfig, TMultiSelectFilterFieldConfig } from "@nerve/types";

type Config = TSingleSelectFilterFieldConfig<string> | TMultiSelectFilterFieldConfig<string>;
type Result = {
  config: Config;
  read: Config["getOptions"];
  attempt: number;
  options: IFilterOption<string>[];
  error: unknown;
};

/** Owns only this input's option read, never a domain directory or permission decision. */
export function useFilterSelectOptions(config: Config) {
  const read = config.getOptions;
  const [attempt, setAttempt] = useState(0);
  const [result, setResult] = useState<Result>();
  const retry = useCallback(() => setAttempt((value) => value + 1), []);
  useEffect(() => {
    let current = true;
    const owner = { config, read, attempt };
    void Promise.resolve()
      .then(() => (typeof read === "function" ? read() : read))
      .then(
        (options) => {
          if (current) setResult({ ...owner, options, error: undefined });
          return undefined;
        },
        (error: unknown) => {
          if (current) setResult({ ...owner, options: [], error: error ?? new Error("Options unavailable") });
        }
      );
    return () => {
      current = false;
    };
  }, [config, read, attempt]);
  const owned = result?.config === config && result.read === read && result.attempt === attempt;
  return {
    options: owned ? result.options : [],
    loading: !owned,
    error: owned ? result.error : undefined,
    retry,
  };
}
