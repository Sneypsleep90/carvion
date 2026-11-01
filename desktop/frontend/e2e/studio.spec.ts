import { expect, test } from "@playwright/test";

test("shows synthetic adapter status and opens the adapter workspace", async ({ page }) => {
  await page.goto("/e2e.html");

  await expect(page.getByRole("heading", { name: "CARVION Studio" })).toBeVisible();
  await expect(page.getByText("2 локальных адаптеров", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Адаптеры", exact: true }).click();
  await expect(page.getByRole("heading", { name: "Адаптеры" })).toBeVisible();
  await expect(page.getByText("CARVION Synthetic Android Demo", { exact: true })).toBeVisible();
  await expect(page.getByText("Только синтетические данные", { exact: true })).toBeVisible();
  await expect(page.getByText("SDK 1.0.0", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("android-strings-xml", { exact: true })).toBeVisible();
  await expect(page.getByText("compatibility-profile", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("ПОСЛЕДНИЙ ВЫЗОВ: ОШИБКА", { exact: true })).toBeVisible();
});

test("marks an unknown vehicle as unknown in the compatibility workspace", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Диагностика" }).click();

  await page.getByLabel("Производитель").fill("Неизвестная лаборатория");
  await page.getByRole("button", { name: "Проверить совместимость" }).click();

  await expect(page.getByRole("status")).toContainText("Неизвестно");
  await expect(page.getByRole("status")).toContainText("vehicle identity is not present in the synthetic demo profiles");
});

test("uses the adapter selected by the user for compatibility checks", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Диагностика" }).click();

  await page.getByLabel("Адаптер проверки").selectOption("adapter-android-demo-alt");
  await page.getByRole("button", { name: "Проверить совместимость" }).click();

  await expect(page.getByRole("status")).toContainText("adapter-android-demo-alt");
});

test("inspects an Android strings XML sample through the adapter workspace", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Локализация" }).click();
  await expect(page.locator('option[value="adapter-android-demo-alt"]')).toHaveCount(0);

  await page.getByRole("button", { name: "Загрузить тестовый XML" }).click();
  await page.getByRole("button", { name: "Проверить строки" }).click();

  await expect(page.getByRole("heading", { name: "Найденные строки" })).toBeVisible();
  await expect(page.getByText("welcome_title", { exact: true })).toBeVisible();
  await expect(page.getByRole("main").getByText("CARVION Studio", { exact: true })).toBeVisible();
});
