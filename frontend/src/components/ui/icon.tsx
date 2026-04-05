import { HugeiconsIcon, type HugeiconsProps } from "@hugeicons/react"
import {
  Add01Icon,
  ArrowDown01Icon,
  ArrowLeft01Icon,
  ArrowLeftRightIcon,
  ArrowRight01Icon,
  ArrowUpDownIcon,
  Cancel01Icon,
  Clock01Icon,
  Copy01Icon,
  CubeIcon,
  Delete02Icon,
  Folder01Icon,
  Folder02Icon,
  Globe02Icon,
  Moon01Icon,
  PencilEdit01Icon,
  Search01Icon,
  Sun01Icon,
  Upload01Icon,
  SidebarLeftIcon,
  SidebarRightIcon,
} from "@hugeicons/core-free-icons"
import { cn } from "@/lib/utils"

const iconMap = {
  add: Add01Icon,
  arrowDown: ArrowDown01Icon,
  arrowLeft: ArrowLeft01Icon,
  arrowLeftRight: ArrowLeftRightIcon,
  arrowRight: ArrowRight01Icon,
  arrowUpDown: ArrowUpDownIcon,
  clear: Cancel01Icon,
  clock: Clock01Icon,
  copy: Copy01Icon,
  cube: CubeIcon,
  delete: Delete02Icon,
  folder: Folder01Icon,
  folderOpen: Folder02Icon,
  globe: Globe02Icon,
  moon: Moon01Icon,
  pencil: PencilEdit01Icon,
  search: Search01Icon,
  sun: Sun01Icon,
  upload: Upload01Icon,
  sidebarCollapse: SidebarLeftIcon,
  sidebarExpand: SidebarRightIcon,
} as const

export type AppIconName = keyof typeof iconMap

interface AppIconProps extends Omit<HugeiconsProps, "icon"> {
  name: AppIconName
  className?: string
}

export function AppIcon({ name, className, size = 16, strokeWidth = 1.8, color = "currentColor", ...props }: AppIconProps) {
  return (
    <HugeiconsIcon
      icon={iconMap[name]}
      size={size}
      strokeWidth={strokeWidth}
      color={color}
      className={cn("shrink-0", className)}
      {...props}
    />
  )
}
