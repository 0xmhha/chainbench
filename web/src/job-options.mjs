// @ts-check
// The run options a job screen offers come from the jobs contract: the
// yes-or-no and number arguments an operation reads, with the title, range and
// meaning the contract gives them.

/** @typedef {{name: string, type: string, title: string, description?: string, minimum?: number, maximum?: number, flag?: string, cli: string}} OperationArgument */

// The job form's server list already offers these.
const SERVER_CHOICE = new Set(['allServers'])

/**
 * @param {{arguments?: OperationArgument[]}|undefined} operation
 * @returns {OperationArgument[]}
 */
export function scalarOptions(operation) {
 return (operation?.arguments ?? []).filter(a => (a.type === 'boolean' || a.type === 'integer') && !SERVER_CHOICE.has(a.name))
}

/**
 * optionArguments turns what the user set into plan arguments. An option left
 * off or empty is left out, so the engine keeps its own choice.
 * @param {OperationArgument[]} options
 * @param {Record<string, unknown>} values
 */
export function optionArguments(options, values) {
 /** @type {Record<string, boolean|number>} */
 const out = {}
 for (const option of options) {
  const value = values[option.name]
  if (option.type === 'boolean') {
   if (value === true) out[option.name] = true
   continue
  }
  if (value === '' || value === null || value === undefined) continue
  const n = Number(value)
  if (!Number.isInteger(n) || (option.minimum !== undefined && n < option.minimum) || (option.maximum !== undefined && n > option.maximum)) {
   throw new Error(`${option.title}은(는) ${rangeText(option)}정수여야 합니다.`)
  }
  out[option.name] = n
 }
 return out
}

/**
 * planBlockers names what the job form still needs before a plan can be made,
 * in the order the form asks for it.
 * @param {{workspace?: boolean, manifest?: boolean, asset?: boolean, server?: boolean, needsBinary?: boolean, needsCase?: boolean, cases?: boolean}} form
 * @returns {string[]}
 */
export function planBlockers(form) {
 const missing = []
 if (!form.workspace) missing.push('Workspace')
 if (form.needsBinary && !form.manifest) missing.push('매니페스트')
 if (form.needsBinary && !form.asset) missing.push('바이너리')
 if (!form.server) missing.push('서버')
 if (form.needsCase && !form.cases) missing.push('실행할 케이스')
 return missing
}

/**
 * rangeText reads the bounds a contract gives, which may be one or none.
 * @param {{ minimum?: number, maximum?: number }} bounds
 */
function rangeText({ minimum, maximum }) {
 if (minimum !== undefined && maximum !== undefined) return `${minimum}부터 ${maximum}까지의 `
 if (minimum !== undefined) return `${minimum} 이상의 `
 if (maximum !== undefined) return `${maximum} 이하의 `
 return ''
}
