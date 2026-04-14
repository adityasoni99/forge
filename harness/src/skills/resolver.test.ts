import { describe, it, expect } from 'vitest';
import { SkillResolver } from './resolver.js';
import type { Skill } from './types.js';

function makeSkill(overrides: Partial<Skill>): Skill {
  return {
    name: 'default',
    version: '1.0',
    description: '',
    whenToUse: '',
    evalScore: 0,
    tags: [],
    bodyPath: '',
    body: '',
    ...overrides,
  };
}

const SKILLS: Skill[] = [
  makeSkill({
    name: 'implement-feature',
    description: 'Implement a new feature from scratch',
    whenToUse: 'When implementing new features or adding functionality',
    tags: ['coding', 'implementation'],
    evalScore: 0.9,
  }),
  makeSkill({
    name: 'code-review',
    description: 'Review code for quality and correctness',
    whenToUse: 'When reviewing pull requests or code changes',
    tags: ['review', 'quality'],
    evalScore: 0.85,
  }),
  makeSkill({
    name: 'bug-fix',
    description: 'Debug and fix software bugs',
    whenToUse: 'When fixing bugs or resolving errors',
    tags: ['debugging', 'fix'],
    evalScore: 0.8,
  }),
];

describe('SkillResolver', () => {
  it('resolves by exact name', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('anything', { skill: 'code-review' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('code-review');
  });

  it('returns null for unknown exact name', () => {
    const resolver = new SkillResolver(SKILLS);
    expect(resolver.resolve('anything', { skill: 'nonexistent' })).toBeNull();
  });

  it('auto-resolves based on task description keywords', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('Review the authentication module code', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('code-review');
  });

  it('auto-resolves implementation task', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('Implement user registration feature', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('implement-feature');
  });

  it('auto-resolves bug fix task', () => {
    const resolver = new SkillResolver(SKILLS);
    const result = resolver.resolve('Fix the login error when password is empty', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('bug-fix');
  });

  it('returns null when no skill matches auto', () => {
    const resolver = new SkillResolver(SKILLS);
    expect(resolver.resolve('Do something completely unrelated', { skill: 'auto' })).toBeNull();
  });

  it('returns null when skill key is absent from config', () => {
    const resolver = new SkillResolver(SKILLS);
    expect(resolver.resolve('Implement something', {})).toBeNull();
  });

  it('prefers higher evalScore on tie', () => {
    const skills = [
      makeSkill({ name: 'a', description: 'implement things', whenToUse: 'implement', evalScore: 0.7, tags: ['coding'] }),
      makeSkill({ name: 'b', description: 'implement things', whenToUse: 'implement', evalScore: 0.9, tags: ['coding'] }),
    ];
    const resolver = new SkillResolver(skills);
    const result = resolver.resolve('implement a widget', { skill: 'auto' });
    expect(result).toBeDefined();
    expect(result!.name).toBe('b');
  });
});
