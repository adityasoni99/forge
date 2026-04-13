import * as fs from 'node:fs/promises';
import * as path from 'node:path';
import type { Skill } from './types.js';
import { parseFrontmatter, frontmatterToSkill } from './types.js';

export class SkillRegistry {
  private skills: Skill[] = [];

  async loadAll(skillsDir: string): Promise<Skill[]> {
    this.skills = [];
    const skillPaths = await findSkillFiles(skillsDir);
    for (const skillPath of skillPaths) {
      try {
        const raw = await fs.readFile(skillPath, 'utf-8');
        const { frontmatter, body } = parseFrontmatter(raw);
        this.skills.push(frontmatterToSkill(frontmatter, body, skillPath));
      } catch {
        // Skip malformed skill files
      }
    }
    return [...this.skills];
  }

  findByName(name: string): Skill | undefined {
    return this.skills.find((s) => s.name === name);
  }

  findByTag(tag: string): Skill[] {
    return this.skills.filter((s) => s.tags.includes(tag));
  }

  all(): Skill[] {
    return [...this.skills];
  }
}

async function findSkillFiles(dir: string): Promise<string[]> {
  const results: string[] = [];
  try {
    const entries = await fs.readdir(dir, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      const subEntries = await fs.readdir(path.join(dir, entry.name), { withFileTypes: true });
      for (const sub of subEntries) {
        if (sub.isDirectory()) {
          const skillFile = path.join(dir, entry.name, sub.name, 'SKILL.md');
          try {
            await fs.access(skillFile);
            results.push(skillFile);
          } catch {
            // No SKILL.md in this directory
          }
        }
      }
    }
  } catch {
    // Directory doesn't exist or can't be read
  }
  return results.sort();
}
