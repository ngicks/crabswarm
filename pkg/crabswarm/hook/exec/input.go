package exec

import (
	"github.com/ngicks/crabswarm/pkg/crabswarm/hook"
	"github.com/ngicks/crabswarm/pkg/filetype"
)

// parseInput reads a Claude / Codex hook envelope into [Data]: it delegates
// envelope parsing and edited-file extraction to [hook.Parse], then
// layers exec-specific filetype and module-root detection on top of the
// first edited file.
func parseInput(raw []byte, ftCfg filetype.Config) (Data, error) {
	p, err := hook.Parse(raw)
	if err != nil {
		return Data{}, err
	}

	var file string
	if len(p.Files) > 0 {
		file = p.Files[0]
	}
	d := Data{
		Input:     p.Input,
		Event:     p.Event,
		ToolName:  p.ToolName,
		SessionID: p.SessionID,
		Cwd:       p.Cwd,
		File:      file,
		Files:     p.Files,
	}
	if d.File != "" {
		if name, ok := ftCfg.Detect(d.File); ok {
			d.Filetype = name
			if root, ok := ftCfg.FindRoot(name, d.File); ok {
				d.Root = root
			}
		}
	}
	return d, nil
}
