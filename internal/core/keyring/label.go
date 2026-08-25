package keyring

// A label is how a network's declaration refers to an identity without knowing
// its address, which is what lets a test say "bp1" and never carry a key.
type Label string
