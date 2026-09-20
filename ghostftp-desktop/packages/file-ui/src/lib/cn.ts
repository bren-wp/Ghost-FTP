import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

// Merge conditional class lists with Tailwind-aware de-duplication. A local copy
// so the package carries no dependency back into a host app's utils.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
