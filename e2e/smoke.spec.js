import { expect, test } from '@playwright/test';

test('demo boots a Managed File and reveals a secret', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(
    page.getByTestId('managed-file').filter({ hasText: '.env.production' }),
  ).toBeVisible();
  await page.getByTestId('reveal').click();
  await expect(page.getByTestId('key-name')).toHaveValue('STRIPE_SECRET');
  await expect(page.getByTestId('key-value')).toHaveValue('sk_test_demo');
});

test('publish runs against the local fake GitHub', async ({ request }) => {
  const demo = await request.get('/demo');
  expect(demo.ok()).toBeTruthy();
  const info = await demo.json();
  expect(info.bobPublicKey).toMatch(/^age1/);
  expect(info.projects.length).toBeGreaterThan(1);

  const listed = await request.post('/invoke', {
    data: { cmd: 'list_managed_files', path: info.project },
  });
  const listBody = await listed.json();
  const file = listBody.result.find((item) => item.name === '.env.production');
  expect(file).toBeTruthy();

  const published = await request.post('/invoke', {
    data: {
      cmd: 'publish_managed_file',
      path: file.path,
      prefix: 'SD_',
      yes: true,
    },
  });
  const body = await published.json();
  expect(body.result).toContain('published');
});

test('GitHub Sync now publishes to the local fake GitHub', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('github-integration').click();
  await expect(page.getByTestId('integration-dialog')).toBeVisible();
  await page.getByTestId('integration-sync').click();
  await expect(page.getByTestId('integration-dialog-status')).toBeVisible();
});

test('Grant Access from the inspector', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('add-team-member').click();
  await expect(page.getByTestId('recipient-key')).toHaveValue(/^age1/);
  await page.getByTestId('grant-access').click();
  await expect(page.getByTestId('access-status')).toContainText('Access granted');
});
