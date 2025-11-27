package main

import (
    "context"
    "flag"
    "fmt"
    "io"
    "os"

    "gopsi/pkg/inventory"
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
        if len(os.Args) == 2 { printUsage(); os.Exit(0) }
        subject := os.Args[2]
        switch subject {
        case "run": usageRun()
        case "inventory": usageInventory()
        case "vault": usageVault()
        case "version": usageVersion()
        default: printUsage()
        }
        os.Exit(0)
    case "version":
        fmt.Printf("Gopsi %s\n", version.Version)
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
        _ = runFlags.Parse(os.Args[2:])
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
        r := runner.NewWithOptions(*forks, *check, *jsonOut)
        hosts := inv.AllHosts(*limit)
        ctx := context.Background()
        if err := r.Run(ctx, hosts, pb); err != nil {
            fmt.Fprintln(os.Stderr, err)
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
        if p == "" { p = os.Getenv("AT_VAULT_PASSWORD") }
        if p == "" { fmt.Fprintln(os.Stderr, "missing passphrase"); os.Exit(2) }
        var data []byte
        if *in == "-" {
            b, _ := io.ReadAll(os.Stdin)
            data = b
        } else {
            b, err := os.ReadFile(*in)
            if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
            data = b
        }
        var outb []byte
        switch *mode {
        case "encrypt":
            b, err := vault.Encrypt(data, []byte(p))
            if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
            outb = b
        case "decrypt":
            b, err := vault.Decrypt(data, []byte(p))
            if err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
            outb = b
        default:
            fmt.Fprintln(os.Stderr, "unknown mode")
            os.Exit(2)
        }
        if *out == "-" {
            os.Stdout.Write(outb)
        } else {
            if err := os.WriteFile(*out, outb, 0600); err != nil { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
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
    fmt.Println("  help        Show detailed help for a command")
    fmt.Println("Examples:")
    fmt.Println("  gopsi run -i inventory.yml play.yml --check")
    fmt.Println("  gopsi run -i inventory.yml play.yml --forks 10 --json")
    fmt.Println("  gopsi inventory --list -i inventory.yml")
    fmt.Println("  gopsi vault --mode encrypt --in vars.yml --out vars.enc --pass '...' ")
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
