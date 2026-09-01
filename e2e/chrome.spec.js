import { expect, test } from '@playwright/test';

async function encryptAndSave(page) {
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save-preview-dialog')).toBeVisible();
  await page.getByTestId('save-preview-confirm').click();
  await expect(page.getByTestId('save-preview-dialog')).toBeHidden();
}

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
    if (data?.cmd === 'inspect_project') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ result: { initialized: true, managed: [], candidates: [] } }),
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

test('GitHub integration offers Sync now', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('github-integration').click();
  await expect(page.getByTestId('integration-dialog')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Sync now' })).toBeVisible();
});

test('robot identity stays open with a copyable private key', async ({ page }) => {
  const commands = [];
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd === 'create_robot_identity') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          result: {
            name: 'Deploy bot',
            public_key: 'age1test',
            private_key: '# public key: age1test\nAGE-SECRET-KEY-TEST',
          },
        }),
      });
      return;
    }
    if (data?.cmd === 'add_recipient' || data?.cmd === 'list_file_access') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ result: data.cmd === 'list_file_access' ? [] : null }),
      });
      return;
    }
    await route.continue();
  });
  page.on('request', (request) => {
    if (!request.url().endsWith('/invoke')) return;
    try {
      commands.push(request.postDataJSON()?.cmd);
    } catch {
      // Ignore non-JSON requests.
    }
  });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByRole('button', { name: 'Add bot / integration account' }).click();
  await page.getByTestId('robot-dialog').getByPlaceholder('Deploy bot').fill('Deploy bot');
  await page.getByRole('button', { name: 'Create identity' }).click();
  await expect(page.getByTestId('robot-dialog')).toBeVisible();
  await expect(page.getByTestId('robot-private-key')).toHaveValue(/AGE-SECRET-KEY-/);
  expect(commands.filter((command) => command === 'copy_text')).toHaveLength(0);
});

test('save preview lists edited keys before commit', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const row = await keyRowByName(page, 'STRIPE_SECRET');
  await row.getByTestId('reveal-key').click();
  await row.getByTestId('key-value').fill('sk_live_demo');
  await expect(page.getByTestId('save')).toBeEnabled();
  await page.getByTestId('save').click();
  await expect(page.getByTestId('save-preview')).toContainText('STRIPE_SECRET');
  await page.getByTestId('save-preview-confirm').click();
  await expect(page.getByTestId('save')).toBeDisabled();
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
  await expect(page.getByTestId('whats-new-tag').first()).toHaveText('Feature');
  await expect(page.getByTestId('whats-new-platform').first()).toHaveText('macOS');
});

test('save preview shows a plaintext secret diff', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const row = await keyRowByName(page, 'STRIPE_SECRET');
  await row.getByTestId('reveal-key').click();
  await row.getByTestId('key-value').fill('sk_review_demo');
  await expect(page.getByTestId('save')).toBeEnabled();
  await page.getByTestId('save').click();
  const out = page.getByTestId('save-preview');
  await expect(out).toBeVisible();
  await expect(out).toContainText('STRIPE_SECRET');
  await expect(out).not.toContainText('ENC[');
  await page.getByTestId('save-preview-cancel').click();
});

test('History lists commits without secret values', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('file-history').click();
  const list = page.getByTestId('file-history-list');
  await expect(list.locator('button')).not.toHaveCount(0);
  await expect(list).not.toContainText('sk_test_demo');
});

test('composer is visible when a Managed File is open', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('key-composer')).toBeVisible();
});

test('key name on an existing row is editable', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const name = (await keyRowByName(page, 'STRIPE_SECRET')).getByTestId('key-name');
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
  await encryptAndSave(page);
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
  await encryptAndSave(page);
  await expect(page.getByTestId('save')).toBeDisabled();
  const row = await keyRowByName(page, 'UI_DELETE_ME');
  await row.getByTestId('delete-key').click();
  await expect(page.getByTestId('save')).toBeEnabled();
  await encryptAndSave(page);
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

test('nested folders group and collapse', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('add-file-name').fill('apps/web/.env.nested');
  await page.getByTestId('add-file').click();
  await expect(page.getByTestId('editor-error')).toBeHidden();
  const folder = page.getByTestId('tree-folder').filter({ hasText: 'apps/web' });
  await expect(folder).toBeVisible();
  const nested = page.getByTestId('managed-file').filter({ hasText: '.env.nested' });
  await expect(nested).toBeVisible();
  await folder.click();
  await expect(nested).toBeHidden();
});

test('recents reopen a Project without the folder picker', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.goto('/?empty=1');
  await expect(page.getByTestId('empty-state')).toBeVisible();
  await page.getByTestId('recent-project').filter({ hasText: 'checkout' }).click();
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(
    page.getByTestId('managed-file').filter({ hasText: '.env.production' }),
  ).toBeVisible();
});

