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

test("imports a synthetic Android resource file and edits a translation", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Локализация" }).click();
  await page.getByRole("button", { name: "Синтетический пример" }).click();
  await page.getByRole("button", { name: "Импортировать ресурсы" }).click();
  await expect(page.getByRole("heading", { name: "Строки проекта" })).toBeVisible();
  await expect(page.getByText("welcome_title", { exact: true })).toBeVisible();
  await expect(page.getByText("Synthetic media", { exact: true })).toBeVisible();

  await page.getByLabel("Поиск ресурсов").fill("welcome_title");
  await page.getByLabel("Перевод welcome_title").fill("Главная");
  await page.getByRole("button", { name: "Сохранить", exact: true }).click();
  await expect(page.getByText("Черновик", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Утвердить", exact: true }).click();
  await expect(page.getByText("Утверждён", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Проверить перевод" }).click();
  await expect(page.getByRole("heading", { name: "Найдены проблемы" })).toBeVisible();
});

test("reimporting a changed source requires an approved translation to be reviewed again", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Локализация" }).click();
  await page.getByRole("button", { name: "Синтетический пример" }).click();
  await page.getByRole("button", { name: "Импортировать ресурсы" }).click();
  await page.getByLabel("Поиск ресурсов").fill("welcome_title");
  await page.getByLabel("Перевод welcome_title").fill("Главная");
  await page.getByRole("button", { name: "Сохранить", exact: true }).click();
  await page.getByRole("button", { name: "Утвердить", exact: true }).click();

  await page.getByRole("button", { name: "Обновить источник" }).click();
  await page.locator('input[type="file"]').first().setInputFiles({
    name: "strings-v2.xml",
    mimeType: "text/xml",
    buffer: Buffer.from('<resources><string name="welcome_title">Updated title</string></resources>'),
  });

  await expect(page.getByText("Проверить", { exact: true })).toBeVisible();
  await expect(page.getByText("Updated title", { exact: true })).toBeVisible();
});

test("runs a complete synthetic simulator localization cycle", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Симулятор" }).click();

  await expect(page.getByRole("heading", { name: "Симулятор CARVION" })).toBeVisible();
  await expect(page.getByText("SIMULATION · ТОЛЬКО ТЕСТОВЫЕ ДАННЫЕ", { exact: true })).toBeVisible();
  await page.getByLabel("Профиль симуляции").selectOption("demo-android-platform-a");
  await page.getByRole("button", { name: "Запустить сценарий" }).click();

  await expect(page.getByRole("status")).toContainText("Сценарий завершён");
  await expect(page.getByText("adapter.selected", { exact: true })).toBeVisible();
  await expect(page.getByText("package.verified", { exact: true })).toBeVisible();
  await expect(page.getByText("Пакет проверен · OEM firmware: false", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Скачать языковой пакет" })).toBeEnabled();
  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Скачать языковой пакет" }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("CARVION-Simulator-ru-RU.carvion.zip");
});

test("stops before localization when firmware is unknown", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Симулятор" }).click();
  await page.getByLabel("Профиль симуляции").selectOption("unknown-firmware");
  await page.getByRole("button", { name: "Запустить сценарий" }).click();

  await expect(page.getByRole("status")).toContainText("Неизвестная сборка");
  await expect(page.getByText("localization.imported", { exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: "Скачать языковой пакет" })).toHaveCount(0);
});

test("stops before localization for incompatible hardware", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Симулятор" }).click();
  await page.getByLabel("Профиль симуляции").selectOption("incompatible-hardware");
  await page.getByRole("button", { name: "Запустить сценарий" }).click();

  await expect(page.getByRole("status")).toContainText("Аппаратная ревизия несовместима");
  await expect(page.getByText("localization.imported", { exact: true })).toHaveCount(0);
});

test("shows a controlled communication failure in the simulator event log", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Симулятор" }).click();
  await page.getByLabel("Профиль симуляции").selectOption("simulated-communication-failure");
  await page.getByRole("button", { name: "Запустить сценарий" }).click();

  await expect(page.getByRole("status")).toContainText("Сценарий завершился с ошибкой");
  await expect(page.getByText("adapter.detect", { exact: true })).toBeVisible();
  await expect(page.getByRole("list", { name: "События симуляции" }).getByText("simulated adapter communication failure", { exact: true })).toBeVisible();
});

test("cancels an active simulator run and shows its final event", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Симулятор" }).click();
  await page.getByLabel("Профиль симуляции").selectOption("demo-android-platform-b");
  await page.getByRole("button", { name: "Запустить сценарий" }).click();
  await page.getByRole("button", { name: "Остановить сценарий" }).click();

  await expect(page.getByRole("status")).toContainText("Сценарий завершился с ошибкой");
  await expect(page.getByText("scenario.cancelled", { exact: true })).toBeVisible();
  await expect(page.getByRole("button", { name: "Скачать языковой пакет" })).toHaveCount(0);
});
