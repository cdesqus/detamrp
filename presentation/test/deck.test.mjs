import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import AdmZip from 'adm-zip';
import { deck } from '../src/content.mjs';

test('approved deck has eleven unique, complete slides', () => {
  assert.equal(deck.slides.length, 11);
  assert.equal(new Set(deck.slides.map(s => s.id)).size, 11);
  for (const slide of deck.slides) assert.ok(slide.title?.trim());
});

test('content respects scope guardrails', () => {
  const visible = JSON.stringify(deck);
  assert.doesNotMatch(visible, /sage|accounting integration|pricing|commercial quotation|tbd|todo/i);
  assert.match(visible, /6–7 weeks/);
  assert.equal(deck.slides.at(-1).steps.length, 6);
});

test('generator writes a valid eleven-slide pptx package', async () => {
  const { writePresentation } = await import('../src/generate.mjs');
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'deta-mrp-'));
  const output = path.join(dir, 'deck.pptx');
  await writePresentation(output);
  assert.ok(fs.statSync(output).size > 100_000);
  const zip = new AdmZip(output);
  assert.ok(zip.getEntry('[Content_Types].xml'));
  for (let i=1; i<=11; i++) assert.ok(zip.getEntry(`ppt/slides/slide${i}.xml`));
  assert.equal(zip.getEntry('ppt/slides/slide12.xml'), null);
});
