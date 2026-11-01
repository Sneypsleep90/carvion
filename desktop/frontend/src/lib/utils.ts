import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ReadonlyArray<ClassValue>): string {
  return twMerge(clsx(inputs));
}

export function formatAdapterCount(count: number): string {
  if (count === 1) return "1 локальный адаптер";
  return `${count} локальных адаптеров`;
}

export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Неизвестная ошибка Go Core";
}
