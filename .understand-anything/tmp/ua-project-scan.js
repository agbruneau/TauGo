#!/usr/bin/env node
"use strict";

const fs = require("fs");
const path = require("path");
const cp = require("child_process");

function fail(msg) {
  process.stderr.write(String(msg) + "\n");
  process.exit(1);
}

const ROOT = process.argv[2];
const OUT = process.argv[3];
if (!ROOT || !OUT) fail("usage: ua-project-scan.js <projectRoot> <outFile>");

let rootStat;
try {
  rootStat = fs.statSync(ROOT);
} catch (e) {
  fail("cannot access project root: " + e.message);
}
if (!rootStat.isDirectory()) fail("project root is not a directory: " + ROOT);

const ROOT_ABS = path.resolve(ROOT);

// ---------- Step 1: File discovery ----------
function gitLsFiles() {
  try {
    const out = cp.execSync("git ls-files", {
      cwd: ROOT_ABS,
      encoding: "utf8",
      maxBuffer: 64 * 1024 * 1024,
      stdio: ["ignore", "pipe", "ignore"],
    });
    const files = out.split(/\r?\n/).map((s) => s.trim()).filter(Boolean);
    if (files.length === 0) return null;
    return files;
  } catch (e) {
    return null;
  }
}

function walkDir() {
  const result = [];
  const skipDirs = new Set([
    "node_modules", ".git", "vendor", "venv", ".venv", "__pycache__",
    "dist", "build", "out", "coverage", ".next", ".cache", ".turbo",
    "target", "obj", ".idea", ".vscode",
  ]);
  function walk(dir) {
    let entries;
    try {
      entries = fs.readdirSync(dir, { withFileTypes: true });
    } catch (e) {
      return;
    }
    for (const ent of entries) {
      const full = path.join(dir, ent.name);
      if (ent.isDirectory()) {
        if (skipDirs.has(ent.name)) continue;
        walk(full);
      } else if (ent.isFile()) {
        result.push(path.relative(ROOT_ABS, full).split(path.sep).join("/"));
      }
    }
  }
  walk(ROOT_ABS);
  return result;
}

let rawFiles = gitLsFiles();
if (!rawFiles) rawFiles = walkDir();

// Normalize to forward slashes
rawFiles = rawFiles.map((f) => f.split(path.sep).join("/"));

// ---------- Step 2: Exclusion filtering ----------
const BUILD_DIR_SEGMENTS = new Set([
  "dist", "build", "out", "coverage", ".next", ".cache", ".turbo", "target", "obj",
]);
const DEP_DIR_SEGMENTS = new Set([
  "node_modules", ".git", "vendor", "venv", ".venv", "__pycache__",
]);
const BINARY_EXT = new Set([
  ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf",
  ".eot", ".mp3", ".mp4", ".pdf", ".zip", ".tar", ".gz",
]);

function basename(p) {
  return p.split("/").pop();
}
function ext(p) {
  const b = basename(p);
  const i = b.lastIndexOf(".");
  return i <= 0 ? "" : b.slice(i).toLowerCase();
}

function isExcludedDefault(f) {
  const segs = f.split("/");
  const dirSegs = segs.slice(0, -1);
  const base = segs[segs.length - 1];
  const e = ext(f);

  // dependency dirs
  for (const s of dirSegs) {
    if (DEP_DIR_SEGMENTS.has(s)) return true;
  }
  // .git/ anywhere as substring path already covered; also .understand-anything (project default per dispatch)
  if (dirSegs.includes(".understand-anything") || base === ".understand-anything") return true;
  // build output dir segments
  for (const s of dirSegs) {
    if (BUILD_DIR_SEGMENTS.has(s)) return true;
  }
  // IDE config
  if (dirSegs.includes(".idea") || dirSegs.includes(".vscode")) return true;

  // lock files
  if (e === ".lock") return true;
  if (base === "package-lock.json" || base === "yarn.lock" || base === "pnpm-lock.yaml") return true;

  // binary/asset
  if (BINARY_EXT.has(e)) return true;

  // generated
  if (base.endsWith(".min.js") || base.endsWith(".min.css") || e === ".map") return true;
  if (/\.generated\./.test(base)) return true;

  // misc non-source
  if (base === "LICENSE") return true;
  if (base === ".gitignore" || base === ".editorconfig" || base === ".prettierrc") return true;
  if (/^\.eslintrc/.test(base)) return true;
  if (e === ".log") return true;

  // dispatch-specified extra defaults: bin/, *.exe, *.db / ruvector.db, *.out, *.prof, *.pgo, coverage.*
  if (dirSegs.includes("bin")) return true;
  if (e === ".exe" || e === ".db" || e === ".out" || e === ".prof" || e === ".pgo") return true;
  if (/^coverage\./.test(base)) return true;

  return false;
}

