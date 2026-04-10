package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aditya-soni/forge/blueprints"
	"github.com/aditya-soni/forge/core/blueprint"
	"github.com/aditya-soni/forge/internal/grpcexec"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "blueprint":
		handleBlueprint(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: forge <command>")
	fmt.Println("Commands:")
	fmt.Println("  blueprint validate <file>  Validate a blueprint YAML file")
	fmt.Println("  blueprint list             List built-in blueprints")
	fmt.Println("  blueprint run [--harness <addr>] <file>  Run blueprint (mock or gRPC harness)")
}

func handleBlueprint(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: forge blueprint <validate|list|run> [file]")
		os.Exit(1)
	}

	switch args[0] {
	case "validate":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: forge blueprint validate <file>")
			os.Exit(1)
		}
		cmdValidate(args[1])
	case "list":
		cmdList()
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: forge blueprint run [--harness <addr>] <file>")
			os.Exit(1)
		}
		harnessAddr, file, err := parseBlueprintRunArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "blueprint run: %v\n", err)
			os.Exit(1)
		}
		cmdRun(file, harnessAddr)
	default:
		fmt.Fprintf(os.Stderr, "unknown blueprint subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func parseBlueprintRunArgs(args []string) (harnessAddr, file string, err error) {
	var files []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--harness":
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--harness requires an address")
			}
			i++
			harnessAddr = args[i]
		case strings.HasPrefix(args[i], "-"):
			return "", "", fmt.Errorf("unknown flag: %s", args[i])
		default:
			files = append(files, args[i])
		}
	}
	if len(files) != 1 {
		return "", "", fmt.Errorf("expected exactly one blueprint file, got %d", len(files))
	}
	return harnessAddr, files[0], nil
}

type echoExecutor struct{}

func (e *echoExecutor) Execute(_ context.Context, prompt string, _ map[string]interface{}) (string, error) {
	return fmt.Sprintf("[mock] Would execute agent with prompt: %s", prompt), nil
}

func cmdValidate(file string) {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
	bp, err := blueprint.ParseBlueprintYAML(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}
	g, err := bp.BuildGraph(&echoExecutor{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		os.Exit(1)
	}
	if err := g.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "validation error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Blueprint %q (v%s) is valid: %d nodes, %d edges\n",
		bp.Name, bp.Version, len(bp.Nodes), len(bp.Edges))
}

func cmdList() {
	entries, err := blueprints.BuiltIn.ReadDir(".")
	if err != nil {
		fmt.Println("No blueprints found in blueprints/")
		return
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if strings.HasSuffix(n, ".yaml") || strings.HasSuffix(n, ".yml") {
			names = append(names, n)
		}
	}
	if len(names) == 0 {
		fmt.Println("No blueprints found in blueprints/")
		return
	}
	sort.Strings(names)
	fmt.Println("Built-in blueprints:")
	for _, name := range names {
		data, err := blueprints.BuiltIn.ReadFile(name)
		if err != nil {
			continue
		}
		bp, err := blueprint.ParseBlueprintYAML(data)
		if err != nil {
			continue
		}
		fmt.Printf("  %-30s %s\n", bp.Name, bp.Description)
	}
}

func cmdRun(file string, harnessAddr string) {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
	bp, err := blueprint.ParseBlueprintYAML(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error: %v\n", err)
		os.Exit(1)
	}

	var exec blueprint.AgentExecutor
	if harnessAddr != "" {
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "getwd: %v\n", err)
			os.Exit(1)
		}
		grpcExec, err := grpcexec.NewGrpcAgentExecutor(harnessAddr, wd,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Fprintf(os.Stderr, "harness client: %v\n", err)
			os.Exit(1)
		}
		defer grpcExec.Close()
		exec = grpcExec
	} else {
		exec = &echoExecutor{}
	}

	g, err := bp.BuildGraph(exec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build error: %v\n", err)
		os.Exit(1)
	}
	engine := blueprint.NewEngine(g, bp.Name)
	if harnessAddr != "" {
		fmt.Printf("Running blueprint %q (harness at %s)...\n", bp.Name, harnessAddr)
	} else {
		fmt.Printf("Running blueprint %q (dry-run with mock executor)...\n", bp.Name)
	}
	state, err := engine.Execute(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "execution error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %s (%d nodes executed)\n", state.Status, len(state.NodeResults))
	for id, r := range state.NodeResults {
		fmt.Printf("  [%s] %s: %s\n", r.Status, id, r.Output)
	}
}
