#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
skill_name="open-browser-use"
skill_dir="${repo_root}/skills/${skill_name}"
dist_dir="${repo_root}/dist/skills"
zip_path="${dist_dir}/${skill_name}-skill.zip"
skill_path="${dist_dir}/${skill_name}.skill"
manifest_path="${dist_dir}/package-manifest.json"

source "${repo_root}/scripts/runtime-tools.sh"

node_bin="$(find_node || true)"
if [[ -z "${node_bin}" ]]; then
  echo "node is required to validate the skill package" >&2
  exit 1
fi

zip_directory_contents() {
  local source_dir="$1"
  local output_path="$2"

  "${node_bin}" "${repo_root}/scripts/zip-tool.mjs" create "${source_dir}" "${output_path}"
}

list_zip_entries() {
  local archive_path="$1"

  "${node_bin}" "${repo_root}/scripts/zip-tool.mjs" list "${archive_path}"
}

if [[ ! -f "${skill_dir}/SKILL.md" ]]; then
  echo "missing skill entrypoint: ${skill_dir}/SKILL.md" >&2
  exit 1
fi

"${node_bin}" - "${skill_dir}/SKILL.md" <<'NODE'
const fs = require("fs");

const skillPath = process.argv[2];
const content = fs.readFileSync(skillPath, "utf8");
const errors = [];

if (!content.startsWith("---\n")) {
  errors.push("SKILL.md must start with YAML frontmatter");
}
if (!/^name:\s*open-browser-use\s*$/m.test(content)) {
  errors.push("SKILL.md frontmatter must include name: open-browser-use");
}
if (!/^description:\s*\S/m.test(content)) {
  errors.push("SKILL.md frontmatter must include a non-empty description");
}

if (errors.length > 0) {
  for (const error of errors) {
    console.error(error);
  }
  process.exit(1);
}
NODE

rm -rf "${dist_dir}"
mkdir -p "${dist_dir}"
staging_dir="${dist_dir}/skill-staging"
mkdir -p "${staging_dir}"
cp -R "${skill_dir}" "${staging_dir}/${skill_name}"

zip_directory_contents "${staging_dir}" "${zip_path}"
rm -rf "${staging_dir}"
cp "${zip_path}" "${skill_path}"

"${node_bin}" - "${zip_path}" "${skill_path}" "${manifest_path}" <<'NODE'
const crypto = require("crypto");
const fs = require("fs");

const zipPath = process.argv[2];
const skillPath = process.argv[3];
const manifestPath = process.argv[4];
const zip = fs.readFileSync(zipPath);
const skill = fs.readFileSync(skillPath);

const payload = {
  name: "open-browser-use",
  rootDirectory: "open-browser-use",
  artifacts: {
    zip: zipPath,
    skill: skillPath
  },
  sha256: {
    zip: crypto.createHash("sha256").update(zip).digest("hex"),
    skill: crypto.createHash("sha256").update(skill).digest("hex")
  },
  generatedAtUtc: new Date().toISOString()
};

if (payload.sha256.zip !== payload.sha256.skill) {
  throw new Error("open-browser-use-skill.zip and open-browser-use.skill must contain identical bytes");
}

fs.writeFileSync(manifestPath, `${JSON.stringify(payload, null, 2)}\n`);
NODE

if ! cmp -s "${zip_path}" "${skill_path}"; then
  echo "open-browser-use-skill.zip and open-browser-use.skill differ" >&2
  exit 1
fi

has_skill_entrypoint=0
entry_count=0
while IFS= read -r entry; do
  entry_count=$((entry_count + 1))
  if [[ "${entry}" != "${skill_name}/"* ]]; then
    echo "skill zip entry must be under ${skill_name}/: ${entry}" >&2
    exit 1
  fi
  if [[ "${entry}" == "${skill_name}/SKILL.md" ]]; then
    has_skill_entrypoint=1
  fi
done < <(list_zip_entries "${zip_path}")

if [[ "${entry_count}" -eq 0 ]]; then
  echo "skill zip is empty" >&2
  exit 1
fi

if [[ "${has_skill_entrypoint}" -ne 1 ]]; then
  echo "skill zip is missing ${skill_name}/SKILL.md" >&2
  exit 1
fi

echo "${zip_path}"
echo "${skill_path}"
