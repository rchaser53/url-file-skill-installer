---
name: url-file-skill-installer
description: Install Codex skills from git repository URLs listed in a local file. Use when the user provides or references a repository list file and wants to fetch and install those skills into the current project's .codex/skills directory.
---

# URL File Skill Installer

Install skills from a YAML file that contains git repository URLs as a string array.

## When to use

Use this skill when a user asks to install skills from a URL list file.

## Input format

The URL file must be YAML.

- Use YAML string array items (`- "..."`)
- Empty lines are ignored
- Lines starting with `#` are ignored
- Optional destination name syntax inside each string: `URL -> skill_name`

## File format for registration

When registering this skill in `skills.sh` or documenting how to use it, describe the input file as a YAML file that contains repository URLs in a string array.

Supported file shapes:

### 1. Top-level array

```yaml
- "https://github.com/rchaser53/summarize-website"
- "git@github.com:org/private-skill.git -> my-private-skill"
```

### 2. `skills` key with an array

```yaml
skills:
 - "https://github.com/rchaser53/summarize-website"
 - "git@github.com:org/private-skill.git -> my-private-skill"
```

Each item is interpreted as one install target.

- Plain URL: installs using the source directory name
- `URL -> skill_name`: installs using the explicit destination name

Unsupported examples:

- Direct `SKILL.md` URLs
- Zip archive URLs
- Arbitrary YAML objects instead of string array items

Supported source types:

- Git repository URL (example: `https://github.com/org/repo.git`)
- GitHub repository URL (example: `https://github.com/org/repo`)

Example `codex-skils.yml`:

```yaml
skills:
 - "https://github.com/rchaser53/summarize-website"
 - "git@github.com:org/private-skill.git -> my-private-skill"
```

## Install workflow

1. Confirm URL file path.
2. Run (default file: project root `codex-skills.yml`, fallback: `codex-skils.yml`):

```bash
bash scripts/install_skills_from_url_file.sh
```

If you want to use another file:

```bash
bash scripts/install_skills_from_url_file.sh /path/to/codex-skils.yml
```

3. Verify installed skills under `./.codex/skills` (project root).

## Notes

- Existing destination directories are replaced.
- The installer clones each repository into a temporary workspace and cleans it automatically.
- `git` must be available in the environment.

