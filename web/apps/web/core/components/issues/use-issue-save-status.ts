/* SPDX-License-Identifier: AGPL-3.0-only */
import { useEffect, useState } from "react";
import type { TNameDescriptionLoader } from "@nerve/types";

/** Each editor owns its result; a sibling confirmation cannot hide a refusal. */
export function useIssueSaveStatus() {
  const [title, setTitleStatus] = useState<TNameDescriptionLoader>("saved");
  const [description, setDescriptionStatus] = useState<TNameDescriptionLoader>("saved");
  const status =
    (["failed", "submitting", "submitted", "saved"] as const).find(
      (candidate) => title === candidate || description === candidate
    ) ?? "saved";
  useEffect(() => {
    if (status !== "submitted") return;
    const finish = (current: TNameDescriptionLoader) => (current === "submitted" ? "saved" : current);
    const timer = setTimeout(() => {
      setTitleStatus(finish);
      setDescriptionStatus(finish);
    }, 2000);
    return () => clearTimeout(timer);
  }, [status]);
  return { status, setTitleStatus, setDescriptionStatus };
}
