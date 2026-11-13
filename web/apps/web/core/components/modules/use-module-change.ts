/**
 * Copyright (c) 2026-present OpenNerve
 * SPDX-License-Identifier: AGPL-3.0-only
 */

import { useLayoutEffect, useRef, useState } from "react";
// nerve imports
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
// hooks
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/**
 * A change of a module that one of its controls sends as the caller picks a value (the progress sidebar's status, days,
 * lead and members; P6a spec 2.6, R11): the control is busy while the change is out, so that no other is picked over
 * it. It shows nerve's value all along: only once nerve has made the change does it say so, the store then giving the
 * new value; refused, it says why, and nerve's value stays. It follows nerve's answer in the session it was sent in,
 * while the control is still there (followInSession): late, the answer reaches the store alone.
 */
export function useModuleChange() {
  // hooks
  const { t } = useTranslation();
  const toastRefusal = useRefusalToast();
  // states
  const [busy, setBusy] = useState(false);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  /** Sends change, and gives once it has settled whether the control is still there. */
  const send = async (change: () => Promise<unknown>): Promise<boolean> => {
    setBusy(true);
    await followInSession(change, {
      done: () => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_module.action.update.success.title"),
          message: t("project_module.action.update.success.description"),
        });
      },
      failed: (error) => {
        if (mounted.current) toastRefusal(error);
      },
    });
    if (!mounted.current) return false;
    setBusy(false);
    return true;
  };

  return { busy, send };
}