test('long file lists truncate with Show more', async ({ page }) => {
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd !== 'inspect_project') {
      await route.continue();
      return;
    }

    const response = await route.fetch();
    const payload = await response.json();
    const state = payload.result || {};
    const files = [...(state.managed || [])];
    const root = files[0]?.path?.replace(/\/[^/]+$/u, '') || '/tmp';
    for (let i = 0; i < 10; i++) {
      const name = `.env.zz-${String(i).padStart(2, '0')}`;
      files.push({ name, path: `${root}/${name}`, rel: name, managed: true });
    }

    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ result: { ...state, managed: files } }),
    });
  });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const extra = page.getByTestId('managed-file').filter({ hasText: '.env.zz-09' });
  await expect(extra).toHaveCount(0);
  await page.getByTestId('tree-show-more').click();
  await expect(extra).toBeVisible();
});

test('demo seed shows several Projects with extras collapsed', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('tree-project').filter({ hasText: 'checkout' })).toBeVisible();
  const atlas = page.getByTestId('tree-project').filter({ hasText: 'atlas-web' });
  const docs = page.getByTestId('tree-project').filter({ hasText: 'docs-site' });
  await expect(atlas).toBeVisible();
  await expect(docs).toBeVisible();
  await expect(atlas).toHaveAttribute('aria-expanded', 'false');
  await expect(docs).toHaveAttribute('aria-expanded', 'false');
  await atlas.click();
  await expect(atlas).toHaveAttribute('aria-expanded', 'true');
  await expect(page.getByTestId('managed-file').filter({ hasText: 'eas.json' })).toBeVisible();
});

test('structured editor hides plaintext fields and edits encrypted paths in a modal', async ({
  page,
}) => {
  await page.route('**/invoke', async (route) => {
    const request = route.request().postDataJSON();
    if (request?.cmd === 'get_managed_file' && request.path?.endsWith('/eas.json')) {
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          ok: true,
          result: [
            { key: 'EXPO_PUBLIC_API_URL', value: 'https://example.test', encrypted: true },
            { key: 'cli.appVersionSource', value: 'remote', encrypted: false },
          ],
        }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('managed-file').filter({ hasText: 'eas.json' }).click();

  const tree = page.getByTestId('json-tree');
  await expect(tree).toContainText('EXPO_PUBLIC_API_URL');
  await expect(tree).not.toContainText('appVersionSource');

  await page.getByTestId('edit-encrypted-paths').click();
  const dialog = page.getByTestId('setup-project-dialog');
  await expect(dialog).toBeVisible();
  await dialog.getByTestId('setup-project-file-disclosure').click();
  const field = dialog.locator(
    '[data-testid="setup-project-key-toggle"][value="cli.appVersionSource"]',
  );
  await expect(field).not.toBeChecked();
  await field.check();
  await dialog.getByTestId('setup-project-init').click();
  await expect(dialog).toBeHidden();
  await expect(tree).toContainText('appVersionSource');
});

test('project file selector adds and removes managed files', async ({ page }) => {
  let projectPath = '';
  let updated = false;
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    const file = (rel, managed = false, keys = []) => ({
      name: rel.split('/').at(-1),
      path: projectPath + '/' + rel,
      rel,
      managed,
      keys,
    });
    if (data?.cmd === 'inspect_project') {
      projectPath = data.path;
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({
          result: {
            initialized: true,
            managed: updated ? [file('eas.json', true)] : [file('.env.production', true)],
            candidates: updated
              ? [file('.env.production')]
              : [file('eas.json', false, ['build.env.EXPO_TOKEN'])],
          },
        }),
      });
      return;
    }
    if (data?.cmd === 'remove_project_file' || data?.cmd === 'add_project_file') {
      updated = true;
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ result: 'managed file updated' }),
      });
      return;
    }
    if (data?.cmd === 'get_managed_file') {
      await route.fulfill({
        contentType: 'application/json',
        body: JSON.stringify({ result: [] }),
      });
      return;
    }
    await route.continue();
  });

  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('edit-project-files').click();
  const dialog = page.getByTestId('setup-project-dialog');
  await expect(dialog.locator('.kicker')).toHaveText('Managed files');
  await expect(dialog.locator('input[value=".env.production"]')).toBeChecked();
  await expect(dialog.locator('input[value="eas.json"]')).not.toBeChecked();
  await dialog.locator('input[value=".env.production"]').uncheck();
  await dialog.locator('input[value="eas.json"]').check();
  await dialog.getByTestId('setup-project-init').click();
  await expect(dialog).toBeHidden();
  await expect(page.getByTestId('managed-file').filter({ hasText: 'eas.json' })).toBeVisible();
  await expect(
    page.getByTestId('managed-file').filter({ hasText: '.env.production' }),
  ).toBeHidden();
});

test('Publish inspector opens GitHub configuration', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('github-integration').click();
  const dialog = page.getByTestId('integration-dialog');
  await expect(dialog).toBeVisible();
  await expect(page.locator('#integration-prefix')).toHaveValue('SD_');
  await expect(page.locator('#integration-repo')).toHaveValue('studio/demo');
  await expect(page.locator('#integration-prune')).not.toBeChecked();
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

test('Add secret toolbar is gone; composer remains', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('add-secret')).toHaveCount(0);
  await expect(page.getByTestId('key-composer')).toBeVisible();
});

