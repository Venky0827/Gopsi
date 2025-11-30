package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"sort"
	"strings"
	"time"

	"gopsi/pkg/inventory"
	"gopsi/pkg/modhelp"
	"gopsi/pkg/module"
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

var defaultModules []string
var defaultSet = map[string]struct{}{}

type namespaced interface{ Namespace() string }

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		printUsage()
		os.Exit(0)
	}
	defaultModules = module.List()
	for _, n := range defaultModules {
		defaultSet[n] = struct{}{}
	}
	loadPlugins()
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
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: gopsi module <name|list|install|remove> [help]")
			os.Exit(2)
		}
		sub := os.Args[2]
		if sub == "list" {
			names := module.List()
			for _, n := range names {
				fmt.Println(n)
			}
			os.Exit(0)
		} else if sub == "install" {
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: gopsi module install <name>")
				os.Exit(2)
			}
			name := os.Args[3]
			switch name {
			case "nutanix":
				if err := installNutanix(); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Println("installed: nutanix")
			default:
				fmt.Fprintln(os.Stderr, "unknown module pack")
				os.Exit(2)
			}
			os.Exit(0)
		} else if sub == "remove" {
			if len(os.Args) < 4 {
				fmt.Fprintln(os.Stderr, "usage: gopsi module remove <name>")
				os.Exit(2)
			}
			name := os.Args[3]
			p := filepath.Join(gopsiHome(), "plugins", name+".so")
			_ = os.Remove(p)
			hp := filepath.Join(gopsiHome(), "plugins", "help", name+".md")
			_ = os.Remove(hp)
			fmt.Println("removed:", name)
			os.Exit(0)
		} else if len(os.Args) >= 4 && os.Args[3] == "help" {
			name := os.Args[2]
			if doc, ok := modhelp.Get(name); ok {
				fmt.Println(doc)
				os.Exit(0)
			}
			hp := filepath.Join(gopsiHome(), "plugins", "help", name+".md")
			b, err := os.ReadFile(hp)
			if err != nil {
				fmt.Println("no manual entry for", name)
				os.Exit(1)
			}
			os.Stdout.Write(b)
			os.Exit(0)
		} else {
			fmt.Fprintln(os.Stderr, "usage: gopsi module <list|install|remove|<name> help>")
			os.Exit(2)
		}
		// removed duplicate case
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
		names := module.List()
		groups := map[string][]string{"default": {}}
		for _, n := range defaultModules {
			groups["default"] = append(groups["default"], n)
		}
		for _, n := range names {
			if _, ok := defaultSet[n]; ok {
				continue
			}
			ns := "custom"
			if m := module.Get(n); m != nil {
				if nm, ok := any(m).(namespaced); ok {
					if s := nm.Namespace(); s != "" {
						ns = s
					}
				}
			}
			groups[ns] = append(groups[ns], n)
		}
		var others []string
		for k := range groups {
			if k != "default" {
				others = append(others, k)
			}
		}
		sort.Strings(others)
		fmt.Println(colorViolet("default") + ":")
		ds := groups["default"]
		sort.Strings(ds)
		for _, n := range ds {
			fmt.Printf("  - %s: %s\n", colorLightYellow(n), colorLightBlue(shortDesc(n)))
		}
		for _, k := range others {
			items := groups[k]
			sort.Strings(items)
			fmt.Println(colorViolet(k) + ":")
			for _, n := range items {
				fmt.Printf("  - %s: %s\n", colorLightYellow(n), colorLightBlue(shortDesc(n)))
			}
		}
		if len(names) == 0 {
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(2)
	}
}

