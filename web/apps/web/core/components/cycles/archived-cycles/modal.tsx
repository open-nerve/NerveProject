/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
// nerve imports
import type { Cycle } from "@nerve/api-client";
import { useTranslation } from "@nerve/i18n";
import { Button } from "@nerve/propel/button";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The cycle to archive, as it was when the dialog was asked for. */
  cycle: Cycle | undefined;
  workspaceSlug: string;
  projectId: string;
};

/**
 * The dialog that archives an ended cycle (P5a spec 2.5), named by its title. It closes only once nerve has archived
 * the cycle, says so and goes to the cycles' list; refused, it stays and says nerve's reason. While the archive is out
 * nothing closes it (Escape, the backdrop, Cancel). Its follow-ups are those of the session it was sent in
 * (followInSession) and of its page: once the page has gone (a Back, another project), nerve's late answer reaches the
 * store alone.
 */
export function ArchiveCycleModal(props: Props) {
  const { isOpen, handleClose, cycle, workspaceSlug, projectId } = props;
  // router
  const navigate = useNavigate();
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { archive } = useCycles();
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
    await followInSession(() => archive(cycle.id), {
      done: () => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_cycles.action.archive.success.title"),
          message: t("project_cycles.action.archive.success.description"),
        });
        handleClose();
        return navigate(`/${workspaceSlug}/projects/${projectId}/cycles`);
      },
      failed: (error) => {
        if (mounted.current) toastRefusal(error);
      },
    });
    out.current = false;
    if (mounted.current) setBusy(false);
  };

  return (
    <ModalCore
      isOpen={isOpen}
      handleClose={busy ? undefined : handleClose}
      position={EModalPosition.CENTER}
      width={EModalWidth.LG}
    >
      <div className="px-5 py-4">
        <ModalTitle className="text-18 font-medium 2xl:text-20">
          {t("project_cycles.archive.title", { name: cycle?.name })}
        </ModalTitle>
        <p className="mt-3 text-13 text-secondary">{t("project_cycles.archive.body")}</p>
        <div className="mt-3 flex justify-end gap-2">
          <Button variant="secondary" size="lg" onClick={handleClose} disabled={busy}>
            {t("cancel")}
          </Button>
          <Button variant="primary" size="lg" onClick={() => void submit()} loading={busy}>
            {busy ? t("common.archiving") : t("archive")}
          </Button>
        </div>
      </div>
    </ModalCore>
  );
}
