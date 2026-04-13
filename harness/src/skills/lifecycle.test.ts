import { describe, it, expect } from 'vitest';
import { SkillLifecycle } from './lifecycle.js';
import type { Skill } from './types.js';

function makeSkill(overrides: Partial<Skill> = {}): Skill {
  return {
    name: 'test-skill',
    version: '1.0',
    description: 'A test skill',
    whenToUse: 'When testing',
    evalScore: 0,
    tags: ['test'],
    bodyPath: '/skills/test/SKILL.md',
    body: 'You are a test skill.',
    ...overrides,
  };
}

describe('SkillLifecycle', () => {
  describe('evaluate', () => {
    it('returns passing result when all cases pass', async () => {
      const skill = makeSkill();
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.evaluate(skill, [
        { input: 'Write a hello world', expectedContains: 'hello' },
        { input: 'Add logging', expectedContains: 'log' },
      ]);
      expect(result.passed).toBe(true);
      expect(result.passRate).toBe(1.0);
      expect(result.results).toHaveLength(2);
    });

    it('returns failing result when case fails', async () => {
      const skill = makeSkill();
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.evaluate(skill, [
        { input: 'Write tests', expectedContains: 'IMPOSSIBLE_STRING_NEVER_FOUND' },
      ]);
      expect(result.passed).toBe(false);
      expect(result.passRate).toBe(0);
    });

    it('handles empty test cases', async () => {
      const skill = makeSkill();
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.evaluate(skill, []);
      expect(result.passed).toBe(true);
      expect(result.passRate).toBe(1.0);
    });
  });

  describe('promote', () => {
    it('bumps version and eval score', () => {
      const skill = makeSkill({ version: '1.0', evalScore: 0.7 });
      const lifecycle = new SkillLifecycle();
      const promoted = lifecycle.promote(skill, '2.0', 0.9);
      expect(promoted.version).toBe('2.0');
      expect(promoted.evalScore).toBe(0.9);
      expect(promoted.name).toBe(skill.name);
    });
  });

  describe('compare', () => {
    it('returns comparison showing better skill', async () => {
      const skillA = makeSkill({ name: 'skill-a', body: 'You implement features with tests.' });
      const skillB = makeSkill({ name: 'skill-b', body: 'You implement features.' });
      const lifecycle = new SkillLifecycle();
      const result = await lifecycle.compare(skillA, skillB, [
        { input: 'Add a user model', expectedContains: 'user' },
      ]);
      expect(result.winnerName).toBeDefined();
      expect(result.scoreA).toBeGreaterThanOrEqual(0);
      expect(result.scoreB).toBeGreaterThanOrEqual(0);
    });
  });
});
