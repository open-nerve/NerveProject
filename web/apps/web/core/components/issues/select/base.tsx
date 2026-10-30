/** Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only. See the LICENSE file for details. */
import { useCallback, useEffect, useId, useRef, useState, type ReactNode } from "react";
import type { Placement } from "@popperjs/core";
import { observer } from "mobx-react";
import { usePopper } from "react-popper";
import { Combobox, Popover } from "@headlessui/react";
import { Field, FieldError } from "@makeplane/propel/components/field";
import { LabelsOutline, LoadingOutline, SearchOutline, TickOutline } from "@makeplane/propel/icons";
import { getRandomLabelColor } from "@nerve/constants";
import { useTranslation } from "@nerve/i18n";
import type { Label } from "@nerve/api-client";
import { cn, isComposing, sortBySelectedFirst } from "@nerve/utils";
import { IssueLabelsList } from "@/components/ui/labels-list";
import { usePlatformOS } from "@/hooks/use-platform-os";
import type { IssueLabelCreation } from "@/components/labels/use-label-changes";

export type TWorkItemLabelSelectBaseProps = {
  buttonClassName?: string;
  buttonContainerClassName?: string;
  className?: string;
  optionsClassName?: string;
  issueName?: string;
  createLabelEnabled?: boolean;
  disabled?: boolean;
  getLabelById: (labelId: string) => Label | undefined;
  label?: ReactNode;
  labelIds: string[];
  isInitializing?: boolean;
  readStatus?: ReactNode;
  /** The caller owns the confirmed write and its refusal. This picker retains input on rejection. */
  onChange: (value: string[]) => void | Promise<void>;
  onDropdownOpen?: () => void;
  onClose?: () => void;
  onCreated?: (label: Label) => void;
  onChangeError?: (error: unknown) => void;
  placement?: Placement;
  onCreateLabel?: IssueLabelCreation;
  tabIndex?: number;
  value: string[];
  /** Existing detail and row pickers show every label; the form groups labels under their parent. */
  grouped?: boolean;
};

/** One popup lifetime, independent of rerenders while a directory or write settles. */
function PickerLifetime(props: { opened: () => void; closed: () => void }) {
  const [callbacks] = useState(() => props);
  useEffect(() => {
    callbacks.opened();
    return callbacks.closed;
  }, [callbacks]);
  return null;
}

