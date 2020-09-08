package address

var (
	// DefaultAllowedDecodedLengths ...
	DefaultAllowedDecodedLengths = []int{1, 2, 4, 8, 32}
	// DefaultAllowedEncodedLengths ...
	DefaultAllowedEncodedLengths = []int{3, 4, 6, 10, 35}
	// DefaultAllowedPrefix ...
	DefaultAllowedPrefix = AllPrefixEnums()
	// DefaultPrefix ...
	DefaultPrefix = FortyTwo
)

// SetDefaultPrefix ...
func SetDefaultPrefix(prefix PrefixEnum) {
	DefaultPrefix = prefix.Type()
}
