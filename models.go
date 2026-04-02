package devtui

// GetLogsArgs defines arguments for the app_get_logs tool.
// ormc:formonly
type GetLogsArgs struct {
	Section string `input:"text"`
}

// ActionArgs defines arguments for the tinywasm/action method.
// ormc:formonly
type ActionArgs struct {
	Key   string `input:"text"`
	Value string `input:"text" json:",omitempty"`
}
