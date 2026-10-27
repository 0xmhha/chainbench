// @ts-check
// The run options a job screen offers come from the jobs contract: the
// yes-or-no and number arguments an operation reads, with the title, range and
// meaning the contract gives them.

/** @typedef {{name: string, type: string, title: string, description?: string, minimum?: number, maximum?: number, flag?: string, cli: string}} OperationArgument */

/**
 * @param {{arguments?: OperationArgument[]}|undefined} operation
 * @returns {OperationArgument[]}
 */
export function scalarOptions(operation) {
 return (operation?.arguments ?? []).filter(a => a.type === 'boolean' || a.type === 'integer')
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
   throw new Error(`${option.title}은(는) ${option.minimum}부터 ${option.maximum}까지의 정수여야 합니다.`)
  }
  out[option.name] = n
 }
 return out
}