export const WorkItemLabelSelectBase = observer(function WorkItemLabelSelectBase(props: TWorkItemLabelSelectBaseProps) {
  const {
    buttonClassName,
    buttonContainerClassName,
    className,
    optionsClassName,
    issueName,
    createLabelEnabled = false,
    disabled = false,
    getLabelById,
    label: buttonLabel,
    labelIds,
    isInitializing,
    readStatus,
    onChange,
    onDropdownOpen,
    onClose,
    onCreated,
    onChangeError,
    placement,
    onCreateLabel,
    tabIndex,
    value,
    grouped = true,
  } = props;
  const { t } = useTranslation();
  const { isMobile } = usePlatformOS();
  const [query, setQuery] = useState("");
  const [nameError, setNameError] = useState<string>();
  const errorId = useId();
  const [submitting, setSubmitting] = useState(false);
  const owner = useRef<symbol | null>(null);
  const pending = useRef<symbol | null>(null);
  const [reference, setReference] = useState<HTMLButtonElement | null>(null);
  const [panel, setPanel] = useState<HTMLElement | null>(null);
  const { styles, attributes } = usePopper(reference, panel, {
    placement: placement ?? "bottom-start",
    modifiers: [{ name: "preventOverflow", options: { padding: 12 } }],
  });
  const focusInput = useCallback(
    (input: HTMLInputElement | null) => {
      if (!isMobile) input?.focus({ preventScroll: true });
    },
    [isMobile]
  );
  const labels = labelIds.map((id) => getLabelById(id)).filter((label) => !!label);
  const matches = (label: Label) => label.name.toLowerCase().includes(query.trim().toLowerCase());
  const visible = grouped
    ? labels.filter(
        (label) =>
          !label.parent_id && (matches(label) || labels.some((child) => child.parent_id === label.id && matches(child)))
      )
    : (
        sortBySelectedFirst(
          labels.filter(matches).map((label) => ({ value: label.id, label })),
          value
        ) ?? []
      ).map((item) => item.label);
  const begin = () => {
    if (disabled || pending.current || !owner.current) return;
    const current = owner.current;
    pending.current = current;
    setSubmitting(true);
    return current;
  };
  const finish = (current: symbol) => {
    if (owner.current !== current) return;
    pending.current = null;
    setSubmitting(false);
  };
  const select = async (ids: string[]) => {
    const current = begin();
    if (!current) return;
    try {
      await onChange([...new Set(ids)]);
    } catch (error) {
      if (owner.current === current) onChangeError?.(error);
    } finally {
      finish(current);
    }
  };
  const create = async () => {
    const name = query.trim();
    if (!name || !createLabelEnabled || !onCreateLabel || isInitializing || readStatus) return;
    const current = begin();
    if (!current) return;
    setNameError(undefined);
    try {
      await onCreateLabel(
        { name, color: getRandomLabelColor() },
        (_field, message) => {
          if (owner.current === current) setNameError(message);
        },
        async (label) => {
          if (owner.current !== current) return;
          onCreated?.(label);
          try {
            await onChange([...new Set([...value, label.id])]);
            if (owner.current === current) setQuery("");
          } catch (error) {
            if (owner.current === current) onChangeError?.(error);
          }
        }
      );
    } finally {
      finish(current);
    }
  };
  const option = (label: Label) => (
    <Combobox.Option
      key={label.id}
      as="li"
      value={label.id}
      className={({ active }) =>
        cn(
          "flex cursor-pointer items-center justify-between gap-2 truncate rounded-sm px-1 py-1.5 text-secondary select-none",
          active && "bg-layer-1"
        )
      }
    >
      {({ selected }) => (
        <>
          <span className="flex items-center gap-2 truncate">
            <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: label.color }} />
            {label.name}
          </span>
          {selected && <TickOutline className="h-3.5 w-3.5 shrink-0" />}
        </>
      )}
    </Combobox.Option>
  );
  return (
    <Popover className={cn("relative h-full shrink-0", className)}>
      {({ open, close }) => (
        <>
          <Popover.Button
            type="button"
            ref={setReference}
            disabled={disabled || submitting}
            tabIndex={tabIndex}
            aria-label={issueName ? `${t("common.labels")}: ${issueName}` : t("common.labels")}
            className={cn(
              "flex h-full items-center gap-2 text-11 focus-visible:outline-1 focus-visible:-outline-offset-1 focus-visible:outline-accent-strong",
              buttonContainerClassName
            )}
          >
            {buttonLabel ??
              (value.length > 0 ? (
                <span className={cn("flex h-full items-center gap-2", buttonClassName)}>
                  <IssueLabelsList labels={value.map((id) => getLabelById(id))} length={3} showLength />
                </span>
              ) : (
                <span
                  className={cn(
                    "flex h-full items-center gap-1 rounded-sm border-[0.5px] border-strong px-2 py-1",
                    buttonClassName
                  )}
                >
                  <LabelsOutline className="h-3 w-3" />
                  {t("labels")}
                </span>
              ))}
          </Popover.Button>
          {open && (
            <PickerLifetime
              opened={() => {
                owner.current = Symbol("label-picker");
                onDropdownOpen?.();
              }}
              closed={() => {
                owner.current = null;
                pending.current = null;
                setSubmitting(false);
                setQuery("");
                setNameError(undefined);
                onClose?.();
              }}
            />
          )}
          <Popover.Panel
            portal
            data-prevent-outside-click
            ref={setPanel}
            style={styles.popper}
            {...attributes.popper}
            className={cn(
              "z-30 my-1 w-48 rounded-sm border-[0.5px] border-strong bg-surface-1 px-2 py-2.5 text-11 shadow-raised-200",
              optionsClassName
            )}
            onKeyDownCapture={(event) => {
              if (event.key !== "Escape") return;
              event.stopPropagation();
              if (isComposing(event.nativeEvent)) return;
              event.preventDefault();
              close();
            }}
          >
            <Combobox
              multiple
              value={value}
              disabled={disabled || submitting}
              onChange={(ids: string[]) => {
                void select(ids);
              }}
            >
              <Combobox.Options static modal={false} as="div">
                <Field name="name" invalid={!!nameError}>
                  <div className="flex items-center gap-1.5 rounded-sm border border-subtle bg-surface-2 px-2">
                    <SearchOutline className="h-3.5 w-3.5 text-placeholder" />
                    <Combobox.Input
                      ref={focusInput}
                      value={query}
                      aria-label={t("common.search.label")}
                      aria-invalid={!!nameError}
                      aria-describedby={nameError ? errorId : undefined}
                      className="w-full bg-transparent py-1 text-11 text-secondary placeholder:text-placeholder focus:outline-none"
                      placeholder={t("common.search.label")}
                      onChange={(event) => {
                        setQuery(event.target.value);
                        setNameError(undefined);
                      }}
                      onKeyDown={(event) => {
                        if (
                          event.key === "Enter" &&
                          !isComposing(event.nativeEvent) &&
                          visible.length === 0 &&
                          query.trim() &&
                          createLabelEnabled
                        ) {
                          event.preventDefault();
                          event.stopPropagation();
                          void create();
                        }
                      }}
                    />
                  </div>
                  {nameError && (
                    <FieldError match size="lg" id={errorId}>
                      {nameError}
                    </FieldError>
                  )}
                </Field>
                <div className="mt-2 max-h-48 overflow-y-auto">
                  {readStatus ??
                    (isInitializing ? (
                      <p>{t("common.loading")}</p>
                    ) : visible.length > 0 ? (
                      <ul className="space-y-1">
                        {visible.map((label) => {
                          const children = grouped ? labels.filter((child) => child.parent_id === label.id) : [];
                          return children.length ? (
                            <li key={label.id} className="border-y border-subtle">
                              <div className="p-2 text-primary">{label.name}</div>
                              <ul>{children.filter((child) => matches(label) || matches(child)).map(option)}</ul>
                            </li>
                          ) : (
                            option(label)
                          );
                        })}
                      </ul>
                    ) : submitting ? (
                      <LoadingOutline className="h-3.5 w-3.5 animate-spin" />
                    ) : createLabelEnabled ? (
                      <button
                        type="button"
                        disabled={disabled || submitting || !query.trim()}
                        onClick={() => {
                          void create();
                        }}
                        className="text-left text-secondary"
                      >
                        {query.trim() ? `${t("common.add_label")}: ${query.trim()}` : t("label.create.type")}
                      </button>
                    ) : (
                      <p>{t("common.search.no_matching_results")}</p>
                    ))}
                </div>
              </Combobox.Options>
            </Combobox>
          </Popover.Panel>
        </>
      )}
    </Popover>
  );
});
