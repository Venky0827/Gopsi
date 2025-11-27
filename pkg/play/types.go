package play

type Playbook struct {
    Plays []Play `yaml:"-"`
    SchemaVersion int
}

type Play struct {
    Hosts   string                 `yaml:"hosts"`
    Become  bool                   `yaml:"become"`
    Serial  int                    `yaml:"serial"`
    Vars    map[string]any         `yaml:"vars"`
    Tasks   []Task                  `yaml:"tasks"`
    Handlers []Task                `yaml:"handlers"`
}

type Task struct {
    Name    string                 `yaml:"name"`
    Module  string                 `yaml:"-"`
    Args    map[string]any         `yaml:"-"`
    Raw     map[string]any         `yaml:",inline"`
    Tags    []string               `yaml:"tags"`
    When    string                 `yaml:"when"`
    Notify  []string               `yaml:"notify"`
    Register string                `yaml:"register"`
}
