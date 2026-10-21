// Parse complete Svelte, HTML and CSS trees with the repository's locked compiler.
// Only identifiers and source locations are persisted; literal credentials are omitted.
const fs = require('node:fs')
const path = require('node:path')
const cp = require('node:child_process')

const [repo, compilerPath, output] = process.argv.slice(2)
if (!repo || !compilerPath || !output) throw new Error('Usage: node parse-web-ast.cjs REPO COMPILER OUTPUT')
const compiler = require(path.resolve(compilerPath))
const files = cp.execFileSync('git', ['ls-files', '-z'], { cwd: repo, timeout: 30000 }).toString().split('\0').filter(f => /\.(svelte|html|css)$/.test(f)).sort()
const records = []
function identifier(node) {
  if (!node) return ''
  if (node.type === 'Identifier') return node.name
  if (node.type === 'MemberExpression') return [identifier(node.object), identifier(node.property)].filter(Boolean).join('.')
  return ''
}
for (const file of files) {
  const source = fs.readFileSync(path.join(repo, file), 'utf8')
  const wrapped = file.endsWith('.css')
  const input = wrapped ? `<style>${source}</style>` : source
  const record = { file, language: path.extname(file).slice(1), lines: source.split('\n').length, astNodes: 0, imports: [], symbols: [], calls: [], components: [], guards: [], errors: [] }
  try {
    const root = compiler.parse(input, { modern: true, filename: file })
    const seen = new WeakSet()
    const stack = [root]
    while (stack.length) {
      const node = stack.pop()
      if (!node || typeof node !== 'object' || seen.has(node)) continue
      seen.add(node)
      if (typeof node.type === 'string') {
        record.astNodes++
        const offset = Math.max(0, (node.start ?? 0) - (wrapped ? 7 : 0))
        const line = source.slice(0, offset).split('\n').length
        if (node.type === 'ImportDeclaration') record.imports.push({ target: node.source.value, line })
        if (['FunctionDeclaration', 'ClassDeclaration', 'VariableDeclarator'].includes(node.type) && node.id?.type === 'Identifier') record.symbols.push({ name: node.id.name, kind: node.type, line })
        if (['CallExpression', 'NewExpression'].includes(node.type)) {
          const target = identifier(node.callee)
          if (target) record.calls.push({ target, line })
        }
        if (node.type === 'Component') record.components.push({ target: node.name, line })
        if (node.type === 'IfBlock' && node.test) {
          const expression = input.slice(node.test.start, node.test.end)
          if (/webSession|session|actor|role|canEdit|writable|administrator|owner|access/.test(expression)) record.guards.push({ expression, line })
        }
      }
      for (const value of Object.values(node)) {
        if (Array.isArray(value)) stack.push(...value)
        else if (value && typeof value === 'object') stack.push(value)
      }
    }
  } catch (error) {
    record.errors.push({ code: error.code ?? 'parse-error', line: error.start?.line ?? null })
  }
  records.push(record)
}
fs.writeFileSync(output, JSON.stringify({ compiler: `svelte@${compiler.VERSION}`, files: records }, null, 2) + '\n')
console.log(JSON.stringify({ files: records.length, errors: records.filter(r => r.errors.length).length, compiler: compiler.VERSION }))
