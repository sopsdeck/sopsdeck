import { expect, test } from '@playwright/test';

test('breadcrumb and inspector use a readable display path', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const breadcrumb = page.getByTestId('breadcrumb');
  const inspector = page.getByTestId('inspector-path');
  await expect(breadcrumb).toContainText('.env.production');
  await expect(inspector).toContainText('.env.production');
  for (const loc of [breadcrumb, inspector]) {
    await expect(loc).not.toContainText('..');
    await expect(loc).not.toContainText('/var/folders');
    await expect(loc).not.toContainText('desktop/../');
  }
});

test('empty state when no Project is open', async ({ page }) => {
  await page.goto('/?empty=1');
  const empty = page.getByTestId('empty-state');
  await expect(empty).toBeVisible();
  await expect(empty).toContainText('No Project yet');
  await expect(page.getByTestId('headline')).toHaveText('Sopsdeck');
  await expect(page.getByTestId('keys')).toBeHidden();
});

test('empty state when a Project has no Managed Files', async ({ page }) => {
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd === 'list_managed_files') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ result: [] }),
      });
      return;
    }

    await route.continue();
  });
  await page.goto('/');
  const empty = page.getByTestId('empty-state');
  await expect(empty).toBeVisible();
  await expect(empty).toContainText('no Managed Files');
});

test('empty state when the open file has no keys', async ({ page }) => {
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd === 'get_managed_file') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ result: [] }),
      });
      return;
    }

    await route.continue();
  });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const empty = page.getByTestId('empty-state');
  await expect(empty).toBeVisible();
  await expect(empty).toContainText('No keys');
});

test('Sync shows loading on the control', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  let release;
  const held = new Promise((resolve) => {
    release = resolve;
  });
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd === 'sync_project') {
      await held;
    }

    await route.continue();
  });
  const clicked = page.getByTestId('sync').click();
  await expect(page.getByTestId('sync')).toHaveAttribute('aria-busy', 'true');
  release();
  await clicked;
  await expect(page.getByTestId('sync')).not.toHaveAttribute('aria-busy', 'true');
});

test('commit message prefills from edited keys', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('reveal').click();
  await page.getByTestId('key-value').fill('sk_live_demo');
  await expect(page.getByTestId('commit-message')).toHaveValue(/STRIPE_SECRET/);
});

test('theme survives reload', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('theme-toggle').click();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
  await page.reload();
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
});