func colorBold(s string) string        { return "\033[1m" + s + "\033[0m" }
func colorViolet(s string) string      { return "\033[95;1m" + s + "\033[0m" }
func colorLightYellow(s string) string { return "\033[93m" + s + "\033[0m" }
func colorLightBlue(s string) string   { return "\033[94m" + s + "\033[0m" }
func colorLightGreen(s string) string  { return "\033[92m" + s + "\033[0m" }
func pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	b := make([]byte, n)
	copy(b, s)
	for i := len(s); i < n; i++ {
		b[i] = ' '
	}
	return string(b)
}
func printGrid(name string, items []string) {
	col := 18
	cols := 2
	inner := col*cols + (cols-1)*2
	top := "┌" + strings.Repeat("─", inner) + "┐"
	sep := "├" + strings.Repeat("─", inner) + "┤"
	bot := "└" + strings.Repeat("─", inner) + "┘"
	fmt.Println(colorBold(name))
	fmt.Println(top)
	line := fmt.Sprintf("│ %s%*s │", pad(fmt.Sprintf("count: %d", len(items)), inner-2), 0, "")
	fmt.Println(line)
	fmt.Println(sep)
	if len(items) == 0 {
		fmt.Println("│ " + pad("(none)", inner-2) + " │")
		fmt.Println(bot)
		return
	}
	rows := (len(items) + cols - 1) / cols
	for i := 0; i < rows; i++ {
		left := items[i]
		r := i + rows
		right := ""
		if r < len(items) {
			right = items[r]
		}
		l := pad(left, col)
		rr := pad(right, col)
		fmt.Println("│ " + l + "  " + rr + " │")
	}
	fmt.Println(bot)
}

func shortDesc(name string) string {
	if doc, ok := modhelp.Get(name); ok {
		lines := strings.Split(doc, "\n")
		for i := 0; i+1 < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "NAME" {
				line := strings.TrimSpace(lines[i+1])
				// format: "<name> - <desc>"
				parts := strings.SplitN(line, "-", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
				return line
			}
		}
	}
	hp := filepath.Join(gopsiHome(), "plugins", "help", name+".md")
	if b, err := os.ReadFile(hp); err == nil {
		lines := strings.Split(string(b), "\n")
		for i := 0; i+1 < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "NAME" {
				line := strings.TrimSpace(lines[i+1])
				parts := strings.SplitN(line, "-", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1])
				}
				return line
			}
		}
	}
	return "No description"
}

func printUsage() {
	fmt.Println(colorViolet("Usage:") + " " + colorLightYellow("gopsi <command> [flags]"))
	fmt.Println(colorViolet("Commands:"))
	fmt.Println("  " + colorLightYellow("run") + "         " + colorLightBlue("Execute playbook(s) against hosts"))
	fmt.Println("  " + colorLightYellow("inventory") + "   " + colorLightBlue("Inspect or list inventory hosts"))
	fmt.Println("  " + colorLightYellow("vault") + "       " + colorLightBlue("Encrypt/decrypt variable files"))
	fmt.Println("  " + colorLightYellow("version") + "     " + colorLightBlue("Show build version info"))
	fmt.Println("  " + colorLightYellow("ping") + "        " + colorLightBlue("Check TCP reachability for inventory hosts"))
	fmt.Println("  " + colorLightYellow("modules") + "     " + colorLightBlue("List registered modules"))
	fmt.Println("  " + colorLightYellow("completion") + "  " + colorLightBlue("Output shell completion script (bash|zsh)"))
	fmt.Println("  " + colorLightYellow("help") + "        " + colorLightBlue("Show detailed help for a command"))
	fmt.Println(colorViolet("Flags by command:"))
	fmt.Println("  " + colorLightYellow("run") + ": " + colorLightBlue("-i, --limit, --forks, --check, --json, -v, -vv, -vvv"))
	fmt.Println("  " + colorLightYellow("inventory") + ": " + colorLightBlue("--list, -i"))
	fmt.Println("  " + colorLightYellow("vault") + ": " + colorLightBlue("--mode, --in, --out, --pass"))
	fmt.Println("  " + colorLightYellow("ping") + ": " + colorLightBlue("-i, --limit, --port, --timeout"))
	fmt.Println("  " + colorLightYellow("modules") + ": " + colorLightBlue("(no flags)"))
	fmt.Println("  " + colorLightYellow("completion") + ": " + colorLightBlue("bash|zsh"))
	fmt.Println("  " + colorLightYellow("help") + ": " + colorLightBlue("help <run|inventory|vault|version|ping|modules>"))
	fmt.Println(colorViolet("Examples:"))
	fmt.Println("  " + colorLightGreen("Dry-run; shows predicted changes without applying"))
	fmt.Println("  " + colorLightYellow("gopsi run -i inventory.yml play.yml --check"))
	fmt.Println("  " + colorLightGreen("Increase parallelism and print per-task JSON"))
	fmt.Println("  " + colorLightYellow("gopsi run -i inventory.yml play.yml --forks 10 --json"))
	fmt.Println("  " + colorLightGreen("List resolved hosts from inventory"))
	fmt.Println("  " + colorLightYellow("gopsi inventory --list -i inventory.yml"))
	fmt.Println("  " + colorLightGreen("Encrypt a vars file using a passphrase"))
	fmt.Println("  " + colorLightYellow("gopsi vault --mode encrypt --in vars.yml --out vars.enc --pass '...' "))
	fmt.Println("  " + colorLightGreen("Check reachability for a group with a custom timeout"))
	fmt.Println("  " + colorLightYellow("gopsi ping -i inventory.yml --limit web --timeout 5"))
}

