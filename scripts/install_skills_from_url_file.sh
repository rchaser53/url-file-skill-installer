#!/usr/bin/env bash
set -euo pipefail

usage() {
 cat <<'EOF'
Usage:
 install_skills_from_url_file.sh [url_list_file] [install_dir]

Arguments:
 url_list_file  Optional. Defaults to ./codex-skills.yml (fallback: ./codex-skils.yml)
 install_dir    Optional target directory (default: $PWD/.codex/skills)

URL list format:
 - YAML string array
 - Empty lines and lines starting with # are ignored
 - Optional alias syntax inside each string: URL -> skill_name
 - Supported sources:
     * Git repository URLs (e.g. https://github.com/org/repo.git)
     * GitHub repository URLs (e.g. https://github.com/org/repo)
EOF
}

require_cmd() {
 if ! command -v "$1" >/dev/null 2>&1; then
   echo "Error: required command not found: $1" >&2
   exit 1
 fi
}

trim() {
 local s="$1"
 s="${s#"${s%%[![:space:]]*}"}"
 s="${s%"${s##*[![:space:]]}"}"
 printf '%s' "$s"
}

strip_wrapping_quotes() {
 local s="$1"
 if [[ "$s" =~ ^\".*\"$ ]]; then
   s="${s:1:${#s}-2}"
 elif [[ "$s" =~ ^\'.*\'$ ]]; then
   s="${s:1:${#s}-2}"
 fi
 printf '%s' "$s"
}

iter_yaml_string_array_items() {
 local yaml_file="$1"
 local from_skills
 from_skills="$(
   awk '
     function indent_of(s,   i) {
       i = match(s, /[^ ]/)
       return (i == 0 ? length(s) : i - 1)
     }

     /^[[:space:]]*#/ { next }
     /^[[:space:]]*$/ { next }

     /^[[:space:]]*skills:[[:space:]]*$/ {
       in_skills = 1
       skills_indent = indent_of($0)
       next
     }

     {
       if (!in_skills) next
       current_indent = indent_of($0)
       if (current_indent <= skills_indent) {
         in_skills = 0
         next
       }
       if ($0 ~ /^[[:space:]]*-[[:space:]]*/) {
         line = $0
         sub(/^[[:space:]]*-[[:space:]]*/, "", line)
         print line
       }
     }
   ' "$yaml_file"
 )"

 if [[ -n "$from_skills" ]]; then
   printf '%s\n' "$from_skills"
   return 0
 fi

 awk '
   /^[[:space:]]*#/ { next }
   /^[[:space:]]*$/ { next }
   /^[[:space:]]*-[[:space:]]*/ {
     line = $0
     sub(/^[[:space:]]*-[[:space:]]*/, "", line)
     print line
   }
 ' "$yaml_file"
}

install_dir_from_source() {
 local source_dir="$1"
 local target_root="$2"
 local alias_name="${3:-}"
 local skill_name

 if [[ -n "$alias_name" ]]; then
   skill_name="$alias_name"
 else
   skill_name="$(basename "$source_dir")"
 fi

 if [[ ! -f "$source_dir/SKILL.md" ]]; then
   echo "Skip: missing SKILL.md in $source_dir" >&2
   return 0
 fi

 mkdir -p "$target_root"
 local destination="$target_root/$skill_name"
 rm -rf "$destination"
 cp -R "$source_dir" "$destination"
 echo "Installed: $skill_name -> $destination"
}

looks_like_git_repo_url() {
 local url="$1"

 [[ "$url" == *.zip* ]] && return 1
 [[ "$url" == *SKILL.md* ]] && return 1
 [[ "$url" == *.md ]] && return 1
 [[ "$url" == git@*:* ]] && return 0
 [[ "$url" == ssh://* ]] && return 0
 [[ "$url" =~ ^https?://github\.com/[^/]+/[^/]+/?(\.git)?$ ]] && return 0
 [[ "$url" =~ ^https?://gitlab\.com/[^/]+/[^/]+/?(\.git)?$ ]] && return 0
 [[ "$url" =~ ^https?://bitbucket\.org/[^/]+/[^/]+/?(\.git)?$ ]] && return 0
 [[ "$url" =~ ^https?://.+\.git/?$ ]] && return 0

 return 1
}

install_from_git_repo_url() {
 local url="$1"
 local target_root="$2"
 local alias_name="${3:-}"
 local tmp_root="$4"
 local repo_dir="$tmp_root/repo"

 require_cmd git

 if ! git clone --depth 1 "$url" "$repo_dir" >/dev/null 2>&1; then
   echo "Error: failed to clone git repository: $url" >&2
   return 1
 fi

 local found=0
 while IFS= read -r -d '' skill_md; do
   found=1
   local skill_dir
   skill_dir="$(dirname "$skill_md")"
   install_dir_from_source "$skill_dir" "$target_root" "$alias_name"
   alias_name=""
 done < <(find "$repo_dir" -type f -name 'SKILL.md' -print0)

 if [[ "$found" -eq 0 ]]; then
   echo "Error: no SKILL.md found in git repository: $url" >&2
   return 1
 fi
}

main() {
 if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
   usage
   exit 0
 fi

 if [[ $# -gt 2 ]]; then
   usage >&2
   exit 1
 fi

 local url_list_file="${1:-}"
 local install_dir="${2:-$PWD/.codex/skills}"

 if [[ -z "$url_list_file" ]]; then
   if [[ -f "$PWD/codex-skills.yml" ]]; then
     url_list_file="$PWD/codex-skills.yml"
   else
     url_list_file="$PWD/codex-skils.yml"
   fi
 fi

 if [[ ! -f "$url_list_file" ]]; then
   echo "Error: file not found: $url_list_file" >&2
   exit 1
 fi

 mkdir -p "$install_dir"

 local line url alias_name
 local n=0
 while IFS= read -r line || [[ -n "$line" ]]; do
   line="$(trim "$line")"
   line="$(strip_wrapping_quotes "$line")"
   [[ -z "$line" ]] && continue

   url="$line"
   alias_name=""

   if [[ "$line" == *"->"* ]]; then
     url="$(trim "${line%%->*}")"
     alias_name="$(trim "${line##*->}")"
   fi

   if [[ -z "$url" ]]; then
     echo "Skip: empty URL line" >&2
     continue
   fi

   n=$((n + 1))
   echo "[$n] Processing: $url"

   local tmp_root
   tmp_root="$(mktemp -d)"

   if looks_like_git_repo_url "$url"; then
     if ! install_from_git_repo_url "$url" "$install_dir" "$alias_name" "$tmp_root"; then
       rm -rf "$tmp_root"
       exit 1
     fi
   else
     echo "Error: unsupported source (expected git repository URL): $url" >&2
     rm -rf "$tmp_root"
     exit 1
   fi

   rm -rf "$tmp_root"
 done < <(iter_yaml_string_array_items "$url_list_file")

 if [[ "$n" -eq 0 ]]; then
   echo "Error: no YAML array items found in $url_list_file" >&2
   exit 1
 fi

 echo "Done. Installed skills are in: $install_dir"
}

main "$@"
