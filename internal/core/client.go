package core

// CLIUserAgentPrefix is the User-Agent prefix sent by the BeeBuzz CLI client.
// The CLI sets a User-Agent of the form "BeeBuzz-CLI/<version>" and the server
// uses this prefix to classify the source of incoming push requests.
const CLIUserAgentPrefix = "BeeBuzz-CLI"
