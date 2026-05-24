// Package gmailmsg provides Gmail helper contracts for payload shaping and
// operator-supplied OAuth2 bootstrap. The fnct helper stays pure and does not
// call Gmail, resolve credentials, or choose accounts. OAuth helpers are used
// only when an explicit trusted runtime or local operator CLI invokes them.
package gmailmsg
