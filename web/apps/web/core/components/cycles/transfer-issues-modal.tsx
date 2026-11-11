/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useLayoutEffect, useRef, useState } from "react";
import { observer } from "mobx-react";
import {
  CloseOutline,
  CyclesOutline,
  SearchOutline,
  TransferWorkItemOutline,
  WarningCircleOutline,
} from "@makeplane/propel/icons";
// nerve imports
import type { Cycle, CycleStatus } from "@nerve/api-client";
import { CYCLE_STATUS } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import { TOAST_TYPE, setToast } from "@nerve/propel/toast";
import { EModalPosition, EModalWidth, Loader, ModalCore, ModalTitle } from "@nerve/ui";
// hooks
import { useCycles } from "@/hooks/store/use-cycles";
import { useRefusalToast } from "@/hooks/use-refusal-toast";
// lib
import { followInSession } from "@/lib/in-session";

/** The cycles that may take an ended cycle's work items: those that have not ended (nerve refuses an ended one). */
const TAKING: ReadonlySet<CycleStatus> = new Set<CycleStatus>(["draft", "upcoming", "current"]);

type Props = {
  isOpen: boolean;
  handleClose: () => void;
  /** The ended cycle whose unfinished work items to transfer, as it was when the dialog was asked for. */
  cycle: Cycle | undefined;
  projectId: string;
};

/**
 * The dialog that transfers an ended cycle's unfinished work items to another of its project's cycles (P5a spec 2.2,
 * 2.5, R12), named by its title. Once the caller chooses that cycle it waits for nerve: it closes and says how many
 * nerve moved (none is an answer too) only once nerve has, the store then reading the counts of both cycles again;
 * refused (the cycle chosen no longer may take them, either is archived, cycles turned off, a role that may not, nerve
 * busy), it stays and says nerve's reason, never success. While the transfer is out nothing closes it (Escape, the
 * backdrop, its close button). Its follow-ups are those of the session it was sent in (followInSession) and of its
 * page, whose host holds it: once the page has gone (a Back, another project), nerve's late answer reaches the store
 * alone.
 */
export const TransferIssuesModal = observer(function TransferIssuesModal(props: Props) {
  const { isOpen, handleClose, cycle, projectId } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { transfer } = useCycles();
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

  const submit = async (target: Cycle) => {
    if (!cycle || out.current) return;
    out.current = true;
    setBusy(true);
    await followInSession(() => transfer(cycle.id, target.id), {
      done: ({ transferred_issues }) => {
        if (!mounted.current) return;
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("project_cycles.action.transfer.success.title"),
          message: t("project_cycles.action.transfer.success.description", {
            count: transferred_issues,
            name: target.name,
          }),
        });
        handleClose();
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
      position={EModalPosition.TOP}
      width={EModalWidth.XXL}
    >
      <div className="flex flex-col gap-4 py-5">
        <div className="flex items-center justify-between px-5">
          <div className="flex items-center gap-1">
            <TransferWorkItemOutline className="w-5 fill-primary" aria-hidden="true" />
            <ModalTitle className="text-18 font-medium text-primary">{t("project_cycles.transfer.title")}</ModalTitle>
          </div>
          <button type="button" aria-label={t("close")} onClick={handleClose} disabled={busy}>
            <CloseOutline className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        {cycle && <TransferTargets source={cycle} projectId={projectId} busy={busy} onChoose={(c) => void submit(c)} />}
      </div>
    </ModalCore>
  );
});

type TargetsProps = {
  source: Cycle;
  projectId: string;
  busy: boolean;
  onChoose: (target: Cycle) => void;
};

/**
 * The cycles that may take the source's work items, searched by name: its project's cycles, in nerve's order, that are
 * not archived and have not ended, other than itself; each a button that says its name and its status. Its search is
 * the dialog's while it is open, and starts empty each time it opens.
 */
const TransferTargets = observer(function TransferTargets(props: TargetsProps) {
  const { source, projectId, busy, onChoose } = props;
  // nerve hooks
  const { t } = useTranslation();
  // store hooks
  const { getProjectCycleIds, getCycleById } = useCycles();
  // states
  const [query, setQuery] = useState("");

  const targets = getProjectCycleIds(projectId)?.flatMap((id) => {
    const target = getCycleById(id);
    const taking = target && target.id !== source.id && target.archived_at === null && TAKING.has(target.status);
    return taking ? [target] : [];
  });
  const found = targets?.filter((target) => target.name.toLowerCase().includes(query.toLowerCase()));

  return (
    <>
      <div className="flex items-center gap-2 border-b border-subtle px-5 pb-3">
        <SearchOutline className="h-4 w-4 text-secondary" aria-hidden="true" />
        <input
          className="w-full text-13 outline-none"
          aria-label={t("project_cycles.transfer.search")}
          placeholder={t("project_cycles.transfer.search")}
          onChange={(e) => setQuery(e.target.value)}
          value={query}
        />
      </div>
      <div className="flex w-full flex-col items-start gap-2 px-5">
        {found === undefined ? (
          <Loader className="w-full space-y-2">
            <Loader.Item height="40px" />
            <Loader.Item height="40px" />
          </Loader>
        ) : found.length > 0 ? (
          <ul className="flex w-full flex-col gap-2">
            {found.map((target) => {
              const status = CYCLE_STATUS.find((one) => one.value === target.status);
              return (
                <li key={target.id}>
                  <button
                    type="button"
                    className="flex w-full items-center gap-4 rounded-sm px-4 py-3 text-13 text-secondary hover:bg-surface-2 disabled:cursor-not-allowed"
                    disabled={busy}
                    onClick={() => onChoose(target)}
                  >
                    <CyclesOutline className="h-5 w-5 flex-shrink-0" aria-hidden="true" />
                    <div className="flex w-full justify-between gap-2 truncate">
                      <span className="truncate">{target.name}</span>
                      <span className="flex flex-shrink-0 items-center rounded-full bg-layer-1 px-2">
                        {status ? t(status.i18n_title) : target.status}
                      </span>
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
        ) : (
          <div className="flex w-full items-center justify-center gap-4 p-5 text-13">
            <WarningCircleOutline className="h-3.5 w-3.5 text-secondary" aria-hidden="true" />
            <span className="text-center text-secondary">
              {targets?.length === 0 ? t("project_cycles.transfer.no_target") : t("project_cycles.no_matching_cycles")}
            </span>
          </div>
        )}
      </div>
    </>
  );
});
