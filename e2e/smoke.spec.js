import { execSync } from 'node:child_process';
import { expect, test } from '@playwright/test';

test('demo boots a Managed File and reveals a secret', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('managed-file')).toContainText('.env.production');
  await page.getByTestId('reveal').click();
  await expect(page.getByTestId('key-name')).toHaveValue('STRIPE_SECRET');
  await expect(page.getByTestId('key-value')).toHaveValue('sk_test_demo');
});

test('publish runs against the local fake GitHub', async ({ request }) => {
  const demo = await request.get('/demo');
  expect(demo.ok()).toBeTruthy();
  const info = await demo.json();
  expect(info.bobPublicKey).toMatch(/^age1/);

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

test('Sync succeeds against the local origin', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('sync').click();
  await expect(page.getByTestId('editor-error')).toBeHidden();
  await expect(page.getByTestId('git-error')).toBeHidden();
});

test('Sync failure sits next to Sync', async ({ page, request }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');

  const demo = await request.get('/demo');
  const info = await demo.json();
  execSync('git branch --unset-upstream', { cwd: info.project });

  await page.getByTestId('sync').click();
  const gitError = page.getByTestId('git-error');
  await expect(gitError).toBeVisible();
  await expect(gitError).toContainText('no upstream');
  await expect(gitError).not.toContainText('git-pull(1)');
  await expect(page.getByTestId('editor-error')).toBeHidden();
  await page.getByTestId('sync').click();
  await expect(gitError).toBeVisible();
});

test('Grant Access and Publish dry-run from the inspector', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('recipient-key')).toHaveValue(/^age1/);
  await page.getByTestId('grant-access').click();
  await expect(page.getByTestId('access-status')).toContainText('Access granted');
  await page.getByTestId('remove-access').click();
  await expect(page.getByTestId('access-status')).toContainText('still decrypt');
  await page.getByTestId('publish').click();
  await expect(page.getByTestId('publish-status')).toContainText('dry-run');
});
