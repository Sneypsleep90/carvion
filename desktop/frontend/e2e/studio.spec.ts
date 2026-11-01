import { expect, test } from "@playwright/test";
import { readFile } from "node:fs/promises";

const testOnlyProfile = JSON.stringify({
  schema: "carvion.geely-research/v1",
  profile_id: "test-only-profile",
  vehicle: { manufacturer: "TEST ONLY", model: "TEST ONLY", model_year: 2024, market: "TEST ONLY", trim: "TEST ONLY" },
  infotainment: { platform: "TEST ONLY", os_name: "TEST ONLY", os_version: "TEST ONLY", hardware_identifier: "TEST ONLY", hardware_revision: "TEST ONLY", firmware_version: "TEST ONLY", firmware_build_id: "TEST ONLY" },
});

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

test("analyzes a file profile, imports XML to Localization, and exports a private diagnostic report", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Исследование Geely" }).click();
  await expect(page.getByRole("heading", { name: "Исследование Geely" })).toBeVisible();

  await page.getByLabel("JSON профиль Geely").setInputFiles({
    name: "test-only-profile.json", mimeType: "application/json", buffer: Buffer.from(testOnlyProfile),
  });
  await page.getByLabel("XML или JSON ресурс").setInputFiles({
    name: "strings.xml", mimeType: "text/xml", buffer: Buffer.from('<resources><string name="title">SYNTHETIC ONLY RESOURCE TEXT</string></resources>'),
  });
  await page.getByRole("button", { name: "Анализировать профиль" }).click();
  await expect(page.getByText("Неизвестно · точной записи нет", { exact: true })).toBeVisible();
  await expect(page.getByText("ИЗ ФАЙЛА · НЕ ПРОВЕРЕНО", { exact: true })).toBeVisible();
  await expect(page.getByText("Synthetic Geely research source fixture", { exact: true })).toBeVisible();
  await expect(page.getByText("Только фикстура Playwright; не является техническим утверждением о мультимедиа Geely.", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Анализировать ресурсы" }).click();
  await expect(page.getByText("SYNTHETIC ONLY RESOURCE TEXT", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Создать проект локализации" }).click();
  await expect(page.getByText(/Создан локальный проект/)).toBeVisible();

  const downloadPromise = page.waitForEvent("download");
  await page.getByRole("button", { name: "Экспортировать отчёт" }).click();
  const download = await downloadPromise;
  expect(download.suggestedFilename()).toBe("carvion-geely-diagnostic-report.json");
  const downloadPath = await download.path();
  expect(downloadPath).not.toBeNull();
  const report = await readFile(downloadPath!, "utf8");
  expect(report).toContain("\"keys\": [\n      \"title\"");
  expect(report).not.toContain("SYNTHETIC ONLY RESOURCE TEXT");
  expect(report).not.toContain("/Users/");

  await page.getByRole("button", { name: "Открыть созданный проект" }).click();
  await expect(page.getByRole("heading", { name: "Строки проекта" })).toBeVisible();
  await expect(page.getByText("SYNTHETIC ONLY RESOURCE TEXT", { exact: true })).toBeVisible();
});

test("inspects a separate versioned JSON localization resource", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Исследование Geely" }).click();
  await page.getByLabel("JSON профиль Geely").setInputFiles({
    name: "test-only-profile.json", mimeType: "application/json", buffer: Buffer.from(testOnlyProfile),
  });
  await page.getByLabel("XML или JSON ресурс").setInputFiles({
    name: "strings.json", mimeType: "application/json",
    buffer: Buffer.from('{"schema":"carvion.localization/v1","resources":[{"key":"demo.title","kind":"string","text":"Synthetic JSON text"}]}'),
  });
  await page.getByRole("button", { name: "Анализировать ресурсы" }).click();
  await expect(page.getByText("Отдельный Android strings.xml или carvion-json-v1", { exact: true })).toBeVisible();
  await expect(page.getByText("demo.title", { exact: true })).toBeVisible();
  await expect(page.getByText("Synthetic JSON text", { exact: true })).toBeVisible();
});

test("rejects oversized profile and resource files before processing them", async ({ page }) => {
  await page.goto("/e2e.html");
  await page.getByRole("button", { name: "Исследование Geely" }).click();

  const oversized = Buffer.alloc(128 * 1024 + 1, 0x61);
  await page.getByLabel("JSON профиль Geely").setInputFiles({
    name: "oversized-profile.json", mimeType: "application/json", buffer: oversized,
  });
  await expect(page.getByRole("alert")).toHaveText("Файл превышает допустимый размер 131072 байт.");
  await expect(page.getByRole("button", { name: "Анализировать профиль" })).toBeDisabled();

  await page.getByLabel("XML или JSON ресурс").setInputFiles({
    name: "oversized-resource.xml", mimeType: "text/xml", buffer: oversized,
  });
  await expect(page.getByRole("alert")).toHaveText("Файл превышает допустимый размер 131072 байт.");
  await expect(page.getByRole("button", { name: "Анализировать ресурсы" })).toBeDisabled();
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
