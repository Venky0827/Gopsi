NAME
  nutanix_vm - manage Nutanix VM operations via Prism API

SYNOPSIS
  - name: power on
    nutanix_vm:
      endpoint: https://prism.example.com:9440
      token: $PRISM_TOKEN
      name: app-01
      action: power_on

ARGS
  endpoint  string   Prism endpoint (https)
  token     string   API token (prefer vault/env)
  username  string   optional if not using token
  password  string   optional if not using token
  name      string   VM name
  action    string   create|power_on|power_off|delete

ARTIFACTS
  vm_name, action, request_id