const filteredFiles = rawFiles.filter((f) => !isExcludedDefault(f));

// ---------- Step 2.5: .understandignore ----------
// All patterns in the project's .understandignore are commented out, and
// @understand-anything/core is not assumed installed. Defaults already applied.
let filteredByIgnore = 0;

const files = filteredFiles.slice();

// ---------- Step 3: Language detection ----------
const EXT_LANG = {
  ".ts": "typescript", ".tsx": "typescript",
  ".js": "javascript", ".jsx": "javascript",
  ".py": "python", ".go": "go", ".rs": "rust", ".java": "java", ".rb": "ruby",
  ".cpp": "cpp", ".cc": "cpp", ".cxx": "cpp", ".h": "cpp", ".hpp": "cpp",
  ".c": "c", ".cs": "csharp", ".swift": "swift", ".kt": "kotlin", ".php": "php",
  ".vue": "vue", ".svelte": "svelte", ".sh": "shell", ".bash": "shell",
  ".ps1": "powershell", ".bat": "batch", ".cmd": "batch",
  ".md": "markdown", ".rst": "markdown",
  ".yaml": "yaml", ".yml": "yaml", ".json": "json", ".jsonc": "jsonc",
  ".toml": "toml", ".sql": "sql", ".graphql": "graphql", ".gql": "graphql",
  ".proto": "protobuf", ".tf": "terraform", ".tfvars": "terraform",
  ".html": "html", ".htm": "html",
  ".css": "css", ".scss": "css", ".sass": "css", ".less": "css",
  ".xml": "xml", ".cfg": "config", ".ini": "config", ".env": "config",
};
const NOEXT_LANG = {
  Dockerfile: "dockerfile", Makefile: "makefile", Jenkinsfile: "jenkinsfile",
};

function detectLanguage(f) {
  const base = basename(f);
  const e = ext(f);
  if (e && EXT_LANG[e]) return EXT_LANG[e];
  if (NOEXT_LANG[base]) return NOEXT_LANG[base];
  // .env.example etc.
  if (base === ".env" || base.startsWith(".env.")) return "config";
  if (e) return e.slice(1);
  return "unknown";
}

// ---------- Step 4: File category ----------
function detectCategory(f) {
  const base = basename(f);
  const e = ext(f);
  const lower = f.toLowerCase();
  const segs = f.split("/");

  // infra (high specificity first)
  if (base === "Dockerfile" || base.startsWith("docker-compose.")) return "infra";
  if (e === ".tf" || e === ".tfvars") return "infra";
  if (base === "Makefile" || base === "Jenkinsfile" || base === "Procfile" || base === "Vagrantfile") return "infra";
  if (segs.includes(".github") && segs.includes("workflows")) return "infra";
  if (base === ".gitlab-ci.yml") return "infra";
  if (segs.includes(".circleci")) return "infra";
  if (base.endsWith(".k8s.yaml") || base.endsWith(".k8s.yml")) return "infra";
  if (segs.includes("k8s") || segs.includes("kubernetes")) return "infra";

  // data
  if (e === ".sql" || e === ".graphql" || e === ".gql" || e === ".proto" || e === ".prisma" || e === ".csv") return "data";
  if (base.endsWith(".schema.json")) return "data";

  // docs
  if ((e === ".md" || e === ".rst" || e === ".txt") && base !== "LICENSE") return "docs";

  // config
  if ([".yaml", ".yml", ".json", ".jsonc", ".toml", ".xml", ".cfg", ".ini", ".env"].includes(e)) return "config";
  if (base === "tsconfig.json" || base === "package.json" || base === "pyproject.toml" || base === "Cargo.toml" || base === "go.mod") return "config";
  if (base === ".env" || base.startsWith(".env.")) return "config";

  // script
  if (e === ".sh" || e === ".bash" || e === ".ps1" || e === ".bat") return "script";

  // markup
  if ([".html", ".htm", ".css", ".scss", ".sass", ".less"].includes(e)) return "markup";

  return "code";
}