test('values column heading reveals and hides values', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const value = page.getByTestId('key-value').first();
  await expect(value).toHaveValue(/•/);
  await page.getByTestId('reveal').click();
  await expect(value).not.toHaveValue(/•/);
  await page.getByTestId('reveal').click();
  await expect(value).toHaveValue(/•/);
});

test('inspector sections collapse and persist', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('inspector-path')).toBeVisible();
  await page.getByTestId('inspector-toggle-file').click();
  await expect(page.getByTestId('inspector-path')).toBeHidden();
  await page.reload();
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('inspector-path')).toBeHidden();
});

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

test('under development banner links to GitHub issues', async ({ page }) => {
  await page.goto('/?empty=1');
  const banner = page.getByTestId('dev-banner');
  await expect(banner).toBeVisible();
  await expect(banner).toContainText('under development');
  await expect(banner.getByRole('link', { name: 'Submit a GitHub issue' })).toHaveAttribute(
    'href',
    'https://github.com/sopsdeck/sopsdeck/issues/new',
  );
});

test('lock badge follows file status', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('file-badge')).toHaveText('Locked');
});

test('project panel shows the open Project', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('project-panel-name')).toHaveText('checkout');
  await expect(page.getByTestId('docs-link')).toHaveAttribute('href', 'https://sopsdeck.com/docs/');
});

test('account modal copies the Age public key and an access request', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('account').click();
  const key = page.getByTestId('account-public-key');
  await expect(key).toBeVisible();
  await expect(key).toHaveValue(/age1/);
  await expect(page.getByTestId('account-request')).toBeVisible();
  await expect(page.getByTestId('account-request-template')).toContainText('Age public key');
  await page.getByTestId('account-copy-key').click();
  await expect(page.getByTestId('account-status')).toContainText('Copied your Age public key');
  await expect(page.getByTestId('access-status')).toBeHidden();
});

test('sidebar stacks Project name above the path', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const project = page.getByTestId('tree-project').filter({ hasText: 'checkout' });
  await expect(project.locator('.project-name')).toHaveText('checkout');
  await expect(project.locator('.project-path')).toBeVisible();
});

test('access empty actions stay hidden when recipients exist', async ({ page }) => {
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd === 'list_file_access') {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          result: [{ name: 'Bob', key: 'age1bobexample', kind: 'person', self: false }],
        }),
      });
      return;
    }

    await route.continue();
  });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.getByTestId('access-list')).toBeVisible();
  await expect(page.getByRole('button', { name: 'Add team member' })).toBeHidden();
});

test('focused Project hides recents and extra folders', async ({ page }) => {
  await page.route('**/demo', async (route) => {
    await route.fulfill({ status: 404, body: 'not found' });
  });
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await expect(page.locator('body')).toHaveClass(/focused-project/);
  await expect(page.getByTestId('recents')).toHaveCount(0);
  await expect(page.getByTestId('add-project')).toBeHidden();
  await expect(page.getByTestId('tree-project').filter({ hasText: 'atlas-web' })).toHaveCount(0);
});

test('missing Access shows a recovery panel instead of a raw error', async ({ page }) => {
  await page.route('**/invoke', async (route) => {
    const data = route.request().postDataJSON();
    if (data?.cmd === 'get_managed_file') {
      await route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'get: no Access to this Managed File' }),
      });
      return;
    }

    await route.continue();
  });
  await page.goto('/');
  await expect(page.getByTestId('access-gate')).toBeVisible();
  await expect(page.getByTestId('access-gate')).toContainText('don’t have Access');
  await expect(page.getByTestId('editor-error')).toBeHidden();
  await page.getByRole('button', { name: 'Open account' }).click();
  await expect(page.getByTestId('account-dialog')).toBeVisible();
});

test('Account reveals a private-key backup on demand', async ({ page }) => {
  await page.goto('/');
  await page.getByTestId('account').click();
  await page.getByRole('button', { name: 'Back up private key' }).click();
  await expect(page.getByTestId('account-backup')).toBeVisible();
  await expect(page.getByTestId('account-private-key')).toHaveValue(/AGE-SECRET-KEY-/);
});

test('JSON files render a tree with encrypt toggles', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  await page.getByTestId('managed-file').filter({ hasText: 'eas.json' }).click();
  await expect(page.getByTestId('json-tree')).toBeVisible();
  await expect(page.getByTestId('encrypt-toggle').first()).toBeVisible();
  await expect(page.getByTestId('file-fields')).toBeVisible();
  await expect(page.locator('.key-head')).not.toContainText('Type');
});

test('secret values can be edited on more than one line', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByTestId('headline')).toHaveText('Production');
  const row = await keyRowByName(page, 'STRIPE_SECRET');
  await row.getByTestId('reveal-key').click();
  const value = row.getByTestId('key-value');
  await expect(value).toHaveJSProperty('tagName', 'TEXTAREA');
  await value.fill('line-one\nline-two');
  await expect(value).toHaveValue('line-one\nline-two');
});
