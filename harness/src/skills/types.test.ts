import { describe, it, expect } from 'vitest';
import { parseFrontmatter, frontmatterToSkill } from './types.js';

const VALID_SKILL = `---
name: implement-feature
version: "1.0"
description: Basic implementation skill
when_to_use: When implementing a new feature
eval_score: 0.85
tags:
  - coding
  - implementation
---
# Implement Feature

You are a skilled software developer. Implement the requested feature.
`;

describe('parseFrontmatter', () => {
  it('parses valid SKILL.md content', () => {
    const { frontmatter, body } = parseFrontmatter(VALID_SKILL);
    expect(frontmatter.name).toBe('implement-feature');
    expect(frontmatter.version).toBe('1.0');
    expect(frontmatter.description).toBe('Basic implementation skill');
    expect(frontmatter.when_to_use).toBe('When implementing a new feature');
    expect(frontmatter.eval_score).toBe(0.85);
    expect(frontmatter.tags).toEqual(['coding', 'implementation']);
    expect(body).toContain('You are a skilled software developer');
  });

  it('throws on missing frontmatter delimiters', () => {
    expect(() => parseFrontmatter('# No frontmatter')).toThrow('missing YAML frontmatter');
  });

  it('throws on missing name', () => {
    const content = `---
version: "1.0"
description: No name
---
Body text`;
    expect(() => parseFrontmatter(content)).toThrow('missing required field: name');
  });

  it('provides defaults for optional fields', () => {
    const content = `---
name: minimal
---
Body`;
    const { frontmatter } = parseFrontmatter(content);
    expect(frontmatter.version).toBe('1.0');
    expect(frontmatter.eval_score).toBe(0);
    expect(frontmatter.tags).toEqual([]);
  });
});

describe('frontmatterToSkill', () => {
  it('converts frontmatter + body to Skill object', () => {
    const { frontmatter, body } = parseFrontmatter(VALID_SKILL);
    const skill = frontmatterToSkill(frontmatter, body, '/skills/coding/implement-feature/SKILL.md');
    expect(skill.name).toBe('implement-feature');
    expect(skill.whenToUse).toBe('When implementing a new feature');
    expect(skill.bodyPath).toBe('/skills/coding/implement-feature/SKILL.md');
    expect(skill.body).toContain('skilled software developer');
  });
});