// ---------- Step 5: Line counting ----------
const lineCounts = {};
function countLines(fileList) {
  for (const f of fileList) {
    const abs = path.join(ROOT_ABS, f);
    try {
      const buf = fs.readFileSync(abs);
      let n = 0;
      for (let i = 0; i < buf.length; i++) {
        if (buf[i] === 10) n++;
      }
      // wc -l counts newline chars; if file non-empty without trailing newline, that's fine.
      lineCounts[f] = n;
    } catch (e) {
      lineCounts[f] = 0;
    }
  }
}
countLines(files);

// ---------- Build file records ----------
const fileSet = new Set(files);
const fileRecords = files.map((f) => ({
  path: f,
  language: detectLanguage(f),
  sizeLines: lineCounts[f] || 0,
  fileCategory: detectCategory(f),
}));
fileRecords.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0));

// ---------- languages ----------
const langSet = new Set(fileRecords.map((r) => r.language));
const languages = Array.from(langSet).sort();

// ---------- Step 6: Frameworks ----------
const frameworks = new Set();

function readFileSafe(rel) {
  try {
    return fs.readFileSync(path.join(ROOT_ABS, rel), "utf8");
  } catch (e) {
    return null;
  }
}

let rawDescription = "";
let projectNameFromManifest = null;

// package.json
const pkgRaw = readFileSafe("package.json");
if (pkgRaw) {
  try {
    const pkg = JSON.parse(pkgRaw);
    if (pkg.name) projectNameFromManifest = projectNameFromManifest || pkg.name;
    if (pkg.description) rawDescription = pkg.description;
    const deps = Object.assign({}, pkg.dependencies, pkg.devDependencies);
    const map = {
      react: "React", vue: "Vue", svelte: "Svelte", "@angular/core": "Angular",
      express: "Express", fastify: "Fastify", koa: "Koa", next: "Next.js",
      nuxt: "Nuxt", vite: "Vite", vitest: "Vitest", jest: "Jest", mocha: "Mocha",
      tailwindcss: "Tailwind CSS", prisma: "Prisma", typeorm: "TypeORM",
      sequelize: "Sequelize", mongoose: "Mongoose", redux: "Redux",
      zustand: "Zustand", mobx: "MobX",
    };
    for (const [d, name] of Object.entries(map)) {
      if (deps[d]) frameworks.add(name);
    }
  } catch (e) {}
}

// go.mod
const goMod = readFileSafe("go.mod");
let goModulePath = null;
if (goMod) {
  const m = goMod.match(/^\s*module\s+(\S+)/m);
  if (m) goModulePath = m[1];
  const goFw = {
    "github.com/gin-gonic/gin": "Gin",
    "github.com/labstack/echo": "Echo",
    "github.com/gofiber/fiber": "Fiber",
    "github.com/go-chi/chi": "Chi",
    "gorm.io/gorm": "GORM",
  };
  for (const [mod, name] of Object.entries(goFw)) {
    if (goMod.includes(mod)) frameworks.add(name);
  }
}

// Cargo.toml
const cargo = readFileSafe("Cargo.toml");
if (cargo) {
  const rustFw = ["actix-web", "axum", "rocket", "diesel", "tokio", "serde", "warp"];
  for (const c of rustFw) {
    const re = new RegExp("^\\s*" + c.replace(/[-]/g, "\\-") + "\\s*=", "m");
    if (re.test(cargo)) frameworks.add(c);
  }
}

// Infra tooling from discovered files
const hasFile = (pred) => files.some(pred);
if (hasFile((f) => basename(f) === "Dockerfile")) frameworks.add("Docker");
if (hasFile((f) => /^docker-compose\.(yml|yaml)$/.test(basename(f)))) frameworks.add("Docker Compose");
if (hasFile((f) => ext(f) === ".tf")) frameworks.add("Terraform");
if (hasFile((f) => { const s = f.split("/"); return s.includes(".github") && s.includes("workflows") && /\.ya?ml$/.test(f); })) frameworks.add("GitHub Actions");
if (hasFile((f) => basename(f) === ".gitlab-ci.yml")) frameworks.add("GitLab CI");
if (hasFile((f) => basename(f) === "Jenkinsfile")) frameworks.add("Jenkins");

