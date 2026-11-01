/** Copyright (c) 2026-present OpenNerve. SPDX-License-Identifier: AGPL-3.0-only */
import type { ICustomSearchSelectOption } from "@nerve/types";
import { CustomSelect } from "./custom-select";
import { CustomSearchSelect } from "./custom-search-select";
import { CustomMenu } from "./custom-menu";

// Compiled by UI's real tsc include; never called or exported from the production barrel.
function selectContracts(arbitraryMultiple: boolean) {
  const numericOption: ICustomSearchSelectOption<number> = { value: 0, query: "Zero", content: "Zero" };
  // @ts-expect-error: a declared numeric option does not accept text.
  const wrongDeclaredOption = <CustomSelect.Option<number> value="zero">Wrong</CustomSelect.Option>;
  const strings = [{ value: "one", query: "One", content: "One" }];
  const numbers = [{ value: 0, query: "Zero", content: "Zero" }];
  const theme = { name: "Dark", colors: ["black", "white"] };
  const single = (
    <CustomSearchSelect value={null} options={strings} onChange={(value: string) => value.toUpperCase()} />
  );
  const number = <CustomSearchSelect value={0} options={numbers} onChange={(value: number) => value.toFixed()} />;
  const multi = (
    <CustomSearchSelect multiple value={null} options={numbers} onChange={(values: number[]) => values.map(String)} />
  );
  const object = (
    <CustomSelect value={theme} onChange={(value: typeof theme) => value.colors}>
      <CustomSelect.Option value={theme}>Dark</CustomSelect.Option>
    </CustomSelect>
  );
  const empty = (
    <CustomSelect value={null} onChange={(value: boolean) => !value}>
      <CustomSelect.Option value={false}>False</CustomSelect.Option>
    </CustomSelect>
  );
  const item = <CustomMenu.MenuItem onClick={(event) => event.currentTarget.focus()}>Open</CustomMenu.MenuItem>;
  const wrongValue = (
    // @ts-expect-error: a numeric selection cannot be passed to a string callback.
    <CustomSelect value={7} onChange={(value: string) => value.toUpperCase()}>
      Wrong
    </CustomSelect>
  );
  // @ts-expect-error: a string option cannot be passed to a number callback.
  const wrongOption = <CustomSearchSelect options={strings} value={0} onChange={(value: number) => value.toFixed()} />;
  const wrongMulti = (
    // @ts-expect-error: multiple mode emits an array, never a scalar.
    <CustomSearchSelect multiple options={strings} value={["one"]} onChange={(value: string) => value.toUpperCase()} />
  );
  // @ts-expect-error: single mode emits one value, never an array.
  const wrongSingle = <CustomSearchSelect options={strings} value="one" onChange={(value: string[]) => value.join()} />;
  const unknownMode = (
    // @ts-expect-error: mode must remain correlated with the value and callback, not an arbitrary boolean.
    <CustomSearchSelect
      multiple={arbitraryMultiple}
      options={strings}
      value="one"
      onChange={(value: string) => value.toUpperCase()}
    />
  );
  // @ts-expect-error: search option values must be React key primitives, not theme objects.
  const objectOption: ICustomSearchSelectOption = { value: theme, query: "Dark", content: "Dark" };
  // @ts-expect-error: a menu item gets a button mouse event, never a selected string.
  const wrongEvent = <CustomMenu.MenuItem onClick={(value: string) => value.toUpperCase()}>Wrong</CustomMenu.MenuItem>;
  // @ts-expect-error: opening is a notification with no arguments.
  const wrongOpen = <CustomMenu menuButtonOnClick={(value: string) => value.toUpperCase()}>Wrong</CustomMenu>;
  return {
    numericOption,
    wrongDeclaredOption,
    single,
    number,
    multi,
    object,
    empty,
    item,
    wrongValue,
    wrongOption,
    wrongMulti,
    wrongSingle,
    unknownMode,
    objectOption,
    wrongEvent,
    wrongOpen,
  };
}
void selectContracts;
