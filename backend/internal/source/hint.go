package source

// ExtractionHint is a per-host learned extraction template produced by an
// offline learning process. It carries enough information for a future
// extraction pass to locate the required fields (Title, Company, Description,
// Location) deterministically in a known page structure, without a live model
// call in the hot path.
//
// This type and the two interfaces below are the seam for the next phase of
// the extraction waterfall. Nothing in this package reads or writes a hint;
// the implementations are left to the next phase.
type ExtractionHint struct {
	// Host is the hostname this hint applies to, e.g. "careers.example.com".
	Host string

	// Version is an opaque string that changes whenever the hint is re-learned.
	// Validation compares the live page structure against this version and
	// invalidates the hint when they diverge.
	Version string

	// Selectors is an opaque map from field name to the extraction selector
	// learned for that field. The map layout is defined by the future implementation.
	Selectors map[string]string
}

// HintValidator is called on every use of an ExtractionHint to verify that the
// hint still matches the page structure. It returns false when the hint has
// become stale — for example, because the page layout changed — which triggers
// InvalidateHint.
type HintValidator interface {
	// Validate returns true when hint is still valid for the given page body.
	Validate(hint ExtractionHint, pageBody string) bool
}

// HintStore is the read/write interface for the per-host hint registry. The
// learning mechanism that populates it is not part of this package.
type HintStore interface {
	// Get returns the hint for host, and false when no hint exists.
	Get(host string) (ExtractionHint, bool)

	// Invalidate removes the hint for host, triggering re-learning on the next
	// collection run. It is called whenever Validate returns false.
	Invalidate(host string)
}
