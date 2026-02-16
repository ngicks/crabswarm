// Command splitpane creates a tmux session and splits panes so you can
// visually inspect the split algorithm. It labels each pane with its
// index and pane ID, then attaches to the session for interactive use.
//
// Usage:
//
//	go run ./pkg/mux/tmux/internal/cmd/splitpane [flags]
//	  -n int      number of panes to add via Split (default 3)
//	  -socket     tmux socket name (default "splitpane-demo")
//	  -tmux       path to tmux binary (default: lookup from PATH)
//	  -no-attach  don't attach; just print the socket/session name and exit
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/ngicks/crabswarm/pkg/mux/tmux"
)

func main() {
	n := flag.Int("n", 3, "number of panes to add via Split")
	socket := flag.String("socket", "splitpane-demo", "tmux socket name")
	tmuxBin := flag.String("tmux", "", "path to tmux binary (empty = lookup PATH)")
	noAttach := flag.Bool("no-attach", false, "don't attach to the session after setup")
	flag.Parse()

	tmuxPath := *tmuxBin
	if tmuxPath == "" {
		p, err := exec.LookPath("tmux")
		if err != nil {
			log.Fatalf("tmux not found: %v", err)
		}
		tmuxPath = p
	}

	ctx := context.Background()
	cfg := tmux.Config{
		Name:       "splitpane-demo",
		TmuxPath:   tmuxPath,
		SocketName: *socket,
	}

	// Kill any previous demo session on this socket.
	cleanup := exec.CommandContext(ctx, tmuxPath, "-L", *socket, "kill-server")
	_ = cleanup.Run()

	sess, err := tmux.New(ctx, cfg)
	if err != nil {
		log.Fatalf("tmux.New: %v", err)
	}
	fmt.Printf("session: id=%s name=%s socket=%s\n", sess.Id(), "splitpane-demo", *socket)

	w, err := sess.GetAt(ctx, 0)
	if err != nil {
		log.Fatalf("GetAt(0): %v", err)
	}
	fmt.Printf("window:  id=%s\n", w.Id())

	if *n > 0 {
		fmt.Printf("splitting %d pane(s)...\n", *n)
		if err := w.Split(ctx, *n); err != nil {
			log.Fatalf("Split(%d): %v", *n, err)
		}
	}

	panes, err := w.List(ctx)
	if err != nil {
		log.Fatalf("List: %v", err)
	}
	fmt.Printf("total panes: %d\n", len(panes))

	// Label each pane with its index and ID.
	for i, p := range panes {
		idx, _ := p.Index(ctx)
		keys := []string{
			fmt.Sprintf("echo 'pane %d (id=%s, idx=%d)'", i, p.Id(), idx),
			"Enter",
		}
		if err := p.SendKeys(ctx, keys); err != nil {
			log.Printf("SendKeys pane %d: %v", i, err)
		}
	}

	if *noAttach {
		fmt.Println("done (not attaching)")
		fmt.Printf("attach manually: tmux -L %s attach\n", *socket)
		fmt.Printf("kill:            tmux -L %s kill-server\n", *socket)
		return
	}

	fmt.Println("attaching... (detach with Ctrl-b d, or exit to clean up)")
	attach := exec.CommandContext(ctx, tmuxPath, "-L", *socket, "attach-session", "-t", "splitpane-demo")
	attach.Stdin = os.Stdin
	attach.Stdout = os.Stdout
	attach.Stderr = os.Stderr
	if err := attach.Run(); err != nil {
		log.Fatalf("attach: %v", err)
	}

	// Clean up after detach/exit.
	kill := exec.CommandContext(ctx, tmuxPath, "-L", *socket, "kill-server")
	_ = kill.Run()
	fmt.Println("server killed")
}
