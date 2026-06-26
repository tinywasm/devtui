package devtui

// ActionArgs defines arguments for the tinywasm/action method.
// It is both a TUI input form and the wire payload for the tinywasm/action
// JSON-RPC call, so ormc generates its form schema AND its fmt codec here.
// ormc:formonly
type ActionArgs struct {
	Key   string `input:"text"`
	Value string `input:"text"`
}
