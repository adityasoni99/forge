import { parse as parseYaml } from 'yaml';

export interface SkillFrontmatter {
  name: string;
  version: string;
  description: string;
  when_to_use: string;
  eval_score: number;
  tags: string[];
}

export interface Skill {
  name: string;
  version: string;
  description: string;
  whenToUse: string;
  evalScore: number;
  tags: string[];
  bodyPath: string;
  body: string;
}

const FRONTMATTER_REGEX = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?([\s\S]*)$/;

export function parseFrontmatter(raw: string): { frontmatter: SkillFrontmatter; body: string } {
  const match = raw.match(FRONTMATTER_REGEX);
  if (!match) {
    throw new Error('SKILL.md missing YAML frontmatter (expected --- delimiters)');
  }

  const parsed = parseYaml(match[1]) as Record<string, unknown>;
  const frontmatter: SkillFrontmatter = {
    name: String(parsed.name ?? ''),
    version: String(parsed.version ?? '1.0'),
    description: String(parsed.description ?? ''),
    when_to_use: String(parsed.when_to_use ?? ''),
    eval_score: Number(parsed.eval_score ?? 0),
    tags: Array.isArray(parsed.tags) ? parsed.tags.map(String) : [],
  };

  if (!frontmatter.name) {
    throw new Error('SKILL.md frontmatter missing required field: name');
  }

  return { frontmatter, body: match[2].trim() };
}

export function frontmatterToSkill(
  fm: SkillFrontmatter,
  body: string,
  bodyPath: string,
): Skill {
  return {
    name: fm.name,
    version: fm.version,
    description: fm.description,
    whenToUse: fm.when_to_use,
    evalScore: fm.eval_score,
    tags: fm.tags,
    bodyPath,
    body,
  };
}