const frameworksArr = Array.from(frameworks).sort();

// ---------- Step 7: Complexity ----------
const totalFiles = fileRecords.length;
let estimatedComplexity;
if (totalFiles <= 30) estimatedComplexity = "small";
else if (totalFiles <= 150) estimatedComplexity = "moderate";
else if (totalFiles <= 500) estimatedComplexity = "large";
else estimatedComplexity = "very-large";

// ---------- Step 8: Project name ----------
let name = projectNameFromManifest;
if (!name && cargo) {
  const m = cargo.match(/\[package\][\s\S]*?\bname\s*=\s*["']([^"']+)["']/);
  if (m) name = m[1];
}
if (!name && goModulePath) {
  name = goModulePath.split("/").pop();
}
if (!name) {
  const pyproj = readFileSafe("pyproject.toml");
  if (pyproj) {
    let m = pyproj.match(/\[project\][\s\S]*?\bname\s*=\s*["']([^"']+)["']/);
    if (!m) m = pyproj.match(/\[tool\.poetry\][\s\S]*?\bname\s*=\s*["']([^"']+)["']/);
    if (m) name = m[1];
  }
}
if (!name) name = path.basename(ROOT_ABS);

// ---------- readmeHead ----------
let readmeHead = "";
const readmeCandidate = files.find((f) => /^readme\.md$/i.test(basename(f)) && !f.includes("/"))
  || files.find((f) => /^readme\.md$/i.test(basename(f)));
if (readmeCandidate) {
  const content = readFileSafe(readmeCandidate);
  if (content) readmeHead = content.split(/\r?\n/).slice(0, 10).join("\n");
}

// ---------- Step 9: Import resolution (Go-focused) ----------
const importMap = {};
for (const f of files) importMap[f] = [];

const codeFiles = fileRecords.filter((r) => r.fileCategory === "code").map((r) => r.path);

const EXT_PROBES = [".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.js", "/index.tsx", "/index.jsx", ".py", ".go", ".rs", ".rb"];

function resolveRelative(fromFile, importPath) {
  const baseDir = path.posix.dirname(fromFile);
  let target = path.posix.normalize(path.posix.join(baseDir, importPath));
  if (target.startsWith("/")) target = target.slice(1);
  if (fileSet.has(target)) return target;
  for (const ext2 of EXT_PROBES) {
    const cand = target + ext2;
    if (fileSet.has(cand)) return cand;
  }
  return null;
}

// Go: resolve module-internal imports. Import path like "module/internal/foo"
// maps to directory "internal/foo"; resolve to .go files in that dir (excluding _test.go for the importer side we still include all).
function resolveGoImport(importPath) {
  if (!goModulePath) return [];
  if (importPath !== goModulePath && !importPath.startsWith(goModulePath + "/")) return [];
  let rel = importPath === goModulePath ? "" : importPath.slice(goModulePath.length + 1);
  // collect all .go files directly in that directory
  const matches = files.filter((f) => {
    if (ext(f) !== ".go") return false;
    const dir = path.posix.dirname(f);
    return (rel === "" && dir === ".") || dir === rel;
  });
  return matches;
}

function extractGoImports(content) {
  const imports = [];
  // block: import ( ... )
  const blockRe = /import\s*\(([\s\S]*?)\)/g;
  let m;
  while ((m = blockRe.exec(content)) !== null) {
    const body = m[1];
    const lineRe = /(?:[A-Za-z0-9_\.]+\s+)?(?:_\s+|\.\s+)?["`]([^"`]+)["`]/g;
    let lm;
    while ((lm = lineRe.exec(body)) !== null) imports.push(lm[1]);
  }
  // single-line: import "path"
  const singleRe = /^\s*import\s+(?:[A-Za-z0-9_\.]+\s+)?["`]([^"`]+)["`]/gm;
  while ((m = singleRe.exec(content)) !== null) imports.push(m[1]);
  return imports;
}