func usageRun() {
	fmt.Println(colorViolet("Usage:") + " " + colorLightYellow("gopsi run [flags] <playbook>"))
	fmt.Println(colorViolet("Description:"))
	fmt.Println("  " + colorLightBlue("Executes YAML playbook tasks across selected hosts using SSH."))
	fmt.Println(colorViolet("Flags:"))
	fmt.Println("  " + colorLightYellow("-i string") + "  " + colorLightGreen("Inventory file path (default 'inventory.yml')"))
	fmt.Println("  " + colorLightYellow("--limit string") + "  " + colorLightGreen("Limit execution to a host or group name"))
	fmt.Println("  " + colorLightYellow("--forks int") + "  " + colorLightGreen("Number of parallel workers (default 5)"))
	fmt.Println("  " + colorLightYellow("--check") + "  " + colorLightGreen("Dry-run; predict changes without applying"))
	fmt.Println("  " + colorLightYellow("--json") + "  " + colorLightGreen("Print per-task results as JSON lines"))
	fmt.Println("  " + colorLightYellow("-v") + ", " + colorLightYellow("-vv") + ", " + colorLightYellow("-vvv") + "  " + colorLightGreen("Increase diagnostics verbosity (1/2/3)"))
	fmt.Println(colorViolet("Ordering:"))
	fmt.Println("  " + colorLightBlue("Flags can appear anywhere; they are normalized before parsing."))
	fmt.Println(colorViolet("Notes:"))
	fmt.Println("  " + colorLightBlue("Concurrency per play can be controlled via 'serial' in the playbook."))
	fmt.Println("  " + colorLightBlue("Facts are gathered automatically and available as 'facts' in templates/when."))
}

func usageInventory() {
	fmt.Println(colorViolet("Usage:") + " " + colorLightYellow("gopsi inventory --list -i <inventory>"))
	fmt.Println(colorViolet("Description:"))
	fmt.Println("  " + colorLightBlue("Lists resolved hostnames from the inventory."))
	fmt.Println(colorViolet("Flags:"))
	fmt.Println("  " + colorLightYellow("--list") + "  " + colorLightGreen("List all hosts in the inventory"))
	fmt.Println("  " + colorLightYellow("-i string") + "  " + colorLightGreen("Path to inventory YAML file (default 'inventory.yml')"))
	fmt.Println(colorViolet("Inventory keys:"))
	fmt.Println("  " + colorLightYellow("host") + "  " + colorLightBlue("IP/DNS of the host"))
	fmt.Println("  " + colorLightYellow("user") + "  " + colorLightBlue("SSH username"))
	fmt.Println("  " + colorLightYellow("ssh_private_key_file") + "  " + colorLightBlue("Path to SSH private key"))
}

