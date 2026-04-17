package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/aditya-soni/forge/blueprints"
	"github.com/aditya-soni/forge/core/blueprint"
	"github.com/aditya-soni/forge/factory/delivery"
	"github.com/aditya-soni/forge/factory/orchestrator"
	"github.com/aditya-soni/forge/factory/sandbox"
	"github.com/aditya-soni/forge/factory/workspace"
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
	case "run":
		cmdForgeRun(os.Args[2:])
	case "blueprint":
		handleBlueprint(os.Args[2:])
	case "plugin":
		cmdPlugin(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: forge <command>")
	fmt.Println("Commands:")
	fmt.Println("  run \"task\" [flags]         Run a task in a Docker sandbox")
	fmt.Println("  blueprint validate <file>  Validate a blueprint YAML file")
	fmt.Println("  blueprint list             List built-in blueprints")
	fmt.Println("  blueprint run [--harness <addr>] [--builtin <name> | <file>] [--task <text>]")
	fmt.Println("  plugin install [--ide auto|cursor|claude-code|windsurf]  Install MCP plugin for IDE")
	fmt.Println("  plugin status                                            Check plugin installation")
}

func cmdForgeRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	blueprintName := fs.String("blueprint", "standard-implementation", "Built-in blueprint name")
	blueprintFile := fs.String("blueprint-file", "", "Path to blueprint YAML file")
	noSandbox := fs.Bool("no-sandbox", false, "Run without Docker sandbox")
	noPR := fs.Bool("no-pr", false, "Skip PR creation")
	adapter := fs.String("adapter", "echo", "Agent adapter (echo, claude)")
	image := fs.String("image", "forge:latest", "Docker image for sandbox")
	baseBranch := fs.String("base-branch", "main", "Base branch for PR")
	harnessAddr := fs.String("harness", "", "Harness gRPC address for local runs")
	fs.Parse(args)

	task := strings.Join(fs.Args(), " ")
	if task == "" {
		fmt.Fprintln(os.Stderr, "usage: forge run [flags] \"task description\"")
		os.Exit(1)
	}

	if *blueprintFile != "" {
		if *blueprintName != "standard-implementation" {
			fmt.Fprintln(os.Stderr, "use either --blueprint or --blueprint-file, not both")
			os.Exit(1)
		}
		*blueprintName = ""
	}

	if *noSandbox {
		if *adapter == "claude" && *harnessAddr == "" {
			fmt.Fprintln(os.Stderr, "--harness is required for --no-sandbox with --adapter claude")
			os.Exit(1)
		}
		builtin := ""
		if *blueprintFile == "" {
			builtin = *blueprintName
		}
		cmdRun(*blueprintFile, builtin, task, *harnessAddr)
		return
	}

	cwd, _ := os.Getwd()
	runner := &sandbox.ExecRunner{}
	pipeline := orchestrator.NewPipeline(
		sandbox.NewDockerSandbox(runner),
		workspace.NewManager(),
		delivery.NewGitDelivery(runner),
	)

	req := orchestrator.RunRequest{
		Task:          task,
		BlueprintName: *blueprintName,
		BlueprintFile: *blueprintFile,
		RepoDir:       cwd,
		Adapter:       *adapter,
		Image:         *image,
		NoPR:          *noPR,
		BaseBranch:    *baseBranch,
	}

	source := *blueprintName
	if *blueprintFile != "" {
		source = *blueprintFile
	}
	fmt.Printf("Forge run: %q (blueprint=%s, image=%s)\n", task, source, *image)
	result, err := pipeline.Execute(context.Background(), req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipeline error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s (%.1fs)\n", result.Status, result.Duration.Seconds())
	if result.PRURL != "" {
		fmt.Printf("PR: %s\n", result.PRURL)
	}
	if result.Status == orchestrator.RunStatusFailed {
		fmt.Fprintf(os.Stderr, "Error: %s\n", result.Error)
		os.Exit(1)
	}
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
			fmt.Fprintln(os.Stderr, "usage: forge blueprint run [--harness <addr>] [--builtin <name> | <file>] [--task <text>]")
			os.Exit(1)
		}
		harnessAddr, file, builtin, task, err := parseBlueprintRunArgs(args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "blueprint run: %v\n", err)
			os.Exit(1)
		}
		cmdRun(file, builtin, task, harnessAddr)
	default:
		fmt.Fprintf(os.Stderr, "unknown blueprint subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func parseBlueprintRunArgs(args []string) (harnessAddr, file, builtin, task string, err error) {
	var files []string
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--harness":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--harness requires an address")
			}
			i++
			harnessAddr = args[i]
		case args[i] == "--builtin":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--builtin requires a name")
			}
			i++
			builtin = args[i]
		case args[i] == "--task":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("--task requires a value")
			}
			i++
			task = args[i]
		case strings.HasPrefix(args[i], "-"):
			return "", "", "", "", fmt.Errorf("unknown flag: %s", args[i])
		default:
			files = append(files, args[i])
		}
	}
	if builtin != "" && len(files) > 0 {
		return "", "", "", "", fmt.Errorf("--builtin and positional file are mutually exclusive")
	}
	if builtin == "" && len(files) != 1 {
		return "", "", "", "", fmt.Errorf("expected exactly one blueprint file, got %d", len(files))
	}
	if len(files) == 1 {
		file = files[0]
	}
	return harnessAddr, file, builtin, task, nil
}

func resolveBlueprintData(file, builtin, task string) ([]byte, string, error) {
	var data []byte
	var label string
	var err error

	switch {
	case file != "":
		data, err = os.ReadFile(file)
		label = file
	case builtin != "":
		data, err = blueprints.BuiltIn.ReadFile(builtin + ".yaml")
		label = builtin
	default:
		return nil, "", fmt.Errorf("no blueprint source specified")
	}
	if err != nil {
		return nil, "", err
	}

	if task != "" {
		data = []byte(strings.ReplaceAll(string(data), "{{task}}", task))
	}
	return data, label, nil
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

func cmdRun(file, builtin, task, harnessAddr string) {
	data, label, err := resolveBlueprintData(file, builtin, task)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
	bp, err := blueprint.ParseBlueprintYAML(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse error (%s): %v\n", label, err)
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
		fmt.Fprintf(os.Stderr, "build error (%s): %v\n", label, err)
		os.Exit(1)
	}
	engine := blueprint.NewEngine(g, bp.Name)
	if harnessAddr != "" {
		fmt.Printf("Running blueprint %q (source %q, harness at %s)...\n", bp.Name, label, harnessAddr)
	} else {
		fmt.Printf("Running blueprint %q (source %q, dry-run with mock executor)...\n", bp.Name, label)
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
