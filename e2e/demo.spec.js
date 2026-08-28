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

async function boot(page) {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await assertReadablePaths(page);
}

async function recordClip(browser, dest, run) {
  const context = await browser.newContext({
    baseURL: process.env.SOPSDECK_DRIVE_URL ?? 'http://127.0.0.1:4174',
    viewport: { width: 1440, height: 900 },
    recordVideo: { dir: 'test-results/clips', size: { width: 1440, height: 900 } },
  });
  const page = await context.newPage();
  await run(page);
  const video = page.video();
  await page.close();
  await video.saveAs(dest);
  await context.close();
}

test('product stills', async ({ page }) => {
  mkdirSync('docs/assets', { recursive: true });
  mkdirSync('site/assets', { recursive: true });
  await boot(page);
  await page.screenshot({ path: stillOf.open, fullPage: true });

  await page.getByTestId('reveal').click();
  await expect(page.getByTestId('key-value')).toHaveValue('sk_test_demo');
  await page.screenshot({ path: stillOf.reveal, fullPage: true });

  await page.getByTestId('key-value').fill('sk_live_demo');
  await expect(page.getByTestId('save')).toBeEnabled();
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save')).toBeDisabled();
  await page.screenshot({ path: stillOf.save, fullPage: true });

  await expect(page.getByTestId('commit-message')).toHaveValue(/STRIPE_SECRET/);
  await page.screenshot({ path: stillOf.commit, fullPage: true });
  await page.getByTestId('commit').click();
  await expect(page.getByTestId('git-error')).toBeHidden();

  await page.getByTestId('sync').click();
  await expect(page.getByTestId('git-error')).toBeHidden();
  await page.screenshot({ path: stillOf.sync, fullPage: true });

  await expect(page.getByTestId('recipient-key')).toHaveValue(/^age1/);
  await page.getByTestId('grant-access').click();
  await expect(page.getByTestId('access-status')).toContainText('Access granted');
  await page.screenshot({ path: stillOf.grant, fullPage: true });

  await page.getByTestId('publish').click();
  await expect(page.getByTestId('publish-status')).toContainText('dry-run');
  await page.screenshot({ path: stillOf.publish, fullPage: true });

  copyFileSync(stillOf.open, 'site/assets/editor.png');
});

test('product clips', async ({ browser }) => {
  mkdirSync('docs/assets', { recursive: true });
  await recordClip(browser, clipOf.open, boot);
  await recordClip(browser, clipOf.reveal, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await expect(page.getByTestId('key-value')).not.toHaveValue(/^•+$/);
  });
  await recordClip(browser, clipOf.save, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await page.getByTestId('key-value').fill('sk_clip_save');
    await page.getByTestId('save').click();
    await expect(page.getByTestId('save')).toBeDisabled();
  });
  await recordClip(browser, clipOf.commit, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await page.getByTestId('key-value').fill('sk_clip_commit');
    await page.getByTestId('save').click();
    await expect(page.getByTestId('save')).toBeDisabled();
    await expect(page.getByTestId('commit-message')).toHaveValue(/STRIPE_SECRET/);
    await page.getByTestId('commit').click();
    await expect(page.getByTestId('git-error')).toBeHidden();
  });
  await recordClip(browser, clipOf.sync, async (page) => {
    await boot(page);
    await page.getByTestId('sync').click();
    await expect(page.getByTestId('git-error')).toBeHidden();
  });
  await recordClip(browser, clipOf.grant, async (page) => {
    await boot(page);
    await page.getByTestId('grant-access').click();
    await expect(page.getByTestId('access-status')).toContainText('Access granted');
  });
  await recordClip(browser, clipOf.publish, async (page) => {
    await boot(page);
    await page.getByTestId('publish').click();
    await expect(page.getByTestId('publish-status')).toContainText('dry-run');
  });
});

test('walkthrough', async ({ browser }) => {
  mkdirSync('docs/assets', { recursive: true });
  await recordClip(browser, `docs/assets/${catalog.walkthrough}`, async (page) => {
    await boot(page);
    await page.getByTestId('reveal').click();
    await page.getByTestId('key-value').fill('sk_walkthrough');
    await page.getByTestId('save').click();
    await expect(page.getByTestId('save')).toBeDisabled();
    await page.getByTestId('commit').click();
    await expect(page.getByTestId('git-error')).toBeHidden();
    await page.getByTestId('sync').click();
    await expect(page.getByTestId('git-error')).toBeHidden();
    await page.getByTestId('grant-access').click();
    await expect(page.getByTestId('access-status')).toContainText('Access granted');
    await page.getByTestId('publish-yes').click();
    await expect(page.getByTestId('publish-status')).toContainText('published');
  });
});
