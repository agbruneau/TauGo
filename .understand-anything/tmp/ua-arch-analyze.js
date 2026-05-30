#!/usr/bin/env node
'use strict';
const fs = require('fs');

function main() {
  const inPath = process.argv[2];
  const outPath = process.argv[3];
  if (!inPath || !outPath) { console.error('usage: script <in> <out>'); process.exit(1); }
  const data = JSON.parse(fs.readFileSync(inPath, 'utf8'));
  const fileNodes = data.fileNodes || [];
  const importEdges = data.importEdges || [];
  const allEdges = data.allEdges || [];

  const idToNode = new Map();
  fileNodes.forEach(n => idToNode.set(n.id, n));

  // Common prefix of file paths
  const paths = fileNodes.map(n => n.filePath || '').filter(Boolean);
  function commonPrefix(arr) {
    if (!arr.length) return '';
    const split = arr.map(p => p.split('/'));
    const first = split[0];
    let prefix = [];
    for (let i = 0; i < first.length - 1; i++) {
      const seg = first[i];
      if (split.every(s => s[i] === seg)) prefix.push(seg); else break;
    }
    return prefix.length ? prefix.join('/') + '/' : '';
  }
  const prefix = commonPrefix(paths);

  // A. Directory grouping
  const directoryGroups = {};
  const fileToGroup = new Map();
  for (const n of fileNodes) {
    let p = n.filePath || '';
    let rest = prefix && p.startsWith(prefix) ? p.slice(prefix.length) : p;
    const segs = rest.split('/');
    let group;
    if (segs.length <= 1) group = '(root)';
    else group = segs[0];
    (directoryGroups[group] = directoryGroups[group] || []).push(n.id);
    fileToGroup.set(n.id, group);
  }

  // B. Node type grouping
  const nodeTypeGroups = {};
  for (const n of fileNodes) (nodeTypeGroups[n.type] = nodeTypeGroups[n.type] || []).push(n.id);

  // C. Import adjacency
  const fanOut = {}, fanIn = {};
  for (const n of fileNodes) { fanOut[n.id] = 0; fanIn[n.id] = 0; }
  for (const e of importEdges) {
    if (fanOut[e.source] !== undefined) fanOut[e.source]++;
    if (fanIn[e.target] !== undefined) fanIn[e.target]++;
  }

  // D. Cross-category edges (by node type, from allEdges)
  const ccMap = {};
  for (const e of allEdges) {
    const s = idToNode.get(e.source), t = idToNode.get(e.target);
    if (!s || !t) continue;
    if (s.type === 'file' && t.type === 'file') continue; // pure code-code handled elsewhere
    const key = s.type + '|' + t.type + '|' + (e.type || 'edge');
    ccMap[key] = (ccMap[key] || 0) + 1;
  }
  const crossCategoryEdges = Object.entries(ccMap).map(([k, count]) => {
    const [fromType, toType, edgeType] = k.split('|');
    return { fromType, toType, edgeType, count };
  }).sort((a, b) => b.count - a.count);

  // E. Inter-group import frequency
  const igMap = {};
  for (const e of importEdges) {
    const g1 = fileToGroup.get(e.source), g2 = fileToGroup.get(e.target);
    if (g1 == null || g2 == null || g1 === g2) continue;
    const key = g1 + '->' + g2;
    igMap[key] = (igMap[key] || 0) + 1;
  }
  const interGroupImports = Object.entries(igMap).map(([k, count]) => {
    const [from, to] = k.split('->');
    return { from, to, count };
  }).sort((a, b) => b.count - a.count);

  // F. Intra-group density
  const intraGroupDensity = {};
  for (const g of Object.keys(directoryGroups)) {
    intraGroupDensity[g] = { internalEdges: 0, totalEdges: 0, density: 0 };
  }
  for (const e of importEdges) {
    const g1 = fileToGroup.get(e.source), g2 = fileToGroup.get(e.target);
    if (g1 == null || g2 == null) continue;
    if (g1 === g2) { intraGroupDensity[g1].internalEdges++; intraGroupDensity[g1].totalEdges++; }
    else { intraGroupDensity[g1].totalEdges++; intraGroupDensity[g2].totalEdges++; }
  }
  for (const g of Object.keys(intraGroupDensity)) {
    const d = intraGroupDensity[g];
    d.density = d.totalEdges ? +(d.internalEdges / d.totalEdges).toFixed(3) : 0;
  }

  // G. Pattern matching
  const dirPatterns = [
    [/^(routes|api|controllers|endpoints|handlers|controller|routers|serializers|blueprints)$/, 'api'],
    [/^(services|core|lib|domain|logic|signals|composables|mailers|jobs|channels|internal)$/, 'service'],
    [/^(models|db|data|persistence|repository|entities|migrations|entity|sql|database)$/, 'data'],
    [/^(components|views|pages|ui|layouts|screens)$/, 'ui'],
    [/^(middleware|plugins|interceptors|guards)$/, 'middleware'],
    [/^(utils|helpers|common|shared|tools|templatetags|pkg)$/, 'utility'],
    [/^(config|constants|env|settings|management|commands)$/, 'config'],
    [/^(__tests__|test|tests|spec|specs)$/, 'test'],
    [/^(types|interfaces|schemas|contracts|dtos|dto|request|response)$/, 'types'],
    [/^hooks$/, 'hooks'],
    [/^(store|state|reducers|actions|slices)$/, 'state'],
    [/^(assets|static|public)$/, 'assets'],
    [/^(cmd|bin)$/, 'entry'],
    [/^(docs|documentation|wiki)$/, 'documentation'],
    [/^(deploy|deployment|infra|infrastructure|docker|k8s|kubernetes|helm|charts|terraform|tf)$/, 'infrastructure'],
    [/^(\.github|\.gitlab|\.circleci)$/, 'ci-cd'],
  ];
  const patternMatches = {};
  for (const g of Object.keys(directoryGroups)) {
    let label = null;
    for (const [re, lab] of dirPatterns) if (re.test(g)) { label = lab; break; }
    if (label) patternMatches[g] = label;
  }

  // File-level pattern helpers
  const isTest = p => /(\.test\.|\.spec\.|_test\.go$|_test\.py$|Test\.java$|_spec\.rb$|Tests\.cs$)/.test(p);
  const isDoc = p => /\.(md|rst)$/i.test(p);

  // H. Deployment topology
  const infraFiles = [];
  let hasDockerfile = false, hasCompose = false, hasK8s = false, hasTerraform = false, hasCI = false;
  for (const n of fileNodes) {
    const p = n.filePath || '';
    const base = p.split('/').pop();
    if (/^Dockerfile/.test(base)) { hasDockerfile = true; infraFiles.push(p); }
    else if (/^docker-compose/.test(base)) { hasCompose = true; infraFiles.push(p); }
    else if (/\.(tf|tfvars)$/.test(base)) { hasTerraform = true; infraFiles.push(p); }
    else if (/(\.github\/workflows|\.gitlab-ci|Jenkinsfile)/.test(p)) { hasCI = true; infraFiles.push(p); }
    else if (/(k8s|kubernetes|helm|charts)/.test(p) && /\.ya?ml$/.test(p)) { hasK8s = true; infraFiles.push(p); }
  }

  // I. Data pipeline
  const dataPipeline = { schemaFiles: [], migrationFiles: [], dataModelFiles: [], apiHandlerFiles: [] };
  for (const n of fileNodes) {
    const p = n.filePath || '';
    if (/\.(sql|graphql|gql|proto|prisma)$/.test(p)) dataPipeline.schemaFiles.push(p);
    if (/migrations?\//.test(p)) dataPipeline.migrationFiles.push(p);
    const tags = (n.tags || []).join(' ');
    if (/model|entity/.test(tags)) dataPipeline.dataModelFiles.push(p);
    if (/api-handler|endpoint|route/.test(tags)) dataPipeline.apiHandlerFiles.push(p);
  }

  // J. Doc coverage
  const groupsWithDocs = new Set();
  for (const n of fileNodes) {
    const g = fileToGroup.get(n.id);
    if (isDoc(n.filePath || '')) groupsWithDocs.add(g);
  }
  const totalGroups = Object.keys(directoryGroups).length;
  const undocumentedGroups = Object.keys(directoryGroups).filter(g => !groupsWithDocs.has(g));
  const docCoverage = {
    groupsWithDocs: groupsWithDocs.size,
    totalGroups,
    coverageRatio: totalGroups ? +(groupsWithDocs.size / totalGroups).toFixed(2) : 0,
    undocumentedGroups,
  };

  // K. Dependency direction
  const pairNet = {};
  for (const { from, to, count } of interGroupImports) {
    const key = [from, to].sort().join('||');
    pairNet[key] = pairNet[key] || {};
    pairNet[key][from + '->' + to] = count;
  }
  const dependencyDirection = [];
  for (const { from, to, count } of interGroupImports) {
    const rev = igMap[to + '->' + from] || 0;
    if (count > rev) dependencyDirection.push({ dependent: from, dependsOn: to, net: count - rev });
  }
  // dedupe (each ordered pair appears once already)
  const seenDir = new Set();
  const ddFinal = [];
  for (const d of dependencyDirection) {
    const k = d.dependent + '->' + d.dependsOn;
    if (seenDir.has(k)) continue; seenDir.add(k);
    ddFinal.push({ dependent: d.dependent, dependsOn: d.dependsOn });
  }

  // Stats
  const filesPerGroup = {};
  for (const g of Object.keys(directoryGroups)) filesPerGroup[g] = directoryGroups[g].length;
  const nodeTypeCounts = {};
  for (const t of Object.keys(nodeTypeGroups)) nodeTypeCounts[t] = nodeTypeGroups[t].length;

  const result = {
    scriptCompleted: true,
    commonPrefix: prefix,
    directoryGroups,
    nodeTypeGroups,
    crossCategoryEdges,
    interGroupImports,
    intraGroupDensity,
    patternMatches,
    deploymentTopology: { hasDockerfile, hasCompose, hasK8s, hasTerraform, hasCI, infraFiles },
    dataPipeline,
    docCoverage,
    dependencyDirection: ddFinal,
    fileStats: { totalFileNodes: fileNodes.length, filesPerGroup, nodeTypeCounts },
    fileFanIn: fanIn,
    fileFanOut: fanOut,
  };
  fs.writeFileSync(outPath, JSON.stringify(result, null, 2));
  process.exit(0);
}
try { main(); } catch (e) { console.error(e && e.stack || e); process.exit(1); }
