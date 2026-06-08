const fs = require('fs');
const path = require('path');

let YAML;
try {
  YAML = require(path.join(process.cwd(), 'frontend', 'node_modules', 'yaml'));
} catch (error) {
  console.log(`SKIPPED: yaml parser is unavailable: ${error.message}`);
  process.exit(0);
}

const workflowDir = path.join(process.cwd(), '.github', 'workflows');
const files = fs
  .readdirSync(workflowDir)
  .filter((name) => /^audit.*\.ya?ml$/.test(name))
  .sort();

for (const file of files) {
  const fullPath = path.join(workflowDir, file);
  YAML.parse(fs.readFileSync(fullPath, 'utf8'));
  console.log(`ok ${file}`);
}
