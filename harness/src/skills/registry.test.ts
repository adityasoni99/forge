import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import * as fs from 'node:fs';
import * as path from 'node:path';
import * as os from 'node:os';
import { SkillRegistry } from './registry.js';

function createSkillDir(baseDir: string, skillPath: string, content: string) {
  const dir = path.join(baseDir, path.dirname(skillPath));
  fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(path.join(baseDir, skillPath), content);
}

const SKILL_A = `---
name: skill-a
version: "1.0"
description: First skill
when_to_use: When doing A
eval_score: 0.9
tags:
  - coding
---
# Skill A
Do the A thing.
`;

const SKILL_B = `---
name: skill-b
version: "2.0"
description: Second skill
when_to_use: When doing B
tags:
  - review
  - quality
---
# Skill B
Do the B thing.
`;

describe('SkillRegistry', () => {
  let tmpDir: string;

  beforeEach(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-registry-'));
    createSkillDir(tmpDir, 'coding/skill-a/SKILL.md', SKILL_A);
    createSkillDir(tmpDir, 'quality/skill-b/SKILL.md', SKILL_B);
  });

  afterEach(() => {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  });

  it('loadAll discovers all skills', async () => {
    const registry = new SkillRegistry();
    const skills = await registry.loadAll(tmpDir);
    expect(skills).toHaveLength(2);
    const names = skills.map((s) => s.name).sort();
    expect(names).toEqual(['skill-a', 'skill-b']);
  });

  it('findByName returns matching skill', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    const skill = registry.findByName('skill-a');
    expect(skill).toBeDefined();
    expect(skill!.name).toBe('skill-a');
    expect(skill!.version).toBe('1.0');
  });

  it('findByName returns undefined for unknown skill', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    expect(registry.findByName('nonexistent')).toBeUndefined();
  });

  it('findByTag returns skills matching tag', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    const review = registry.findByTag('review');
    expect(review).toHaveLength(1);
    expect(review[0].name).toBe('skill-b');
  });

  it('findByTag returns empty for unknown tag', async () => {
    const registry = new SkillRegistry();
    await registry.loadAll(tmpDir);
    expect(registry.findByTag('unknown')).toEqual([]);
  });

  it('handles empty directory', async () => {
    const emptyDir = fs.mkdtempSync(path.join(os.tmpdir(), 'forge-empty-'));
    try {
      const registry = new SkillRegistry();
      const skills = await registry.loadAll(emptyDir);
      expect(skills).toEqual([]);
    } finally {
      fs.rmSync(emptyDir, { recursive: true, force: true });
    }
  });

  it('skips directories without SKILL.md', async () => {
    fs.mkdirSync(path.join(tmpDir, 'empty-dir/no-skill'), { recursive: true });
    fs.writeFileSync(path.join(tmpDir, 'empty-dir/no-skill/README.md'), '# Not a skill');
    const registry = new SkillRegistry();
    const skills = await registry.loadAll(tmpDir);
    expect(skills).toHaveLength(2);
  });
});
