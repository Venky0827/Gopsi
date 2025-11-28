package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopsi/pkg/inventory"
	"gopsi/pkg/module"
	"gopsi/pkg/modhelp"
	_ "gopsi/pkg/modules/command"
	_ "gopsi/pkg/modules/copy"
	_ "gopsi/pkg/modules/cron"
	_ "gopsi/pkg/modules/file"
	_ "gopsi/pkg/modules/get_url"
	_ "gopsi/pkg/modules/git"
	_ "gopsi/pkg/modules/lineinfile"
	_ "gopsi/pkg/modules/package"
	_ "gopsi/pkg/modules/pip"
	_ "gopsi/pkg/modules/service"
	_ "gopsi/pkg/modules/shell"
	_ "gopsi/pkg/modules/template"
	_ "gopsi/pkg/modules/unarchive"
	"gopsi/pkg/play"
	"gopsi/pkg/runner"
	"gopsi/pkg/vault"
	"gopsi/pkg/version"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}
	cmd := os.Args[1]
	switch cmd {
	case "help":
		if len(os.Args) == 2 {
			printUsage()
			os.Exit(0)
		}
		subject := os.Args[2]
		switch subject {
		case "run":
			usageRun()
		case "inventory":
			usageInventory()
		case "vault":
			usageVault()
		case "version":
			usageVersion()
		case "ping":
			usagePing()
		case "modules":
			usageModules()
		default:
			printUsage()
		}
		os.Exit(0)
	case "version":
		fmt.Printf("Gopsi %s\n", version.Version)
	case "module":
		if len(os.Args) < 4 { fmt.Fprintln(os.Stderr, "usage: gopsi module <name> help"); os.Exit(2) }
		name := os.Args[2]
		action := os.Args[3]
		if action != "help" { fmt.Fprintln(os.Stderr, "supported action: help"); os.Exit(2) }
		if doc, ok := modhelp.Get(name); ok { fmt.Println(doc) } else { fmt.Println(modhelp.FormatNotFound(name)); os.Exit(1) }
	case "completion":
		if len(os.Args) < 3 {
			usageCompletion()
			os.Exit(2)
		}
		shell := os.Args[2]
		switch shell {
		case "bash":
			printBashCompletion()
		case "zsh":
			printZshCompletion()
		default:
			fmt.Fprintln(os.Stderr, "supported shells: bash, zsh")
			os.Exit(2)
		}
	case "inventory":
		invFile := flag.NewFlagSet("inventory", flag.ExitOnError)
		invFile.Usage = usageInventory
		list := invFile.Bool("list", false, "list hosts")
		file := invFile.String("i", "inventory.yml", "inventory file")
		_ = invFile.Parse(os.Args[2:])
		inv, err := inventory.LoadFromFile(*file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if *list {
			for _, h := range inv.AllHosts("") {
				fmt.Println(h.Name)
			}
		}
	case "run":
		runFlags := flag.NewFlagSet("run", flag.ExitOnError)
		runFlags.Usage = usageRun
		invPath := runFlags.String("i", "inventory.yml", "inventory file")
		limit := runFlags.String("limit", "", "limit hosts/group")
		forks := runFlags.Int("forks", 5, "parallel forks")
		check := runFlags.Bool("check", false, "check mode")
		jsonOut := runFlags.Bool("json", false, "json output")
		v := runFlags.Bool("v", false, "increase verbosity")
		vv := runFlags.Bool("vv", false, "increase verbosity more")
		vvv := runFlags.Bool("vvv", false, "maximum verbosity")
		argv := os.Args[2:]
		var fl, ar []string
		for i := 0; i < len(argv); i++ {
			tok := argv[i]
			if strings.HasPrefix(tok, "-") {
				fl = append(fl, tok)
				if !strings.Contains(tok, "=") && i+1 < len(argv) && !strings.HasPrefix(argv[i+1], "-") {
					fl = append(fl, argv[i+1])
					i++
				}
			} else {
				ar = append(ar, tok)
			}
		}
		reordered := append(fl, ar...)
		_ = runFlags.Parse(reordered)
		args := runFlags.Args()
		if len(args) < 1 {
			fmt.Fprintln(os.Stderr, "run requires playbook path")
			os.Exit(2)
		}
		playPath := args[0]
		inv, err := inventory.LoadFromFile(*invPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		pb, err := play.LoadPlaybook(playPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		verbosity := 0
		if *v {
			verbosity = 1
		}
		if *vv && verbosity < 2 {
			verbosity = 2
		}
		if *vvv && verbosity < 3 {
			verbosity = 3
		}
		r := runner.NewWithOptions(*forks, *check, *jsonOut, verbosity)
		hosts := inv.AllHosts(*limit)
		ctx := context.Background()
		if err := r.Run(ctx, hosts, pb); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "ping":
		pf := flag.NewFlagSet("ping", flag.ExitOnError)
		pf.Usage = usagePing
		invPath := pf.String("i", "inventory.yml", "inventory file")
		limit := pf.String("limit", "", "limit hosts/group")
		port := pf.Int("port", 22, "TCP port to check")
		timeoutSec := pf.Int("timeout", 5, "ssh timeout seconds")
		_ = pf.Parse(os.Args[2:])
		inv, err := inventory.LoadFromFile(*invPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		hosts := inv.AllHosts(*limit)
		if len(hosts) == 0 {
			fmt.Fprintln(os.Stderr, "no hosts to ping")
			os.Exit(2)
		}
		timeout := time.Duration(*timeoutSec) * time.Second
		unreachable := 0
		for _, h := range hosts {
			addr := h.Addr
			if addr == "" {
				addr = h.Name
			}
			target := net.JoinHostPort(addr, fmt.Sprintf("%d", *port))
			c, derr := net.DialTimeout("tcp", target, timeout)
			if derr != nil {
				fmt.Printf("%s | unreachable | %v\n", h.Name, derr)
				unreachable++
				continue
			}
			_ = c.Close()
			fmt.Printf("%s | reachable\n", h.Name)
		}
		if unreachable > 0 {
			os.Exit(1)
		}
	case "vault":
		vf := flag.NewFlagSet("vault", flag.ExitOnError)
		vf.Usage = usageVault
		mode := vf.String("mode", "encrypt", "encrypt|decrypt")
		in := vf.String("in", "-", "input file or - for stdin")
		out := vf.String("out", "-", "output file or - for stdout")
		pass := vf.String("pass", "", "passphrase (use AT_VAULT_PASSWORD env if empty)")
		_ = vf.Parse(os.Args[2:])
		p := *pass
		if p == "" {
			p = os.Getenv("AT_VAULT_PASSWORD")
		}
		if p == "" {
			fmt.Fprintln(os.Stderr, "missing passphrase")
			os.Exit(2)
		}
		var data []byte
		if *in == "-" {
			b, _ := io.ReadAll(os.Stdin)
			data = b
		} else {
			b, err := os.ReadFile(*in)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			data = b
		}
		var outb []byte
		switch *mode {
		case "encrypt":
			b, err := vault.Encrypt(data, []byte(p))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			outb = b
		case "decrypt":
			b, err := vault.Decrypt(data, []byte(p))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			outb = b
		default:
			fmt.Fprintln(os.Stderr, "unknown mode")
			os.Exit(2)
		}
		if *out == "-" {
			os.Stdout.Write(outb)
		} else {
			if err := os.WriteFile(*out, outb, 0600); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

	case "modules":
		// ensure module packages are imported for init() side-effects
		names := module.List()
		for _, n := range names {
			fmt.Println(n)
		}
		if len(names) == 0 {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(2)
	}
}

func printUsage() {
	fmt.Println("Usage: gopsi <command> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  run         Execute playbook(s) against hosts")
	fmt.Println("  inventory   Inspect or list inventory hosts")
	fmt.Println("  vault       Encrypt/decrypt variable files")
	fmt.Println("  version     Show build version info")
	fmt.Println("  ping        Check TCP reachability for inventory hosts")
	fmt.Println("  modules     List registered modules")
	fmt.Println("  completion  Output shell completion script (bash|zsh)")
	fmt.Println("  help        Show detailed help for a command")
	fmt.Println("Examples:")
	fmt.Println("  gopsi run -i inventory.yml play.yml --check")
	fmt.Println("  gopsi run -i inventory.yml play.yml --forks 10 --json")
	fmt.Println("  gopsi inventory --list -i inventory.yml")
	fmt.Println("  gopsi vault --mode encrypt --in vars.yml --out vars.enc --pass '...' ")
	fmt.Println("  gopsi ping -i inventory.yml --limit web --timeout 5")
}

func usageRun() {
	fmt.Println("Usage: gopsi run [flags] <playbook>")
	fmt.Println("Description:")
	fmt.Println("  Executes YAML playbook tasks across selected hosts using SSH.")
	fmt.Println("Flags:")
	fmt.Println("  -i string       Path to inventory file (default 'inventory.yml').")
	fmt.Println("  -limit string   Limit execution to a host or group name.")
	fmt.Println("                  Example: --limit web or --limit host1")
	fmt.Println("  -forks int      Number of parallel workers (default 5).")
	fmt.Println("  -check          Check mode; shows predicted changes without applying.")
	fmt.Println("  -json           Print per-task results as JSON lines.")
	fmt.Println("  -v              Increase diagnostics verbosity.")
	fmt.Println("  -vv             More verbose diagnostics.")
	fmt.Println("  -vvv            Maximum verbosity.")
	fmt.Println("Ordering:")
	fmt.Println("  Flags can appear anywhere; they are normalized before parsing.")
	fmt.Println("Notes:")
	fmt.Println("  - Concurrency per play can be controlled via 'serial' in the playbook.")
	fmt.Println("  - Facts are gathered automatically and available as 'facts' in templates/when.")
}

func usageInventory() {
	fmt.Println("Usage: gopsi inventory --list -i <inventory>")
	fmt.Println("Description:")
	fmt.Println("  Lists resolved hostnames from the inventory.")
	fmt.Println("Flags:")
	fmt.Println("  --list          List all hosts in the inventory.")
	fmt.Println("  -i string       Path to inventory YAML file (default 'inventory.yml').")
	fmt.Println("Inventory keys:")
	fmt.Println("  host                IP/DNS of the host")
	fmt.Println("  user                SSH username")
	fmt.Println("  ssh_private_key_file Path to SSH private key")
}

func usageVault() {
	fmt.Println("Usage: gopsi vault --mode <encrypt|decrypt> --in <file|-> --out <file|-> [--pass <str>]")
	fmt.Println("Description:")
	fmt.Println("  Encrypts or decrypts YAML variable files using AES-GCM.")
	fmt.Println("Flags:")
	fmt.Println("  --mode string    Operation: encrypt or decrypt (default 'encrypt').")
	fmt.Println("  --in string      Input file path or '-' for stdin (default '-').")
	fmt.Println("  --out string     Output file path or '-' for stdout (default '-').")
	fmt.Println("  --pass string    Passphrase; if omitted, uses AT_VAULT_PASSWORD env var.")
	fmt.Println("Environment:")
	fmt.Println("  AT_VAULT_PASSWORD  Optional passphrase source for non-interactive use.")
}

func usageVersion() {
	fmt.Println("Usage: gopsi version")
	fmt.Println("Description:")
	fmt.Println("  Shows tool version and build metadata.")
}

func usagePing() {
	fmt.Println("Usage: gopsi ping [flags]")
	fmt.Println("Description:")
	fmt.Println("  Checks TCP port reachability for hosts defined in the inventory.")
	fmt.Println("Flags:")
	fmt.Println("  -i string       Path to inventory file (default 'inventory.yml').")
	fmt.Println("  -limit string   Limit execution to a host or group name.")
	fmt.Println("  -port int       TCP port to check (default 22).")
	fmt.Println("  -timeout int    connection timeout in seconds (default 5).")
}

func usageModules() {
	fmt.Println("Usage: gopsi modules")
	fmt.Println("Description:")
	fmt.Println("  Lists module names registered via init() side-effects.")
}

func usageCompletion() {
	fmt.Println("Usage: gopsi completion <bash|zsh>")
	fmt.Println("Description:")
	fmt.Println("  Prints shell completion script to stdout. Source it or install it.")
}

func printBashCompletion() {
	fmt.Println(`# bash completion for gopsi
_gopsi()
{
    local cur prev words cword
    _init_completion || return
    local cmds="run inventory vault version help ping modules completion"
    case ${COMP_WORDS[1]} in
        run)
            COMPREPLY=( $(compgen -W "-i --limit --forks --check --json -v -vv -vvv" -- "$cur") )
            ;;
        inventory)
            COMPREPLY=( $(compgen -W "--list -i" -- "$cur") )
            ;;
        vault)
            COMPREPLY=( $(compgen -W "--mode --in --out --pass" -- "$cur") )
            ;;
        ping)
            COMPREPLY=( $(compgen -W "-i --limit --port --timeout" -- "$cur") )
            ;;
        modules)
            COMPREPLY=()
            ;;
        completion)
            COMPREPLY=( $(compgen -W "bash zsh" -- "$cur") )
            ;;
        *)
            COMPREPLY=( $(compgen -W "$cmds" -- "$cur") )
            ;;
    esac
}
complete -F _gopsi gopsi`)
}

func printZshCompletion() {
	fmt.Println(`# zsh completion for gopsi
_gopsi() {
  local -a cmds
  cmds=(run inventory vault version help ping modules completion)
  local state
  _arguments \
    '1: :->cmd' \
    '*::arg:->args'
  case $state in
    cmd)
      _describe -t commands 'gopsi commands' cmds
      ;;
    args)
      case $words[2] in
        run)
          _arguments '-i[Inventory file]' '--limit[Limit hosts/group]' '--forks[Parallel]' '--check[Check mode]' '--json[JSON output]' '(-v -vv -vvv)-v[Verbose]' '(-v -vv -vvv)-vv[More verbose]' '(-v -vv -vvv)-vvv[Max verbose]'
          ;;
        inventory)
          _arguments '--list[List hosts]' '-i[Inventory file]'
          ;;
        vault)
          _arguments '--mode[encrypt|decrypt]' '--in[input]' '--out[output]' '--pass[passphrase]'
          ;;
        ping)
          _arguments '-i[Inventory file]' '--limit[Limit hosts/group]' '--port[TCP port]' '--timeout[Seconds]'
          ;;
        completion)
          _arguments '1: :(bash zsh)'
          ;;
      esac
      ;;
  esac
}
compdef _gopsi gopsi`)
}

func stringVar(vars map[string]any, key string) string {
	v, ok := vars[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return os.ExpandEnv(s)
}

func expandHome(path string) string {
	if path == "" {
		return path
	}
	if path[0] == '~' {
		return filepath.Join(os.Getenv("HOME"), path[1:])
	}
	return path
}
