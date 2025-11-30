package modhelp

import "fmt"

var docs = map[string]string{
    "command": `NAME
  command - run a command with optional guards

SYNOPSIS
  - name: run
    command: echo "hello"
    args:
      creates: /tmp/ok.flag
      removes: /tmp/old.flag
      env: { KEY: VAL }
      become: true

ARGS
  _         string   command to run (module key value)
  creates   string   path; skip apply if exists
  removes   string   path; skip apply if absent
  env       map      environment variables
  become    bool     sudo

ARTIFACTS
  stdout, stderr, exit, cmd, sudo
`,
    "shell": `NAME
  shell - run a shell command through bash

SYNOPSIS
  - name: pipeline
    shell: |
      echo "pipeline" | tr a-z A-Z
    args:
      become: true

ARGS
  _         string   shell command
  env       map      environment
  become    bool     sudo

ARTIFACTS
  stdout, stderr, exit, cmd, sudo
`,
    "file": `NAME
  file - ensure a file present or absent with content and permissions

SYNOPSIS
  - name: create file
    file: { path: /etc/app, file_name: app.cfg, state: present, content: "key=val", mode: "0644" }
  - name: remove file
    file: { path: /etc/app, file_name: app.cfg, state: absent }

ARGS
  path       string   required directory path
  file_name  string   required file name
  state      string   present|absent
  content    string   required when state=present
  mode       string   optional octal permissions

ARTIFACTS
  path, file_name, dest, before, after, mode
`,
    "copy": `NAME
  copy - copy local content to remote path

SYNOPSIS
  - name: copy cfg
    copy: { src: files/app.cfg, dest: /etc/app/app.cfg, mode: "0644" }
  - name: copy inline
    copy: { content: "hello", dest: /tmp/hello.txt }

ARGS
  src       string   local file path
  content   string   inline content
  dest      string   required
  mode      string   octal permissions

ARTIFACTS
  dest, before, after
`,
    "template": `NAME
  template - render template with vars and copy to remote

SYNOPSIS
  - name: render
    template: { src: templates/app.tmpl, dest: /etc/app/app.cfg, mode: "0644" }

ARGS
  src       string   local template path
  dest      string   required
  mode      string   octal permissions

ARTIFACTS
  dest, before, after, mode
`,
    "lineinfile": `NAME
  lineinfile - ensure a line in a text file

SYNOPSIS
  - name: set sysctl
    lineinfile:
      path: /etc/sysctl.conf
      regexp: '^net\\.ipv4\\.ip_forward='
      line: 'net.ipv4.ip_forward=1'
      state: present

ARGS
  path      string   required
  line      string   required
  regexp    string   optional
  state     string   present|absent

ARTIFACTS
  path, stdout, stderr, exit
`,
    "get_url": `NAME
  get_url - download a URL to a file

SYNOPSIS
  - name: download
    get_url: { url: https://example.com/app.tar.gz, dest: /tmp/app.tar.gz }

ARGS
  url       string   required
  dest      string   required
  checksum  string   optional sha256
  become    bool     sudo

ARTIFACTS
  url, dest, exit, stderr
`,
    "unarchive": `NAME
  unarchive - extract an archive

SYNOPSIS
  - name: extract
    unarchive: { src: /tmp/app.tar.gz, dest: /opt/app }

ARGS
  src       string   required (.tar.gz|.tgz|.zip)
  dest      string   required
  become    bool     sudo

ARTIFACTS
  src, dest, exit, stderr
`,
    "git": `NAME
  git - clone or update a git repository

SYNOPSIS
  - name: checkout
    git: { repo: https://github.com/org/app.git, dest: /opt/app, version: main }

ARGS
  repo      string   required
  dest      string   required
  version   string   optional branch/tag
  become    bool     sudo

ARTIFACTS
  repo, dest, version, exit, stderr
`,
    "pip": `NAME
  pip - manage Python packages

SYNOPSIS
  - name: install requests
    pip: { name: requests, state: present }
  - name: venv install
    pip: { name: flask, state: present, virtualenv: /opt/venv }

ARGS
  name      string   required
  state     string   present|absent
  virtualenv string  optional venv path
  become    bool     sudo

ARTIFACTS
  name, state, venv, exit, stderr
`,
    "package": `NAME
  package - install or remove packages via auto-detected manager

SYNOPSIS
  - name: install tmux
    package: { name: tmux, action: install }

ARGS
  name      string   required
  action    string   install|remove
  become    bool     sudo

ARTIFACTS
  name, manager, cmd, exit
`,
    "service": `NAME
  service - manage services via systemd

SYNOPSIS
  - name: start sshd
    service: { name: sshd, state: started }

ARGS
  name      string   required
  state     string   started|stopped|restarted

ARTIFACTS
  name, state, active, cmd
`,
    "cron": `NAME
  cron - manage cron entries

SYNOPSIS
  - name: nightly
    cron:
      name: nightly_backup
      user: root
      minute: "0"
      hour: "2"
      job: "/usr/local/bin/backup"
      state: present

ARGS
  name      string   required
  user      string   required
  job       string   required
  minute    string   default '*'
  hour      string   default '*'
  day       string   default '*'
  month     string   default '*'
  weekday   string   default '*'
  state     string   present|absent

ARTIFACTS
  user, name, exit, stderr
`,
}

func Get(name string) (string, bool) { s, ok := docs[name]; return s, ok }
func List() []string {
    out := make([]string, 0, len(docs))
    for k := range docs { out = append(out, k) }
    return out
}
func FormatNotFound(name string) string { return fmt.Sprintf("no manual entry for %s", name) }
