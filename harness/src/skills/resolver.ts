import type { Skill } from './types.js';

interface ResolveConfig {
  skill?: string;
  [key: string]: unknown;
}

export class SkillResolver {
  constructor(private readonly skills: Skill[]) {}

  resolve(taskDescription: string, config: ResolveConfig): Skill | null {
    const skillRef = config.skill;
    if (!skillRef) return null;

    if (skillRef !== 'auto') {
      return this.skills.find((s) => s.name === skillRef) ?? null;
    }

    return this.autoResolve(taskDescription);
  }

  private autoResolve(taskDescription: string): Skill | null {
    const words = taskDescription.toLowerCase().split(/\s+/);
    let bestSkill: Skill | null = null;
    let bestScore = 0;

    for (const skill of this.skills) {
      const score = this.scoreMatch(words, skill);
      if (score > bestScore || (score === bestScore && skill.evalScore > (bestSkill?.evalScore ?? 0))) {
        bestScore = score;
        bestSkill = skill;
      }
    }

    return bestScore > 0 ? bestSkill : null;
  }

  private scoreMatch(taskWords: string[], skill: Skill): number {
    const targets = [
      skill.description.toLowerCase(),
      skill.whenToUse.toLowerCase(),
      ...skill.tags.map((t) => t.toLowerCase()),
      skill.name.replace(/-/g, ' ').toLowerCase(),
    ].join(' ');

    const targetWords = new Set(targets.split(/\s+/));
    let matches = 0;
    for (const word of taskWords) {
      if (word.length < 3) continue;
      if (targetWords.has(word)) {
        matches++;
      }
      for (const tw of targetWords) {
        if (tw.length >= 4 && (tw.includes(word) || word.includes(tw))) {
          matches++;
          break;
        }
      }
    }
    return matches;
  }
}
