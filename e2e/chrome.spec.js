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
  await expect(page.getByTestId('key-composer')).toBeVisible();
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
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save')).toBeDisabled();
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

test("What's new shows bundled notes", async ({ page }) => {
  await page.goto('/?empty=1');
  await page.getByTestId('whats-new').click();
  const dialog = page.getByTestId('whats-new-dialog');
  await expect(dialog).toBeVisible();
  await expect(dialog).toContainText('Unreleased');
  await expect(page.getByTestId('whats-new-list').locator('li')).not.toHaveCount(0);
  await expect(page.getByTestId('whats-new-list').locator('li').first()).toHaveClass(
    /whats-new-item/,
  );
});

test('Review shows a plaintext secret diff after save', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('reveal').click();
  await page.getByTestId('key-value').fill('sk_review_demo');
  await expect(page.getByTestId('save')).toBeEnabled();
  await page.getByTestId('save').click();
  await page.getByTestId('review').click();
  const out = page.getByTestId('review-out');
  await expect(out).toBeVisible();
  await expect(out).toContainText('STRIPE_SECRET');
  await expect(out).toContainText('sk_review_demo');
  await expect(out).not.toContainText('ENC[');
});

test('History lists commits without secret values', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('history').click();
  const list = page.getByTestId('history-list');
  await expect(list.locator('button')).not.toHaveCount(0);
  await expect(list).not.toContainText('sk_test_demo');
});

async function keyNames(page) {
  return page.getByTestId('key-name').evaluateAll((els) => els.map((el) => el.value));
}

async function keyRowByName(page, key) {
  const rows = page.getByTestId('key-row');
  const count = await rows.count();
  for (let i = 0; i < count; i++) {
    const row = rows.nth(i);
    if ((await row.getByTestId('key-name').inputValue()) === key) {
      return row;
    }
  }

  throw new Error(`no row for ${key}`);
}

test('composer is visible when a Managed File is open', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('key-composer')).toBeVisible();
});

test('key name on an existing row is editable', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const name = page.getByTestId('key-name').first();
  await expect(name).toHaveValue('STRIPE_SECRET');
  await expect(name).not.toHaveAttribute('readonly');
  await name.click();
  await name.fill('STRIPE_SECRET_EDIT');
  await expect(name).toHaveValue('STRIPE_SECRET_EDIT');
});

test('key rows have reveal, copy, and delete icon controls', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const row = page.getByTestId('key-row').first();
  await expect(row.getByTestId('reveal-key')).toBeVisible();
  await expect(row.getByTestId('copy-value')).toBeVisible();
  await expect(row.getByTestId('delete-key')).toBeVisible();
  await expect(row.getByTestId('copy-key')).toBeVisible();
});

test('composer adds a key that survives reload', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('key-composer').fill('UI_ADD_ME=composer_saved');
  await page.getByTestId('key-composer').press('Enter');
  await expect(page.getByTestId('save')).toBeEnabled();
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save')).toBeDisabled();
  await page.reload();
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect.poll(async () => keyNames(page)).toContain('UI_ADD_ME');
});

test('deleting a key and saving removes it', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('key-composer').fill('UI_DELETE_ME=gone');
  await page.getByTestId('key-composer').press('Enter');
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save')).toBeDisabled();
  const row = await keyRowByName(page, 'UI_DELETE_ME');
  await row.getByTestId('delete-key').click();
  await expect(page.getByTestId('save')).toBeEnabled();
  await expect(page.getByTestId('commit-message')).toHaveValue(/Remove UI_DELETE_ME/);
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save')).toBeDisabled();
  await expect.poll(async () => keyNames(page)).not.toContain('UI_DELETE_ME');
  await page.reload();
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect.poll(async () => keyNames(page)).not.toContain('UI_DELETE_ME');
});

test('sidebar adds a Managed File', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('add-file-name').fill('.env.ui-added');
  await page.getByTestId('add-file').click();
  await expect(page.getByTestId('editor-error')).toBeHidden();
  await expect(page.getByTestId('managed-file').filter({ hasText: '.env.ui-added' })).toBeVisible();
  await expect(page.getByTestId('breadcrumb')).toContainText('.env.ui-added');
  await expect(page.getByTestId('key-composer')).toBeVisible();
});

test('sidebar rejects a Managed File path outside the Project', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('add-file-name').fill('../escape.env');
  await page.getByTestId('add-file').click();
  const error = page.getByTestId('editor-error');
  await expect(error).toBeVisible();
  await expect(error).toContainText('inside the Project');
});

test('Publish inspector shows mapping and prune off', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('publish-prefix')).toHaveValue('SD_');
  await expect(page.getByTestId('publish-repo')).toContainText('studio/demo');
  await expect(page.getByTestId('publish-environment')).toHaveText('—');
  await expect(page.getByTestId('publish-prune')).not.toBeChecked();
});

test('window does not scroll empty body chrome', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const metrics = await page.evaluate(() => {
    const el = document.scrollingElement;
    return { scrollHeight: el.scrollHeight, clientHeight: el.clientHeight };
  });
  expect(metrics.scrollHeight).toBeLessThanOrEqual(metrics.clientHeight + 8);
});

async function dispatchPaste(page, text) {
  await page.getByTestId('keys').evaluate((el, value) => {
    const dt = new DataTransfer();
    dt.setData('text/plain', value);
    const event = new ClipboardEvent('paste', { bubbles: true, cancelable: true });
    Object.defineProperty(event, 'clipboardData', { value: dt });
    el.dispatchEvent(event);
  }, text);
}

test('bulk paste previews key names without values', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await dispatchPaste(page, 'NEW=supersecretvalue\n');
  const preview = page.getByTestId('paste-preview');
  await expect(preview).toBeVisible();
  await expect(preview).toContainText('NEW');
  await expect(preview).not.toContainText('supersecretvalue');
  await page.getByTestId('paste-confirm').click();
  await expect(preview).toBeHidden();
  await expect.poll(async () => keyNames(page)).toContain('NEW');
  await expect(page.getByTestId('save')).toBeEnabled();
});
