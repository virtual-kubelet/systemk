package provider

// supportedOptions contains a mappnig for supported systemd options. If an option
// is supported the key name will be returned. Unsupported either return an
// empty string (really not supported) or an alternative option that's better
// than nothing at all.
var supportedOptions = map[string]string{
	"BindReadOnlyPaths": "BindReadOnlyPaths",
}

// ProbeSupportedOptions checks it the options in SupportedOptions are
// supported by the systemd version running on this system. It will emit Info
// logs for each unsupported option.
func ProbeSupportedOptions() {
	for option := range supportedOptions {
		ok := probe(option)
		switch option {
		case "BindReadOnlyPaths":
			if !ok {
				supportedOptions[option] = "BindPaths" // drop the RO bit
			}
		}
	}
}

// probe probes system to see if option is supported.
func probe(option string) bool {
	return true
}

// Option return the option that is supported by the detected systemd.
func Option(option string) string {
	opt, ok := supportedOptions[option]
	if !ok {
		// not found in map, return option as-is
		return option
	}
	return opt
}
