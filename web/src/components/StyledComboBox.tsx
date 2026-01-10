import {
  ComboBox,
  ComboBoxProps,
  Input,
  Button,
  Label,
  ListBox,
  ListBoxItem,
  ListBoxItemProps,
  Popover,
} from "react-aria-components";
import { IconChevronDown } from "@tabler/icons-react";

export interface MyComboBoxProps<T extends object>
  extends Omit<ComboBoxProps<T>, "children"> {
  label?: string;
  children: React.ReactNode | ((item: T) => React.ReactNode);
}

export function StyledComboBox<T extends object>({
  label,
  children,
  ...props
}: MyComboBoxProps<T>) {
  return (
    <ComboBox {...props} className="group flex flex-col gap-1">
      {label && (
        <Label className="block text-sm font-medium text-slate-700">
          {label}
        </Label>
      )}
      <div
        className={`flex items-center rounded-md shadow-xs border border-gray-300 group-focus-within:ring-1 group-focus-within:ring-emerald-500 group-focus-within:border-emerald-500`}
      >
        <Input className="block w-full px-3 py-2 bg-transparent focus:outline-hidden sm:text-sm" />
        <Button className="px-2">
          <IconChevronDown size={20} />
        </Button>
      </div>
      <Popover className="w-(--trigger-width) max-h-60 overflow-auto rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5">
        <ListBox className="outline-hidden">{children}</ListBox>
      </Popover>
    </ComboBox>
  );
}

export function StyledItem(props: ListBoxItemProps) {
  return (
    <ListBoxItem
      {...props}
      className="group flex items-center gap-2 cursor-default select-none py-2 px-4 outline-hidden focus:bg-emerald-500 focus:text-white"
    />
  );
}
