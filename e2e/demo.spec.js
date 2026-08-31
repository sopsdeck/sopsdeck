import { copyFileSync, mkdirSync, readFileSync } from 'node:fs';
import { expect, test } from '@playwright/test';

const catalog = JSON.parse(
  readFileSync(new URL('../docs/assets/catalog.json', import.meta.url), 'utf8'),
);
const stillOf = Object.fromEntries(
  catalog.items.map((item) => [item.id, `docs/assets/${item.still}`]),
);
const clipOf = Object.fromEntries(
  catalog.items.map((item) => [item.id, `docs/assets/${item.clip}`]),
);
const skipClips = process.env.SOPSDECK_SKIP_CLIPS === '1';

function assertReadablePaths(page) {
  return Promise.all(
    ['breadcrumb', 'inspector-path'].map(async (id) => {
      const loc = page.getByTestId(id);
      await expect(loc).not.toContainText('..');
      await expect(loc).not.toContainText('/var/folders');
      await expect(loc).not.toContainText('desktop/../');
    }),
  );
}

async function hold(page, ms) {
  await page.waitForTimeout(ms);
}

async function boot(page) {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await assertReadablePaths(page);
}

async function typeValue(page, value) {
  const field = page.getByTestId('key-value');
  await field.click();
  await field.fill('');
  await field.pressSequentially(value, { delay: 55 });
}

async function encryptAndSave(page) {
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save-preview-dialog')).toBeVisible();
  await page.getByTestId('save-preview-confirm').click();
  await expect(page.getByTestId('save-preview-dialog')).toBeHidden();
  await expect(page.getByTestId('save')).toBeDisabled();
}

async function grantAccess(page) {
  const add = page.getByTestId('add-team-member');
  if (await add.isVisible()) await add.click();
  await expect(page.getByTestId('recipient-key')).toHaveValue(/^age1/);
  await page.getByTestId('grant-access').click();
  await expect(page.getByTestId('access-status')).toContainText('Access granted');
}

async function openGitHubSync(page) {
  await page.getByTestId('github-integration').click();
  await expect(page.getByTestId('integration-dialog')).toBeVisible();
}

async function syncGitHub(page) {
  await openGitHubSync(page);
  await page.getByTestId('integration-sync').click();
  await expect(page.getByTestId('integration-dialog-status')).toBeVisible();
}

async function recordClip(browser, dest, run) {
  const context = await browser.newContext({
    baseURL: process.env.SOPSDECK_DRIVE_URL ?? 'http://127.0.0.1:4174',
    viewport: { width: 1440, height: 900 },
    recordVideo: { dir: 'test-results/clips', size: { width: 1440, height: 900 } },
  });
  const page = await context.newPage();
  await hold(page, 900);
  await run(page);
  await hold(page, 1600);
  const video = page.video();
  await page.close();
  await video.saveAs(dest);
  await context.close();
}

test('product stills', async ({ page }) => {
  mkdirSync('docs/assets', { recursive: true });
  mkdirSync('site/public/assets', { recursive: true });
  await boot(page);
  await page.screenshot({ path: stillOf.open, fullPage: true });

  await page.getByTestId('reveal').click();
  await expect(page.getByTestId('key-value')).toHaveValue('sk_test_demo');
  await page.screenshot({ path: stillOf.reveal, fullPage: true });

  await page.getByTestId('key-value').fill('sk_live_demo');
  await expect(page.getByTestId('save')).toBeEnabled();
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save-preview')).toContainText('STRIPE_SECRET');
  await page.screenshot({ path: stillOf.save, fullPage: true });
  await page.getByTestId('save-preview-confirm').click();
  await expect(page.getByTestId('save')).toBeDisabled();
  await page.screenshot({ path: stillOf.commit, fullPage: true });

  await openGitHubSync(page);
  await page.screenshot({ path: stillOf.sync, fullPage: true });
  await page.getByTestId('integration-sync').click();
  await expect(page.getByTestId('integration-dialog-status')).toBeVisible();
  await page.screenshot({ path: stillOf.publish, fullPage: true });
  await page.keyboard.press('Escape');

  await grantAccess(page);
  await page.screenshot({ path: stillOf.grant, fullPage: true });

  copyFileSync(stillOf.open, 'site/public/assets/editor.png');
});

test('product clips', async ({ browser }) => {
  test.skip(skipClips, 'clips recorded by webreel');
  test.setTimeout(600_000);
  mkdirSync('docs/assets', { recursive: true });
  await recordClip(browser, clipOf.open, boot);
  await recordClip(browser, clipOf.reveal, async (page) => {
    await boot(page);
    await hold(page, 400);
    await page.getByTestId('reveal').click();
    await expect(page.getByTestId('key-value')).not.toHaveValue(/^•+$/);
    await hold(page, 700);
  });
  await recordClip(browser, clipOf.save, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await typeValue(page, 'sk_clip_save');
    await encryptAndSave(page);
  });
  await recordClip(browser, clipOf.commit, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await typeValue(page, 'sk_clip_commit');
    await encryptAndSave(page);
  });
  await recordClip(browser, clipOf.sync, async (page) => {
    await boot(page);
    await openGitHubSync(page);
  });
  await recordClip(browser, clipOf.grant, async (page) => {
    await boot(page);
    await grantAccess(page);
  });
  await recordClip(browser, clipOf.publish, async (page) => {
    await boot(page);
    await syncGitHub(page);
  });
});

test('walkthrough', async ({ browser }) => {
  test.skip(skipClips, 'clips recorded by webreel');
  test.setTimeout(600_000);
  mkdirSync('docs/assets', { recursive: true });
  await recordClip(browser, `docs/assets/${catalog.walkthrough}`, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await typeValue(page, 'sk_walkthrough');
    await encryptAndSave(page);
    await hold(page, 400);
    await grantAccess(page);
    await hold(page, 400);
    await syncGitHub(page);
  });
});
