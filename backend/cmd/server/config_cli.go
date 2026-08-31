package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"tsunagu/backend/internal/config"
	"tsunagu/backend/internal/db"
	"tsunagu/backend/internal/db/sqlcgen"
)

func runConfigCLI(args []string, opts config.Options) {
	usage := "usage: tsunagu config <list|get KEY|set KEY VALUE|unset KEY|reset>"
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}

	bootCfg, tomlPath, active, err := config.Load(opts)
	if err != nil {
		fatal("loading config: %v", err)
	}
	conn, err := db.Open(bootCfg.DBPath)
	if err != nil {
		fatal("opening db: %v", err)
	}
	defer conn.Close()
	store := config.NewStore(bootCfg, sqlcgen.New(conn), tomlPath, active)
	ctx := context.Background()
	if err := store.Sync(ctx); err != nil {
		fatal("syncing config: %v", err)
	}

	switch args[0] {
	case "list":
		w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(w, "KEY\tVALUE\tSOURCE\tKIND\tSCOPE")
		for _, s := range store.List(ctx) {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Key, s.Value, s.Source, s.Kind, s.Scope)
		}
		w.Flush()
	case "get":
		if len(args) != 2 {
			fatal("%s", usage)
		}
		s, err := store.Get(ctx, args[1])
		if err != nil {
			fatal("%v", err)
		}
		fmt.Println(s.Value)
	case "set":
		if len(args) != 3 {
			fatal("%s", usage)
		}
		s, err := store.Set(ctx, args[1], args[2])
		if err != nil {
			fatal("%v", err)
		}
		fmt.Printf("%s = %s", s.Key, s.Value)
		if s.RestartRequired() {
			fmt.Print("  (restart required)")
		}
		fmt.Println()
	case "unset":
		if len(args) != 2 {
			fatal("%s", usage)
		}
		if _, err := store.Unset(ctx, args[1]); err != nil {
			fatal("%v", err)
		}
		fmt.Printf("%s reset to default\n", args[1])
	case "reset":
		for _, s := range store.List(ctx) {
			if s.Editable {
				_, _ = store.Unset(ctx, s.Key)
			}
		}
		fmt.Println("all runtime settings reset to default")
	default:
		fatal("%s", usage)
	}
}

func fatal(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
