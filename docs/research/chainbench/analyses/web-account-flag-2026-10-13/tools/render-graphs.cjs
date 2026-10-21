// Render complete package and frontend import graphs without remote resources.
const fs = require('node:fs')
const path = require('node:path')
const [directory, vizModule] = process.argv.slice(2)
if (!directory || !vizModule) throw new Error('Usage: node render-graphs.cjs GRAPH_DIRECTORY VIZ_MODULE')
const { instance } = require(path.resolve(vizModule))
const go = JSON.parse(fs.readFileSync(path.join(directory, 'go-package-graph.json')))
const source = JSON.parse(fs.readFileSync(path.join(directory, 'source-graph.json')))
const quote = value => JSON.stringify(value)
const attributes = 'graph [rankdir=LR, bgcolor="white", pad=0.3]; node [shape=box, style="rounded,filled", fillcolor="#f1f5f9", color="#94a3b8", fontname="Helvetica", fontsize=11]; edge [color="#64748b", arrowsize=0.6];'
const highlight = file => /dashboard|webcli|internal\/app|LoginPanel|App\.svelte|web-api|SettingsPanel|JobPanel/.test(file)
function graph(nodes, edges) {
  const lines = ['digraph imports {', attributes]
  for (const node of nodes) lines.push(`${quote(node.id)} [label=${quote(node.label)}, fillcolor=${quote(highlight(node.id) ? '#dbeafe' : '#f1f5f9')}];`)
  for (const edge of edges) lines.push(`${quote(edge.from)} -> ${quote(edge.to)};`)
  return [...lines, '}'].join('\n')
}
async function main() {
  const viz = await instance()
  const graphs = {
    'go-packages': graph(go.packages.map(p => ({ id: p.path, label: `${p.path}\n${p.files} source / ${p.testFiles} test` })), go.edges),
    'web-files': graph(source.files.filter(f => f.file.startsWith('web/')).map(f => ({ id: f.file, label: f.file })), source.edges.filter(e => e.kind === 'file-import' && e.from.startsWith('web/'))),
  }
  for (const [name, dot] of Object.entries(graphs)) {
    fs.writeFileSync(path.join(directory, `${name}.dot`), dot + '\n')
    fs.writeFileSync(path.join(directory, `${name}.svg`), viz.renderString(dot, { format: 'svg' }))
  }
  const summary = {
    commit: source.commit,
    coverage: source.coverage,
    files: source.fileCount,
    parseErrorFiles: source.parseErrorFiles,
    astNodes: source.files.reduce((n, f) => n + f.astNodes, 0),
    syntacticCalls: source.files.reduce((n, f) => n + f.calls.length, 0),
    fileImportEdges: source.edges.filter(e => e.kind === 'file-import').length,
    goPackageImportOccurrences: source.edges.filter(e => e.kind === 'go-package-import').length,
    goPackages: go.packages.length,
    goPackageEdges: go.edges.length,
    goSymbols: go.symbols.length,
    layerViolationsReportedByInventoryTool: go.violations,
    goProductionFiles: go.packages.reduce((n, p) => n + p.files, 0),
    goTestFiles: go.packages.reduce((n, p) => n + p.testFiles, 0),
    topGoFanIn: [...go.packages].sort((a, b) => b.fanIn - a.fanIn || a.path.localeCompare(b.path)).slice(0, 10),
    goEntrypoint: go.edges.filter(e => e.from === 'cmd/chainbench-dashboard'),
    webFiles: source.files.filter(f => f.file.startsWith('web/')).length,
  }
  fs.writeFileSync(path.join(directory, 'summary.json'), JSON.stringify(summary, null, 2) + '\n')
  console.log(JSON.stringify({ files: summary.files, packages: summary.goPackages, edges: summary.goPackageEdges, astNodes: summary.astNodes, webFiles: summary.webFiles }))
}
main().catch(error => { console.error(error); process.exitCode = 1 })
