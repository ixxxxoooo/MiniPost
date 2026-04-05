import * as React from "react"
import * as SelectPrimitive from "@radix-ui/react-select"
import { AppIcon } from "@/components/ui/icon"
import { cn } from "@/lib/utils"

const Select = SelectPrimitive.Root
const SelectGroup = SelectPrimitive.Group
const SelectValue = SelectPrimitive.Value

const SelectTrigger = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Trigger>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Trigger>
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Trigger
    ref={ref}
    className={cn(
      "flex items-center justify-between rounded-[var(--radius-input)] border border-[var(--border-color)]",
      "bg-[var(--surface)] h-[var(--size-input)] px-3 text-[length:var(--size-font-sm)]",
      "text-[var(--fg)] placeholder:text-[var(--fg-muted)]",
      "focus:outline-none focus:ring-2 focus:ring-[var(--accent)]",
      "disabled:cursor-not-allowed disabled:opacity-50 [&>span]:line-clamp-1",
      className
    )}
    {...props}
  >
    {children}
    <SelectPrimitive.Icon asChild>
      <AppIcon name="arrowDown" size={14} strokeWidth={1.9} className="ml-1 opacity-50" />
    </SelectPrimitive.Icon>
  </SelectPrimitive.Trigger>
))
SelectTrigger.displayName = SelectPrimitive.Trigger.displayName

const SelectContent = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Content>
>(({ className, children, position = "popper", ...props }, ref) => (
  <SelectPrimitive.Portal>
    <SelectPrimitive.Content
      ref={ref}
      className={cn(
        "relative z-50 max-h-96 overflow-hidden",
        "w-max min-w-[var(--radix-select-trigger-width)]",
        "rounded-[var(--radius-menu)] border border-[var(--border-color)]",
        "bg-[var(--surface-elevated)] text-[var(--fg)] shadow-[var(--shadow-lg)]",
        "data-[state=open]:animate-fade-in",
        position === "popper" && "data-[side=bottom]:translate-y-1 data-[side=top]:-translate-y-1",
        className
      )}
      position={position}
      {...props}
    >
      <SelectPrimitive.Viewport
        className={cn(
          "p-1",
          position === "popper" && "w-auto min-w-[var(--radix-select-trigger-width)]"
        )}
      >
        {children}
      </SelectPrimitive.Viewport>
    </SelectPrimitive.Content>
  </SelectPrimitive.Portal>
))
SelectContent.displayName = SelectPrimitive.Content.displayName

const SelectItem = React.forwardRef<
  React.ElementRef<typeof SelectPrimitive.Item>,
  React.ComponentPropsWithoutRef<typeof SelectPrimitive.Item>
>(({ className, children, ...props }, ref) => (
  <SelectPrimitive.Item
    ref={ref}
    className={cn(
      "relative flex w-full cursor-default select-none items-center",
      "rounded-[var(--radius-sm)] py-1.5 px-2 text-[length:var(--size-font-sm)] outline-none",
      "whitespace-nowrap",
      "focus:bg-[var(--selected-bg)] focus:text-[var(--fg)]",
      "data-[state=checked]:bg-[var(--selected-bg)] data-[state=checked]:text-[var(--fg)]",
      "data-[disabled]:pointer-events-none data-[disabled]:opacity-40",
      className
    )}
    {...props}
  >
    <SelectPrimitive.ItemText>{children}</SelectPrimitive.ItemText>
  </SelectPrimitive.Item>
))
SelectItem.displayName = SelectPrimitive.Item.displayName

export {
  Select,
  SelectGroup,
  SelectValue,
  SelectTrigger,
  SelectContent,
  SelectItem,
}
