package main

import (
    "context"
    "fmt"

    "gopsi/pkg/module"
)

type vmMod struct{}

func (vmMod) Name() string { return "nutanix_vm" }
func (vmMod) Namespace() string { return "nutanix" }
func (vmMod) Validate(args map[string]any) error {
    if str(args["endpoint"]) == "" || str(args["name"]) == "" || str(args["action"]) == "" { return fmt.Errorf("nutanix_vm requires endpoint, name, action") }
    return nil
}
func (vmMod) Check(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    action := str(args["action"]) 
    // TODO: call Prism API to check VM state; for now, simulate change
    changed := action != "power_on" // placeholder
    return module.Result{Changed: changed, Artifacts: map[string]any{"vm_name": name, "action": action}}, nil
}
func (vmMod) Apply(ctx context.Context, c module.Conn, args map[string]any) (module.Result, error) {
    name := str(args["name"]) 
    action := str(args["action"]) 
    // TODO: call Prism API to perform operation; simulate success
    return module.Result{Changed: true, Artifacts: map[string]any{"vm_name": name, "action": action, "request_id": "simulated"}}, nil
}

func Register() { module.Register(vmMod{}) }
func str(v any) string { if v == nil { return "" }; return fmt.Sprintf("%v", v) }
