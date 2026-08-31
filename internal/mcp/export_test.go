package mcp

// SaveNetwork opens the unexported registry writer to the external test
// package, which seeds attached networks before exercising the tools.
var SaveNetwork = saveNetwork
