import type { TDropdownProps } from "../types";

export type MemberDropdownProps = TDropdownProps & {
  dropdownArrow?: boolean;
  dropdownArrowClassName?: string;
  placeholder?: string;
  tooltipContent?: string;
  onClose?: () => void;
  showUserDetails?: boolean;
  /** As one of a form's fields: whether nerve refused its value (its select's button is aria-invalid). */
  invalid?: boolean;
  /** As one of a form's fields: the id of the message under it, which describes its select's button. */
  describedBy?: string;
} & (
    | {
        multiple: false;
        onChange: (val: string | null) => void;
        value: string | null;
      }
    | {
        multiple: true;
        onChange: (val: string[]) => void;
        value: string[];
      }
  );
