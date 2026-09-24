package statemachine

// The bases each machine's messages are numbered from.
//
// This file knows the numbers and not the domain, which is the point: a child
// machine's messages pass through its controller, so the two must not overlap,
// and something has to hand out the ranges without knowing what either machine
// is for. The reference does the same thing in one place for the same reason
// (IpClient gives DHCPv4 the 1000s and DHCPv6 the 2000s).
//
// Inside a base, the value says which way a message goes. A machine's own
// protocol.go lays them out:
//
//	base + 0x001  a Cmd the outside sends down
//	base + 0x040  a Cmd this machine sends up to its controller
//	base + 0x100  an Event this machine says to itself
//	base + 0x180  an Event something below posts up to it
//
// So a value alone tells a reader the direction, the way PUBLIC_BASE and
// PRIVATE_BASE do in the reference. Whether the name is exported says the same
// thing to the compiler.
const (
	// BaseChain numbers the chain composition machine's messages.
	BaseChain What = 0x1000
	// BaseTest numbers the test run machine's messages.
	BaseTest What = 0x8000
)