func usageVault() {
	fmt.Println(colorViolet("Usage:") + " " + colorLightYellow("gopsi vault --mode <encrypt|decrypt> --in <file|-> --out <file|-> [--pass <str>]"))
	fmt.Println(colorViolet("Description:"))
	fmt.Println("  " + colorLightBlue("Encrypts or decrypts YAML variable files using AES-GCM."))
	fmt.Println(colorViolet("Flags:"))
	fmt.Println("  " + colorLightYellow("--mode string") + "  " + colorLightGreen("Operation: encrypt or decrypt (default 'encrypt')"))
	fmt.Println("  " + colorLightYellow("--in string") + "  " + colorLightGreen("Input file path or '-' for stdin (default '-')"))
	fmt.Println("  " + colorLightYellow("--out string") + "  " + colorLightGreen("Output file path or '-' for stdout (default '-')"))
	fmt.Println("  " + colorLightYellow("--pass string") + "  " + colorLightGreen("Passphrase; if omitted, uses AT_VAULT_PASSWORD env var"))
	fmt.Println(colorViolet("Environment:"))
	fmt.Println("  " + colorLightYellow("AT_VAULT_PASSWORD") + "  " + colorLightBlue("Optional passphrase source for non-interactive use"))
}

func usageVersion() {
	fmt.Println("Usage: gopsi version")
	fmt.Println("Description:")
	fmt.Println("  Shows tool version and build metadata.")
}

func usagePing() {
	fmt.Println(colorViolet("Usage:") + " " + colorLightYellow("gopsi ping [flags]"))
	fmt.Println(colorViolet("Description:"))
	fmt.Println("  " + colorLightBlue("Checks TCP port reachability for hosts defined in the inventory."))
	fmt.Println(colorViolet("Flags:"))
	fmt.Println("  " + colorLightYellow("-i string") + "  " + colorLightGreen("Path to inventory file (default 'inventory.yml')"))
	fmt.Println("  " + colorLightYellow("--limit string") + "  " + colorLightGreen("Limit execution to a host or group name"))
	fmt.Println("  " + colorLightYellow("--port int") + "  " + colorLightGreen("TCP port to check (default 22)"))
	fmt.Println("  " + colorLightYellow("--timeout int") + "  " + colorLightGreen("Connection timeout in seconds (default 5)"))
}

func usageModules() {
	fmt.Println("Usage: gopsi modules")
	fmt.Println("Description:")
	fmt.Println("  Lists module names registered via init() side-effects.")
}

func gopsiHome() string {
	h := os.Getenv("GOPSI_HOME")
	if h == "" {
		h = filepath.Join(os.Getenv("HOME"), ".gopsi")
	}
	_ = os.MkdirAll(filepath.Join(h, "plugins"), 0755)
	_ = os.MkdirAll(filepath.Join(h, "plugins", "help"), 0755)
	return h
}

func loadPlugins() {
	dir := filepath.Join(gopsiHome(), "plugins")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".so") {
			continue
		}
		p := filepath.Join(dir, e.Name())
		pl, err := plugin.Open(p)
		if err != nil {
			continue
		}
		sym, err := pl.Lookup("Register")
		if err != nil {
			continue
		}
		if f, ok := sym.(func()); ok {
			f()
		}
	}
}

func installNutanix() error {
	out := filepath.Join(gopsiHome(), "plugins", "nutanix.so")
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", out, "./plugins/nutanix")
	cmd.Env = os.Environ()
	cmd.Dir = filepath.Dir(os.Args[0])
	if err := cmd.Run(); err != nil {
		return err
	}
	src := filepath.Join(filepath.Dir(os.Args[0]), "plugins", "nutanix", "help", "nutanix_vm.md")
	dst := filepath.Join(gopsiHome(), "plugins", "help", "nutanix_vm.md")
	b, err := os.ReadFile(src)
	if err == nil {
		_ = os.WriteFile(dst, b, 0644)
	}
	return nil
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
