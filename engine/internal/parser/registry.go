package parser

import "io"

// Registry holds registered parsers and supports lookup by format or auto-detection.
type Registry struct {
	parsers []Parser
	byFmt   map[Format]Parser
}

// NewRegistry creates an empty parser registry.
func NewRegistry() *Registry {
	return &Registry{
		byFmt: make(map[Format]Parser),
	}
}

// Register adds a parser to the registry.
func (r *Registry) Register(p Parser) {
	r.parsers = append(r.parsers, p)
	r.byFmt[p.Format()] = p
}

// Get returns the parser for the given format.
func (r *Registry) Get(f Format) (Parser, bool) {
	p, ok := r.byFmt[f]
	return p, ok
}

// Detect iterates registered parsers and returns the first one whose
// Detect method returns true. The reader position is reset between attempts.
func (r *Registry) Detect(rs io.ReadSeeker) (Parser, bool) {
	for _, p := range r.parsers {
		if _, err := rs.Seek(0, io.SeekStart); err != nil {
			continue
		}
		if p.Detect(rs) {
			if _, err := rs.Seek(0, io.SeekStart); err != nil {
				continue
			}
			return p, true
		}
	}
	return nil, false
}

// Formats returns all registered format identifiers.
func (r *Registry) Formats() []Format {
	fmts := make([]Format, len(r.parsers))
	for i, p := range r.parsers {
		fmts[i] = p.Format()
	}
	return fmts
}
