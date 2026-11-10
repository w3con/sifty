package matchcontract

// RulesIdentity returns a canonical identity for the complete effective rule set. It covers the
// declared version and every content field, so changing a weight, requirement weight or the
// declared version changes the identity. Stored results compare this value instead of trusting
// a manually bumped version string alone.
func RulesIdentity(r Rules) (string, error) {
	return hashJSON(struct {
		Rules Rules `json:"rules"`
	}{Rules: r})
}

// SourcePolicyIdentity returns a canonical identity for the complete effective source-field
// policy. Editing a source rule without bumping SourcePolicy.Version still changes the identity,
// so old facts and results cannot remain current by accident.
func SourcePolicyIdentity(p SourcePolicy) (string, error) {
	return hashJSON(struct {
		Policy SourcePolicy `json:"policy"`
	}{Policy: p})
}
