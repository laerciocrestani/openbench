package app

// Progress desacopla app.Run* da apresentação (CLI ANSI ou TUI).
// Migração incremental: quando nil, runner usa ui.Session.
type Progress interface {
	Step(label string, fn func() error) error
	StepQuiet(fn func() error) error
	Detail(msg string)
	Info(msg string)
	Warn(msg string)
	Success(msg string)
}
