/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router";
// nerve imports
import type { Cycle } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { AlertModalCore } from "@nerve/ui";
// components
import { Translated } from "@/components/common/translated";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The cycle to delete, as it was when the dialog was asked for. */
  cycle: Cycle | undefined;
  workspaceSlug: string;
  projectId: string;
};

/**
 * The dialog that deletes a cycle (P5a spec 2.5). It closes only once nerve has deleted the cycle, says so and, when
 * the page shows the cycle (its own page, its preview), goes back to the cycles' list; refused, it stays and says
 * nerve's reason. While the deletion is out nothing closes it (Escape, the backdrop, Cancel). Its follow-ups are those
 * of the session it was sent in (followInSession) and of its page: once the page has gone (a Back, another project),
 * nerve's late answer reaches the store alone.
 */
export function CycleDeleteModal(props: Props) {
  const { isOpen, handleClose, cycle, workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  const { cycleId: pageCycleId } = useParams();
  const [searchParams] = useSearchParams();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { remove } = useCycles();
  const toastRefusal = useRefusalToast();
  // states
  const [busy, setBusy] = useState(false);
  const out = useRef(false);
  const mounted = useRef(false);
  useLayoutEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const submit = async () => {
    if (!cycle || out.current) return;
    out.current = true;
    setBusy(true);
    const shownByPage = pageCycleId === cycle.id || searchParams.get("peekCycle") === cycle.id;
    await followInSession(() => remove(cycle.id), {
      done: () => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_cycles.action.delete.success.title"),
          message: t("project_cycles.action.delete.success.description"),
        });
        handleClose();
        if (shownByPage) return navigate(`/${workspaceSlug}/projects/${projectId}/cycles`);
      },
      failed: (error) => {
        if (mounted.current) toastRefusal(error);
      },
    });
    out.current = false;
    if (mounted.current) setBusy(false);
  };

  return (
    <AlertModalCore
      handleClose={handleClose}
      handleSubmit={() => void submit()}
      isSubmitting={busy}
      isOpen={isOpen}
      title={t("project_cycles.delete.title")}
      primaryButtonText={{ default: t("delete"), loading: t("deleting") }}
      secondaryButtonText={t("cancel")}
      content={
        <Translated
          i18nKey="project_cycles.delete.body"
          values={{ name: <span className="font-medium break-words text-primary">{cycle?.name}</span> }}
        />
      }
    />
  );
}
