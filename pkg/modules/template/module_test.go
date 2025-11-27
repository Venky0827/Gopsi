package template

import (
    "bytes"
    "context"
    "io"
    "os"
    "testing"
)

type fakeConn struct{}

func (fakeConn) Exec(ctx context.Context, cmd string, env map[string]string, sudo bool) (string, string, int, error) { return "", "", 0, nil }
func (fakeConn) Put(ctx context.Context, src io.Reader, dst string, mode os.FileMode) error { return nil }
func (fakeConn) Get(ctx context.Context, src string) (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(nil)), nil }

func TestTemplateCheckNoRemote(t *testing.T) {
    f, err := os.CreateTemp(t.TempDir(), "tmpl-*.tmpl")
    if err != nil { t.Fatal(err) }
    if _, err := f.WriteString("Hello"); err != nil { t.Fatal(err) }
    _ = f.Close()
    m := mod{}
    args := map[string]any{"src": f.Name(), "dest": "/tmp/x"}
    _, err = m.Check(context.Background(), fakeConn{}, args)
    if err != nil { t.Fatal(err) }
}
