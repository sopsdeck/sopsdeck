import { mkdirSync } from 'node:fs';
import { expect, test } from '@playwright/test';

test('product screenshots', async ({ page }) => {
  mkdirSync('docs/assets', { recursive: true });
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.screenshot({ path: 'docs/assets/editor.png', fullPage: true });
  await page.getByTestId('reveal').click();
  await expect(page.getByTestId('key-value')).toHaveValue('sk_test_demo');
  await page.screenshot({
    path: 'docs/assets/editor-revealed.png',
    fullPage: true,
  });
});