function extractJsImports(content) {
  const imports = [];
  const re1 = /import\s+(?:[\s\S]*?\s+from\s+)?["']([^"']+)["']/g;
  const re2 = /require\(\s*["']([^"']+)["']\s*\)/g;
  let m;
  while ((m = re1.exec(content)) !== null) imports.push(m[1]);
  while ((m = re2.exec(content)) !== null) imports.push(m[1]);
  return imports.filter((p) => p.startsWith("./") || p.startsWith("../"));
}

function extractPyImports(content) {
  const imports = { rel: [], abs: [] };
  const lines = content.split(/\r?\n/);
  for (const line of lines) {
    let m;
    if ((m = line.match(/^\s*from\s+(\.+)([\w\.]*)\s+import\s+(.+)$/))) {
      imports.rel.push({ dots: m[1].length, mod: m[2], names: m[3] });
    } else if ((m = line.match(/^\s*from\s+([\w\.]+)\s+import\s+(.+)$/))) {
      imports.abs.push({ mod: m[1], names: m[2] });
    } else if ((m = line.match(/^\s*import\s+([\w\.]+(?:\s*,\s*[\w\.]+)*)/))) {
      for (const part of m[1].split(",")) {
        imports.abs.push({ mod: part.trim(), names: null });
      }
    }
  }
  return imports;
}

function resolvePyAbs(mod) {
  const out = [];
  const base = mod.replace(/\./g, "/");
  const asFile = base + ".py";
  const asPkg = base + "/__init__.py";
  if (fileSet.has(asFile)) out.push(asFile);
  else if (fileSet.has(asPkg)) out.push(asPkg);
  return out;
}

for (const f of codeFiles) {
  const e = ext(f);
  const content = readFileSafe(f);
  if (content === null) continue;
  const resolved = new Set();

  if (e === ".go") {
    for (const imp of extractGoImports(content)) {
      for (const r of resolveGoImport(imp)) {
        if (r !== f) resolved.add(r);
      }
    }
  } else if ([".ts", ".tsx", ".js", ".jsx"].includes(e)) {
    for (const imp of extractJsImports(content)) {
      const r = resolveRelative(f, imp);
      if (r && r !== f) resolved.add(r);
    }
  } else if (e === ".py") {
    const imps = extractPyImports(content);
    const fromDir = path.posix.dirname(f);
    for (const r of imps.rel) {
      // resolve relative module
      let up = fromDir;
      for (let i = 1; i < r.dots; i++) up = path.posix.dirname(up);
      const modPath = r.mod ? r.mod.replace(/\./g, "/") : "";
      const base = modPath ? path.posix.join(up, modPath) : up;
      const cands = [base + ".py", base + "/__init__.py"];
      for (const c of cands) if (fileSet.has(c) && c !== f) resolved.add(c);
    }
    for (const r of imps.abs) {
      const matches = resolvePyAbs(r.mod);
      for (const mm of matches) if (mm !== f) resolved.add(mm);
      if (matches.length && matches[0].endsWith("/__init__.py") && r.names) {
        const pkgBase = r.mod.replace(/\./g, "/");
        for (const nm of r.names.split(",").map((s) => s.trim().replace(/\s+as\s+.*/, "")).filter(Boolean)) {
          if (nm === "*") continue;
          const c1 = pkgBase + "/" + nm + ".py";
          const c2 = pkgBase + "/" + nm + "/__init__.py";
          if (fileSet.has(c1) && c1 !== f) resolved.add(c1);
          else if (fileSet.has(c2) && c2 !== f) resolved.add(c2);
        }
      }
    }
  }

  importMap[f] = Array.from(resolved).sort();
}

// ---------- Output ----------
const result = {
  scriptCompleted: true,
  name,
  rawDescription,
  readmeHead,
  languages,
  frameworks: frameworksArr,
  files: fileRecords,
  totalFiles,
  filteredByIgnore,
  estimatedComplexity,
  importMap,
};

try {
  fs.writeFileSync(OUT, JSON.stringify(result, null, 2), "utf8");
} catch (e) {
  fail("cannot write output: " + e.message);
}
process.exit(0);
